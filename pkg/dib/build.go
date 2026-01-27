package dib

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/pkg/buildkit"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/goss"
	"github.com/radiofrance/dib/pkg/logger"
	"github.com/radiofrance/dib/pkg/ratelimit"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
	"gopkg.in/yaml.v3"
)

type BuildOpts struct {
	// Root options
	BuildPath        string `mapstructure:"build_path"`
	RegistryURL      string `mapstructure:"registry_url"`
	PlaceholderTag   string `mapstructure:"placeholder_tag"`
	HashListFilePath string `mapstructure:"hash_list_file_path"`

	// Build specific options
	BuildkitHost string   `mapstructure:"buildkit_host"`
	NoGraph      bool     `mapstructure:"no_graph"`
	NoTests      bool     `mapstructure:"no_tests"`
	IncludeTests []string `mapstructure:"include_tests"`
	ReportsDir   string   `mapstructure:"reports_dir"`
	DryRun       bool     `mapstructure:"dry_run"`
	ForceRebuild bool     `mapstructure:"force_rebuild"`
	NoRetag      bool     `mapstructure:"no_retag"`
	LocalOnly    bool     `mapstructure:"local_only"`
	Push         bool     `mapstructure:"push"`
	Release      bool     `mapstructure:"release"`
	Backend      string   `mapstructure:"backend"`
	File         string   `mapstructure:"file"`
	Target       string   `mapstructure:"target"`
	Progress     string   `mapstructure:"progress"`

	Goss      goss.Config     `mapstructure:"goss"`
	Buildkit  buildkit.Config `mapstructure:"buildkit"`
	RateLimit int             `mapstructure:"rate_limit"`
	BuildArg  []string        `mapstructure:"build_arg"`
}

// RebuildGraph iterates over the graph to rebuild all the images that are marked to be rebuilt.
//
//nolint:musttag
func (p *Builder) RebuildGraph(
	ctx context.Context,
	builder types.ImageBuilder,
	rateLimiter ratelimit.RateLimiter,
	buildArgs map[string]string,
) *report.Report {
	buildOpts, err := yaml.Marshal(&p.BuildOpts)
	if err != nil {
		logger.Fatalf("cannot marshal build options: %v", err)
	}

	res := report.Init(p.Version, p.ReportsDir, p.NoGraph, p.TestRunners, string(buildOpts))
	buildReportsChan := make(chan report.BuildReport)

	go p.rebuildGraph(
		ctx,
		buildReportsChan,
		builder,
		rateLimiter,
		res.GetBuildReportDir(),
		res.GetJunitReportDir(),
		buildArgs,
	)

	for buildReport := range buildReportsChan {
		res.BuildReports = append(res.BuildReports, buildReport)
	}

	return res
}

func (p *Builder) rebuildGraph(
	ctx context.Context,
	buildReportsChan chan report.BuildReport,
	builder types.ImageBuilder,
	rateLimiter ratelimit.RateLimiter,
	buildReportDir, junitReportDir string,
	buildArgs map[string]string,
) {
	p.Graph.
		WalkParallel(
			func(node *dag.Node) {
				img := node.Image
				if !img.NeedsRebuild && !img.NeedsTests {
					img.RebuildFailed = false
					return
				}

				buildReport := report.BuildReport{Image: *img}

				// Return if any parent build failed
				for _, parent := range node.Parents() {
					if parent.Image.RebuildFailed {
						img.RebuildFailed = true

						buildReportsChan <- buildReport

						return
					}
				}

				if img.NeedsRebuild {
					meta := LoadCommonMetadata(&exec.ShellExecutor{})

					opts := types.ImageBuilderOpts{
						BuildkitHost: p.BuildkitHost,
						Context:      img.Dockerfile.ContextPath,
						File:         p.File,
						LocalOnly:    p.LocalOnly,
						Target:       p.Target,
						Tags: []string{
							img.CurrentRef(),
						},
						Labels: meta.WithImage(img).ToLabels(),
						// TODO fix this flag there is mix between push and local, is totally different
						Push:      p.Push,
						BuildArgs: buildArgs,
						Progress:  p.Progress,
					}

					err := buildNode(ctx, node, opts, builder, rateLimiter,
						p.PlaceholderTag, buildReportDir,
					)
					if err != nil {
						img.RebuildFailed = true

						buildReportsChan <- buildReport.WithError(err)

						return
					}

					buildReport.BuildStatus = report.BuildStatusSuccess
				}

				if !img.NeedsTests {
					buildReportsChan <- buildReport
					return
				}

				err := testImage(ctx, p.TestRunners, types.RunTestOptions{
					ImageName:         img.ShortName,
					ImageReference:    img.CurrentRef(),
					BuildkitHost:      p.BuildkitHost,
					DockerContextPath: img.Dockerfile.ContextPath,
					ReportJunitDir:    junitReportDir,
				})
				if err != nil {
					buildReport.TestsStatus = report.TestsStatusFailed
					buildReport.FailureMessage = err.Error()
				} else {
					buildReport.TestsStatus = report.TestsStatusPassed
				}

				buildReportsChan <- buildReport
			})
	close(buildReportsChan)
}

func buildNode(
	ctx context.Context,
	node *dag.Node,
	opts types.ImageBuilderOpts,
	builder types.ImageBuilder,
	rateLimiter ratelimit.RateLimiter,
	placeholderTag string,
	buildReportDir string,
) error {
	rateLimiter.Acquire()
	defer rateLimiter.Release()

	img := node.Image
	// Before building the image, we need to replace all references to tags
	// of any dib-managed images used as dependencies in the Dockerfile.
	tagsToReplace := make(map[string]string)
	for _, parent := range node.Parents() {
		tagsToReplace[parent.Image.DockerRef(placeholderTag)] = parent.Image.CurrentRef()
	}

	err := dockerfile.ReplaceInFile(
		path.Join(img.Dockerfile.ContextPath, img.Dockerfile.Filename), tagsToReplace)
	if err != nil {
		return fmt.Errorf("failed to replace tag in dockerfile %s: %w", img.Dockerfile.ContextPath, err)
	}

	defer func() {
		err := dockerfile.ResetFile(
			path.Join(img.Dockerfile.ContextPath, img.Dockerfile.Filename), tagsToReplace)
		if err != nil {
			logger.Warnf("failed to reset tag in dockerfile %s: %v", img.Dockerfile.ContextPath, err)
		}
	}()

	err = os.MkdirAll(buildReportDir, 0o750)
	if err != nil {
		return fmt.Errorf("failed to create folder %s: %w", buildReportDir, err)
	}

	filePath := path.Join(buildReportDir, fmt.Sprintf("%s.txt", strings.ReplaceAll(img.ShortName, "/", "_")))

	opts.LogOutput, err = os.Create(filePath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}

	logger.Infof("Building \"%s\" in context \"%s\"", img.CurrentRef(), img.Dockerfile.ContextPath)

	err = builder.Build(ctx, opts)
	if err != nil {
		return fmt.Errorf("building image %s failed: %w", img.ShortName, err)
	}

	return nil
}
