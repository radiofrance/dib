package report

import (
	"fmt"
	"html/template"
	"os"
	"path"
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
	patternSpecialChars = regexp.MustCompile(`[^a-zA-Z0-9 _-]`)
)

var templateFuncs = template.FuncMap{
	"sanitize": sanitize,
}

func InitDibReport(dir string) (*Report, error) {
	generationDate := time.Now()
	name := generationDate.Format("20060102150405") // equivalent of `$ date +%Y%m%d%H%M%S`

	dibReport := Report{
		Name:           name,
		Dir:            dir,
		GenerationDate: generationDate,
		BuildReports:   []BuildReport{},
	}

	// Create Report root directory
	if err := os.MkdirAll(path.Join(dibReport.Dir, dibReport.Name), 0o755); err != nil {
		return nil, err
	}

	// Create Report build logs directory
	if err := os.MkdirAll(dibReport.GetBuildLogsDir(), 0o755); err != nil {
		return nil, err
	}

	// Create Report Junit reports directory
	if err := os.MkdirAll(dibReport.GetJunitReportDir(), 0o755); err != nil {
		return nil, err
	}

	// Create Trivy scan reports directory
	if err := os.MkdirAll(dibReport.GetTrivyReportDir(), 0o755); err != nil {
		return nil, err
	}

	return &dibReport, nil
}

// renderTemplate Parse and execute given template by its name, taking care of inheritance,
// then write it on the disk, inside the report folder.
func (r Report) renderTemplate(name string, data any) error {
	// The order matter for inheritance
	files := []string{
		path.Join(templatesDir, "layout.go.html"),
		path.Join(templatesDir, fmt.Sprintf("%s.go.html", name)),
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

	if err = tpl.ExecuteTemplate(writer, "layout", data); err != nil {
		return err
	}

	return nil
}

// sortBuildReport sort BuildReport by image name.
func sortBuildReport(buildReports []BuildReport) []BuildReport {
	sort.SliceStable(buildReports, func(i, j int) bool {
		return buildReports[i].ImageName < buildReports[j].ImageName
	})
	return buildReports
}

// sortTrivyScan sort Trivy scan report by severity.
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
	return path.Join(r.Dir, r.Name)
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
