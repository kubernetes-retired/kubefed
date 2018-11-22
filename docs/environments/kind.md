# `kind` - `k`ubernetes `in` `d`ocker

[kind](https://github.com/kubernetes-sigs/kind) provides the quickest way to
set up clusters for use with the Federation v2 control plane.

## Download and Install kind

If you don't yet have `kind` installed, you can run the following script to
download and install a known working version.

```bash
./scripts/download-e2e-binaries.sh
```

Make sure that your `GOBIN` directory is in your path as that is where `kind`
will be installed. Your `GOBIN` directory should be at `$(go env GOPATH)/bin`:

## Create Clusters

You can proceed to create clusters once you have `kind` available in your path.
The `NUM_CLUSTERS` is `2` by default. Set that variable before invoking the
script if you'd like to change the default.

### Configure Insecure Container Registry

Please answer the following questions to determine if you need to set up an
insecure container registry on your host:

1. Is this the first time you're running the `create-clusters.sh` script?
2. Are you planning on creating container images locally without pushing to a
public container registry such as quay.io?

If you answered yes to both of these questions, then you will need to
configure the docker daemon on your host for an insecure container registry.
Configuring a container registry is necessary if you want your kind clusters to
pull images that you built locally on your host without pushing them to a
public container registry. The reason for an insecure registry is to simplify
the container registry setup by not enabling TLS. See the [docker
docs](https://docs.docker.com/registry) for more details.

In order to configure an insecure container registry, you can pass the
`CONFIGURE_INSECURE_REGISTRY` flag to `create-clusters.sh` as follows:

```bash
CONFIGURE_INSECURE_REGISTRY=y ./scripts/create-clusters.sh
```

This will automatically create the necessary dockerd daemon config and reload
the docker daemon for you. Keep in mind that it will **not** do this for you
if a config already exists, or your docker daemon is already configured with an
`--insecure-registry` command line option.

If you would like to manually make the changes to your docker daemon instead,
add `172.17.0.1:5000` as an insecure registry host and reload or restart your
docker daemon.

### Run Script

Run the following command to create `2` `kind` clusters:

```bash
./scripts/create-clusters.sh
```

## Delete Clusters

Run the following command to delete your `2` `kind` clusters. If you've
specified a different `NUM_CLUSTERS` than the default, make sure to set it
again before invoking the script.

```bash
./scripts/delete-clusters.sh
```
