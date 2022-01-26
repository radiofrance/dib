package kaniko_test

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/radiofrance/dib/kaniko"
)

const dockerSecretName = "some_kubernetes_secret_name" //nolint:gosec

func Test_KubernetesExecutor_ExecuteRequiresDockerSecret(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := kaniko.NewKubernetesExecutor(clientSet, kaniko.PodConfig{})

	err := executor.Execute(context.Background(), []string{"kaniko-arg1", "kaniko-arg2"})

	assert.EqualError(t, err, "the DockerConfigSecret option is required")
}

func Test_KubernetesExecutor_ExecuteFailsOnInvalidContainerYamlOverride(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := kaniko.NewKubernetesExecutor(clientSet, kaniko.PodConfig{})
	executor.DockerConfigSecret = dockerSecretName
	executor.PodConfig = kaniko.PodConfig{
		ContainerOverride: "{\n",
	}

	err := executor.Execute(context.Background(), []string{"kaniko-arg1", "kaniko-arg2"})

	assert.EqualError(t, err, "invalid yaml override for type *v1.Container: unexpected EOF")
}

func Test_KubernetesExecutor_ExecuteFailsOnInvalidPodTemplateYamlOverride(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := kaniko.NewKubernetesExecutor(clientSet, kaniko.PodConfig{})
	executor.DockerConfigSecret = dockerSecretName
	executor.PodConfig = kaniko.PodConfig{
		PodOverride: "{\n",
	}

	err := executor.Execute(context.Background(), []string{"kaniko-arg1", "kaniko-arg2"})

	assert.EqualError(t, err, "invalid yaml override for type *v1.Pod: unexpected EOF")
}

func Test_KubernetesExecutor_Execute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		success bool
	}{
		{"build succeeded", true},
		{"build failed", false},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			clientSet := fake.NewSimpleClientset()
			podConfig := kaniko.PodConfig{
				Name: "name-overridden-by-name-generator",
				NameGenerator: func() string {
					return "kaniko-pod"
				},
				Namespace: "kaniko-ns",
				Labels: map[string]string{
					"some_label": "some_value",
				},
				Image: "my-kaniko-image:tag",
				ImagePullSecrets: []string{
					"my-pull-secret",
				},
				Env: map[string]string{
					"MY_ENV_VAR": "env_value",
				},
				EnvSecrets: []string{
					"my-env-secret",
				},
				ContainerOverride: `
resources:
  limits:
    cpu: 2
  requests:
    memory: 1Gi
`,
				PodOverride: `
spec:
  restartPolicy: OnFailure
`,
			}

			expectedLabels := map[string]string{
				"app.kubernetes.io/name":      "kaniko",
				"app.kubernetes.io/component": "build-pod",
				"app.kubernetes.io/instance":  "kaniko-pod",
				"some_label":                  "some_value",
			}

			executor := kaniko.NewKubernetesExecutor(clientSet, podConfig)
			executor.DockerConfigSecret = dockerSecretName

			go func() {
				// Wait for the Pod to be created before running assertions
				<-time.After(1 * time.Second)

				// Check the created Pod
				pod, err := clientSet.CoreV1().Pods("kaniko-ns").Get(context.Background(), "kaniko-pod", metav1.GetOptions{})
				require.NoError(t, err)

				// Pod assertions
				assert.Equal(t, expectedLabels, pod.Labels)
				assert.Len(t, pod.Spec.Containers, 1)
				assert.Equal(t, expectedLabels, pod.Labels)
				assert.Contains(t, pod.Spec.ImagePullSecrets, corev1.LocalObjectReference{
					Name: "my-pull-secret",
				})
				assert.Equal(t, corev1.RestartPolicyOnFailure, pod.Spec.RestartPolicy)

				assert.Len(t, pod.Spec.Volumes, 1)
				volume := pod.Spec.Volumes[0]
				assert.Equal(t, dockerSecretName, volume.Name)
				assert.Equal(t, dockerSecretName, volume.VolumeSource.Secret.SecretName)
				assert.Equal(t, int32(420), *volume.VolumeSource.Secret.DefaultMode)

				// Container assertions
				container := pod.Spec.Containers[0]
				assert.ElementsMatch(t, container.Args, []string{
					"kaniko-arg1",
					"kaniko-arg2",
				})
				assert.ElementsMatch(t, container.Env, []corev1.EnvVar{
					{
						Name:  "DOCKER_CONFIG",
						Value: "/kaniko/.docker",
					},
					{
						Name:  "MY_ENV_VAR",
						Value: "env_value",
					},
				})
				assert.Len(t, container.EnvFrom, 1)
				assert.Equal(t, "my-env-secret", container.EnvFrom[0].SecretRef.Name)

				assert.True(t, container.Resources.Limits[corev1.ResourceCPU].Equal(resource.MustParse("2")))
				assert.True(t, container.Resources.Requests[corev1.ResourceMemory].Equal(resource.MustParse("1Gi")))

				assert.Equal(t, "my-kaniko-image:tag", container.Image)

				assert.ElementsMatch(t, container.VolumeMounts, []corev1.VolumeMount{
					{
						Name:      dockerSecretName,
						MountPath: "/kaniko/.docker",
						ReadOnly:  true,
					},
				})

				simulatePodExecution(t, clientSet, pod, test.success)
			}()

			// Run the executor
			err := executor.Execute(context.Background(), []string{"kaniko-arg1", "kaniko-arg2"})
			if test.success {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// simulatePodExecution simulates the default behaviour of a Kubernetes pod controller
// by creating a pod, and also simulates the pod lifecycle until it reaches completion.
func simulatePodExecution(t *testing.T, clientSet kubernetes.Interface, pod *corev1.Pod, isSuccess bool) {
	t.Helper()

	// Wait a moment to simulate the pod taking time to complete its task.
	<-time.After(3 * time.Second)

	// Set pod status to completed
	if isSuccess {
		pod.Status.Phase = corev1.PodSucceeded
	} else {
		pod.Status.Phase = corev1.PodFailed
	}

	_, err := clientSet.CoreV1().Pods(pod.Namespace).Update(context.Background(), pod, metav1.UpdateOptions{})
	require.NoError(t, err)
}

func Test_UniquePodName(t *testing.T) {
	t.Parallel()

	dataset := []struct {
		identifier     string
		expectedPrefix string
	}{
		{
			identifier:     "dib",
			expectedPrefix: "kaniko-dib-",
		},
		{
			identifier:     "semicolon:slashes/dib",
			expectedPrefix: "kaniko-semicolon-slashes-dib-",
		},
		{
			identifier:     "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong",
			expectedPrefix: "kaniko-veryveryveryveryveryveryveryveryveryveryveryver-",
		},
	}

	// Only alphanumeric characters, or dashes, maximum 63 chars
	validationRegexp := regexp.MustCompile(`^[a-z0-9\-]{1,63}`)

	for _, ds := range dataset {
		podName := kaniko.UniquePodName(ds.identifier)()

		assert.Truef(t, strings.HasPrefix(podName, ds.expectedPrefix),
			"Pod name %s does not have prefix %s", podName, ds.expectedPrefix)

		assert.Regexp(t, validationRegexp, podName)
	}
}
