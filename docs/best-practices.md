DIB Best Practices
==================

### Pin dependencies versions in Dockerfiles

As DIB only rebuilds images when something changes in the build context (including the Dockerfile), external 
dependencies should always be pinned to a specific version, so upgrading the dependency triggers a rebuild.

Example:
```dockerfile
RUN apt-get install package@1.0.0
```

### Use .dockerignore

The `.dockerignore` lists file patterns that should not be included in the build context. DIB also ignores those files
when it computes the checksum, so no rebuild is triggered when they are modified.
