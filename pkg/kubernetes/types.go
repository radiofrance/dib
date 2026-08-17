package kubernetes

// PodConfig hold the configuration for the kubernetes pod to create.
type PodConfig struct {
	// Kubernetes generic configuration.
	Name             string            // The name of the pod. Must be unique to avoid collisions with an existing pod.
	NameGenerator    func() string     // A function that generates the pod name. Will override the Name option.
	Namespace        string            // The namespace where the pod should be created.
	Image            string            // The image for the container.
	ImagePullSecrets []string          // A list of `imagePullSecret` secret names.
	Env              map[string]string // A map of key/value env variables.
	EnvSecrets       []string          // A list of `envFrom` secret names.

	// Advanced customizations (raw YAML overrides and additional values)
	ContainerOverride     string            // YAML string to override the container object.
	PodOverride           string            // YAML string to override the pod object.
	AdditionalLabels      map[string]string // A map of additional labels.
	AdditionalAnnotations map[string]string // A map of additional annotations.
}
