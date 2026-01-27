package report

import (
	"fmt"
	"html/template"
	"os"
	"path"
	"regexp"
	"sort"

	"github.com/radiofrance/dib/pkg/types"
)

const (
	BuildReportDir = "builds"
	JunitReportDir = "junit"
)

var (
	patternAnsiColors   = regexp.MustCompile(`\x1B\[([0-9]{1,3}(;[0-9]{1,2})?)?[mGK]`)
	patternKanikoLogs   = regexp.MustCompile(`time=".*" level=.* msg="(?P<message>.*)"`)
	patternSpecialChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

var templateFuncs = template.FuncMap{
	"sanitize": sanitize,
}

// renderTemplate Parse and execute given template by its name, taking care of inheritance,
// then write it on the disk, inside the report folder.
func (r Report) renderTemplate(name string, reportOpts Options, reportData any) error {
	// The order matter for inheritance
	files := []string{
		path.Join(templatesDir, "_layout.go.html"),               // base layout
		path.Join(templatesDir, "_nav.go.html"),                  // navbar
		path.Join(templatesDir, fmt.Sprintf("%s.go.html", name)), // report page
	}

	tpl, err := template.New("layout").Funcs(templateFuncs).ParseFS(templatesFS, files...)
	if err != nil {
		return err
	}

	writer, err := os.Create(fmt.Sprintf("%s.html", path.Join(r.GetRootDir(), name)))
	if err != nil {
		return err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(writer)

	templateData := map[string]any{
		"Name": name,
		"Opt":  reportOpts,
		"Data": reportData,
	}

	return tpl.ExecuteTemplate(writer, "layout", templateData)
}

// isTestRunnerEnabled return true if given types.TestRunner is enabled. False instead.
func isTestRunnerEnabled(name string, testRunners []types.TestRunner) bool {
	for _, runner := range testRunners {
		if name == runner.Name() {
			return true
		}
	}

	return false
}

// sortBuildReport sort BuildReport by image name.
func sortBuildReport(buildReports []BuildReport) []BuildReport {
	sort.SliceStable(buildReports, func(i, j int) bool {
		return buildReports[i].Image.ShortName < buildReports[j].Image.ShortName
	})

	return buildReports
}

func beautifyBuildsLogs(rawBuildLogs []byte) string {
	unescapedBuildLogs := RemoveTerminalColors(rawBuildLogs)
	return StripKanikoBuildLogs(unescapedBuildLogs)
}

// sanitize removes characters from string that are not allowed in document.querySelector calls.
func sanitize(input string) string {
	return patternSpecialChars.ReplaceAllString(input, "")
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
