package buildkit

import (
	"path/filepath"
)

// getBuildkitHostAddress returns the address of the buildkit host for unix distributions.
func getBuildkitHostAddress() string {
	run := getRuntimeVariableDataDir()
	return "unix://" + filepath.Join(run, "buildkit/buildkitd.sock")
}
