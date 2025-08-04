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
