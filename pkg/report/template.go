package report

import (
	"fmt"
	"io/fs"
	"os"
	"path"
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

// Generate create a Report on the filesystem.
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

	if err := copyAssetsFiles(dibReport); err != nil {
		return fmt.Errorf("unable to create report static file: %w", err)
	}

	if err := generateReportData(dibReport, dag); err != nil {
		return fmt.Errorf("unable to render report templates: %w", err)
	}

	logger.Infof("Generated HTML report: \"%s\"", dibReport.GetURL())

	return nil
}

// copyAssetsFiles iterate recursively on the "client" embed filesystem and copy it inside the report folder.
func copyAssetsFiles(dibReport *Report) error {
	subFS, err := fs.Sub(svelteclient.AssetsFS, svelteclient.AssetsRootDir)
	if err != nil {
		return err
	}

	return fs.WalkDir(subFS, ".", func(embedFilePath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			return os.MkdirAll(path.Join(dibReport.GetRootDir(), embedFilePath), 0o755)
		}

		data, err := fs.ReadFile(subFS, embedFilePath)
		if err != nil {
			return err
		}

		return os.WriteFile(path.Join(dibReport.GetRootDir(), embedFilePath), data, 0o644) //nolint:gosec
	})
}
