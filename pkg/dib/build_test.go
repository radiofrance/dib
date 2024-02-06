package dib_test

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/radiofrance/dib/pkg/mock"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRebuildGraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		buildGraph      func() *dag.DAG
		testRunners     []types.TestRunner
		expBuildReports []report.BuildReport
		expNumBuilds    int
	}{
		{
			name: "Graph with 1 node with nothing to do",
			buildGraph: func() *dag.DAG {
				graph := &dag.DAG{}
				node := newTestNode(
					false,
					false,
					false)
				graph.AddNode(node)
				return graph
			},
			testRunners:     []types.TestRunner{},
			expBuildReports: []report.BuildReport{},
			expNumBuilds:    0,
		},
		{
			name: "Graph with 1 node that needs test only",
			buildGraph: func() *dag.DAG {
				graph := &dag.DAG{}
				node := newTestNode(
					false,
					true,
					false)
				graph.AddNode(node)
				return graph
			},
			testRunners: []types.TestRunner{},
			expBuildReports: []report.BuildReport{
				{
					BuildStatus: report.BuildStatusSkipped,
					TestsStatus: report.TestsStatusPassed,
				},
			},
			expNumBuilds: 0,
		},
		{
			name: "Graph with 1 node that needs rebuild and test, test is successful",
			buildGraph: func() *dag.DAG {
				graph := &dag.DAG{}
				node := newTestNode(
					true,
					true,
					false)
				graph.AddNode(node)
				return graph
			},
			testRunners: []types.TestRunner{},
			expBuildReports: []report.BuildReport{
				{
					BuildStatus: report.BuildStatusSuccess,
					TestsStatus: report.TestsStatusPassed,
				},
			},
			expNumBuilds: 1,
		},
		{
			name: "Graph with 1 node that needs rebuild and test, test is failing",
			buildGraph: func() *dag.DAG {
				graph := &dag.DAG{}
				node := newTestNode(
					true,
					true,
					false)
				graph.AddNode(node)
				return graph
			},
			testRunners: []types.TestRunner{&mock.TestRunner{
				ReturnedError: fmt.Errorf("mock test failed"),
			}},
			expBuildReports: []report.BuildReport{
				{
					BuildStatus:    report.BuildStatusSuccess,
					TestsStatus:    report.TestsStatusFailed,
					FailureMessage: "mock test failed",
				},
			},
			expNumBuilds: 1,
		},
		{
			name: "Graph with 1 parent and 2 children nodes, rebuild and test successful on all nodes",
			buildGraph: func() *dag.DAG {
				graph := &dag.DAG{}
				parentNode := newTestNode(
					true,
					true,
					false)
				childNode1 := newTestNode(
					true,
					true,
					false)
				childNode2 := newTestNode(
					true,
					true,
					false)
				parentNode.AddChild(childNode1)
				parentNode.AddChild(childNode2)
				graph.AddNode(parentNode)
				return graph
			},
			testRunners: []types.TestRunner{},
			expBuildReports: []report.BuildReport{
				{
					BuildStatus: report.BuildStatusSuccess,
					TestsStatus: report.TestsStatusPassed,
				},
				{
					BuildStatus: report.BuildStatusSuccess,
					TestsStatus: report.TestsStatusPassed,
				},
				{
					BuildStatus: report.BuildStatusSuccess,
					TestsStatus: report.TestsStatusPassed,
				},
			},
			expNumBuilds: 3,
		},
		{
			name: "Graph with 1 parent and 2 children nodes, rebuild failed on parent",
			buildGraph: func() *dag.DAG {
				graph := &dag.DAG{}
				parentNode := newTestNode(
					true,
					false,
					true)
				childNode1 := newTestNode(
					true,
					true,
					false)
				childNode2 := newTestNode(
					true,
					true,
					false)
				parentNode.AddChild(childNode1)
				parentNode.AddChild(childNode2)
				graph.AddNode(parentNode)
				return graph
			},
			testRunners: []types.TestRunner{},
			expBuildReports: []report.BuildReport{
				{
					BuildStatus: report.BuildStatusSuccess,
					TestsStatus: report.TestsStatusSkipped,
				},
				{
					BuildStatus: report.BuildStatusSkipped,
					TestsStatus: report.TestsStatusSkipped,
				},
				{
					BuildStatus: report.BuildStatusSkipped,
					TestsStatus: report.TestsStatusSkipped,
				},
			},
			expNumBuilds: 1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			builder := mock.NewBuilder()
			dibBuilder := dib.Builder{
				Version:     "v1.0.0",
				Graph:       test.buildGraph(),
				TestRunners: test.testRunners,
				BuildOpts: dib.BuildOpts{
					ReportsDir: mock.ReportsDir,
				},
			}

			res := dibBuilder.RebuildGraph(builder, mock.RateLimiter{})

			assert.Len(t, res.BuildReports, len(test.expBuildReports))
			for i, buildReport := range res.BuildReports {
				assert.Equal(t, test.expBuildReports[i].BuildStatus, buildReport.BuildStatus)
				assert.Equal(t, test.expBuildReports[i].TestsStatus, buildReport.TestsStatus)
				assert.Equal(t, test.expBuildReports[i].FailureMessage, buildReport.FailureMessage)
			}

			assert.Equal(t, test.expNumBuilds, countFilesInDirectory(path.Join(mock.ReportsDir, builder.ID)))
		})
	}
}

func newTestNode(needsRebuild, needsTests, rebuildFailed bool) *dag.Node {
	return dag.NewNode(&dag.Image{
		Name:          uuid.NewString(),
		Dockerfile:    &testDockerfile,
		NeedsRebuild:  needsRebuild,
		NeedsTests:    needsTests,
		RebuildFailed: rebuildFailed,
	})
}

func countFilesInDirectory(dirPath string) int {
	if err := os.MkdirAll(dirPath, 0o755); err != nil && !os.IsExist(err) {
		panic(err)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}

	count := 0
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			count++
		}
	}

	return count
}

var testDockerfile = dockerfile.Dockerfile{
	ContextPath: "../../test/fixtures/build",
	Filename:    "Dockerfile",
}
