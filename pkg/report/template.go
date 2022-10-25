package report

import (
	"embed"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/graphviz"
	"github.com/sirupsen/logrus"
)

const (
	assetsDir    = "assets"
	templatesDir = "templates"
)

var (
	//go:embed assets/*
	assetsFS embed.FS
	//go:embed templates/*.go.html
	templatesFS embed.FS
)

// Generate create a Report on the filesystem.
func Generate(dibReport Report, dag dag.DAG) error {
	logrus.Infof("generating html report in the %s folder...", dibReport.GetRootDir())
	if err := graphviz.GenerateGraph(&dag, dibReport.GetRootDir()); err != nil {
		return fmt.Errorf("unable to generate graph: %w", err)
	}

	if err := copyAssetsFiles(dibReport); err != nil {
		return fmt.Errorf("unable to create report static file: %w", err)
	}

	if err := renderTemplates(dibReport); err != nil {
		return fmt.Errorf("unable to render report templates: %w", err)
	}

	return nil
}

// copyAssetsFiles iterate recursively on the "assets" embed filesystem and copy it inside the report folder.
func copyAssetsFiles(dibReport Report) error {
	return fs.WalkDir(assetsFS, assetsDir, func(itemPath string, itemEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if itemEntry.IsDir() {
			if err = os.MkdirAll(path.Join(dibReport.GetRootDir(), itemPath), 0o755); err != nil {
				return err
			}
			return nil
		}

		data, err := fs.ReadFile(assetsFS, itemPath)
		if err != nil {
			return err
		}

		err = os.WriteFile(path.Join(dibReport.GetRootDir(), itemPath), data, 0o644) //nolint:gosec
		if err != nil {
			return err
		}

		return nil
	})
}

// renderTemplates compile & render Report templates files then create it inside the report folder.
func renderTemplates(dibReport Report) error {
	data := make(map[string]any)
	data["buildUID"] = dibReport.Name
	data["generationDate"] = dibReport.GenerationDate
	data["buildReport"] = dibReport.BuildReports

	// Generate index.html
	if err := dibReport.renderTemplate("index", data); err != nil {
		return err
	}

	// Generate graph.html
	if err := dibReport.renderTemplate("graph", nil); err != nil {
		return err
	}

	// Generate build.html
	buildLogsData, err := parseBuildLogs(dibReport)
	if err != nil {
		return err
	}

	if err := dibReport.renderTemplate("build", buildLogsData); err != nil {
		return err
	}

	// Generate scan.html
	if err := dibReport.renderTemplate("scan", nil); err != nil {
		return err
	}

	// Generate test.html
	dgossLogsData, err := parseDgossLogs(dibReport)
	if err != nil {
		return err
	}

	if err := dibReport.renderTemplate("test", dgossLogsData); err != nil {
		return err
	}

	return nil
}

// parseBuildLogs iterate over built Dockerfiles and read their respective build logs file.
// Then, it put in a map that will be used later in Go template.
func parseBuildLogs(dibReport Report) (map[string]string, error) {
	buildLogsData := make(map[string]string)

	for _, buildReport := range dibReport.BuildReports {
		rawImageBuildLogs, err := os.ReadFile(path.Join(dibReport.GetBuildLogsDir(), buildReport.ImageName) + ".txt")
		if err != nil {
			return nil, err
		}

		buildLogsData[buildReport.ImageName] = string(rawImageBuildLogs)
	}

	return buildLogsData, nil
}

// parseDgossLogs iterate over each dgoss tests (in junit format) and read their respective logs file.
// Then, it put in a map that will be used later in Go template.
func parseDgossLogs(dibReport Report) (map[string]Testsuite, error) {
	dgossTestsLogsData := make(map[string]Testsuite)

	for _, buildReport := range dibReport.BuildReports {
		dgossTestLogsFile := fmt.Sprintf("%s/junit-%s.xml", dibReport.GetJunitReportDir(), buildReport.ImageName)
		rawDgossTestLogs, err := os.ReadFile(dgossTestLogsFile)
		if err != nil {
			return nil, err
		}

		parsedDgossTestLogs, err := convertJunitReportXmlToHumanReadableFormat(rawDgossTestLogs)
		if err != nil {
			return nil, err
		}

		dgossTestsLogsData[buildReport.ImageName] = parsedDgossTestLogs
	}

	return dgossTestsLogsData, nil
}

type Testsuite struct {
	XMLName   xml.Name   `xml:"testsuite"`
	Name      string     `xml:"name,attr"`
	Errors    string     `xml:"errors,attr"`
	Tests     string     `xml:"tests,attr"`
	Failures  string     `xml:"failures,attr"`
	Skipped   string     `xml:"skipped,attr"`
	Time      string     `xml:"time,attr"`
	Timestamp string     `xml:"timestamp,attr"`
	TestCase  []TestCase `xml:"testcase"`
}

type TestCase struct {
	XMLName   xml.Name `xml:"testcase"`
	ClassName string   `xml:"classname,attr"`
	File      string   `xml:"file,attr"`
	Name      string   `xml:"name,attr"`
	Time      string   `xml:"time,attr"`
	SystemOut string   `xml:"system-out"`
}

func convertJunitReportXmlToHumanReadableFormat(rawDgossTestLogs []byte) (Testsuite, error) {
	testSuite := Testsuite{}
	err := xml.Unmarshal(rawDgossTestLogs, &testSuite)
	if err != nil {
		return testSuite, err
	}

	return testSuite, nil
}
