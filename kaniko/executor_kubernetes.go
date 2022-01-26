package kaniko

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

// PodConfig hold the configuration for the kubernetes pod to create.
type PodConfig struct {
	// Kubernetes generic configuration.
	Name             string            // The name of the pod. Must be unique to avoid collisions with an existing pod.
	NameGenerator    func() string     // A function that generates the pod name. Will override the Name option.
	Namespace        string            // The namespace where the pod should be created.
	Labels           map[string]string // A map of key/value labels.
	Image            string            // The image for the kaniko container.
	ImagePullSecrets []string          // A list of `imagePullSecret` secret names.
	Env              map[string]string // A map of key/value env variables.
	EnvSecrets       []string          // A list of `envFrom` secret names.

	// Advanced customisations (raw YAML overrides)
	ContainerOverride string // YAML string to override the Kaniko container object.
	PodOverride       string // YAML string to override the pod object.
}

// KubernetesExecutor will run Kaniko in a Kubernetes cluster.
type KubernetesExecutor struct {
	clientSet          kubernetes.Interface
	DockerConfigSecret string    // Name of the secret containing the docker config used by Kaniko (required).
	PodConfig          PodConfig // The default pod configuration used to run Kaniko builds.
}

// NewKubernetesExecutor creates a new instance of KubernetesExecutor.
func NewKubernetesExecutor(clientSet kubernetes.Interface, config PodConfig) *KubernetesExecutor {
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

	name := e.PodConfig.Name
	if e.PodConfig.NameGenerator != nil {
		name = e.PodConfig.NameGenerator()
	}

	labels := map[string]string{
		"app.kubernetes.io/name":      "kaniko",
		"app.kubernetes.io/component": "build-pod",
		"app.kubernetes.io/instance":  name,
	}
	// Merge the default labels with those provided in the options.
	for k, v := range e.PodConfig.Labels {
		labels[k] = v
	}

	objectMeta := metav1.ObjectMeta{
		Name:      name,
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
		Name:            "kaniko",
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
	err := mergeObjectWithYaml(&container, e.PodConfig.ContainerOverride)
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
							DefaultMode: pointer.Int32Ptr(420),
						},
					},
				},
			},
		},
	}
	err = mergeObjectWithYaml(&pod, e.PodConfig.PodOverride)
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

	readyChan := make(chan struct{})
	doneChan := make(chan error)
	go func() {
		for {
			select {
			case event := <-watcher.ResultChan():
				pod, ok := event.Object.(*corev1.Pod)
				if !ok {
					logrus.Fatalf("type assertion failed for object %s, type *corev1.Pod", event.Object.GetObjectKind())
				}

				logrus.Debugf("Pod %s/%s %s, status %s", pod.ObjectMeta.Namespace,
					pod.ObjectMeta.Name, event.Type, pod.Status.Phase)
				switch pod.Status.Phase { // nolint: exhaustive
				case corev1.PodRunning:
					readyChan <- struct{}{}
					logrus.Infof("Kaniko pod %s/%s is Running, build in progress", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
				case corev1.PodSucceeded:
					logrus.Infof("Kaniko pod %s/%s succeeded", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
					doneChan <- nil
				case corev1.PodFailed:
					logrus.Infof("Kaniko pod %s/%s failed", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
					doneChan <- fmt.Errorf("build failed, pod %s has failed", pod.Name)
				}
			case <-time.After(1 * time.Hour):
				doneChan <- fmt.Errorf("build timeout, pod %s could not finish in less than one hour", pod.Name)
			}
		}
	}()

	go printPodLog(ctx, readyChan, output, e.clientSet, e.PodConfig.Namespace, name)
	_, err = e.clientSet.CoreV1().Pods(e.PodConfig.Namespace).Create(ctx, &pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kaniko pod: %w", err)
	}

	err = <-doneChan
	return err
}

// UniquePodName generates a unique pod name with random characters.
// An identifier string passed as argument will be included in the generated pod name.
func UniquePodName(identifier string) func() string {
	return func() string {
		identifier = strings.ReplaceAll(identifier, ":", "-")
		identifier = strings.ReplaceAll(identifier, "/", "-")
		base := fmt.Sprintf("kaniko-%s", identifier)
		maxNameLength, randomLength := 63, 8
		maxGeneratedNameLength := maxNameLength - randomLength - 1
		if len(base) > maxGeneratedNameLength {
			base = base[:maxGeneratedNameLength]
		}

		return strings.ToLower(fmt.Sprintf("%s-%s", base, rand.String(randomLength)))
	}
}

// mergeObjectWithYaml unmarshalls the YAML from the yamlOverride argument into the provided object.
// The `obj` argument typically is a pointer to a kubernetes type (with `json` tags).
// Existing values inside the `obj` will be erased if the YAML explicitly overrides it.
// All values within the object that are not explicitly overridden will not be modified.
func mergeObjectWithYaml(obj interface{}, yamlOverride string) error {
	if yamlOverride == "" {
		return nil
	}

	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(yamlOverride), 1024)
	if err := decoder.Decode(&obj); err != nil {
		return fmt.Errorf("invalid yaml override for type %T: %w", obj, err)
	}

	return nil
}
