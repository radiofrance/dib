package dib

import (
	"fmt"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

// testImage runs the tests on an image.
func testImage(img *dag.Image, testRunners []types.TestRunner, newTag string) error {
	logrus.Infof("Running tests for \"%s:%s\"", img.Name, newTag)

	opts := types.RunTestOptions{
		ImageName:         img.ShortName,
		ImageReference:    fmt.Sprintf("%s:%s", img.Name, newTag),
		DockerContextPath: img.Dockerfile.ContextPath,
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
