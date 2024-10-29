Extra Tags
==========

Images managed by dib will get tagged with the human-readable version of the computed hash. This is not very convenient
in some cases, for instance if we want to tag an image with the explicit version of the contained software.

dib allows additional tags to be definedusing a label in the Dockerfile:
```dockerfile
LABEL dib.extra-tags="v1.0.0,v1.0,v1"
```

The label may contain a coma-separated list of tags to be created when the image
gets promoted with the `--release` flag.
