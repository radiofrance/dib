# DIB (Docker Image Builder)

## Use-cases

### Get version

```
dib -v
dib version
```

### Dry run

```
dib build --dry-run
```

### Construire uniquement une sous-arborescence du dossier docker

```
# dib build [DOCKER_BUILD_PATH:docker/]
dib build docker/bullseye/nodejs14
```

### Forcer la reconstruction d'une arborescence même s'il n'y a pas de diff

```
dib build --force-rebuild <path:docker/>
```

### Générer un graph png

Graphe avec des couleurs
- Image unchanged    => grey
- Image need retag   => yellow
- Image need rebuild => red

```
dib graph
```

### Générer le hash du dossier docker

(Utiliser https://pkg.go.dev/golang.org/x/mod/sumdb/dirhash#HashDir)

```
dib hash
```
