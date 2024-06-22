package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/radiofrance/dib/pkg/junit"
)

func parseGossResults(buildReport BuildReport, imageDataBaseDir string, gossTestsLogsDir string) {
	if buildReport.TestsStatus == TestsStatusSkipped {
		return
	}

	gossReportPath := path.Join(gossTestsLogsDir, fmt.Sprintf("junit-%s.xml", buildReport.Image.ShortName))
	rawGossTestLogs, err := os.ReadFile(gossReportPath)
	if err != nil {
		return
	}

	parsedGossTestLogs, err := junit.ParseRawLogs(rawGossTestLogs)
	if err != nil {
		return
	}

	data, err := json.MarshalIndent(parsedGossTestLogs, "", "\t")
	if err != nil {
		return
	}

	err = os.WriteFile(path.Join(imageDataBaseDir, "goss.json"), data, 0o644) //nolint:gosec
	if err != nil {
		return
	}
}
