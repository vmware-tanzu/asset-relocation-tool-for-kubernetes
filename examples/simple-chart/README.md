# Simple Chart

This example shows how the Asset Relocation Tool for Kubernetes can be used to relocate a simple Helm chart.

## Inputs

### Helm chart

In this example, we are using the [Bitnami MySQL](https://bitnami.com/stack/mysql/helm) chart.
This chart references three images and does not contain any subcharts*.

* This chart actually does contain a subchart, `bitnami/common`, but that chart does not itself contain any image references, so it effectively makes no difference to the example.

### Image hints file

`relok8s` requires an image hints file to know how the chart encodes the image references.
Specifically, the main MySQL image is referenced in the chart like this:

```yaml
image:
  registry: docker.io
  repository: bitnami/mysql
  tag: 8.0.26-debian-10-r0
```

So, our hints file includes this line:

```yaml
- "{{ .image.registry }}/{{ .image.repository }}:{{ .image.tag }}"
```

This is repeated for the other two images.

## Running `relok8s`

To relocate the chart we will run this command:

```bash
relok8s chart move mysql-8.7.3 --image-patterns mysql-8.7.3-image-hints.yaml --registry harbor-repo.vmware.com --repo-prefix pwall
```

Breaking down this command:

```bash
relok8s chart move ...
```

indicates that we want to relocate a Helm chart

```bash
... mysql-8.7.3 ...
```

this part is the path to the chart

```bash
... --image-patterns mysql-8.7.3-image-hints.yaml ...
```

this part is the path to the image patterns file that we created for this chart

```bash
... --registry harbor-repo.vmware.com ...
```

this flag says that we want to change the image registry

```bash
... --repo-prefix pwall
```

and this flag says that we want to replace the first part of the image reference with `pwall`.

When the command runs, it will:

1. Fetch the images
1. Check the remote registry for the rewritten image
1. Prompt for confirmation
1. Push the rewritten images
1. Write the modified chart

```bash
$ ../../build/relok8s chart move mysql-8.7.3 --image-patterns mysql-8.7.3-image-hints.yaml --registry harbor-repo.vmware.com --repo-prefix pwall
Pulling index.docker.io/bitnami/mysql:8.0.26-debian-10-r0...
Done
Pulling index.docker.io/bitnami/mysqld-exporter:0.13.0-debian-10-r43...
Done
Pulling index.docker.io/bitnami/bitnami-shell:10-debian-10-r140...
Done
Checking harbor-repo.vmware.com/pwall/mysql:8.0.26-debian-10-r0 (sha256:ca9c38c676ac322aed686d3e0aa99279f29fed50cdb4b424392543ce01239447)...
Push required
Checking harbor-repo.vmware.com/pwall/mysqld-exporter:0.13.0-debian-10-r43 (sha256:7ae5705c51731f3c8168c88de2003076edbd814f59b3845ce207ec19cd30c842)...
Push required
Checking harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r140 (sha256:d0f7ca4e02e3c64b201ff32e8d0fa1910a636e679c4b9e23ee98e19c82f91e1e)...
Push required

Images to be pushed:
  harbor-repo.vmware.com/pwall/mysql:8.0.26-debian-10-r0 (sha256:ca9c38c676ac322aed686d3e0aa99279f29fed50cdb4b424392543ce01239447)
  harbor-repo.vmware.com/pwall/mysqld-exporter:0.13.0-debian-10-r43 (sha256:7ae5705c51731f3c8168c88de2003076edbd814f59b3845ce207ec19cd30c842)
  harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r140 (sha256:d0f7ca4e02e3c64b201ff32e8d0fa1910a636e679c4b9e23ee98e19c82f91e1e)

Changes written to mysql/values.yaml:
  .image.registry: harbor-repo.vmware.com
  .image.repository: pwall/mysql
  .metrics.image.registry: harbor-repo.vmware.com
  .metrics.image.repository: pwall/mysqld-exporter
  .volumePermissions.image.registry: harbor-repo.vmware.com
  .volumePermissions.image.repository: pwall/bitnami-shell
Would you like to proceed? (y/N)
y
Pushing harbor-repo.vmware.com/pwall/mysql:8.0.26-debian-10-r0...
Done
Pushing harbor-repo.vmware.com/pwall/mysqld-exporter:0.13.0-debian-10-r43...
Done
Pushing harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r140...
Done
Writing chart files... Done
```

## Outputs

### Modified Helm chart

The output of the command is a rewritten Helm chart, with the values of the image references put into the chart's values.yaml file:

```bash
$ ls mysql-8.7.3.relocated.tgz 
mysql-8.7.3.relocated.tgz
$ diff mysql-8.7.3/values.yaml <(tar zxfO mysql-8.7.3.relocated.tgz mysql/values.yaml)
56,57c56,57
<   registry: docker.io
<   repository: bitnami/mysql
---
>   registry: harbor-repo.vmware.com
>   repository: pwall/mysql
828,829c828,829
<     registry: docker.io
<     repository: bitnami/bitnami-shell
---
>     registry: harbor-repo.vmware.com
>     repository: pwall/bitnami-shell
859,860c859,860
<     registry: docker.io
<     repository: bitnami/mysqld-exporter
---
>     registry: harbor-repo.vmware.com
>     repository: pwall/mysqld-exporter
```
