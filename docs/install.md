Installation Guide
==================

=== "Install with go"

    Install the latest release on macOS or Linux with:

    ```bash
    go install github.com/radiofrance/dib@latest
    ```

=== "From binary"

    Binaries are available to download from the [GitHub releases](https://github.com/radiofrance/dib/releases) page.

=== "Container image"

    A multi-arch (`linux/amd64`, `linux/arm64`) image is published to `ghcr.io/radiofrance/dib` for each release,
    tagged with the release version (without the `v` prefix) and `latest`.

    It is meant for running dib in CI jobs with the BuildKit backend and the [Kubernetes executor](executors.md#kubernetes),
    where builds and tests run in pods. The image only contains dib and a busybox shell: it does not ship
    `buildctl`, `docker`, `goss`, `ctr` or `graphviz`, so local builds (`--local-only`), the Docker backend
    and graph rendering are not supported with it.

    The image runs as the non-root user `65532` (distroless `nonroot`), so the job's working directory
    must be writable by that user.

    Images are signed with [cosign](https://docs.sigstore.dev/cosign/) using the release workflow's GitHub identity.
    To check a signature:

    ```shell
    cosign verify ghcr.io/radiofrance/dib:<version> \
      --certificate-identity-regexp '^https://github\.com/radiofrance/dib/\.github/workflows/release\.yml@refs/tags/v' \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com
    ```

    Example GitLab CI job:

    ```yaml
    build-images:
      image:
        name: ghcr.io/radiofrance/dib:latest
        entrypoint: [""]
      script:
        - dib build
    ```

## Shell autocompletion

Configure your shell to load dib completions:

=== "Bash"

    To load completion run:
    
    ```shell
    . <(dib completion bash)
    ```

    To configure your bash shell to load completions for each session add to your bashrc:

    ```shell
    # ~/.bashrc or ~/.bash_profile
    command -v dib >/dev/null && . <(dib completion bash)
    ```

    If you have an alias for dib, you can extend shell completion to work with that alias:

    ```shell
    # ~/.bashrc or ~/.bash_profile
    alias tm=dib
    complete -F __start_dib tm
    ```

=== "Fish"

    To configure your fish shell to [load completions](http://fishshell.com/docs/current/index.html#completion-own)
    for each session write this script to your completions dir:
    
    ```shell
    dib completion fish > ~/.config/fish/completions/dib.fish
    ```

=== "Powershell"

    To load completion run:

    ```shell
    . <(dib completion powershell)
    ```

    To configure your powershell shell to load completions for each session add to your powershell profile:
    
    Windows:

    ```shell
    cd "$env:USERPROFILE\Documents\WindowsPowerShell\Modules"
    dib completion >> dib-completion.ps1
    ```
    Linux:

    ```shell
    cd "${XDG_CONFIG_HOME:-"$HOME/.config/"}/powershell/modules"
    dib completion >> dib-completions.ps1
    ```

=== "Zsh"

    To load completion run:
    
    ```shell
    . <(dib completion zsh) && compdef _dib dib
    ```

    To configure your zsh shell to load completions for each session add to your zshrc:
    
    ```shell
    # ~/.zshrc or ~/.profile
    command -v dib >/dev/null && . <(dib completion zsh) && compdef _dib dib
    ```

    or write a cached file in one of the completion directories in your ${fpath}:
    
    ```shell
    echo "${fpath// /\n}" | grep -i completion
    dib completion zsh > _dib
    
    mv _dib ~/.oh-my-zsh/completions  # oh-my-zsh
    mv _dib ~/.zprezto/modules/completion/external/src/  # zprezto
    ```
