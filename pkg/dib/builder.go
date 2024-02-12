package dib

import (
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/types"
)

type Builder struct {
	Version     string
	Graph       *dag.DAG
	TestRunners []types.TestRunner
	BuildOpts
}
