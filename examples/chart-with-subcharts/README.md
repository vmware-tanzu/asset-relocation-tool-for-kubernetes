# Chart with Subcharts

This example shows how the Asset Relocation Tool for Kubernetes can be used to relocate a Helm chart that uses subcharts.
This will not go into as much detail as the [Simple Chart](../simple-chart) example, but will highlight the differences.

## Inputs

### Helm chart

In this example, we are using the [Bitnami Wordpress](https://bitnami.com/stack/wordpress/helm) chart.
This chart depends on the [Bitnami MariaDB](https://bitnami.com/stack/mariadb/helm) and [Bitnami Memcached](https://bitnami.com/stack/memcached/helm) charts. Each chart references three images.

### Image hints file

`relok8s` only requires a single image hints file to find all of the images in the chart and subcharts.
Images in the subcharts are pre-pended with the subchart name.

## Running `relok8s`

To relocate the chart we will run this command:

```bash
$ relok8s chart move wordpress-11.1.5 --image-patterns wordpress-11.1.5-image-hints.yaml --registry harbor-repo.vmware.com --repo-prefix pwall
Pulling index.docker.io/bitnami/wordpress:5.7.2-debian-10-r45...
Done
Pulling index.docker.io/bitnami/apache-exporter:0.9.0-debian-10-r33...
Done
Pulling index.docker.io/bitnami/bitnami-shell:10-debian-10-r134...
Done
Pulling index.docker.io/bitnami/mariadb:10.5.11-debian-10-r0...
Done
Pulling index.docker.io/bitnami/mysqld-exporter:0.13.0-debian-10-r19...
Done
Pulling index.docker.io/bitnami/bitnami-shell:10-debian-10-r115...
Done
Pulling index.docker.io/bitnami/memcached:1.6.9-debian-10-r194...
Done
Pulling index.docker.io/bitnami/memcached-exporter:0.9.0-debian-10-r85...
Done
Pulling index.docker.io/bitnami/bitnami-shell:10-debian-10-r120...
Done
Checking harbor-repo.vmware.com/pwall/wordpress:5.7.2-debian-10-r45 (sha256:187d539c69e4da11706d63fead255a870cae79a34719b67f8d1cbfd8f7653ff8)...
Push required
Checking harbor-repo.vmware.com/pwall/apache-exporter:0.9.0-debian-10-r33 (sha256:c64fa482ae9cbadbb53487d1155a0a135141116bbfecd8fded0cd365f9646407)...
Push required
Checking harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r134 (sha256:1d385c55c7d8efddc1ac7c9a1d847e3b040803b8bcbf58fba41715e77706add7)...
Push required
Checking harbor-repo.vmware.com/pwall/mariadb:10.5.11-debian-10-r0 (sha256:160902dddb9c7d9640dcfc33ae0dbbed9346f786eeb653fa1b427c76f4673126)...
Push required
Checking harbor-repo.vmware.com/pwall/mysqld-exporter:0.13.0-debian-10-r19 (sha256:ad0993ebdf34a6b6ee0ec469384a3c92ed020ff7b23277c90d150ddd4ed01020)...
Push required
Checking harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r115 (sha256:400d6b412a753845c65c656b311a1d032b605cf1c63e14c25929c1d9e9c423c8)...
Push required
Checking harbor-repo.vmware.com/pwall/memcached:1.6.9-debian-10-r194 (sha256:3dcf3a49f162f55ae9f7407d022ae53021cccf49f8d43276080708bd56857e78)...
Push required
Checking harbor-repo.vmware.com/pwall/memcached-exporter:0.9.0-debian-10-r85 (sha256:979154c8afa2027194fe2721351b112d4daacf96ccf6ff853525e4794d1bbe49)...
Push required
Checking harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r120 (sha256:9eeeefd2e9abeed0ee111a43e4c5c19b2983b74a2a38adb641649a77929ac59b)...
Push required

Images to be pushed:
  harbor-repo.vmware.com/pwall/wordpress:5.7.2-debian-10-r45 (sha256:187d539c69e4da11706d63fead255a870cae79a34719b67f8d1cbfd8f7653ff8)
  harbor-repo.vmware.com/pwall/apache-exporter:0.9.0-debian-10-r33 (sha256:c64fa482ae9cbadbb53487d1155a0a135141116bbfecd8fded0cd365f9646407)
  harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r134 (sha256:1d385c55c7d8efddc1ac7c9a1d847e3b040803b8bcbf58fba41715e77706add7)
  harbor-repo.vmware.com/pwall/mariadb:10.5.11-debian-10-r0 (sha256:160902dddb9c7d9640dcfc33ae0dbbed9346f786eeb653fa1b427c76f4673126)
  harbor-repo.vmware.com/pwall/mysqld-exporter:0.13.0-debian-10-r19 (sha256:ad0993ebdf34a6b6ee0ec469384a3c92ed020ff7b23277c90d150ddd4ed01020)
  harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r115 (sha256:400d6b412a753845c65c656b311a1d032b605cf1c63e14c25929c1d9e9c423c8)
  harbor-repo.vmware.com/pwall/memcached:1.6.9-debian-10-r194 (sha256:3dcf3a49f162f55ae9f7407d022ae53021cccf49f8d43276080708bd56857e78)
  harbor-repo.vmware.com/pwall/memcached-exporter:0.9.0-debian-10-r85 (sha256:979154c8afa2027194fe2721351b112d4daacf96ccf6ff853525e4794d1bbe49)
  harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r120 (sha256:9eeeefd2e9abeed0ee111a43e4c5c19b2983b74a2a38adb641649a77929ac59b)

Changes written to wordpress/values.yaml:
  .image.registry: harbor-repo.vmware.com
  .image.repository: pwall/wordpress
  .metrics.image.registry: harbor-repo.vmware.com
  .metrics.image.repository: pwall/apache-exporter
  .volumePermissions.image.registry: harbor-repo.vmware.com
  .volumePermissions.image.repository: pwall/bitnami-shell

Changes written to wordpress/charts/mariadb/values.yaml:
  .mariadb.image.registry: harbor-repo.vmware.com
  .mariadb.image.repository: pwall/mariadb
  .mariadb.metrics.image.registry: harbor-repo.vmware.com
  .mariadb.metrics.image.repository: pwall/mysqld-exporter
  .mariadb.volumePermissions.image.registry: harbor-repo.vmware.com
  .mariadb.volumePermissions.image.repository: pwall/bitnami-shell

Changes written to wordpress/charts/memcached/values.yaml:
  .memcached.image.registry: harbor-repo.vmware.com
  .memcached.image.repository: pwall/memcached
  .memcached.metrics.image.registry: harbor-repo.vmware.com
  .memcached.metrics.image.repository: pwall/memcached-exporter
  .memcached.volumePermissions.image.registry: harbor-repo.vmware.com
  .memcached.volumePermissions.image.repository: pwall/bitnami-shell
Would you like to proceed? (y/N)
y
Pushing harbor-repo.vmware.com/pwall/wordpress:5.7.2-debian-10-r45...
Done
Pushing harbor-repo.vmware.com/pwall/apache-exporter:0.9.0-debian-10-r33...
Done
Pushing harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r134...
Done
Pushing harbor-repo.vmware.com/pwall/mariadb:10.5.11-debian-10-r0...
Done
Pushing harbor-repo.vmware.com/pwall/mysqld-exporter:0.13.0-debian-10-r19...
Done
Pushing harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r115...
Done
Pushing harbor-repo.vmware.com/pwall/memcached:1.6.9-debian-10-r194...
Done
Pushing harbor-repo.vmware.com/pwall/memcached-exporter:0.9.0-debian-10-r85...
Done
Pushing harbor-repo.vmware.com/pwall/bitnami-shell:10-debian-10-r120...
Done
Writing chart files... Done
```
## Outputs

### Modified Helm chart

The output of the command is a rewritten Helm chart, with the values of the image references put into the chart's and subchart's values.yaml files:

```bash
$ ls wordpress-11.1.5.relocated.tgz 
wordpress-11.1.5.relocated.tgz
$ diff wordpress-11.1.5/values.yaml <(tar zxfO wordpress-11.1.5.relocated.tgz wordpress/values.yaml)
55,56c55,56
<   registry: docker.io
<   repository: bitnami/wordpress
---
>   registry: harbor-repo.vmware.com
>   repository: pwall/wordpress
629,630c629,630
<     registry: docker.io
<     repository: bitnami/bitnami-shell
---
>     registry: harbor-repo.vmware.com
>     repository: pwall/bitnami-shell
703,704c703,704
<     registry: docker.io
<     repository: bitnami/apache-exporter
---
>     registry: harbor-repo.vmware.com
>     repository: pwall/apache-exporter
$ diff wordpress-11.1.5/charts/mariadb/values.yaml <(tar zxfO wordpress-11.1.5.relocated.tgz wordpress/charts/mariadb/values.yaml)
56,57c56,57
<   registry: docker.io
<   repository: bitnami/mariadb
---
>   registry: harbor-repo.vmware.com
>   repository: pwall/mariadb
783,784c783,784
<     registry: docker.io
<     repository: bitnami/bitnami-shell
---
>     registry: harbor-repo.vmware.com
>     repository: pwall/bitnami-shell
816,817c816,817
<     registry: docker.io
<     repository: bitnami/mysqld-exporter
---
>     registry: harbor-repo.vmware.com
>     repository: pwall/mysqld-exporter
$ diff wordpress-11.1.5/charts/memcached/values.yaml <(tar zxfO wordpress-11.1.5.relocated.tgz wordpress/charts/memcached/values.yaml)
15,16c15,16
<   registry: docker.io
<   repository: bitnami/memcached
---
>   registry: harbor-repo.vmware.com
>   repository: pwall/memcached
301,302c301,302
<     registry: docker.io
<     repository: bitnami/memcached-exporter
---
>     registry: harbor-repo.vmware.com
>     repository: pwall/memcached-exporter
400,401c400,401
<     registry: docker.io
<     repository: bitnami/bitnami-shell
---
>     registry: harbor-repo.vmware.com
>     repository: pwall/bitnami-shell
```
