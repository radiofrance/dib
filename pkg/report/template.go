package report

import (
	"fmt"
	"os"
	"time"

	svelteclient "github.com/radiofrance/dib/client"
	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/graphviz"
	"github.com/radiofrance/dib/pkg/types"
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
			WithTrivy:      isTestRunnerEnabled(types.TestRunnerTrivy, testRunners),
		},
	}
}

// Generate func create a html report on the filesystem at the end of the "dib build" cmd.
func Generate(dibReport *Report, dag *dag.DAG) error {
	if len(dibReport.BuildReports) == 0 {
		return nil
	}

	logger.Infof("Generating HTML report in the %s folder...", dibReport.GetRootDir())

	if err := os.MkdirAll(dibReport.GetRootDir(), 0o755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("unable to create report folder: %w", err)
	}

	if err := graphviz.GenerateGraph(dag, dibReport.GetRootDir()); err != nil {
		return fmt.Errorf("unable to generate graph: %w", err)
	}

	if err := copyAssetsFiles(svelteclient.AssetsFS, svelteclient.AssetsRootDir, dibReport); err != nil {
		return fmt.Errorf("unable to create report static file: %w", err)
	}

	if err := generateReportData(dibReport, dag); err != nil {
		return fmt.Errorf("unable to render report templates: %w", err)
	}

	logger.Infof("Generated HTML report: \"%s\"", dibReport.GetURL())

	return nil
}
