package kubernetes_test

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func Test_UniquePodName(t *testing.T) {
	t.Parallel()

	dataset := []struct {
		identifier     string
		expectedPrefix string
	}{
		{
			identifier:     "dib",
			expectedPrefix: "dib-",
		},
		{
			identifier:     "semicolon:slashes/dib",
			expectedPrefix: "semicolon-slashes-dib-",
		},
		{
			identifier:     "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong",
			expectedPrefix: "veryveryveryveryveryveryveryveryveryveryveryveryveryve-",
		},
	}

	// Only alphanumeric characters, or dashes, maximum 63 chars
	validationRegexp := regexp.MustCompile(`^[a-z0-9\-]{1,63}`)

	for _, ds := range dataset {
		podName := k8sutils.UniquePodName(ds.identifier)()

		assert.Truef(t, strings.HasPrefix(podName, ds.expectedPrefix),
			"Pod name %s does not have prefix %s", podName, ds.expectedPrefix)

		assert.Regexp(t, validationRegexp, podName)
	}
}

func Test_UniquePodNameWithImage(t *testing.T) {
	t.Parallel()

	dataset := []struct {
		identifier     string
		imageName      string
		expectedPrefix string
	}{
		{
			identifier:     "buildkit-dib",
			imageName:      "nginx",
			expectedPrefix: "buildkit-dib-nginx-",
		},
		{
			identifier:     "buildkit-dib",
			imageName:      "registry.example.com/nginx:1.19",
			expectedPrefix: "buildkit-dib-registry.example.com-nginx-1.19-",
		},
		{
			identifier:     "buildkit-dib",
			imageName:      "aquasec_trivy",
			expectedPrefix: "buildkit-dib-aquasec-trivy-",
		},
		{
			identifier:     "semicolon:slashes/dib",
			imageName:      "image:with/special:chars",
			expectedPrefix: "semicolon-slashes-dib-image-with-special-chars-",
		},
		{
			identifier:     "short",
			imageName:      "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong",
			expectedPrefix: "short-veryveryveryveryveryveryveryveryveryveryveryvery-",
		},
		{
			identifier:     "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong",
			imageName:      "short",
			expectedPrefix: "veryveryveryveryveryveryveryveryveryveryveryveryveryve-",
		},
		{
			identifier:     "UPPERCASE",
			imageName:      "MixedCase",
			expectedPrefix: "uppercase-mixedcase-",
		},
	}

	// Only alphanumeric characters, or dashes, maximum 63 chars
	validationRegexp := regexp.MustCompile(`^[a-z0-9\-]{1,63}`)

	for _, ds := range dataset {
		podName := k8sutils.UniquePodNameWithImage(ds.identifier, ds.imageName)()

		assert.Truef(t, strings.HasPrefix(podName, ds.expectedPrefix),
			"Pod name %s does not have prefix %s", podName, ds.expectedPrefix)

		assert.Regexp(t, validationRegexp, podName)
	}
}

func Test_MonitorPod_Disruption(t *testing.T) {
	t.Parallel()

	newPod := func(phase corev1.PodPhase, conditions ...corev1.PodCondition) *corev1.Pod {
		return &corev1.Pod{
			Name:      "dib-buildkit-test",
			Namespace: "ci",
			Status:    corev1.PodStatus{Phase: phase, Conditions: conditions},
		}
	}

	evictedCondition := corev1.PodCondition{
		Type:   corev1.DisruptionTarget,
		Status: corev1.ConditionTrue,
		Reason: "EvictionByEvictionAPI",
	}

	tests := []struct {
		name   string
		send   func(fake *watch.FakeWatcher)
		cancel bool
		assert func(t *testing.T, err error)
	}{
		{
			name: "evicted",
			send: func(fake *watch.FakeWatcher) { fake.Modify(newPod(corev1.PodFailed, evictedCondition)) },
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, k8sutils.ErrPodDisrupted)
				assert.ErrorContains(t, err, "EvictionByEvictionAPI")
			},
		},
		{
			name: "deleted",
			send: func(fake *watch.FakeWatcher) { fake.Delete(newPod(corev1.PodRunning)) },
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, k8sutils.ErrPodDisrupted)
			},
		},
		{
			name: "deleted gracefully",
			send: func(fake *watch.FakeWatcher) {
				pod := newPod(corev1.PodFailed)
				pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				fake.Modify(pod)
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, k8sutils.ErrPodDisrupted)
				assert.ErrorContains(t, err, "deleted")
			},
		},
		{
			name: "build failure",
			send: func(fake *watch.FakeWatcher) { fake.Modify(newPod(corev1.PodFailed)) },
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.EqualError(t, err, "pod dib-buildkit-test terminated (failed)")
				assert.NotErrorIs(t, err, k8sutils.ErrPodDisrupted)
			},
		},
		{
			name: "disruption condition false",
			send: func(fake *watch.FakeWatcher) {
				fake.Modify(newPod(corev1.PodFailed, corev1.PodCondition{
					Type:   corev1.DisruptionTarget,
					Status: corev1.ConditionFalse,
				}))
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.NotErrorIs(t, err, k8sutils.ErrPodDisrupted)
			},
		},
		{
			name:   "context cancelled",
			cancel: true,
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, context.Canceled)
				assert.NotErrorIs(t, err, k8sutils.ErrPodDisrupted)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			fake := watch.NewFake()
			defer fake.Stop()

			_, errChan := k8sutils.MonitorPod(ctx, fake)

			if test.cancel {
				cancel()
			} else {
				go test.send(fake)
			}

			test.assert(t, <-errChan)
		})
	}
}
