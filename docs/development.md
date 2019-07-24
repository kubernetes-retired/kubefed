<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Development Guide](#development-guide)
  - [Prerequisites](#prerequisites)
    - [Binaries](#binaries)
    - [kubernetes](#kubernetes)
  - [Prerequisites](#prerequisites-1)
    - [docker](#docker)
  - [Adding a new API type](#adding-a-new-api-type)
  - [Running E2E Tests](#running-e2e-tests)
    - [Setup Clusters and Deploy the KubeFed Control Plane](#setup-clusters-and-deploy-the-kubefed-control-plane)
    - [Running Tests](#running-tests)
    - [Running Tests With In-Memory Controllers](#running-tests-with-in-memory-controllers)
    - [Cleanup](#cleanup)
  - [Test Your Changes](#test-your-changes)
    - [Automated Deployment](#automated-deployment)
  - [Test Latest Master Changes (`canary`)](#test-latest-master-changes-canary)
  - [Test Latest Stable Version (`latest`)](#test-latest-stable-version-latest)
  - [Updating Document](#updating-document)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Development Guide

If you would like to contribute to the KubeFed project, this guide will
help you get started.

## Prerequisites

### Binaries

The KubeFed deployment depends on `kubebuilder`, `etcd`, `kubectl`, and
`kube-apiserver` >= v1.13 being installed in the path. The `kubebuilder`
([v1.0.8](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v1.0.8)
as of this writing) release packages all of these dependencies together.

These binaries can be installed via the `download-binaries.sh` script, which
downloads them to `./bin`:

```bash
./scripts/download-binaries.sh
export PATH=$(pwd)/bin:${PATH}
```

### kubernetes

The KubeFed deployment requires kubernetes version >= 1.13. To see a detailed list of binaries required, see the prerequisites section in the [user guide](./userguide.md#prerequisites)

## Prerequisites

### docker

This repo requires `docker` >= 1.12 to do the docker build work.

Set up your [Docker environment](https://docs.docker.com/install/).

## Adding a new API type

As per the
[docs](http://book.kubebuilder.io/quick_start.html)
for kubebuilder, bootstrapping a new KubeFed API type can be
accomplished as follows:

```bash
# Bootstrap and commit a new type
$ kubebuilder create api --group <your-group> --version v1alpha1 --kind <your-kind>
$ git add .
$ git commit -m 'Bootstrapped a new api resource <your-group>.kubefed.io./v1alpha1/<your-kind>'

# Modify and commit the bootstrapped type
vi pkg/apis/<your-group>/v1alpha1/<your-kind>_types.go
git commit -a -m 'Added fields to <your-kind>'

# Update the generated code and commit
make generate
git add .
git commit -m 'Updated generated code'
```

The generated code will need to be updated whenever the code for a
type is modified. Care should be taken to separate generated from
non-generated code in the commit history.

## Running E2E Tests

The KubeFed E2E tests must be executed against a KubeFed control plane
with one or more registered clusters.  Optionally, the KubeFed
controllers can be run in-memory to enable debugging.

Many of the tests validate CRUD operations for each of the federated
types enabled by default:

1. the objects are created in the target clusters.
1. a label update is reflected in the objects stored in the target
   clusters.
1. a placement update for the object is reflected in the target clusters.
1. deleted resources are removed from the target clusters.

The read operation is implicit.

### Setup Clusters and Deploy the KubeFed Control Plane

In order to run E2E tests, you first need to:

1. Create clusters
   - See the [user guide for a way to deploy clusters](userguide.md#create-clusters)
     for testing KubeFed.
1. Deploy the KubeFed control plane
   - To deploy the latest version of the KubeFed control plane, follow
     the [Helm chart deployment in the user guide](../charts/kubefed/README.md#installing-the-chart).
   - To deploy your own changes, follow the [Test Your Changes](#test-your-changes)
     section of this guide.

Once completed, return here for instructions on running the e2e tests.

### Running Tests

Follow the below instructions to run E2E tests against a KubeFed control plane.

To run E2E tests for all types:

```bash
cd test/e2e
go test -args -kubeconfig=/path/to/kubeconfig -test.v
```

To run E2E tests for a single type:

```bash
cd test/e2e
go test -args -kubeconfig=/path/to/kubeconfig -test.v \
    --ginkgo.focus='Federated "secrets"'
```

It may be helpful to use the [delve
debugger](https://github.com/derekparker/delve) to gain insight into
the components involved in the test:

```bash
cd test/e2e
dlv test -- -kubeconfig=/path/to/kubeconfig -test.v \
    --ginkgo.focus='Federated "secrets"'
```

### Running Tests With In-Memory Controllers

Running the KubeFed controllers in-memory for a test run allows the
controllers to be targeted by a debugger (e.g. delve) or the golang
race detector.  The prerequisite for this mode is scaling down the
KubeFed controller manager:

1. Reduce the `kubefed-controller-manager` deployment replicas to 0. This way
   we can launch the necessary KubeFed controllers ourselves via the test
   binary.

   ```bash
   kubectl scale deployments kubefed-controller-manager -n kube-federation-system --replicas=0
   ```

   Once you've reduced the replicas to 0, you should see the
   `kubefed-controller-manager` deployment update to show 0 pods running:

   ```bash
   kubectl -n kube-federation-system get deployment.apps kubefed-controller-manager
   NAME                            DESIRED   CURRENT   AGE
   kubefed-controller-manager   0         0         14s
   ```

1. Run tests.

   ```bash
   cd test/e2e
   go test -race -args -kubeconfig=/path/to/kubeconfig -in-memory-controllers=true \
       --v=4 -test.v --ginkgo.focus='Federated "secrets"'
   ```

   Additionally, you can run delve to debug the test:

   ```bash
   cd test/e2e
   dlv test -- -kubeconfig=/path/to/kubeconfig -in-memory-controllers=true \
       -v=4 -test.v --ginkgo.focus='Federated "secrets"'
   ```

### Cleanup

Follow the [cleanup instructions in the user guide](../charts/kubefed/README.md#uninstalling-the-chart).

## Test Your Changes

In order to test your changes on your kubernetes cluster, you'll need
to build an image and a deployment config.

**NOTE:** When KubeFed CRDs are changed, you need to run:
```bash
make generate
```
This step ensures that the CRD resources in helm chart are synced.
Ensure binaries from kubebuilder for `etcd` and `kube-apiserver` are in the path (see [prerequisites](#prerequisites)).

### Automated Deployment

If you just want to have this automated, then run the following command
specifying your own image. This assumes you've used the steps [documented
above](#setup-clusters-and-deploy-the-kubefed-control-plane) to
set up two `kind` or `minikube` clusters (`cluster1` and `cluster2`):

```bash
./scripts/deploy-kubefed.sh <containerregistry>/<username>/kubefed:test cluster2
```

**NOTE:** You can list multiple joining cluster names in the above command.
Also, please make sure the joining cluster name(s) provided matches the joining
cluster context from your kubeconfig. This will already be the case if you used
the steps [documented
above](#setup-clusters-and-deploy-the-kubefed-control-plane)
to create your clusters.

## Test Latest Master Changes (`canary`)

In order to test the latest master changes (tagged as `canary`) on your
kubernetes cluster, you'll need to generate a config that specifies the correct
image and generated CRDs. To do that, run the following command:

```bash
make generate
./scripts/deploy-kubefed.sh <containerregistry>/<username>/kubefed:canary cluster2
```

## Test Latest Stable Version (`latest`)

In order to test the latest stable released version (tagged as `latest`) on
your kubernetes cluster, follow the
[Helm Chart Deployment](../charts/kubefed/README.md#installing-the-chart) instructions from the user guide.

## Updating Document

If you are going to add some new sections for the document, make sure to update the table
of contents. This can be done manually or with [doctoc](https://github.com/thlorenz/doctoc).
