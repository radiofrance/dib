package dib

import (
	"github.com/radiofrance/dib/pkg/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// testImage runs the tests on an image.
func testImage(testRunners []types.TestRunner, runTestOpts types.RunTestOptions) error {
	logrus.Infof("Running tests for \"%s\"", runTestOpts.ImageReference)
	errG := new(errgroup.Group)
	for _, runner := range testRunners {
		runner := runner
		errG.Go(func() error {
			if !runner.Supports(runTestOpts) {
				return nil
			}
			if err := runner.RunTest(runTestOpts); err != nil {
				logrus.Errorf("Test runner %s failed on image %s with error: %v", runner.Name(), runTestOpts.ImageName, err)
				return err
			}
			return nil
		})
	}
	return errG.Wait()
}
