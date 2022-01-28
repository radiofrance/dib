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

	for _, runner := range testRunners {
		if err := runner.RunTest(types.RunTestOptions{
			ImageName:         img.ShortName,
			ImageReference:    fmt.Sprintf("%s:%s", img.Name, newTag),
			DockerContextPath: img.Dockerfile.ContextPath,
		}); err != nil {
			return err
		}
	}
	return nil
}
