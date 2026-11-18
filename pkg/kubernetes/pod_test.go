package kubernetes_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/radiofrance/dib/pkg/types"
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

	deleting := newPod(corev1.PodFailed)
	deleting.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	tests := []struct {
		name      string
		event     func(fake *watch.FakeWatcher) // nil cancels the context instead
		disrupted bool
		contains  string
	}{
		{
			name: "evicted",
			event: func(fake *watch.FakeWatcher) {
				fake.Modify(newPod(corev1.PodFailed, corev1.PodCondition{
					Type: corev1.DisruptionTarget, Status: corev1.ConditionTrue, Reason: "EvictionByEvictionAPI",
				}))
			},
			disrupted: true,
			contains:  "pod dib-buildkit-test terminated (failed): pod disrupted (EvictionByEvictionAPI)",
		},
		{
			name:      "deleted",
			event:     func(fake *watch.FakeWatcher) { fake.Delete(newPod(corev1.PodRunning)) },
			disrupted: true,
			contains:  "was deleted",
		},
		{
			name:      "deleted gracefully",
			event:     func(fake *watch.FakeWatcher) { fake.Modify(deleting) },
			disrupted: true,
			contains:  "(deleted)",
		},
		{
			name:     "build failure",
			event:    func(fake *watch.FakeWatcher) { fake.Modify(newPod(corev1.PodFailed)) },
			contains: "pod dib-buildkit-test terminated (failed)",
		},
		{
			name: "disruption condition false",
			event: func(fake *watch.FakeWatcher) {
				fake.Modify(newPod(corev1.PodFailed, corev1.PodCondition{
					Type: corev1.DisruptionTarget, Status: corev1.ConditionFalse,
				}))
			},
		},
		{
			name:     "context cancelled",
			contains: context.Canceled.Error(),
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

			if test.event == nil {
				cancel()
			} else {
				go test.event(fake)
			}

			err := <-errChan
			require.Error(t, err)
			assert.Equal(t, test.disrupted, errors.Is(err, types.ErrPodDisrupted))
			assert.ErrorContains(t, err, test.contains)
		})
	}
}
