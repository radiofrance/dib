package kaniko

import (
	"context"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

// JobConfig hold the configuration for the kubernetes job to create.
type JobConfig struct {
	// Kubernetes generic configuration.
	Name             string            // The name of the job. Must be unique to avoid collisions with an existing job.
	NameGenerator    func() string     // A function that generates the job name. Will override the Name option.
	Namespace        string            // The namespace where the job should be created.
	Labels           map[string]string // A map of key/value labels.
	Image            string            // The image for the kaniko container.
	ImagePullSecrets []string          // A list of `imagePullSecret` secret names.
	Env              map[string]string // A map of key/value env variables.
	EnvSecrets       []string          // A list of `envFrom` secret names.

	// Advanced customisations (raw YAML overrides)
	ContainerOverride   string // YAML string to override the Kaniko container object.
	PodTemplateOverride string // YAML string to override the job.spec.template object.
}

// KubernetesExecutor will run Kaniko in a Kubernetes cluster.
type KubernetesExecutor struct {
	clientSet          kubernetes.Interface
	DockerConfigSecret string    // Name of the secret containing the docker config used by Kaniko (required).
	JobConfig          JobConfig // The default job configuration used to run Kaniko builds.
}

// NewKubernetesExecutor creates a new instance of KubernetesExecutor.
func NewKubernetesExecutor(clientSet kubernetes.Interface, config JobConfig) *KubernetesExecutor {
	return &KubernetesExecutor{
		clientSet:          clientSet,
		DockerConfigSecret: "",
		JobConfig:          config,
	}
}

// Execute the Kaniko build using a Kubernetes Job.
func (e KubernetesExecutor) Execute(ctx context.Context, args []string) error {
	if e.DockerConfigSecret == "" {
		return fmt.Errorf("the DockerConfigSecret option is required")
	}

	name := e.JobConfig.Name
	if e.JobConfig.NameGenerator != nil {
		name = e.JobConfig.NameGenerator()
	}

	labels := map[string]string{
		"app.kubernetes.io/name":      "kaniko",
		"app.kubernetes.io/component": "build-job",
		"app.kubernetes.io/instance":  name,
	}
	// Merge the default labels with those provided in the options.
	for k, v := range e.JobConfig.Labels {
		labels[k] = v
	}

	objectMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: e.JobConfig.Namespace,
		Labels:    labels,
	}

	var imagePullSecrets []corev1.LocalObjectReference
	for _, secretName := range e.JobConfig.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: secretName,
		})
	}

	var envVars []corev1.EnvVar
	for k, v := range e.JobConfig.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	var envFrom []corev1.EnvFromSource
	for _, secretName := range e.JobConfig.EnvSecrets {
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
		Image:           e.JobConfig.Image,
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
	err := mergeObjectWithYaml(&container, e.JobConfig.ContainerOverride)
	if err != nil {
		return err
	}

	podTemplate := corev1.PodTemplateSpec{
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
										"app.kubernetes.io/component": "build-job",
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
										"app.kubernetes.io/component": "build-job",
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
	err = mergeObjectWithYaml(&podTemplate, e.JobConfig.PodTemplateOverride)
	if err != nil {
		return err
	}

	job := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: objectMeta,
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template:     podTemplate,
		},
	}

	_, err = e.clientSet.BatchV1().Jobs(e.JobConfig.Namespace).Create(ctx, &job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kaniko job: %w", err)
	}

	return watchJob(ctx, e.clientSet, e.JobConfig.Namespace, name)
}

// UniqueJobName generates an unique job name with random characters.
// An identifier string passed as argument will be included in the generated job name.
func UniqueJobName(identifier string) string {
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

// mergeObjectWithYaml unmarshalls the YAML from the yamlOverride argument into the provided object.
// The `obj` argument typically is a pointer to a kubernetes type (with `json` tags).
// Existing values inside the `obj` will be erased if the YAML explicitely overrides it.
// All values within the object that are not explicitely overriden will not be modified.
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
