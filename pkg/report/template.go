package report

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/graphviz"
	"github.com/radiofrance/dib/pkg/junit"
	"github.com/radiofrance/dib/pkg/trivy"
	"github.com/radiofrance/dib/pkg/types"
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

// Init function initialise and return a Report struct.
func Init(
	version string,
	rootDir string,
	disableGenerateGraph bool,
	testRunners []types.TestRunner,
	buildCfg string,
) *Report {
	generationDate := time.Now()
	return &Report{
		BuildReports: []BuildReport{},
		Options: Options{
			RootDir:        rootDir,
			Name:           generationDate.Format("20060102150405"),
			GenerationDate: generationDate,
			Version:        fmt.Sprintf("v%s", version),
			BuildCfg:       buildCfg,
			WithGraph:      !disableGenerateGraph,
			WithGoss:       isTestRunnerEnabled(types.TestRunnerGoss, testRunners),
			WithTrivy:      isTestRunnerEnabled(types.TestRunnerTrivy, testRunners),
		},
	}
}

// Generate create a Report on the filesystem.
func Generate(dibReport Report, dag dag.DAG) error {
	if len(dibReport.BuildReports) == 0 {
		return nil
	}

	logger.Debugf("generating html report in the %s folder...", dibReport.GetRootDir())
	if err := graphviz.GenerateGraph(&dag, dibReport.GetRootDir()); err != nil {
		return fmt.Errorf("unable to generate graph: %w", err)
	}

	if err := copyAssetsFiles(dibReport); err != nil {
		return fmt.Errorf("unable to create report static file: %w", err)
	}

	if err := renderTemplates(dibReport, dag); err != nil {
		return fmt.Errorf("unable to render report templates: %w", err)
	}

	logger.Infof("Generated HTML report: \"%s\"", dibReport.GetURL())

	return nil
}

// copyAssetsFiles iterate recursively on the "assets" embed filesystem and copy it inside the report folder.
func copyAssetsFiles(dibReport Report) error {
	return fs.WalkDir(assetsFS, assetsDir, func(itemPath string, itemEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if itemEntry.IsDir() {
			return os.MkdirAll(path.Join(dibReport.GetRootDir(), itemPath), 0o755)
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
func renderTemplates(dibReport Report, dag dag.DAG) error {
	// Generate index.html
	if err := dibReport.renderTemplate("index", dibReport.Options, sortBuildReport(dibReport.BuildReports)); err != nil {
		return err
	}

	// Generate build.html
	buildLogsData := parseBuildLogs(dibReport)
	if err := dibReport.renderTemplate("build", dibReport.Options, buildLogsData); err != nil {
		return err
	}

	// Generate debug.html
	if err := dibReport.renderTemplate("debug", dibReport.Options, dag.ListImage()); err != nil {
		return err
	}

	// Generate graph.html
	if dibReport.Options.WithGraph {
		if err := dibReport.renderTemplate("graph", dibReport.Options, nil); err != nil {
			return err
		}
	}

	// Generate test.html
	if dibReport.Options.WithGoss {
		gossLogsData := parseGossLogs(dibReport)
		if err := dibReport.renderTemplate("test", dibReport.Options, gossLogsData); err != nil {
			return err
		}
	}

	// Generate scan.html
	if dibReport.Options.WithTrivy {
		trivyScanLogsData := parseTrivyReports(dibReport)
		if err := dibReport.renderTemplate("scan", dibReport.Options, trivyScanLogsData); err != nil {
			return err
		}
	}

	return nil
}

// parseBuildLogs iterate over built Dockerfiles and read their respective build logs file.
// Then, it put in a map that will be used later in Go template.
func parseBuildLogs(dibReport Report) map[string]string {
	buildLogsData := make(map[string]string)

	for _, buildReport := range dibReport.BuildReports {
		if buildReport.BuildStatus == statusSkipped {
			buildLogsData[buildReport.Image.ShortName] = buildSkippedWording
			continue
		}

		rawImageBuildLogs, err := os.ReadFile(path.Join(dibReport.GetBuildLogsDir(), buildReport.Image.ShortName) + ".txt")
		if err != nil {
			buildLogsData[buildReport.Image.ShortName] = err.Error()
			continue
		}

		buildLogsData[buildReport.Image.ShortName] = beautifyBuildsLogs(rawImageBuildLogs)
	}

	return buildLogsData
}

// parseGossLogs iterate over each Goss tests (in junit format) and read their respective logs file.
// Then, it put in a map that will be used later in Go template.
func parseGossLogs(dibReport Report) map[string]any {
	gossTestsLogsData := make(map[string]any)

	for _, buildReport := range dibReport.BuildReports {
		if buildReport.TestsStatus == statusSkipped {
			gossTestsLogsData[buildReport.Image.ShortName] = testSkippedWording
			continue
		}

		gossTestLogsFile := fmt.Sprintf("%s/junit-%s.xml", dibReport.GetJunitReportDir(), buildReport.Image.ShortName)
		rawGossTestLogs, err := os.ReadFile(gossTestLogsFile)
		if err != nil {
			gossTestsLogsData[buildReport.Image.ShortName] = err.Error()
			continue
		}

		parsedDgossTestLogs, err := junit.ParseRawLogs(rawGossTestLogs)
		if err != nil {
			gossTestsLogsData[buildReport.Image.ShortName] = err.Error()
			continue
		}

		gossTestsLogsData[buildReport.Image.ShortName] = parsedDgossTestLogs
	}

	return gossTestsLogsData
}

// parseTrivyReports iterates over each trivy report (in json format) and read their respective report file.
// Then, the reports are put together in a map that will be used later in Go template.
func parseTrivyReports(dibReport Report) map[string]any {
	trivyScanData := make(map[string]any)

	for _, buildReport := range dibReport.BuildReports {
		if buildReport.TestsStatus == statusSkipped {
			trivyScanData[buildReport.Image.ShortName] = scanSkippedWording
			continue
		}

		trivyScanFile := fmt.Sprintf("%s/%s.json", dibReport.GetTrivyReportDir(), buildReport.Image.ShortName)
		rawTrivyReport, err := os.ReadFile(trivyScanFile)
		if err != nil {
			trivyScanData[buildReport.Image.ShortName] = err.Error()
			continue
		}

		parsedTrivyReport, err := trivy.ParseTrivyReport(rawTrivyReport)
		if err != nil {
			trivyScanData[buildReport.Image.ShortName] = err.Error()
			continue
		}

		trivyScanData[buildReport.Image.ShortName] = sortTrivyScan(parsedTrivyReport)
	}

	return trivyScanData
}
