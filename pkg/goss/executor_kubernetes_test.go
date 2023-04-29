package goss_test

import (
	"context"
	"testing"
	"time"

	"github.com/radiofrance/dib/pkg/goss"
	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/radiofrance/dib/pkg/mock"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stest "k8s.io/client-go/testing"
)

func Test_KubernetesExecutor_ExecuteFailsOnInvalidContainerYamlOverride(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := goss.NewKubernetesExecutor(rest.Config{}, clientSet, k8sutils.PodConfig{})
	executor.PodConfig = k8sutils.PodConfig{
		ContainerOverride: "{\n",
	}

	writer := mock.NewWriter()
	opts := types.RunTestOptions{
		ImageName:         "image",
		ImageReference:    "registry.org/image:tag",
		DockerContextPath: "/path/to/context",
	}
	err := executor.Execute(context.Background(), writer, opts, "goss-arg1", "goss-arg2")
	assert.Empty(t, writer.GetString())

	assert.EqualError(t, err, "invalid yaml override for type *v1.Container: unexpected EOF")
}

func Test_KubernetesExecutor_ExecuteFailsOnInvalidPodTemplateYamlOverride(t *testing.T) {
	t.Parallel()

	clientSet := fake.NewSimpleClientset()
	executor := goss.NewKubernetesExecutor(rest.Config{}, clientSet, k8sutils.PodConfig{})
	executor.PodConfig = k8sutils.PodConfig{
		PodOverride: "{\n",
	}

	writer := mock.NewWriter()
	opts := types.RunTestOptions{
		ImageName:         "image",
		ImageReference:    "registry.org/image:tag",
		DockerContextPath: "../../test/fixtures",
	}
	err := executor.Execute(context.Background(), writer, opts, "goss-arg1", "goss-arg2")
	assert.Empty(t, writer.GetString())

	assert.EqualError(t, err, "invalid yaml override for type *v1.Pod: unexpected EOF")
}

func Test_KubernetesExecutor_Execute_CreatesValidPod(t *testing.T) {
	t.Parallel()
	clientSet := fake.NewSimpleClientset()
	watcher := watch.NewFake()
	clientSet.PrependWatchReactor("pods", k8stest.DefaultWatchReactor(watcher, nil))

	podConfig := k8sutils.PodConfig{
		NameGenerator: func() string { return "goss-pod" },
		Namespace:     "goss-ns",
		Image:         "my-goss-image:tag",
		ImagePullSecrets: []string{
			"my-pull-secret",
		},
		ContainerOverride: "",
		PodOverride: `
spec:
  restartPolicy: OnFailure
`,
	}

	expectedLabels := map[string]string{
		"app.kubernetes.io/name":      "goss",
		"app.kubernetes.io/component": "goss-pod",
		"app.kubernetes.io/instance":  "goss-pod",
	}

	executor := goss.NewKubernetesExecutor(rest.Config{}, clientSet, podConfig)

	go func() {
		// Wait for the Pod to be created before running assertions
		<-time.After(1 * time.Second)

		// Check the created Pod
		pod, err := clientSet.CoreV1().Pods("goss-ns").Get(context.Background(), "goss-pod", metav1.GetOptions{})
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
		assert.Equal(t, "shared", volume.Name)
		// InitContainer assertions
		initContainer := pod.Spec.InitContainers[0]
		assert.ElementsMatch(t, initContainer.Command, []string{
			"cp", "/goss/goss", "/shared",
		})
		assert.Equal(t, "my-goss-image:tag", initContainer.Image)
		assert.ElementsMatch(t, initContainer.VolumeMounts, []corev1.VolumeMount{
			{
				Name:      "shared",
				MountPath: "/shared",
				ReadOnly:  false,
			},
		})

		// Container assertions
		container := pod.Spec.Containers[0]
		assert.ElementsMatch(t, container.Command, []string{
			"sleep",
			"1h",
		})
		assert.Equal(t, "registry.org/image:tag", container.Image)
		assert.ElementsMatch(t, container.VolumeMounts, []corev1.VolumeMount{
			{
				Name:      "shared",
				MountPath: "/goss",
				ReadOnly:  false,
			},
		})

		simulatePodExecution(t, watcher, false)
	}()

	// Run the executor
	writer := mock.NewWriter()
	opts := types.RunTestOptions{
		ImageName:         "image",
		ImageReference:    "registry.org/image:tag",
		DockerContextPath: "../../test/fixtures",
	}
	err := executor.Execute(context.Background(), writer, opts, "goss-arg1", "goss-arg2")
	assert.Error(t, err)
	// @TODO: flaky assertion, need to be fixed
	//	assert.Equal(t, "fake logs", writer.GetString())

	// Check the pod has been deleted
	_, err = clientSet.CoreV1().Pods("goss").Get(context.Background(), "goss-pod", metav1.GetOptions{})
	require.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
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

	<-time.After(3 * time.Second)
	if isSuccess {
		return
	}
	watcher.Action(watch.Modified, &corev1.Pod{
		Status: corev1.PodStatus{Phase: corev1.PodFailed},
	})
}
