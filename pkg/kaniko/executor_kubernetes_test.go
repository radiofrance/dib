package kaniko_test

import (
	"testing"
	"time"

	"github.com/radiofrance/dib/pkg/kaniko"
	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/radiofrance/dib/pkg/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stest "k8s.io/client-go/testing"
)

const dockerSecretName = "some_kubernetes_secret_name" //nolint:gosec

func Test_KubernetesExecutor_ExecuteRequiresDockerSecret(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := kaniko.NewKubernetesExecutor(clientSet, k8sutils.PodConfig{})

	writer := mock.NewWriter()
	err := executor.Execute(t.Context(), writer, []string{"kaniko-arg1", "kaniko-arg2"})
	assert.Empty(t, writer.GetString())
	require.EqualError(t, err, "the DockerConfigSecret option is required")
}

func Test_KubernetesExecutor_ExecuteFailsOnInvalidContainerYamlOverride(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := kaniko.NewKubernetesExecutor(clientSet, k8sutils.PodConfig{})
	executor.DockerConfigSecret = dockerSecretName
	executor.PodConfig = k8sutils.PodConfig{
		ContainerOverride: "{\n",
	}

	writer := mock.NewWriter()
	err := executor.Execute(t.Context(), writer, []string{"kaniko-arg1", "kaniko-arg2"})
	assert.Empty(t, writer.GetString())
	require.EqualError(t, err, "invalid yaml override for type *v1.Container: unexpected EOF")
}

func Test_KubernetesExecutor_ExecuteFailsOnInvalidPodTemplateYamlOverride(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := kaniko.NewKubernetesExecutor(clientSet, k8sutils.PodConfig{})
	executor.DockerConfigSecret = dockerSecretName
	executor.PodConfig = k8sutils.PodConfig{
		PodOverride: "{\n",
	}

	writer := mock.NewWriter()
	err := executor.Execute(t.Context(), writer, []string{"kaniko-arg1", "kaniko-arg2"})
	assert.Empty(t, writer.GetString())
	require.EqualError(t, err, "invalid yaml override for type *v1.Pod: unexpected EOF")
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
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			clientSet := fake.NewSimpleClientset()
			watcher := watch.NewFake()
			clientSet.PrependWatchReactor("pods", k8stest.DefaultWatchReactor(watcher, nil))

			podConfig := k8sutils.PodConfig{
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
				pod, err := clientSet.CoreV1().Pods("kaniko-ns").Get(t.Context(), "kaniko-pod", metav1.GetOptions{})
				assert.NoError(t, err)

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
				assert.Equal(t, dockerSecretName, volume.Secret.SecretName)
				assert.Equal(t, int32(420), *volume.Secret.DefaultMode)

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

				simulatePodExecution(t, watcher, test.success)
			}()

			// Run the executor
			writer := mock.NewWriter()

			err := executor.Execute(t.Context(), writer, []string{"kaniko-arg1", "kaniko-arg2"})
			if test.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			assert.Equal(t, "fake logs", writer.GetString())

			// Check the pod has been deleted
			_, err = clientSet.CoreV1().Pods("kaniko-ns").Get(t.Context(), "kaniko-pod", metav1.GetOptions{})
			require.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
		})
	}
}

// simulatePodExecution simulates the default behaviour of a Kubernetes pod controller
// by creating a pod, and also simulates the pod lifecycle until it reaches completion.
func simulatePodExecution(t *testing.T, watcher *watch.FakeWatcher, isSuccess bool) {
	t.Helper()

	watcher.Action(watch.Added, &corev1.Pod{
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	})

	<-time.After(1 * time.Second)
	watcher.Action(watch.Modified, &corev1.Pod{
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	})

	<-time.After(1 * time.Second)

	if isSuccess {
		watcher.Action(watch.Modified, &corev1.Pod{
			Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
		})
	} else {
		watcher.Action(watch.Modified, &corev1.Pod{
			Status: corev1.PodStatus{Phase: corev1.PodFailed},
		})
	}
}
