<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [`kind` - `k`ubernetes `in` `d`ocker](#kind---kubernetes-in-docker)
  - [Download and Install kind](#download-and-install-kind)
  - [Create Clusters](#create-clusters)
  - [Delete Clusters](#delete-clusters)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# `kind` - `k`ubernetes `in` `d`ocker

[kind](https://github.com/kubernetes-sigs/kind) provides the quickest way to
set up clusters for use with the KubeFed control plane.

## Download and Install kind

If you don't yet have `kind` installed, you can run the following script to
download and install a known working version.

```bash
./scripts/download-e2e-binaries.sh
```

Make sure that your `GOBIN` directory is in your `PATH` as that is where `kind`
will be installed. Your `GOBIN` directory should be at `$(go env GOPATH)/bin`:

## Create Clusters

You can proceed to create clusters once you have `kind` available in your path.

Run the following command to create `2` `kind` clusters:

```bash
./scripts/create-clusters.sh
```

The `NUM_CLUSTERS` is `2` by default. Set that variable before invoking the
script if you'd like to change the default:

```bash
NUM_CLUSTERS=<num> ./scripts/create-clusters.sh
```

The `KIND_TAG` is `v1.21.1@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6` by default.
Image `kindest/node:v1.21.1@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6` is used as
node docker image for booting the cluster.

You can use `KIND_IMAGE` or `KIND_TAG` to specify the image as you want.
```bash
KIND_TAG=v1.19.4@sha256:796d09e217d93bed01ecf8502633e48fd806fe42f9d02fdd468b81cd4e3bd40b ./scripts/create-clusters.sh
```

```bash
KIND_IMAGE=kindest/node:v1.19.4@sha256:796d09e217d93bed01ecf8502633e48fd806fe42f9d02fdd468b81cd4e3bd40b ./scripts/create-clusters.sh
```

## Delete Clusters

Run the following command to delete `2` `kind` clusters:

```bash
./scripts/delete-clusters.sh
```

The `NUM_CLUSTERS` is `2` by default. Set that variable before invoking the script
if you'd like to change the default:

```bash
NUM_CLUSTERS=<num> ./scripts/delete-clusters.sh
```
