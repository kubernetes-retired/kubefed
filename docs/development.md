<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Development Guide](#development-guide)
  - [Prerequisites](#prerequisites)
    - [kubernetes](#kubernetes)
    - [docker](#docker)
  - [Adding a new API type](#adding-a-new-api-type)
  - [Running Tests](#running-tests)
    - [Environment Setup](#environment-setup)
    - [E2E](#e2e)
      - [Managed](#managed)
      - [Unmanaged and Hybrid Cluster Setup](#unmanaged-and-hybrid-cluster-setup)
        - [Setup Clusters, Deploy the Cluster Registry and Federation-v2 Control Plane](#setup-clusters-deploy-the-cluster-registry-and-federation-v2-control-plane)
        - [Unmanaged](#unmanaged)
        - [Hybrid](#hybrid)
        - [Unmanaged and Hybrid Cleanup](#unmanaged-and-hybrid-cleanup)
  - [Test Your Changes](#test-your-changes)
    - [Automated Deployment](#automated-deployment)
    - [Manual Deployment](#manual-deployment)
      - [Build Federation Container Image](#build-federation-container-image)
      - [Create Deployment Config](#create-deployment-config)
  - [Test Latest Master Changes (`canary`)](#test-latest-master-changes-canary)
  - [Test Latest Stable Version (`latest`)](#test-latest-stable-version-latest)
  - [Updating Document](#updating-document)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Development Guide

If you would like to contribute to the federation v2 project, this guide will
help you get started.

## Prerequisites

### kubernetes

The federation v2 deployment requires kubernetes version >= 1.11. To see a
detailed list of binaries required, [see the prerequisites section in the
user guide](userguide.md#prerequisites).

### docker

This repo depends on `docker` >= 1.12 to do the docker build work. 
Set up your [Docker environment](https://docs.docker.com/install/)

## Adding a new API type

As per the
[docs](http://book.kubebuilder.io/quick_start.html)
for kubebuilder, bootstrapping a new federation v2 API type can be
accomplished as follows:

```bash
# Bootstrap and commit a new type
$ kubebuilder create resource --group <your-group> --version v1alpha1 --kind <your-kind>
$ git add .
$ git commit -m 'Bootstrapped a new api resource <your-group>.federation.k8s.io./v1alpha1/<your-kind>'

# Modify and commit the bootstrapped type
$ vi pkg/apis/<your-group>/v1alpha1/<your-kind>_types.go
$ git commit -a -m 'Added fields to <your-kind>'

# Update the generated code and commit
$ kubebuilder generate
$ git add .
$ git commit -m 'Updated generated code'
```

The generated code will need to be updated whenever the code for a
type is modified. Care should be taken to separate generated from
non-generated code in the commit history.

## Running Tests

### Environment Setup

Before running tests, make sure your environment is setup.

- Ensure binaries for `etcd` and `kube-apiserver` are in the path (see
  [prerequisites](#prerequisites)).
- Export required variables:
  ```bash
  export TEST_ASSET_PATH="$(pwd)/bin"
  export TEST_ASSET_ETCD="${TEST_ASSET_PATH}/etcd"
  export TEST_ASSET_KUBE_APISERVER="${TEST_ASSET_PATH}/kube-apiserver"
  ```

### E2E

The federation-v2 E2E tests can run in an _unmanaged_, _managed_, or _hybrid_
modes. For both unmanaged and hybrid modes, you will need to bring your own
clusters. The managed mode will spin up a test-managed federation consisting
of a kube API server (hosting the federation and cluster registry APIs) and 2
member clusters. All of these modes run CRUD operations. CRUD here means that
the tests will run through each of the requested federated types and verify
that:

1. the objects are created in the target clusters.
1. an annotation update is reflected in the objects stored in the target
   clusters.
1. a placement update for the object is reflected in the target clusters.
1. deleted resources are removed from the target clusters.

The read operation is implicit.

#### Managed

The E2E managed tests will spin up a federation consisting of kube
api + cluster registry api + federation api + 2 member clusters and
run [CRUD (create-read-update-delete)
checks](https://github.com/kubernetes-sigs/federation-v2/blob/master/test/e2e/crud.go)
for federated types against that federation. To run:

- ensure the same binaries are available as described in the
  [Environment Setup](development.md#environment-setup) section.

To run tests for all types:

```bash
cd test/e2e
go test -v
```

To run tests for a single type:

```bash
cd test/e2e
go test -args -v=4 -test.v --ginkgo.focus='Federated "secrets"'
```

It may be helpful to use the [delve
debugger](https://github.com/derekparker/delve) to gain insight into
the components involved in the test:

```bash
cd test/e2e
dlv test -- -v=4 -test.v --ginkgo.focus='Federated "secrets"'
```

#### Unmanaged and Hybrid Cluster Setup

The difference between unmanaged and hybrid is that with hybrid, you run
the federation-v2 controllers in-process as part of executing the test. This
helps in running a debugger to debug anything in the controllers. On the other
hand with unmanaged, the controllers are already running in the K8s cluster.
Both methods require the clusters to have already been joined.

##### Setup Clusters, Deploy the Cluster Registry and Federation-v2 Control Plane

In order to run E2E tests in an unmanaged or hybrid setup, you first need to:

1. Create clusters
   - See the [user guide for a way to deploy clusters](userguide.md#create-clusters)
     for testing federation-v2.
1. Deploy the federation-v2 control plane
   - To deploy the latest version of the federation-v2 control plane, follow
     the [automated deployment instructions in the user guide](userguide.md#automated-deployment).
   - To deploy your own changes, follow the [Test Your Changes](#test-your-changes)
     section of this guide.

Once completed, return here for instructions on running tests in an unmanaged or hybrid setup.

From here, the unmanaged and hybrid setups differ slightly. Proceed to the
corresponding subsection depending on what you're interested in. If you are unsure,
the hybrid setup is best for debugging as you can run the controllers
in-process with delve. If you're just wanting to run E2E tests, use unmanaged.

##### Unmanaged

Follow the below instructions to run E2E tests in your unmanaged federation setup.

To run E2E tests for all types:

```bash
cd test/e2e
go test -args -kubeconfig=/path/to/kubeconfig -v=4 -test.v
```

To run E2E tests for a single type:

```bash
cd test/e2e
go test -args -kubeconfig=/path/to/kubeconfig -v=4 -test.v \
    --ginkgo.focus='Federated "secrets"'
```

It may be helpful to use the [delve
debugger](https://github.com/derekparker/delve) to gain insight into
the components involved in the test:

```bash
cd test/e2e
dlv test -- -kubeconfig=/path/to/kubeconfig -v=4 -test.v \
    --ginkgo.focus='Federated "secrets"'
```

##### Hybrid

Since hybrid mode runs the federation-v2 controllers as part of the test
executable to aid in debugging, we need to kill the existing
`federation-controller-manager` pod so that they will not step on each other. Follow
these steps:

1. Reduce the `federation-controller-manager` deployment replicas to 0. This way
   we can launch the necessary federation-v2 controllers ourselves via the test
   binary.

   ```bash
   kubectl -n federation-system patch deployment.apps \
       federation-controller-manager -p '{"spec":{"replicas": 0}}'
   ```

   Once you've reduced the replicas to 0, you should see the
   `federation-controller-manager` deployment update to show 0 pods running:

   ```bash
   kubectl -n federation-system get deployment.apps federation-controller-manager
   NAME                            DESIRED   CURRENT   AGE
   federation-controller-manager   0         0         14s
   ```

1. Run tests.

   ```bash
   cd test/e2e
   go test -args -kubeconfig=/path/to/kubeconfig -in-memory-controllers=true \
       --v=4 -test.v --ginkgo.focus='Federated "secrets"'
   ```

   Additionally, you can run delve to debug the test:

   ```bash
   cd test/e2e
   dlv test -- -kubeconfig=/path/to/kubeconfig -in-memory-controllers=true \
       -v=4 -test.v --ginkgo.focus='Federated "secrets"'
   ```

##### Unmanaged and Hybrid Cleanup

Follow the [cleanup instructions in the user guide](userguide.md#cleanup).

## Test Your Changes

In order to test your changes on your kubernetes cluster, you'll need
to build an image and a deployment config.

**NOTE:** When federation CRDs are changed, you need to run:
```bash
./scripts/sync-up-helm-chart.sh
```
This script ensures that the CRD resources in helm chart can be synced.
Ensure binaries from kubebuilder for `etcd` and `kube-apiserver` are in the path (see [prerequisites](#prerequisites)).

### Automated Deployment

If you just want to have this automated, then run the following command
specifying your own image. This assumes you've used the steps [documented
above](#setup-clusters-deploy-the-cluster-registry-and-federation-v2-control-plane) to
set up two `kind` or `minikube` clusters (`cluster1` and `cluster2`):

```bash
./scripts/deploy-federation.sh <containerregistry>/<username>/federation-v2:test cluster2
```

**NOTE:** You can list multiple joining cluster names in the above command.
Also, please make sure the joining cluster name(s) provided matches the joining
cluster context from your kubeconfig. This will already be the case if you used
the steps [documented
above](#setup-clusters-deploy-the-cluster-registry-and-federation-v2-control-plane)
to create your clusters.

### Manual Deployment

If you'd like to understand what the script is automating for you, then proceed
by following the below instructions.

#### Build Federation Container Image

Run the following commands using the committed `images/federation-v2/Dockerfile` to build
and push a container image to use for deployment:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o images/federation-v2/hyperfed cmd/hyperfed/main.go
docker build images/federation-v2 -t <containerregistry>/<username>/federation-v2:test
docker push <containerregistry>/<username>/federation-v2:test
```

If intending to use the docker hub `docker.io` as the `<containerregistry>` to push
the federation image to, make sure to login to the local docker daemon
to ensure credentials are available for push:

```bash
docker login --username <username>
```

#### Create Deployment Config

Run the following command to build the deployment config `hack/install.yaml`
that includes all the necessary kubernetes resources:

```bash
INSTALL_YAML="hack/install.yaml"
IMAGE_NAME="<containerregistry>/<username>/federation-v2:test"
INSTALL_YAML="${INSTALL_YAML}" IMAGE_NAME="${IMAGE_NAME}" scripts/generate-install-yaml.sh
```

Once the installation YAML config `hack/install.yaml` is created, you are able
to apply this configuration by following the [manual deployment steps in the
user guide](userguide.md#manual-deployment). Be sure to use this newly
generated configuration instead of `hack/install-latest.yaml`.

## Test Latest Master Changes (`canary`)

In order to test the latest master changes (tagged as `canary`) on your
kubernetes cluster, you'll need to generate a config that specifies the correct
image. To do that, run the following command:

```bash
INSTALL_YAML="hack/install.yaml"
IMAGE_NAME="quay.io/kubernetes-multicluster/federation-v2:canary"
INSTALL_YAML="${INSTALL_YAML}" IMAGE_NAME="${IMAGE_NAME}" scripts/generate-install-yaml.sh
```

Once the installation YAML config `hack/install.yaml` is created, you are able
to apply this configuration by following the [manual deployment steps in the
user guide](userguide.md#manual-deployment). Be sure to use this newly
generated configuration instead of `hack/install-latest.yaml`.

## Test Latest Stable Version (`latest`)

In order to test the latest stable released version (tagged as `latest`) on
your kubernetes cluster, follow the
[automated](userguide.md#automated-deployment) or
[manual](userguide.md#manual-deployment) instructions from the user guide.

## Updating Document

If you are going to add some new sections for the document, make sure to update the table
of contents. This can be done manually or with [doctoc](https://github.com/thlorenz/doctoc).
