package dib

import (
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/sirupsen/logrus"
)

// testImage runs the tests on an image.
func testImage(img *dag.Image, testRunners []types.TestRunner, dibReport *report.Report) error {
	ref := img.CurrentRef()
	logrus.Infof("Running tests for \"%s\"", ref)

	opts := types.RunTestOptions{
		ImageName:         img.ShortName,
		ImageReference:    ref,
		DockerContextPath: img.Dockerfile.ContextPath,
		ReportJunitDir:    dibReport.GetJunitReportDir(),
		ReportRootDir:     dibReport.GetRootDir(),
	}
	for _, runner := range testRunners {
		if !runner.Supports(opts) {
			continue
		}
		if err := runner.RunTest(opts); err != nil {
			return err
		}
	}
	return nil
}
