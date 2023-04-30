package report_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/radiofrance/dib/pkg/goss"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/trivy"
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
		buildCfg             string
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
					goss.TestRunner{},
					trivy.TestRunner{},
				},
				"",
			},
			expected: report.Report{
				Options: report.Options{
					RootDir:   "report",
					Version:   "v1.0.0",
					BuildCfg:  "",
					WithGraph: true,
					WithGoss:  true,
					WithTrivy: true,
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
					BuildCfg:  "log_level: info\nbackend: docker\nlocal_only: true",
					WithGraph: false,
					WithGoss:  false,
					WithTrivy: false,
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := report.Init(
				test.input.version,
				test.input.rootDir,
				test.input.disableGenerateGraph,
				test.input.testRunners,
				test.input.buildCfg,
			)
			assert.Equal(t, test.expected.Options.RootDir, actual.Options.RootDir)
			assert.Regexp(t, reportNameRegex, actual.Options.Name)
			assert.WithinDuration(t, time.Now(), actual.Options.GenerationDate, 5*time.Second)
			assert.Equal(t, test.expected.Options.Version, actual.Options.Version)
			assert.Equal(t, test.expected.Options.BuildCfg, actual.Options.BuildCfg)
			assert.Equal(t, test.expected.Options.WithGraph, actual.Options.WithGraph)
			assert.Equal(t, test.expected.Options.WithGoss, actual.Options.WithGoss)
			assert.Equal(t, test.expected.Options.WithTrivy, actual.Options.WithTrivy)
		})
	}
}
