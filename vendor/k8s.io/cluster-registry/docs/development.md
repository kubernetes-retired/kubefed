# Development

## Prerequisites

You'll need to clone the repository before doing any work. Make sure to clone
into $GOPATH/src/k8s.io/cluster-registry, since much of the tooling expects
this.

Before doing any development work, you must (in order, from the repository root
directory, after cloning):

1.  run `bazel run //:gazelle`

### bazel

You must have a recent version of [bazel](https://bazel.io) installed. Bazel is
the recommended way to build and test the cluster registry. Bazel is designed to
maintain compatibility with standard Go tooling, but this is not tested on a
regular basis, and some scripts/tooling in this project are built around Bazel.

NOTE: There is an issue with bazel 0.6.x. As a workaround, use 0.5.x, or pass
the flag `--incompatible_comprehension_variables_do_not_leak=false` to bazel
0.6.x invocations.

### `docker-credential-gcr`

To push an image to Google Container Registry you'll also have to have
`docker-credential-gcr` installed and configured. This allows for Docker clients
v1.11+ to easily make authenticated requests to GCR's repositories (gcr.io,
eu.gcr.io, etc.):

1.  Run `gcloud components install docker-credential-gcr`
1.  Run `docker-credential-gcr configure-docker`

### dep

This repository maintains its `vendor` directory with
[dep](https://github.com/golang/dep). You must have v0.3.2 or newer of the tool
installed if you intend to update the vendored dependencies.

### hg

If you plan to use `dep`, you will need to install the Mercurial `hg` command.
This is because of a transitive dependency created by `k8s.io/apiserver` for
the repo `goautoneg` that is hosted at bitbucket.org.  Otherwise the `dep`
commands may hang on you unexpectedly.

## Building `clusterregistry`

`clusterregistry` is the binary for the Kubernetes API server that serves the
cluster registry API.

To build it, from the root of the repository:

1.  Run `bazel build //cmd/clusterregistry`. (This may take a while the first
    time you run it.)
1.  If you want to build a docker image, run
    `bazel build //cmd/clusterregistry:clusterregistry-image`
1.  To push an image to Google Container registry, you'll need to run
    `bazel run //cmd/clusterregistry:push-clusterregistry-image --define repository=<your_gcr_repository_path>`
     where `your_gcr_repository_path` is your GCP project name followed by
     your image name, e.g., `myproject/myimage`.

## Building `crinit`

`crinit` is a command-line tool to bootstrap a cluster registry into a Kubernetes
cluster.

To build it, from the root of the repository:

1.  Run `bazel build //cmd/crinit`. (This may take a while the first time you
    run it.)

## Run all tests

You can run all the unit tests by running
`bazel test -- //... -//vendor/... -//cmd/clusterregistry:push-clusterregistry-image -//pkg/client/...`
from the repository root. (This may take a while the first time you run it.)

## Updating Bazel files

You will need to update the BUILD and BUILD.bazel files when making changes that
cause the Go imports to change.

1.  Run `./hack/update-bazel.sh`
1.  Add the updated `BUILD.bazel` and `BUILD` files to your commit.

## Updating vendored dependencies

The `dep` tool is currently only marginally supported by k/ repos. There are
some warts.

1.  Use the `dep` tool and/or modify the `Gopkg.toml` file to reference the
    new dependency versions. Refer to the [dep
    docs](https://github.com/golang/dep#usage) for more info.
1.  Run `dep prune`.
1.  At this point, there may be other modifications necessary either to the
    vendored dependencies or the `Gopkg.toml` file. The known ones are noted
    below. Make these and any additional necessary ones, and add them to this
    list.
    -   As of #58, it was necessary to modify the BUILD file in
        `vendor/k8s.io/client-go/util/cert` to have the go_library not reference
        testdata.
1.  [Update the BUILD and BUILD.bazel files](#updating-bazel-files).
1.  Run the tests and fix any breakages.
1.  When sending out a PR, please put the handmade changes in one commit and the
    generated updates in another commit so that it's easier for reviewers to
    see what's been done.

## Updating generated code

If you modify any files in `pkg/apis`, you will likely need to regenerate the
generated clients and other generated files.

1.  Run `./hack/update-codegen.sh` to update the files.
1.  Add the generated files to your PR, preferably in a separate, generated-only
    commit so that they are easier to review.

## Verify Go source files

You can run the Go source file verification script to verify and fix your Go source
files:

1. Run `./hack/verify-go-src.sh`

This runs all the Go source verification scripts in
[`./hack/go-tools/`](/hack/go-tools/).

You can also run any of the scripts individually. For example:

1. Run `./hack/go-tools/verify-govet.sh`

The return code of the script indicates success or failure.

## Interacting with the k8s-bot

The cluster-registry repo is monitored by the k8s-ci-robot. You can find a list
of the commands it accepts
[here](https://github.com/kubernetes/test-infra/blob/master/commands.md). Note
that some of the commands are not relevant for the cluster registry, namely as
`/approve`, `/area`, `/hold`, `/release-note` and `/status`.

## Release and build versioning

Refer to [release.md](release.md) for information about doing a release.

[`pkg/version`](/pkg/version) contains infrastructure for generating version
information for builds of the cluster registry. Version info is provided to the
go_binary build rules in the `x_refs` parameter by
[`pkg/version/def.bzl`](/pkg/version/def.bzl). The information is derived from
the Git repository state and build state by
[`hack/print-workspace-status.sh`](/hack/print-workspace-status.sh). This script
is run on each `bazel build` invocation by way of a [`.bazelrc`](/.bazelrc) file
in the repository's root directory. There is some more info about bazel build
stamping
[here](https://www.kchodorow.com/blog/2017/03/27/stamping-your-builds/). Builds
done without `bazel` will get default version information.

### Tagging

The version information is derived largely from annotated git tags. Tags for a
release should be of the form `vX.Y.Z`. Release candidates should be of the form
`vX.Y.Z-rc.N`, where `N` starts at 0 and is incremented with each release
candidate.

This tagging scheme is subject to change as the cluster registry moves through
alpha and beta.

## Nightly releases

The cluster registry has a script, [`hack/release.sh`](../hack/release.sh), that
is used to build releases and push them for public consumption. This script is
run nightly by Prow.

## Versioned releases

[`hack/release.sh`](../hack/release.sh) can be called with one argument, a
version name, to build a versioned release. e.g.,

```sh
./hack/release.sh v0.0.2-rc0
```

This will push a container image with the `latest` and `v0.0.2-rc0` tags and
binaries into a `v0.0.2-rc0` subdirectory in the GCS bucket. *You must add a
corresponding Git tag and release to the cluster-registry repo.*

> Currently, there is no verification that the binaries pushed by
> `hack/release.sh` match the tag that is provided as an argument to the script.
> This will be fixed as the release process evolves.

### Binaries

Binaries are stored in Google Cloud Storage, in the `crreleases` bucket.
Currently there are only binary releases for 64-bit Linux.

#### Nightlies

To get the latest nightly client library (i.e., `crinit`), run:

```sh
PACKAGE=client
LATEST=$(curl https://storage.googleapis.com/crreleases/nightly/latest)
curl -O http://storage.googleapis.com/crreleases/nightly/$LATEST/clusterregistry-$PACKAGE.tar.gz
```

To verify, run:

```sh
curl http://storage.googleapis.com/crreleases/nightly/$LATEST/clusterregistry-$PACKAGE.tar.gz.sha | sha256sum -c -
```

To get the latest nightly server binaries, run the commands above but replace
`PACKAGE=client` with `PACKAGE=server`. To get a nightly build from a specific
day, replace `LATEST=...` with `LATEST=YYYYMMDD`, where `YYYYMMDD` is a date,
e.g., 20171201.

#### Released

To get the latest released version, run the commands above, but remove
`nightly/` from all URL paths, e.g.,

```sh
PACKAGE=client
LATEST=$(curl https://storage.googleapis.com/crreleases/latest)
curl -O http://storage.googleapis.com/crreleases/$LATEST/clusterregistry-$PACKAGE.tar.gz
```

### Images

A Docker image for the [`clusterregistry`](../cmd/clusterregistry) binary is
pushed to GCR nightly and for each release.

To pull the latest nightly image, run:

```sh
docker pull gcr.io/crreleases/clusterregistry:latest_nightly
```

To pull a nightly image from a specific date, replace `latest_nightly` with the
date in `YYYYMMDD` format. For example,

```sh
docker pull gcr.io/crreleases/clusterregistry:20171201
```

To pull the latest released image, run:
```sh
docker pull gcr.io/crreleases/clusterregistry:latest
```

To pull a specific version, replace `latest` with a version tag, e.g.,

```sh
docker pull gcr.io/crreleases/clusterregistry:v0.0.1
```

The tags will map to tags in the cluster-registry repository, which you can
find [here](https://github.com/kubernetes/cluster-registry/tags).
