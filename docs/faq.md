# Frequently Asked Questions (FAQ)

### How to run dib with existing containerd (standalone or created by docker)?

If you already have containerd running on your system (either standalone or as part of Docker), you can configure dib to use it through BuildKit. Here's how:

1. First, make sure BuildKit is installed and running on your system. You can use this Docker image to help with that:

   ```bash
   docker run --privileged --pid=host \
     -e CONTAINERD_ADDRESS=/run/containerd/containerd.sock \
     -e BUILDKIT_VERSION=v0.12.0 \
     <buildkit-nsenter>
   ```

2. Once BuildKit is running and connected to your containerd instance, you can configure dib to use it by setting the `buildkit_host` option in your configuration:

   ```yaml
   # In .dib.yaml
   buildkit_host: unix:///run/buildkit/buildkitd.sock
   ```

   Or you can set it via environment variable:

   ```bash
   export DIB_BUILDKIT_HOST=unix:///run/buildkit/buildkitd.sock
   ```

3. Now when you run dib, it will use your existing BuildKit daemon, which is connected to your containerd instance:

   ```bash
   dib build
   ```

This approach gives you the best of both worlds - you can use dib's powerful image building capabilities while leveraging your existing containerd setup for efficient container operations.

### Are files stored with Git LFS supported?

Yes. dib hashes a [Git LFS](https://git-lfs.com/) pointer with the `oid` it records, which is the sha256 of the
real file. So `dib list` computes the same hashes in a clone made without the LFS objects (e.g. by Renovate) as in a
full clone.

`dib build` refuses to build from a Git LFS pointer, because the image would hold the pointer instead of the real file
under the same hash. Run `git lfs pull` before building.

## Troubleshooting

### Error: "number of layers and diffIDs don't match: X != Y"

If you encounter an error message like `number of layers and diffIDs don't match: X != Y` when pulling or running an image built with `dib`, it's likely due to a bug in older versions of `containerd` regarding OCI image compression.

This issue is known to affect clusters running `containerd` versions:

- older than **2.0.1**
- older than **1.7.29**

To resolve this, you should set the compression to `gzip` when building your images:

```shell
dib build --compression=gzip
```

Or in your `.dib.yaml`:

```yaml
compression: gzip
```

