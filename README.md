# Relok8s

Relok8s is for relocating Helm Charts with rewritten image references.
It can relocate the images as well as modifying the charts.

Relok8s can work with charts in directory format, or TGZ format. It can handle charts with dependent charts.

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

Relok8s requires a few inputs for the various commands.

### Chart

Each command requires a Helm chart.
The chart can be in directory format, or TGZ bundle.
It can contain dependent charts.

### Image Patterns File

Relok8s requires an image patterns file. This file is used to determine the list of images encoded in the helm chart.

```yaml
---
- "{{ .image }}:{{ .tag }}",
- "{{ .proxy.image }}:{{ .proxy.tag }}",
```

This file is a list of strings, which can be evaluated like a template to reference the fully detailed image path.

### Rules

Relok8s allows for two rules to be specified on the command line:


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

## Development

Relok8s is built with Golang 1.16.

### Running tests

There are three types of tests, unit tests, feature tests and external tests.

Unit tests exercise the internals of the code. They can be run with:

```bash
make test-units
```

Feature tests exercise Relok8s from outside in by building and executing it as CLI. They can be run with:

```bash
make test-features
```

External tests are similar to feature tests except that they execute tests directly against external resources.
They can report false negatives if that resource is offline or if access to that resource is limited in some way.
However, they can also assure that Relok8s is correctly integrating with that resource.

They can be run with:

```bash
make test-external
```

External tests require credentials to talk to the internal VMware Harbor registry, ask Pete if you need access.

All local tests can be run with:

```bash
make test
```
Those are safe to run always, even without credentials setup.

To run all tests, including `test-external` do:
```bash
make test-all
```
