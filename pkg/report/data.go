package report

import (
	"fmt"
	"os"
	"path"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dag"
)

func generateReportData(report *Report, _ *dag.DAG) error {
	dataBaseDir := path.Join(report.GetRootDir(), "data")
	err := os.MkdirAll(dataBaseDir, 0o755)
	if err != nil {
		return err
	}

	generateReportMap(report)

	for _, buildReport := range report.BuildReports {
		imageDataBaseDir := path.Join(dataBaseDir, buildReport.Image.ShortName)
		if err = os.MkdirAll(imageDataBaseDir, 0o755); err != nil {
			return err
		}

		parseImageBuildLogs(buildReport, imageDataBaseDir, report.GetBuildReportDir())
		parseGossResults(buildReport, imageDataBaseDir, report.GetJunitReportDir())
		parseTrivyResults(buildReport, imageDataBaseDir, report.GetTrivyReportDir())
	}

	if err := cleanReport(report); err != nil {
		logger.Warnf("%s", err)
	}

	return nil
}

func generateReportMap(report *Report) {
	buildReportMap := "const dib_images = [\n"
	for _, buildReport := range report.BuildReports {
		buildReportMap += fmt.Sprintf("  '%s',\n", buildReport.Image.ShortName)
	}
	buildReportMap += "];\n"

	err := os.WriteFile(path.Join(report.GetRootDir(), "map.js"), []byte(buildReportMap), 0o644) //nolint:gosec
	if err != nil {
		return
	}
}

func cleanReport(report *Report) error {
	_, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.RemoveAll(report.GetBuildReportDir()); err != nil {
		return err
	}

	if err := os.RemoveAll(report.GetJunitReportDir()); err != nil {
		return err
	}

	if err := os.RemoveAll(report.GetTrivyReportDir()); err != nil {
		return err
	}

	if err := os.RemoveAll(path.Join(report.GetRootDir(), "dib.dot")); err != nil {
		return err
	}

	return nil
}
