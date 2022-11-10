package report

import (
	"fmt"
	"html/template"
	"os"
	"path"
	"regexp"
	"sort"
	"time"
)

const (
	RootReportDirectory = "reports"
	BuildLogsDir        = "builds"
	JunitReportDir      = "junit"
)

var (
	patternAnsiColors = regexp.MustCompile(`\x1B\[([0-9]{1,3}(;[0-9]{1,2})?)?[mGK]`)
	patternKanikoLogs = regexp.MustCompile(`time=".*" level=.* msg="(?P<message>.*)"`)
)

func InitDibReport() (*Report, error) {
	generationDate := time.Now()
	name := generationDate.Format("20060102150405") // equivalent of `$ date +%Y%m%d%H%M%S`

	// If we are in dev|test mode, generate a static report name for convenience
	_, exist := os.LookupEnv("IS_TEST")
	if exist {
		name = "test_report"
	}

	dibReport := Report{
		Name:           name,
		GenerationDate: generationDate,
		BuildReports:   []BuildReport{},
	}

	// Create Report root directory
	if err := os.MkdirAll(path.Join(RootReportDirectory, dibReport.Name), 0o755); err != nil {
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

	tpl, err := template.ParseFS(templatesFS, files...)
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

	// We always execute root template, which also render sub templates too
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

func beautifyBuildsLogs(rawBuildLogs []byte) string {
	unescapedBuildLogs := RemoveTerminalColors(rawBuildLogs)
	return StripKanikoBuildLogs(unescapedBuildLogs)
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
	return path.Join(RootReportDirectory, r.Name)
}

// GetBuildLogsDir return the path of the Report "builds" directory.
func (r Report) GetBuildLogsDir() string {
	return path.Join(r.GetRootDir(), BuildLogsDir)
}

// GetJunitReportDir return the path of the Report "Junit reports" directory.
func (r Report) GetJunitReportDir() string {
	return path.Join(r.GetRootDir(), JunitReportDir)
}
