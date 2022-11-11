package report

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_printReportUrl_Gitlab(t *testing.T) { //nolint:paralleltest
	t.Setenv("CI_JOB_URL", "https://gitlab.com/example-repository/-/jobs/123456")

	dibReport := Report{Name: "20220823183000"}
	actual := printReportURL(dibReport)

	expected := "https://gitlab.com/example-repository/-/jobs/123456/artifacts/file/reports/20220823183000/index.html"
	assert.Equal(t, expected, actual)
}

func Test_printReportUrl_Local(t *testing.T) { //nolint:paralleltest
	dibReport := Report{Name: "20220823183000"}
	actual := printReportURL(dibReport)

	expected := "reports/20220823183000/index.html"
	assert.True(t, strings.Contains(actual, expected))
}
