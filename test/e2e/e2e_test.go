package e2e_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/radiofrance/dib/cmd"
	"github.com/rogpeppe/go-internal/testscript"
)

const (
	// envRegistry is the address of the plain HTTP registry used by build scenarios, e.g. localhost:5000.
	envRegistry = "DIB_E2E_REGISTRY"
	// envStrict makes the suite fail, instead of skipping scenarios, when the stack is not reachable.
	envStrict = "DIB_E2E_STRICT"
	// envUpdate rewrites the golden files embedded in the scripts with the actual output.
	envUpdate = "DIB_E2E_UPDATE"

	probeTimeout = 10 * time.Second
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"dib": cmd.Execute,
	})
}

func TestE2E(t *testing.T) {
	t.Parallel()

	stack := probeStack(t.Context())

	if os.Getenv(envStrict) != "" {
		for _, err := range []error{stack.buildkitErr, stack.registryErr} {
			if err != nil {
				t.Fatalf("%s is set but the e2e stack is not ready: %v", envStrict, err)
			}
		}
	}

	testscript.Run(t, testscript.Params{
		Dir:                 "testdata/script",
		RequireExplicitExec: true,
		RequireUniqueNames:  true,
		UpdateScripts:       os.Getenv(envUpdate) != "",
		Setup:               stack.setup,
		Condition:           stack.condition,
		Cmds: map[string]func(*testscript.TestScript, bool, []string){
			"envsubst":      cmdEnvsubst,
			"registry-tags": cmdRegistryTags,
		},
	})
}

// stack holds the state of the external services needed by build scenarios.
type stack struct {
	buildkitHost string
	buildkitErr  error
	registry     string
	registryErr  error
}

func probeStack(ctx context.Context) stack {
	s := stack{
		buildkitHost: os.Getenv("BUILDKIT_HOST"),
		registry:     os.Getenv(envRegistry),
	}
	s.buildkitErr = probeBuildkit(ctx, s.buildkitHost)
	s.registryErr = probeRegistry(ctx, s.registry)

	return s
}

func probeBuildkit(ctx context.Context, host string) error {
	if host == "" {
		return errors.New("BUILDKIT_HOST is not set")
	}

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "buildctl", "--addr", host, "debug", "workers").CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("buildctl cannot reach %s: %w: %s", host, err, out)
	}

	return nil
}

func probeRegistry(ctx context.Context, registry string) error {
	if registry == "" {
		return fmt.Errorf("%s is not set", envRegistry)
	}

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+registry+"/v2/", nil)
	if err != nil {
		return fmt.Errorf("invalid registry %q: %w", registry, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("registry %s is not reachable: %w", registry, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry %s answered %s", registry, resp.Status)
	}

	return nil
}

func (s stack) condition(cond string) (bool, error) {
	switch cond {
	case "buildkit":
		return s.buildkitErr == nil, nil
	case "registry":
		return s.registryErr == nil, nil
	default:
		return false, fmt.Errorf("unknown condition %q", cond)
	}
}

func (s stack) setup(env *testscript.Env) error {
	// buildctl writes its auth state under ~/.docker, and testscript's default
	// HOME (/no-home) is not writable for non-root users such as CI runners.
	home := filepath.Join(env.WorkDir, ".home")

	err := os.Mkdir(home, 0o700)
	if err != nil {
		return err
	}

	env.Setenv("HOME", home)

	if s.buildkitHost != "" {
		env.Setenv("BUILDKIT_HOST", s.buildkitHost)
	}

	if s.registryErr == nil {
		// The registry outlives test runs, so each script pushes under its own namespace.
		env.Setenv("TEST_REGISTRY", s.registry+"/e2e-"+strings.ToLower(rand.Text()[:8]))
	}

	return nil
}

// cmdEnvsubst expands $VAR and ${VAR} in the given files, in place.
// Scripts use it to point Dockerfiles at their registry namespace.
func cmdEnvsubst(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) == 0 {
		ts.Fatalf("usage: envsubst file...")
	}

	for _, file := range args {
		expanded := os.Expand(ts.ReadFile(file), ts.Getenv)
		ts.Check(os.WriteFile(ts.MkAbs(file), []byte(expanded), 0o600))
	}
}

// cmdRegistryTags prints the tags of a repository, one per line.
// Negated, it succeeds only if the repository has no tags.
func cmdRegistryTags(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		ts.Fatalf("usage: [!] registry-tags repository")
	}

	repo, err := name.NewRepository(args[0])
	ts.Check(err)

	tags, err := remote.List(repo)

	var terr *transport.Error
	if errors.As(err, &terr) && terr.StatusCode == http.StatusNotFound {
		tags, err = nil, nil
	}

	ts.Check(err)

	if neg {
		if len(tags) > 0 {
			ts.Fatalf("%s has tags %v, want none", args[0], tags)
		}

		return
	}

	if len(tags) == 0 {
		ts.Fatalf("%s has no tags", args[0])
	}

	slices.Sort(tags)

	_, err = fmt.Fprintln(ts.Stdout(), strings.Join(tags, "\n"))
	ts.Check(err)
}
