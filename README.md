# Chart Mover

Chart Mover is for rewriting Helm Charts with rewritten image references.
It can relocate the images as well as modifying the charts.

Chart Mover can work with charts in directory format, or TGZ format. It can handle charts with dependent charts.

```bash
$ chart-mover --rules-file rules.yaml --image-templates my-image-list.yaml /path/to/mychart
```

## Running chart-mover

```bash
$ chart-mover --image-templates my-image-list.yaml  --rules-file private-registry.yaml /path/to/mychart | jq .
TBD
```

## Supporting commands

### List images

This command lists the images embedded in the chart using the image template file

```bash
$ chart-mover list-images --image-templates my-image-list.yaml /path/to/mychart | jq .
[
  "docker.io/library/ubuntu:latest"
  "docker.io/library/nginx:1.19"
]
```

Adding the `--pull` flag will attempt to pull those images from the remote repository.

```bash
$ chart-mover list-images --image-templates my-image-list.yaml /path/to/mychart --pull | jq .
Pulling docker.io/library/ubuntu:latest... Done
Pulling docker.io/library/nginx:1.19... Done
[
  "docker.io/library/ubuntu:latest"
  "docker.io/library/nginx:1.19"
]
```

### Rewrite images

This command lists the images after applpying the rules in the given rewrite rules file.

```bash
$ chart-mover rewrite-images --image-templates my-image-list.yaml  --rules-file private-registry.yaml /path/to/mychart | jq .
[
  "my-registry.example.com/library/ubuntu:latest"
  "my-registry.example.com/library/nginx:1.19"
]
```

Adding the `--push` flag will attempt to pull the original, unmodified images, then tag and push the images with the rewritten image references.

```bash
TBD
```

## Inputs

Chart mover requires a few inputs for the various commands.

### Chart

Each command requires a Helm chart.
The chart can be in directory format, or TGZ bundle.
It can contain dependent charts.

### Image Template File

Chart Mover requires an image list file. This file is used to determine the list of images encoded in the helm chart.

```yaml
---
- "{{ .Values.image }}:{{ .Values.tag }}",
- "{{ .Values.proxy.image }}:{{ .Values.proxy.tag }}",
```

This file is a list of strings, which can be evaluated like a helm template to reference the fully detailed image path.

### Rules

Chart Mover uses a rules file to determine how to rewrite the image.

```yaml
---
  registry: "harbor-repo.vmware.com"
  repositoryPrefix: "mycompany"
```

The rules file can support multiple different options:

Rule                | Example                               | Input                             | Output
------------------- | ------------------------------------- | --------------------------------- | -----------------------------------------------
Registry            | `registry: "harbor-repo.vmware.com"`  | `docker.io/mycompany/myapp:1.2.3` | `harbor-repo.vmware.com/mycompany/myapp:1.2.3`
Repository          | `repository: "mycompany/otherapp"`    | `docker.io/mycompany/myapp:1.2.3` | `docker.io/mycompany/otherapp:1.2.3`
Repository Prefix   | `repositoryprefix: "mytenant"`        | `docker.io/mycompany/myapp:1.2.3` | `docker.io/mytenant/mycompany/myapp:1.2.3`
Tag                 | `tag: "imported"`                     | `docker.io/mycompany/myapp:1.2.3` | `docker.io/mycompany/myapp:imported`
Digest              | `digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"` | `docker.io/mycompany/myapp@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa` | `docker.io/mycompany/myapp@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee`