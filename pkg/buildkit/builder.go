package buildkit

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/distribution/reference"
	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/executor"
	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/radiofrance/dib/pkg/strutil"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/radiofrance/kubecli"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/utils/ptr"
)

// ContextProvider provides a layer of abstraction for different build context sources.
type ContextProvider interface {
	// PrepareContext allows to do some operations on the build context before the executor runs,
	// like moving it to a remote location in order to be accessible by remote executors.
	// It must return a URL compatible with Buildkit's `--context` flag.
	PrepareContext(opts types.ImageBuilderOpts) (string, error)
}

type bkShellExecutor struct {
	shellExecutor  executor.ShellExecutor
	buildctlBinary string
}

type bkKubernetesExecutor struct {
	KubernetesExecutor executor.KubernetesExecutor
	buildctlBinary     string
	dockerConfigSecret string             // Name of the secret containing the docker config used by Buildkit (required).
	podConfig          k8sutils.PodConfig // The default pod configuration used to run Buildkit builds.
}
type Builder struct {
	bkShellExecutor      bkShellExecutor
	bkKubernetesExecutor bkKubernetesExecutor
	contextProvider      ContextProvider
}

// Config holds the configuration for the Buildkit build backend.
type Config struct {
	Context struct {
		S3 struct {
			Bucket string `mapstructure:"bucket"`
			Region string `mapstructure:"region"`
		} `mapstructure:"s3"`
	} `mapstructure:"context"`
	Executor struct {
		Kubernetes struct {
			Namespace           string   `mapstructure:"namespace"`
			Image               string   `mapstructure:"image"`
			DockerConfigSecret  string   `mapstructure:"docker_config_secret"`
			ImagePullSecrets    []string `mapstructure:"image_pull_secrets"`
			EnvSecrets          []string `mapstructure:"env_secrets"`
			ContainerOverride   string   `mapstructure:"container_override"`
			PodTemplateOverride string   `mapstructure:"pod_template_override"`
		} `mapstructure:"kubernetes"`
	} `mapstructure:"executor"`
}

// NewBuilder creates a new instance of Builder.
func NewBKBuilder(cfg Config, workingDir string, binary string, localOnly bool) (*Builder, error) {
	var (
		err             error
		k8sExecutor     executor.KubernetesExecutor
		shellExecutor   executor.ShellExecutor
		contextProvider ContextProvider
	)

	if localOnly {
		shellExecutor = exec.NewShellExecutor(workingDir, nil)
		contextProvider = NewLocalContextProvider()
	} else {
		k8sExecutor, err = createBuildkitKubernetesExecutor()
		if err != nil {
			logger.Fatalf("cannot create buidkit kubernetes executor: %v", err)
		}

		awsCfg, err := config.LoadDefaultConfig(context.Background(),
			config.WithRegion(cfg.Context.S3.Region))
		if err != nil {
			logger.Fatalf("cannot load AWS config: %v", err)
		}
		s3 := NewS3Uploader(awsCfg, cfg.Context.S3.Bucket)
		contextProvider = NewRemoteContextProvider(s3)
	}

	return &Builder{
		bkShellExecutor: bkShellExecutor{
			shellExecutor,
			binary,
		},
		bkKubernetesExecutor: bkKubernetesExecutor{
			KubernetesExecutor: k8sExecutor,
			buildctlBinary:     binary,
			dockerConfigSecret: cfg.Executor.Kubernetes.DockerConfigSecret,
			podConfig: k8sutils.PodConfig{
				Namespace:     cfg.Executor.Kubernetes.Namespace,
				NameGenerator: k8sutils.UniquePodName("buildkit-dib"),
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "dib",
				},
				Image:            cfg.Executor.Kubernetes.Image,
				ImagePullSecrets: cfg.Executor.Kubernetes.ImagePullSecrets,
				EnvSecrets:       cfg.Executor.Kubernetes.EnvSecrets,
				Env: map[string]string{
					"AWS_REGION": cfg.Context.S3.Region,
					"container":  "kube", // Fix for https://github.com/GoogleContainerTools/kaniko/issues/1542
				},
				PodOverride:       cfg.Executor.Kubernetes.PodTemplateOverride,
				ContainerOverride: cfg.Executor.Kubernetes.ContainerOverride,
			},
		},
		contextProvider: contextProvider,
	}, nil
}

// Build the image using the Buildkit backend.
func (b Builder) Build(opts types.ImageBuilderOpts) error {
	var err error

	opts.Context, err = b.contextProvider.PrepareContext(opts)
	if err != nil {
		return fmt.Errorf("cannot prepare buildkit build context: %w", err)
	}
	buildctlArgs, err := generateBuildctlArgs(opts)
	if err != nil {
		return err
	}
	// `shellExecutor` or `kubernetesExecutor` are mutually exclusive.
	if b.bkShellExecutor.shellExecutor != nil {
		//nolint:lll
		if err := b.bkShellExecutor.shellExecutor.ExecuteStdout(b.bkShellExecutor.buildctlBinary, buildctlArgs...); err != nil {
			return err
		}
	} else {
		pod, err := buildPod(b.bkKubernetesExecutor.dockerConfigSecret, b.bkKubernetesExecutor.podConfig, buildctlArgs)
		if err != nil {
			return err
		}
		//nolint:lll
		if err := b.bkKubernetesExecutor.KubernetesExecutor.ApplyWithWriters(context.Background(), opts.LogOutput, opts.LogOutput, pod, "buildkit"); err != nil {
			return err
		}
	}

	return nil
}

func createBuildkitKubernetesExecutor() (*exec.KubernetesExecutor, error) {
	k8sClient, err := kubecli.New("")
	if err != nil {
		return nil, fmt.Errorf("could not get kube client from context: %w", err)
	}

	executor := exec.NewKubernetesExecutor(k8sClient.ClientSet)
	return executor, nil
}

func generateBuildctlArgs(opts types.ImageBuilderOpts) ([]string, error) {
	output := "type=image"

	if tags := strutil.DedupeStrSlice(opts.Tags); len(tags) > 0 {
		for _, tag := range tags {
			// Normalize the tag by transforming it from a familiar name used in Docker UI to a fully qualified reference.
			parsedReference, err := reference.ParseNormalizedNamed(tag)
			if err != nil {
				return nil, err
			}
			output += ",name=" + parsedReference.String()
		}
	} else {
		output += ",dangling-name-prefix=<none>"
	}

	if opts.Push {
		output += ",push=true"
	}

	buildctlArgs := buildctlBaseArgs(opts.BuildkitHost)

	var contextArg string
	if opts.LocalOnly {
		contextArg = "--local=context=" + opts.Context
	} else {
		// We use a pre-signed URL to securely fetch the context from a remote source, ensuring proper access control.
		contextArg = "--opt=context=" + opts.Context
	}
	buildctlArgs = append(buildctlArgs, []string{
		"build",
		"--progress=" + opts.Progress,
		"--frontend=dockerfile.v0",
		contextArg,
		"--output=" + output,
	}...)

	if opts.LocalOnly {
		// Set the directory and filename for the Dockerfile,
		// as the Dockerfile path may differ from the build context path.
		dir := opts.Context
		file := defaultDockerfileName
		if opts.File != "" {
			dir, file = filepath.Split(opts.File)

			if dir == "" {
				dir = "."
			}
		}

		var err error
		dir, file, err = buildKitFile(dir, file)
		if err != nil {
			return nil, err
		}
		buildctlArgs = append(buildctlArgs, "--local=dockerfile="+dir)
		buildctlArgs = append(buildctlArgs, "--opt=filename="+file)
	}

	// The target option specifies the build stage to build.
	if opts.Target != "" {
		buildctlArgs = append(buildctlArgs, "--opt=target="+opts.Target)
	}

	for key, val := range opts.BuildArgs {
		buildctlArgs = append(buildctlArgs, "--opt=build-arg:"+key+"="+val)
	}

	for _, l := range opts.Labels {
		buildctlArgs = append(buildctlArgs, "--opt=label="+l)
	}

	return buildctlArgs, nil
}

func buildPod(dockerConfigSecret string, podConfig k8sutils.PodConfig, args []string) (*corev1.Pod, error) {
	logger.Infof("Building image with Buildkit kubernetes executor")
	if dockerConfigSecret == "" {
		return nil, errors.New("the DockerConfigSecret option is required")
	}

	podName := podConfig.Name
	if podConfig.NameGenerator != nil {
		podName = podConfig.NameGenerator()
	}
	containerName := "buildkit"

	labels := map[string]string{
		"app.kubernetes.io/name":      "buildkit",
		"app.kubernetes.io/component": "build-pod",
		"app.kubernetes.io/instance":  podName,
	}
	// Merge the default labels with those provided in the options.
	for k, v := range podConfig.Labels {
		labels[k] = v
	}

	objectMeta := metav1.ObjectMeta{
		Name:      podName,
		Namespace: podConfig.Namespace,
		Labels:    labels,
	}

	var imagePullSecrets []corev1.LocalObjectReference
	for _, secretName := range podConfig.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: secretName,
		})
	}

	var envVars []corev1.EnvVar
	for k, v := range podConfig.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	var envFrom []corev1.EnvFromSource
	for _, secretName := range podConfig.EnvSecrets {
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
		Image:           podConfig.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            args,
		EnvFrom:         envFrom,
		Env: append([]corev1.EnvVar{
			{
				Name:  "DOCKER_CONFIG",
				Value: "/buildkit/.docker",
			},
		}, envVars...),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      dockerConfigSecret,
				MountPath: "/buildkit/.docker",
				ReadOnly:  true,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"buildctl", "debug", "workers"},
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       30,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"buildctl", "debug", "workers"},
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       30,
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:  ptr.To[int64](RemoteUserId),
			RunAsGroup: ptr.To[int64](RemoteGroupId),
			// Needs Kubernetes >= 1.19
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeUnconfined,
			},
			// Needs Kubernetes >= 1.30
			// https://github.com/rootless-containers/rootlesskit/pull/421
			AppArmorProfile: &corev1.AppArmorProfile{
				Type: corev1.AppArmorProfileTypeUnconfined,
			},
		},
	}
	err := k8sutils.MergeObjectWithYaml(&container, podConfig.ContainerOverride)
	if err != nil {
		return nil, err
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
										"app.kubernetes.io/name":      "buildkit",
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
										"app.kubernetes.io/name":      "buildkit",
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
					Name: dockerConfigSecret,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  dockerConfigSecret,
							DefaultMode: ptr.To[int32](420),
						},
					},
				},
			},
		},
	}
	err = k8sutils.MergeObjectWithYaml(&pod, podConfig.PodOverride)
	if err != nil {
		return nil, err
	}

	return &pod, nil
}
