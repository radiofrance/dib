package report_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/radiofrance/dib/pkg/goss"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
)

var reportNameRegex = regexp.MustCompile(`[0-9]{14}`)

func TestReport_Init(t *testing.T) {
	t.Parallel()

	type input struct {
		version              string
		rootDir              string
		disableGenerateGraph bool
		testRunners          []types.TestRunner
		buildOpts            string
	}

	tests := []struct {
		name     string
		input    input
		expected report.Report
	}{
		{
			name: "valid 1",
			input: input{
				"1.0.0",
				"report",
				false,
				[]types.TestRunner{
					&goss.TestRunner{},
				},
				"",
			},
			expected: report.Report{
				Options: report.Options{
					RootDir:   "report",
					Version:   "v1.0.0",
					BuildOpts: "",
					WithGraph: true,
					WithGoss:  true,
				},
			},
		},
		{
			name: "valid 2",
			input: input{
				"0.17.x",
				"/tmp/dib/report",
				true,
				nil,
				"log_level: info\nbackend: docker\nlocal_only: true",
			},
			expected: report.Report{
				Options: report.Options{
					RootDir:   "/tmp/dib/report",
					Version:   "v0.17.x",
					BuildOpts: "log_level: info\nbackend: docker\nlocal_only: true",
					WithGraph: false,
					WithGoss:  false,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := report.Init(
				test.input.version,
				test.input.rootDir,
				test.input.disableGenerateGraph,
				test.input.testRunners,
				test.input.buildOpts,
			)
			assert.Equal(t, test.expected.Options.RootDir, actual.Options.RootDir)
			assert.Regexp(t, reportNameRegex, actual.Options.Name)
			assert.WithinDuration(t, time.Now(), actual.Options.GenerationDate, 5*time.Second)
			assert.Equal(t, test.expected.Options.Version, actual.Options.Version)
			assert.Equal(t, test.expected.Options.BuildOpts, actual.Options.BuildOpts)
			assert.Equal(t, test.expected.Options.WithGraph, actual.Options.WithGraph)
			assert.Equal(t, test.expected.Options.WithGoss, actual.Options.WithGoss)
		})
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	type args struct {
		dibReport *report.Report
		dag       func() *dag.DAG
	}

	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			args: args{
				dibReport: &report.Report{
					Options: report.Options{
						RootDir:        "/tmp/dib/report",
						Name:           "report-" + uuid.NewString(),
						GenerationDate: time.Now(),
						Version:        "v1.0.0",
						BuildOpts:      "some build opts",
					},
					BuildReports: []report.BuildReport{
						{
							Image: dag.Image{
								Name:         "image1",
								ShortName:    "image1",
								Dockerfile:   &testDockerfile,
								NeedsRebuild: false,
								NeedsTests:   false,
							},
							BuildStatus: report.BuildStatusSuccess,
							TestsStatus: report.TestsStatusPassed,
						},
					},
				},
				dag: func() *dag.DAG {
					graph := &dag.DAG{}
					node := newTestNode(
						false,
						false,
						false)
					graph.AddNode(node)

					return graph
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "no reports",
			args: args{
				dibReport: &report.Report{
					BuildReports: []report.BuildReport{},
				},
				dag: func() *dag.DAG {
					return &dag.DAG{}
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			test.wantErr(t, report.Generate(context.Background(), test.args.dibReport, test.args.dag()))
		})
	}
}

func newTestNode(needsRebuild, needsTests, rebuildFailed bool) *dag.Node {
	return dag.NewNode(&dag.Image{
		Name:          "image1",
		ShortName:     "image1",
		Dockerfile:    &testDockerfile,
		NeedsRebuild:  needsRebuild,
		NeedsTests:    needsTests,
		RebuildFailed: rebuildFailed,
	})
}

var testDockerfile = dockerfile.Dockerfile{
	ContextPath: "../../test/fixtures/build",
	Filename:    "Dockerfile",
}
