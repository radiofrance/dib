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

const (
	ScanStatusSkipped ScanStatus = iota
	ScanStatusPassed
	ScanStatusFailed
)

type (
	BuildStatus int
	TestsStatus int
	ScanStatus  int
)

type Report struct {
	Name           string
	Dir            string
	GenerationDate time.Time
	BuildReports   []BuildReport
}

// BuildReport holds the status of the build/tests.
type BuildReport struct {
	ImageName      string
	BuildStatus    BuildStatus
	TestsStatus    TestsStatus
	ScanStatus     ScanStatus
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

	logrus.Info("Scan report")
	for _, report := range reports {
		switch report.ScanStatus {
		case ScanStatusPassed:
			logrus.Infof("\t[%s]: PASSED", report.ImageName)
		case ScanStatusSkipped:
			logrus.Infof("\t[%s]: SKIPPED", report.ImageName)
		case ScanStatusFailed:
			logrus.Errorf("\t[%s]: FAILED: %s", report.ImageName, report.FailureMessage)
		}
	}
}

// CheckError looks for failures in build reports and returns an error if any is found.
func CheckError(reports []BuildReport) error {
	for _, report := range reports {
		if report.BuildStatus == BuildStatusError {
			return fmt.Errorf("one of the image build failed, see report for more details")
		}

		if report.TestsStatus == TestsStatusFailed {
			return fmt.Errorf("some tests failed, see report for more details")
		}

		if report.TestsStatus == TestsStatusFailed {
			return fmt.Errorf("some scan failed, see report for more details")
		}
	}

	return nil
}
