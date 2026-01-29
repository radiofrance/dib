package report

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/logger"
)

const (
	BuildStatusSkipped BuildStatus = iota
	BuildStatusSuccess
	BuildStatusError
)

const (
	TestsStatusSkipped TestsStatus = iota
	TestsStatusPassed
	TestsStatusFailed
)

type (
	BuildStatus int
	TestsStatus int
)

type Report struct {
	Options      Options
	BuildReports []BuildReport
}

type Options struct {
	RootDir        string
	Name           string
	GenerationDate time.Time
	Version        string
	BuildOpts      string
	WithGraph      bool
	WithGoss       bool
}

// BuildReport holds the status of the build/tests.
type BuildReport struct {
	Image          dag.Image
	BuildStatus    BuildStatus
	TestsStatus    TestsStatus
	FailureMessage string
}

// GetRootDir return the path of the Report "root" directory.
func (r Report) GetRootDir() string {
	return path.Join(r.Options.RootDir, r.Options.Name)
}

// GetBuildReportDir return the path of the Report "builds" directory.
func (r Report) GetBuildReportDir() string {
	return path.Join(r.GetRootDir(), BuildReportDir)
}

// GetJunitReportDir return the path of the Report "Junit reports" directory.
func (r Report) GetJunitReportDir() string {
	return path.Join(r.GetRootDir(), JunitReportDir)
}

// GetURL return a string representing the path from which we can browse Report.
func (r Report) GetURL() string {
	// GitLab context
	gitlabJobURL := os.Getenv("CI_JOB_URL")
	if gitlabJobURL != "" {
		return fmt.Sprintf("%s/artifacts/file/%s/index.html", gitlabJobURL, r.GetRootDir())
	}

	// Local context
	finalReportURL, err := filepath.Abs(r.GetRootDir())
	if err != nil {
		return r.GetRootDir()
	}

	return fmt.Sprintf("file://%s/index.html", finalReportURL)
}

// Print display Report.BuildReports to the user.
func (r Report) Print() {
	logger.Infof("Build report")

	for _, buildReport := range r.BuildReports {
		switch buildReport.BuildStatus {
		case BuildStatusSuccess:
			logger.Infof("\t[%s]: SUCCESS", buildReport.Image.ShortName)
		case BuildStatusSkipped:
			logger.Infof("\t[%s]: SKIPPED", buildReport.Image.ShortName)
		case BuildStatusError:
			logger.Errorf("\t[%s]: FAILURE: %s", buildReport.Image.ShortName, buildReport.FailureMessage)
		}
	}

	logger.Infof("Tests report")

	for _, buildReport := range r.BuildReports {
		switch buildReport.TestsStatus {
		case TestsStatusPassed:
			logger.Infof("\t[%s]: PASSED", buildReport.Image.ShortName)
		case TestsStatusSkipped:
			logger.Infof("\t[%s]: SKIPPED", buildReport.Image.ShortName)
		case TestsStatusFailed:
			logger.Errorf("\t[%s]: FAILED: %s", buildReport.Image.ShortName, buildReport.FailureMessage)
		}
	}
}

// CheckError looks for failures in Report.BuildReports and returns an error if any is found.
func (r Report) CheckError() error {
	for _, buildReport := range r.BuildReports {
		if buildReport.BuildStatus == BuildStatusError {
			return fmt.Errorf("one of the image build failed, see the report for more details")
		}

		if buildReport.TestsStatus == TestsStatusFailed {
			return fmt.Errorf("some tests failed, see report for more details")
		}
	}

	return nil
}

// WithError returns a BuildReport.
func (r BuildReport) WithError(err error) BuildReport {
	r.BuildStatus = BuildStatusError
	r.FailureMessage = err.Error()

	return r
}
