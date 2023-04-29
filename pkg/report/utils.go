package report

import (
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/radiofrance/dib/pkg/trivy"
)

const (
	BuildLogsDir   = "builds"
	JunitReportDir = "junit"
	TrivyReportDir = "trivy"
)

var (
	patternAnsiColors   = regexp.MustCompile(`\x1B\[([0-9]{1,3}(;[0-9]{1,2})?)?[mGK]`)
	patternKanikoLogs   = regexp.MustCompile(`time=".*" level=.* msg="(?P<message>.*)"`)
	patternSpecialChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

var templateFuncs = template.FuncMap{
	"sanitize": sanitize,
}

func InitDibReport(dir, version string, enabledTestsRunner []string, disableGenerateGraph bool) *Report {
	generationDate := time.Now()
	name := generationDate.Format("20060102150405") // equivalent of `$ date +%Y%m%d%H%M%S`

	return &Report{
		Name:         name,
		BuildReports: []BuildReport{},
		Options: Options{
			Version:        fmt.Sprintf("v%s", version),
			GenerationDate: generationDate,
			RootDir:        dir,
			WithGraph:      false,
			WithGoss:       false,
			WithTrivy:      false,
		},
	}
}

// renderTemplate Parse and execute given template by its name, taking care of inheritance,
// then write it on the disk, inside the report folder.
func (r Report) renderTemplate(name string, reportOpt Options, reportData any) error {
	// The order matter for inheritance
	files := []string{
		path.Join(templatesDir, "_layout.go.html"),               // base layout
		path.Join(templatesDir, "_nav.go.html"),                  // navbar
		path.Join(templatesDir, "_functions.go.html"),            // helpers & utils functions
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
		"Opt":  reportOpt,
		"Data": reportData,
	}
	return tpl.ExecuteTemplate(writer, "layout", templateData)
}

// sortBuildReport sort BuildReport by image name.
func sortBuildReport(buildReports []BuildReport) []BuildReport {
	sort.SliceStable(buildReports, func(i, j int) bool {
		return buildReports[i].ImageName < buildReports[j].ImageName
	})
	return buildReports
}

// sortTrivyScan sorts Trivy scan reports by severity.
func sortTrivyScan(parsedTrivyReport trivy.ScanReport) trivy.ScanReport {
	order := map[string]int{
		"CRITICAL": 1,
		"HIGH":     2,
		"MEDIUM":   3,
		"LOW":      4,
		"UNKNOWN":  5,
	}

	for _, result := range parsedTrivyReport.Results {
		sort.SliceStable(result.Vulnerabilities, func(i, j int) bool {
			return order[result.Vulnerabilities[i].Severity] < order[result.Vulnerabilities[j].Severity]
		})
	}
	return parsedTrivyReport
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

// StripKanikoBuildLogs Improve readability of kaniko builds logs by removing unwanted stuff from a logrus
// standard logs message.
func StripKanikoBuildLogs(input []byte) string {
	results := patternKanikoLogs.ReplaceAll(input, []byte("$message"))

	return string(results)
}

// GetRootDir return the path of the Report "root" directory.
func (r Report) GetRootDir() string {
	return path.Join(r.Options.RootDir, r.Name)
}

// GetBuildLogsDir return the path of the Report "builds" directory.
func (r Report) GetBuildLogsDir() string {
	return path.Join(r.GetRootDir(), BuildLogsDir)
}

// GetJunitReportDir return the path of the Report "Junit reports" directory.
func (r Report) GetJunitReportDir() string {
	return path.Join(r.GetRootDir(), JunitReportDir)
}

// GetTrivyReportDir return the path of the Report "Trivy reports" directory.
func (r Report) GetTrivyReportDir() string {
	return path.Join(r.GetRootDir(), TrivyReportDir)
}

// GetReportURL return a string representing the path from which we can browse HTML report.
func (r Report) GetReportURL() string {
	// GitLab context
	gitlabJobURL := os.Getenv("CI_JOB_URL")
	if gitlabJobURL != "" {
		return fmt.Sprintf("%s/artifacts/file/%s/index.html", gitlabJobURL, r.GetRootDir())
	}

	// Local context
	finalReportURL, err := filepath.Abs(r.GetRootDir())
	if err != nil {
		return r.GetRootDir()
	}

	return fmt.Sprintf("file://%s/index.html", finalReportURL)
}
