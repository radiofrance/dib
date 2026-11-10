End-to-end tests
================

The end-to-end tests run the real `dib` command against a real BuildKit daemon and a real registry.
They are written as [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) scenarios,
the format the Go team uses to test the `go` command.

They live in `test/e2e`:

- `testdata/script/*.txtar`: one scenario per file, with its inputs (`.dib.yaml`, Dockerfiles...) embedded.
- `compose.yaml`: the BuildKit daemon and the registry used by build scenarios.
- `e2e_test.go`: the test harness.

## Running the tests locally

Requirements:

- Docker, to run the stack.
- [`buildctl`](https://github.com/moby/buildkit/releases) in your `PATH` (`brew install buildkit` on macOS).
- [Graphviz](https://graphviz.org/download/), `dib build` uses `dot` to render the graph of the report.

```shell
make e2e.up    # start buildkitd (tcp://127.0.0.1:1234) and a registry (localhost:5000)
make e2e       # run the scenarios
make e2e.down  # stop the stack
```

To run a single scenario:

```shell
make e2e.up
BUILDKIT_HOST=tcp://127.0.0.1:1234 DIB_E2E_REGISTRY=localhost:5000 \
  go test -v -count=1 -run 'TestE2E/build_push' ./test/e2e/...
```

Without the stack, `go test ./...` still runs the scenarios that don't build images and skips the others.
In CI, `DIB_E2E_STRICT=true` turns these skips into a failure, so the build scenarios can't be skipped silently.

## Writing a scenario

A scenario is a list of commands, followed by the files it needs. For example:

```text
# dib list discovers the images managed by dib.
exec dib list
stdout 'base'

-- .dib.yaml --
registry_url: registry.invalid/e2e
-- docker/base/Dockerfile --
FROM busybox:1.37.0
LABEL name="base"
```

Each scenario runs in its own temporary directory. On top of the
[built-in commands](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript#hdr-Script_Language)
(`exec`, `stdout`, `stderr`, `cmp`, `cp`, `env`...), the harness provides:

| Command / condition       | Description                                                                         |
|---------------------------|-------------------------------------------------------------------------------------|
| `[buildkit]`              | True when `buildctl` can reach `$BUILDKIT_HOST`.                                    |
| `[registry]`              | True when the registry `$DIB_E2E_REGISTRY` is reachable.                            |
| `$TEST_REGISTRY`          | A registry namespace unique to the scenario, e.g. `localhost:5000/e2e-x1y2z3a4`.    |
| `envsubst file...`        | Expands `$VAR` and `${VAR}` in files, e.g. to use `$TEST_REGISTRY` in a Dockerfile. |
| `registry-tags repo`      | Prints the tags of a repository. With `!`, checks that it has no tags.             |

Scenarios that build images start with:

```text
[!buildkit] skip 'needs buildkitd, see docs/e2e-tests.md'
[!registry] skip 'needs the e2e registry, see docs/e2e-tests.md'

env DIB_REGISTRY_URL=$TEST_REGISTRY
envsubst docker/child/Dockerfile
```

Some scenarios compare the output with a file embedded in the scenario (`cmp stdout file`).
To refresh these files after an intended change, run the tests with `DIB_E2E_UPDATE=1`.
