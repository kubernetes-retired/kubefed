# Development

The cluster registry repository is maintained using the
[`kubebuilder`](https://github.com/kubernetes-sigs/kubebuilder) tool. This tool
manages code generation, documentation generation, CRD definition generation and
the vendor/ directory.

## Prerequisites

You'll need to clone the repository before doing any work. Make sure to clone
into `$GOPATH/src/k8s.io/cluster-registry`, since much of the tooling expects
this.

You must install [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
in order to work in this repository. `kubebuilder` is in active development, and
we expect that the structure of this repository will need to change as
`kubebuilder` is improved. All of the commands below are verified to work with
`kubebuilder` `0.1.9`.

## Run all tests

To set up the testing environment, run:

```
$ kubebuilder update vendor --overwrite-dep-manifest
$ KUBEBUILDER_PATH=<path_to_your_kubebuilder_install>
$ export TEST_ASSET_ETCD=$KUBEBUILDER_PATH/bin/etcd
$ export TEST_ASSET_KUBE_APISERVER=$KUBEBUILDER_PATH/bin/kube-apiserver
$ export TEST_ASSET_KUBECTL=$KUBEBUILDER_PATH/bin/kubectl
```

Note that this will create a `/vendor` directory, which should not be checked
in.

After this, you can run all the project's tests by running `go test ./test/...`
from the repository root.

## Updating vendored dependencies

The cluster-registry does not have a checked-in `Gopkg.toml` or `Gopkg.lock`, or
a checked-in `/vendor` directory, in order to facilitate integration into other
projects and reduce the amount of work necessary to keep the vendored
dependencies up-to-date. `kubebuilder` operations do not require a checked-in
vendor tree.

## Updating generated code/docs

To update generated code after modifying the cluster type, run these commands
from the repo root:

```
$ kubebuilder update vendor --overwrite-dep-manifest
$ kubebuilder generate
$ kubebuilder docs  # This will fail because of missing dependencies.
$ dep ensure
$ kubebuilder docs
$ chown -R $USER docs/reference/build  # The generated docs are owned by root.
$ kubebuilder create config --crds --output cluster-registry-crd.yaml
```

These will update the generated client code, update the generated docs and
OpenAPI spec in `docs/reference/openapi-spec`, and update the CRD YAML
definition in the repo root.

After running the commands, edit `cluster-registry-crd.yaml` to modify the
`type` of the `caBundle` field from `array` to `string`. (See
https://github.com/kubernetes-sigs/kubebuilder/issues/220 for more details).

## Verify Go source files

You can run the Go source file verification script to verify and fix your Go
source files:

1.  Run `./hack/verify-go-src.sh`

This runs all the Go source verification scripts in
[`./hack/go-tools/`](/hack/go-tools/).

You can also run any of the scripts individually. For example:

1.  Run `./hack/go-tools/verify-govet.sh`

The return code of the script indicates success or failure.

## Interacting with the k8s-bot

The cluster-registry repo is monitored by the k8s-ci-robot. You can find a list
of the commands it accepts
[here](https://github.com/kubernetes/test-infra/blob/master/commands.md). Note
that some of the commands are not relevant for the cluster registry, namely as
`/approve`, `/area`, `/hold`, `/release-note` and `/status`.

## Release

Refer to [release.md](release.md) for information about doing a release.

### Tagging

The version information is derived largely from annotated git tags. Tags for a
release should be of the form `vX.Y.Z`. Release candidates should be of the form
`vX.Y.Z-rc.N`, where `N` starts at 0 and is incremented with each release
candidate.

This tagging scheme is subject to change as the cluster registry moves through
alpha and beta.
