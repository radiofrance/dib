package report

import (
	"regexp"

	"github.com/radiofrance/dib/pkg/types"
)

const (
	BuildReportDir = "builds"
	JunitReportDir = "junit"
	TrivyReportDir = "trivy"
)

var (
	patternAnsiColors = regexp.MustCompile(`\x1B\[([0-9]{1,3}(;[0-9]{1,2})?)?[mGK]`)
	patternKanikoLogs = regexp.MustCompile(`time=".*" level=.* msg="(?P<message>.*)"`)
)

// isTestRunnerEnabled return true if given types.TestRunner is enabled. False instead.
func isTestRunnerEnabled(name string, testRunners []types.TestRunner) bool {
	for _, runner := range testRunners {
		if name == runner.Name() {
			return true
		}
	}
	return false
}

func beautifyBuildsLogs(rawBuildLogs []byte) string {
	unescapedBuildLogs := RemoveTerminalColors(rawBuildLogs)
	return StripKanikoBuildLogs(unescapedBuildLogs)
}

// RemoveTerminalColors strips all ANSI escape codes from the given string.
func RemoveTerminalColors(input []byte) []byte {
	results := patternAnsiColors.ReplaceAll(input, []byte{})

	return results
}

// StripKanikoBuildLogs Improve readability of kaniko builds logs by removing unwanted stuff from a
// standard logs message.
func StripKanikoBuildLogs(input []byte) string {
	results := patternKanikoLogs.ReplaceAll(input, []byte("$message"))

	return string(results)
}
