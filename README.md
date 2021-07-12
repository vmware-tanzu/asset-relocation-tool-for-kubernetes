# Asset Relocation Tool for Kubernetes

The Asset Relocation Tool for Kubernetes is a tool used for relocating Kubernetes assets from one place to another.
It's first focus is on relocating Helm Charts, which is done by:
1. Copying the container images referenced in the chart to a new image registry, and 
2. Modifying the chart with the updated image references.

The tool comes in the form of a CLI, named `relok8s`.

## Running relok8s

```bash
$ relok8s chart move mysql-8.5.8.tgz --image-patterns mysql.images.yaml --registry harbor-repo.vmware.com
Pulling docker.io/bitnami/mysql:8.0.25-debian-10-r0... Done

Images to be pushed:
  harbor-repo.vmware.com/bitnami/mysql:8.0.25-debian-10-r0 (sha256:ae8c4c719352a58abc99c866986ee11578bc43e90d794c6705f7b1eb12c7289e)

Changes written to mysql/values.yaml:
  .image.registry: harbor-repo.vmware.com
Would you like to proceed? (y/N)
y
Pushing harbor-repo.vmware.com/bitnami/mysql:8.0.25-debian-10-r0...Done

New chart: mysql-8.5.8.rewritten.tgz
```

## Inputs

The Asset Relocation Tool for Kubernetes requires a few inputs for the various commands.

### Chart

Each command requires a Helm chart.
The chart can be in directory format, or TGZ bundle.
It can contain dependent charts.

### Image Patterns File

The Asset Relocation Tool for Kubernetes requires an image patterns file. This file determines the list of images encoded in the helm chart.

```yaml
---
- "{{ .image }}:{{ .tag }}",
- "{{ .proxy.image }}:{{ .proxy.tag }}",
```

This file is a list of strings, which can be evaluated like a template to reference the fully detailed image path.

### Rules

The Asset Relocation Tool for Kubernetes allows for two rules to be specified on the command line:

#### Registry
```bash
--registry <registry>
```
This overwrites the image registry

#### Repository Prefix
```bash
--repo-prefix <string>
```
This modifies the image repository name for all parts except for the final word.

Rule                | Example                   | Input                             | Output
------------------- | ------------------------- | --------------------------------- | -----------------------------------------------
Registry            | `harbor-repo.vmware.com`  | `docker.io/mycompany/myapp:1.2.3` | `harbor-repo.vmware.com/mycompany/myapp:1.2.3`
Repository Prefix   | `mytenant`                | `docker.io/mycompany/myapp:1.2.3` | `docker.io/mytenant/myapp:1.2.3`


## Running in CI

It may be useful to run `relok8s` inside a CI pipeline to automatically move a chart when there are updates.
An example [Concourse](https://concourse-ci.org/) pipeline can be found here: [docs/example-pipeline.yaml](docs/example-pipeline.yaml)

## Development

See [Development](DEVELOPMENT.md)
