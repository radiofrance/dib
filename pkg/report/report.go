package report

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
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
	Name           string
	Dir            string
	GenerationDate time.Time
	BuildReports   []BuildReport
	Version        string
}

// BuildReport holds the status of the build/tests.
type BuildReport struct {
	ImageName      string
	BuildStatus    BuildStatus
	TestsStatus    TestsStatus
	FailureMessage string
}

// WithError returns a BuildReport.
func (r BuildReport) WithError(err error) BuildReport {
	r.BuildStatus = BuildStatusError
	r.FailureMessage = err.Error()

	return r
}

// PrintReports prints the reports to the user.
func PrintReports(reports []BuildReport) {
	logrus.Info("Build report")
	for _, report := range reports {
		switch report.BuildStatus {
		case BuildStatusSuccess:
			logrus.Infof("\t[%s]: SUCCESS", report.ImageName)
		case BuildStatusSkipped:
			logrus.Infof("\t[%s]: SKIPPED", report.ImageName)
		case BuildStatusError:
			logrus.Errorf("\t[%s]: FAILURE: %s", report.ImageName, report.FailureMessage)
		}
	}

	logrus.Info("Tests report")
	for _, report := range reports {
		switch report.TestsStatus {
		case TestsStatusPassed:
			logrus.Infof("\t[%s]: PASSED", report.ImageName)
		case TestsStatusSkipped:
			logrus.Infof("\t[%s]: SKIPPED", report.ImageName)
		case TestsStatusFailed:
			logrus.Errorf("\t[%s]: FAILED: %s", report.ImageName, report.FailureMessage)
		}
	}
}

// CheckError looks for failures in build reports and returns an error if any is found.
func CheckError(reports []BuildReport) error {
	for _, report := range reports {
		if report.BuildStatus == BuildStatusError {
			return fmt.Errorf("one of the image build failed, see the report for more details")
		}

		if report.TestsStatus == TestsStatusFailed {
			return fmt.Errorf("some tests failed, see report for more details")
		}
	}

	return nil
}
