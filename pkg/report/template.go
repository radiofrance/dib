package report

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/graphviz"
	"github.com/radiofrance/dib/pkg/junit"
	"github.com/radiofrance/dib/pkg/logger"
	"github.com/radiofrance/dib/pkg/types"
)

const (
	assetsDir    = "assets"
	templatesDir = "templates"

	statusSkipped       = 0
	testSkippedWording  = "Goss tests skipped because the docker image failed to build"
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
	version,
	rootDir string,
	disableGenerateGraph bool,
	testRunners []types.TestRunner,
	buildOpts string,
) *Report {
	generationDate := time.Now()

	return &Report{
		BuildReports: []BuildReport{},
		Options: Options{
			RootDir:        rootDir,
			Name:           generationDate.Format("20060102150405"),
			GenerationDate: generationDate,
			Version:        fmt.Sprintf("v%s", version),
			BuildOpts:      buildOpts,
			WithGraph:      !disableGenerateGraph,
			WithGoss:       isTestRunnerEnabled(types.TestRunnerGoss, testRunners),
		},
	}
}

// Generate create a Report on the filesystem.
func Generate(ctx context.Context, dibReport *Report, dag *dag.DAG) error {
	if len(dibReport.BuildReports) == 0 {
		return nil
	}

	logger.Infof("Generating HTML report in the %s folder...", dibReport.GetRootDir())

	err := os.MkdirAll(dibReport.GetRootDir(), 0o750)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("unable to create report folder: %w", err)
	}

	err = graphviz.GenerateGraph(ctx, dag, dibReport.GetRootDir())
	if err != nil {
		return fmt.Errorf("unable to generate graph: %w", err)
	}

	err = copyAssetsFiles(dibReport)
	if err != nil {
		return fmt.Errorf("unable to create report static file: %w", err)
	}

	err = renderTemplates(dibReport, dag)
	if err != nil {
		return fmt.Errorf("unable to render report templates: %w", err)
	}

	logger.Infof("Generated HTML report: \"%s\"", dibReport.GetURL())

	return nil
}

// copyAssetsFiles iterate recursively on the "assets" embed filesystem and copy it inside the report folder.
func copyAssetsFiles(dibReport *Report) error {
	return fs.WalkDir(assetsFS, assetsDir, func(itemPath string, itemEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if itemEntry.IsDir() {
			return os.MkdirAll(path.Join(dibReport.GetRootDir(), itemPath), 0o750)
		}

		data, err := fs.ReadFile(assetsFS, itemPath)
		if err != nil {
			return err
		}

		err = os.WriteFile(path.Join(dibReport.GetRootDir(), itemPath), data, 0o644)
		if err != nil {
			return err
		}

		return nil
	})
}

// renderTemplates compile & render Report templates files then create it inside the report folder.
func renderTemplates(dibReport *Report, dag *dag.DAG) error {
	// Generate index.html
	err := dibReport.renderTemplate("index", dibReport.Options, sortBuildReport(dibReport.BuildReports))
	if err != nil {
		return err
	}

	// Generate build.html
	buildLogsData := parseBuildLogs(dibReport)

	err = dibReport.renderTemplate("build", dibReport.Options, buildLogsData)
	if err != nil {
		return err
	}

	// Generate debug.html
	err = dibReport.renderTemplate("debug", dibReport.Options, dag.ListImage())
	if err != nil {
		return err
	}

	// Generate graph.html
	if dibReport.Options.WithGraph {
		err := dibReport.renderTemplate("graph", dibReport.Options, nil)
		if err != nil {
			return err
		}
	}

	// Generate test.html
	if dibReport.Options.WithGoss {
		gossLogsData := parseGossLogs(dibReport)

		err := dibReport.renderTemplate("test", dibReport.Options, gossLogsData)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseBuildLogs iterate over built Dockerfiles and read their respective build logs file.
// Then, it put in a map that will be used later in Go template.
func parseBuildLogs(dibReport *Report) map[string]string {
	buildLogsData := make(map[string]string)

	for _, buildReport := range dibReport.BuildReports {
		if buildReport.BuildStatus == statusSkipped {
			buildLogsData[buildReport.Image.ShortName] = buildSkippedWording
			continue
		}

		rawImageBuildLogs, err := os.ReadFile(path.Join(dibReport.GetBuildReportDir(), buildReport.Image.ShortName) + ".txt")
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
func parseGossLogs(dibReport *Report) map[string]any {
	gossTestsLogsData := make(map[string]any)

	for _, buildReport := range dibReport.BuildReports {
		if buildReport.TestsStatus == statusSkipped {
			gossTestsLogsData[buildReport.Image.ShortName] = testSkippedWording
			continue
		}

		gossTestLogsFile := fmt.Sprintf("%s/junit-%s.xml", dibReport.GetJunitReportDir(), buildReport.Image.ShortName)

		rawGossTestLogs, err := os.ReadFile(gossTestLogsFile) //nolint:gosec
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
