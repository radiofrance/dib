package report

import (
	"os"
	"path"
)

func parseTrivyResults(buildReport BuildReport, imageDataBaseDir string, trivyScanLogsDir string) {
	rawTrivyScanLogs, err := os.ReadFile(path.Join(trivyScanLogsDir, buildReport.Image.ShortName) + ".json")
	if err != nil {
		return
	}

	err = os.WriteFile(path.Join(imageDataBaseDir, "trivy.json"), rawTrivyScanLogs, 0o644) //nolint:gosec
	if err != nil {
		return
	}
}
