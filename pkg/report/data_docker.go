package report

import (
	"os"
	"path"
)

func parseImageBuildLogs(buildReport BuildReport, imageDataBaseDir string, imageBuildLogsDir string) {
	if buildReport.BuildStatus == BuildStatusSkipped {
		return
	}

	rawImageBuildLogs, err := os.ReadFile(path.Join(imageBuildLogsDir, buildReport.Image.ShortName+".txt"))
	if err != nil {
		return
	}

	data := beautifyBuildsLogs(rawImageBuildLogs)
	err = os.WriteFile(path.Join(imageDataBaseDir, "docker.txt"), []byte(data), 0o644) //nolint:gosec
	if err != nil {
		return
	}
}
