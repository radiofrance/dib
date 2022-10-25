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
