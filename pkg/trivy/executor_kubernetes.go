package trivy

import (
	"context"
	"fmt"
	"io"
	"strings"

	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

// KubernetesExecutor will run Trivy in a Kubernetes cluster.
type KubernetesExecutor struct {
	clientSet          kubernetes.Interface
	DockerConfigSecret string             // Name of the secret containing the docker config used by Trivy (required).
	PodConfig          k8sutils.PodConfig // The default pod configuration used to run Trivy scans.
}

// NewKubernetesExecutor creates a new instance of KubernetesExecutor.
func NewKubernetesExecutor(clientSet kubernetes.Interface, config k8sutils.PodConfig) *KubernetesExecutor {
	return &KubernetesExecutor{
		clientSet:          clientSet,
		DockerConfigSecret: "",
		PodConfig:          config,
	}
}

func (e KubernetesExecutor) Execute(ctx context.Context, output io.Writer, args ...string,
) error {
	logrus.Info("Running trivy scan with kubernetes executor")
	logrus.Debugf("Running container with args '%s'", strings.Join(args, " "))

	if e.DockerConfigSecret == "" {
		return fmt.Errorf("the DockerConfigSecret option is required")
	}

	podName := e.PodConfig.Name
	if e.PodConfig.NameGenerator != nil {
		podName = e.PodConfig.NameGenerator()
	}
	containerName := "trivy"

	labels := map[string]string{
		"app.kubernetes.io/name":      "trivy",
		"app.kubernetes.io/component": "trivy-pod",
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
		// Command:         []string{"sleep"},
		Args: args,
		// Args:    []string{"3600"},
		Env:     envVars,
		EnvFrom: envFrom,
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
			Volumes: []corev1.Volume{
				{
					Name: e.DockerConfigSecret,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  e.DockerConfigSecret,
							DefaultMode: pointer.Int32(420),
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
