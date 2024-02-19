Configuration
=============

DIB can be configured either by command-line flags, environment variables or configuration file.

The command-line flags have the highest priority, then environment variables, then config file. You can set some
default values in the configuration file, and then override with environment variables of command-line flags.

### Command-line flags

Example:
```shell
dib build --registry-url=gcr.io/project --build-arg=foo=bar
```

### Environment variables

DIB auto-discovers configuration from environment variables prefixed with `DIB_`, followed by the capitalized, 
snake_cased flag name.

Example:
```shell
export DIB_REGISTRY_URL=gcr.io/project
dib build
```

### Configuration file

DIB uses a YAML configuration file in addition to command-line arguments. It will look for a file names `.dib.yaml`
in the current working directory. You can change the file location by setting the `--config` (`-c`) flag.

The YAML keys are equivalent to the flag names, in snake_case.

Example:
```yaml
# .dib.yaml
registryUrl: gcr.io/project
...
```

You can find more examples [here](https://github.com/radiofrance/dib/tree/main/docs/examples/config). See also the 
[reference configuration file](configuration-reference.md).
