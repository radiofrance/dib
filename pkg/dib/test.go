package dib

import (
	"github.com/radiofrance/dib/pkg/logger"
	"github.com/radiofrance/dib/pkg/types"
	"golang.org/x/sync/errgroup"
)

// testImage runs the tests on an image.
func testImage(testRunners []types.TestRunner, runTestOpts types.RunTestOptions) error {
	logger.Infof("Running tests for \"%s\"", runTestOpts.ImageReference)

	errG := new(errgroup.Group)
	for _, runner := range testRunners {
		errG.Go(func() error {
			if !runner.IsConfigured(runTestOpts) {
				return nil
			}

			err := runner.RunTest(runTestOpts)
			if err != nil {
				logger.Errorf("Test runner %s failed on image %s with error: %v",
					runner.Name(), runTestOpts.ImageName, err)

				return err
			}

			return nil
		})
	}

	return errG.Wait()
}
