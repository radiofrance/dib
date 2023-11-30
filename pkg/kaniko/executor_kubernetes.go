package kaniko

import (
	"context"
	"fmt"
	"io"

	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

// KubernetesExecutor will run Kaniko in a Kubernetes cluster.
type KubernetesExecutor struct {
	clientSet          kubernetes.Interface
	DockerConfigSecret string             // Name of the secret containing the docker config used by Kaniko (required).
	PodConfig          k8sutils.PodConfig // The default pod configuration used to run Kaniko builds.
}

// NewKubernetesExecutor creates a new instance of KubernetesExecutor.
func NewKubernetesExecutor(clientSet kubernetes.Interface, config k8sutils.PodConfig) *KubernetesExecutor {
	return &KubernetesExecutor{
		clientSet:          clientSet,
		DockerConfigSecret: "",
		PodConfig:          config,
	}
}

// Execute the Kaniko build using a Kubernetes Pod.
func (e KubernetesExecutor) Execute(ctx context.Context, output io.Writer, args []string) error {
	logrus.Info("Building image with kaniko kubernetes executor")
	if e.DockerConfigSecret == "" {
		return fmt.Errorf("the DockerConfigSecret option is required")
	}

	podName := e.PodConfig.Name
	if e.PodConfig.NameGenerator != nil {
		podName = e.PodConfig.NameGenerator()
	}
	containerName := "kaniko"

	labels := map[string]string{
		"app.kubernetes.io/name":      "kaniko",
		"app.kubernetes.io/component": "build-pod",
		"app.kubernetes.io/instance":  podName,
	}
	// Merge the default labels with those provided in the options.
	for k, v := range e.PodConfig.Labels {
		labels[k] = v
	}

	objectMeta := metav1.ObjectMeta{
		Name:      podName,
		Namespace: e.PodConfig.Namespace,
		Labels:    labels,
	}

	var imagePullSecrets []corev1.LocalObjectReference
	for _, secretName := range e.PodConfig.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: secretName,
		})
	}

	var envVars []corev1.EnvVar
	for k, v := range e.PodConfig.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	var envFrom []corev1.EnvFromSource
	for _, secretName := range e.PodConfig.EnvSecrets {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
			},
		})
	}

	container := corev1.Container{
		Name:            containerName,
		Image:           e.PodConfig.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            args,
		EnvFrom:         envFrom,
		Env: append([]corev1.EnvVar{
			{
				Name:  "DOCKER_CONFIG",
				Value: "/kaniko/.docker",
			},
		}, envVars...),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      e.DockerConfigSecret,
				MountPath: "/kaniko/.docker",
				ReadOnly:  true,
			},
		},
	}
	err := k8sutils.MergeObjectWithYaml(&container, e.PodConfig.ContainerOverride)
	if err != nil {
		return err
	}

	pod := corev1.Pod{
		ObjectMeta: objectMeta,
		Spec: corev1.PodSpec{
			ImagePullSecrets: imagePullSecrets,
			Containers: []corev1.Container{
				container,
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"app.kubernetes.io/name":      "kaniko",
										"app.kubernetes.io/component": "build-pod",
									},
								},
								TopologyKey: "kubernetes.io/hostname",
							},
							Weight: 50,
						},
						{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"app.kubernetes.io/name":      "kaniko",
										"app.kubernetes.io/component": "build-pod",
									},
								},
								TopologyKey: "topology.kubernetes.io/zone",
							},
							Weight: 100,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: e.DockerConfigSecret,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  e.DockerConfigSecret,
							DefaultMode: ptr.To(int32(420)),
						},
					},
				},
			},
		},
	}
	err = k8sutils.MergeObjectWithYaml(&pod, e.PodConfig.PodOverride)
	if err != nil {
		return err
	}

	watcher, err := e.clientSet.CoreV1().Pods(e.PodConfig.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", pod.Name),
		Watch:         true,
	})
	if err != nil {
		return fmt.Errorf("failed to watch pod: %w", err)
	}
	defer watcher.Stop()

	readyChan, errChan := k8sutils.WaitPodReady(ctx, watcher)
	go func() {
		<-readyChan
		k8sutils.PrintPodLogs(ctx, output, e.clientSet, e.PodConfig.Namespace, podName, containerName)
	}()

	_, err = e.clientSet.CoreV1().Pods(e.PodConfig.Namespace).Create(ctx, &pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kaniko pod: %w", err)
	}
	defer func() {
		err := e.clientSet.CoreV1().Pods(e.PodConfig.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil {
			logrus.Warnf("Failed to delete kaniko pod %s, ignoring: %v", pod.Name, err)
		}
	}()

	err = <-errChan
	if err != nil {
		return fmt.Errorf("error watching kaniko pod: %w", err)
	}
	return nil
}
