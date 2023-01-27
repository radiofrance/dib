package dib

import (
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"os"
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

	if err := os.MkdirAll(dibReport.GetJunitReportDir(), 0o755); err != nil {
		return err
	}

	errG := new(errgroup.Group)
	for _, runner := range testRunners {
		runner := runner
		errG.Go(func() error {
			if !runner.Supports(opts) {
				return nil
			}
			if err := runner.RunTest(opts); err != nil {
				logrus.Errorf("Test runner %s failed on image %s with error: %v",
					runner.Name(), opts.ImageName, err)
				return err
			}
			return nil
		})
	}
	return errG.Wait()
}
