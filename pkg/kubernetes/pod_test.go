package kubernetes_test

import (
	"regexp"
	"strings"
	"testing"

	k8sutils "github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
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
