package report

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/graphviz"
	"github.com/radiofrance/dib/pkg/junit"
	"github.com/radiofrance/dib/pkg/trivy"
	"github.com/sirupsen/logrus"
)

const (
	assetsDir    = "assets"
	templatesDir = "templates"

	statusSkipped       = 0
	testSkippedWording  = "Goss tests skipped because the docker image failed to build"
	scanSkippedWording  = "Trivy scans skipped because the docker image failed to build"
	buildSkippedWording = "Build skipped because a parent image failed to build"
)

var (
	//go:embed assets/*
	assetsFS embed.FS
	//go:embed templates/*.go.html
	templatesFS embed.FS
)

// Generate create a Report on the filesystem.
func Generate(dibReport Report, dag dag.DAG) error {
	if len(dibReport.BuildReports) == 0 {
		return nil
	}

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

	finalReportURL := getReportURL(dibReport)
	logrus.Infof("Generated HTML report: \"%s\"", finalReportURL)

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
	data["buildReport"] = sortBuildReport(dibReport.BuildReports)

	// Generate index.html
	if err := dibReport.renderTemplate("index", data); err != nil {
		return err
	}

	// Generate graph.html
	if err := dibReport.renderTemplate("graph", nil); err != nil {
		return err
	}

	// Generate build.html
	buildLogsData := parseBuildLogs(dibReport)
	if err := dibReport.renderTemplate("build", buildLogsData); err != nil {
		return err
	}

	// Generate test.html
	gossLogsData := parseGossLogs(dibReport)
	if err := dibReport.renderTemplate("test", gossLogsData); err != nil {
		return err
	}

	// Generate scan.html
	trivyScanLogsData := parseTrivyReports(dibReport)
	if err := dibReport.renderTemplate("scan", trivyScanLogsData); err != nil {
		return err
	}

	return nil
}

// getReportURL return a string representing the path from which we can browse HTML report.
func getReportURL(dibReport Report) string {
	// GitLab context
	gitlabJobURL := os.Getenv("CI_JOB_URL")
	if gitlabJobURL != "" {
		return fmt.Sprintf("%s/artifacts/file/%s/index.html", gitlabJobURL, dibReport.GetRootDir())
	}

	// Local context
	finalReportURL, err := filepath.Abs(dibReport.GetRootDir())
	if err != nil {
		return dibReport.GetRootDir()
	}

	return fmt.Sprintf("file://%s/index.html", finalReportURL)
}

// parseBuildLogs iterate over built Dockerfiles and read their respective build logs file.
// Then, it put in a map that will be used later in Go template.
func parseBuildLogs(dibReport Report) map[string]string {
	buildLogsData := make(map[string]string)

	for _, buildReport := range dibReport.BuildReports {
		if buildReport.BuildStatus == statusSkipped {
			buildLogsData[buildReport.ImageName] = buildSkippedWording
			continue
		}

		rawImageBuildLogs, err := os.ReadFile(path.Join(dibReport.GetBuildLogsDir(), buildReport.ImageName) + ".txt")
		if err != nil {
			buildLogsData[buildReport.ImageName] = err.Error()
			continue
		}

		buildLogsData[buildReport.ImageName] = beautifyBuildsLogs(rawImageBuildLogs)
	}

	return buildLogsData
}

// parseGossLogs iterate over each Goss tests (in junit format) and read their respective logs file.
// Then, it put in a map that will be used later in Go template.
func parseGossLogs(dibReport Report) map[string]any {
	gossTestsLogsData := make(map[string]any)

	for _, buildReport := range dibReport.BuildReports {
		if buildReport.TestsStatus == statusSkipped {
			gossTestsLogsData[buildReport.ImageName] = testSkippedWording
			continue
		}

		gossTestLogsFile := fmt.Sprintf("%s/junit-%s.xml", dibReport.GetJunitReportDir(), buildReport.ImageName)
		rawGossTestLogs, err := os.ReadFile(gossTestLogsFile)
		if err != nil {
			gossTestsLogsData[buildReport.ImageName] = err.Error()
			continue
		}

		parsedDgossTestLogs, err := junit.ParseRawLogs(rawGossTestLogs)
		if err != nil {
			gossTestsLogsData[buildReport.ImageName] = err.Error()
			continue
		}

		gossTestsLogsData[buildReport.ImageName] = parsedDgossTestLogs
	}

	return gossTestsLogsData
}

// parseTrivyReports iterates over each trivy report (in json format) and read their respective report file.
// Then, the reports are put together in a map that will be used later in Go template.
func parseTrivyReports(dibReport Report) map[string]any {
	trivyScanData := make(map[string]any)

	for _, buildReport := range dibReport.BuildReports {
		if buildReport.TestsStatus == statusSkipped {
			trivyScanData[buildReport.ImageName] = scanSkippedWording
			continue
		}

		trivyScanFile := fmt.Sprintf("%s/%s.json", dibReport.GetTrivyReportDir(), buildReport.ImageName)
		rawTrivyReport, err := os.ReadFile(trivyScanFile)
		if err != nil {
			trivyScanData[buildReport.ImageName] = err.Error()
			continue
		}

		parsedTrivyReport, err := trivy.ParseTrivyReport(rawTrivyReport)
		if err != nil {
			trivyScanData[buildReport.ImageName] = err.Error()
			continue
		}

		trivyScanData[buildReport.ImageName] = sortTrivyScan(parsedTrivyReport)
	}

	return trivyScanData
}
