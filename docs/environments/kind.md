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
