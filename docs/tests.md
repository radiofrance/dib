Tests
=====

DIB can execute tests suites to make assertions on images that it just built. This is useful to prevent regressions, 
and ensure everything work as expected at runtime.


## Goss

[Goss](https://github.com/goss-org/goss) is a YAML-based serverspec alternative tool for validating a server’s configuration. DIB runs a container from the 
image to test, and injects the goss binary and configuration, then execute the test itself.

To get started with goss tests, follow the steps below:

1. Install goss locally (for local builds only)

    Follow the procedure from the [official docs](https://github.com/goss-org/goss#installation)

1. Ensure the goss tests are enabled in configuration:
    ```yaml
    # .dib.yaml
    include_tests:
      - goss
    ```

1. Create a `goss.yml` file next to the Dockerfile of the image to test
    ```
    debian/
    ├── Dockerfile
    └── goss.yml
    ```

1. Add some assertions in the `goss.yml`
    Basic Example:
    ```yaml
    command:
      'echo "Hello World !"':
        exit-status: 0
        stdout:
          - 'Hello World !'
    ```

Read the [Goss documentation](https://github.com/goss-org/goss#full-documentation) to learn all possible assertions.
