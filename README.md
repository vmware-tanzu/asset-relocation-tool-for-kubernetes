# Chart Mover

Chart Mover is a tool to rewrite a Helm chart, given some rewrite rules.

```bash
$ chart-mover --rules-file rules.yaml --images my-image-list.yaml /path/to/mychart
```

## Rules

Chart Mover uses a rules file to determine how to rewrite the image.

```yaml
---
  registry: "harbor-repo.vmware.com"
  repositoryPrefix: "mycompany"
```

## Image List

Chart Mover requires an image list file. This file is used to determine the list of images encoded in the helm chart.

```yaml
---
- "{{ .Values.image }}:{{ .Values.tag }}",
- "{{ .Values.proxy.image }}:{{ .Values.proxy.tag }}",
```

This file is a list of strings, which can be evaluated like a helm template to reference the fully detailed image path.

## Commands

### Rewrite chart

This command is run by default

### List images

This command just renders the image template file with the given chart

```bash
$ chart-mover list-images --images my-image-list.yaml /path/to/mychart | jq .
[
  "docker.io/library/ubuntu:latest"
  "docker.io/library/nginx:1.19"
]
```

### Rewrite images

This command applies the rewrite rules to the images and lists them

```bash
$ chart-mover list-images --images my-image-list.yaml /path/to/mychart | jq .
[
  "harbor-repo.vmware.com/mycompany/library/ubuntu:latest"
  "harbor-repo.vmware.com/mycompany/library/nginx:1.19"
]
```
~~~~