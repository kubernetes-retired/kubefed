# Development Guide

If you would like to contribute to the federation v2 project, this guide will
help you get started.

## Prerequisites

### apiserver-builder

This repo depends on
[apiserver-builder](https://github.com/kubernetes-incubator/apiserver-builder)
to generate code and build binaries. Download a [recent
release](https://github.com/kubernetes-incubator/apiserver-builder/releases)
and install it in your `PATH`.

## Adding a new API type

As per the
[docs](https://github.com/kubernetes-incubator/apiserver-builder/blob/master/docs/tools_user_guide.md#create-an-api-resource)
for apiserver-builder, bootstrapping a new federation v2 API type can be
accomplished as follows:

```
# Bootstrap and commit a new type
$ apiserver-boot create group version resource --group federation --version v1alpha1 --kind <your-kind>
$ git add .
$ git commit -m 'Bootstrapped a new api resource federation.k8s.io./v1alpha1/<your-kind>'

# Modify and commit the bootstrapped type
$ vi pkg/apis/federation/v1alpha1/<your-kind>_types.go
$ git commit -a -m 'Added fields to <your-kind>'

# Update the generated code and commit
$ apiserver-boot build generated
$ git add .
$ git commit -m 'Updated generated code'
```

The generated code will need to be updated whenever the code for a
type is modified. Care should be taken to separate generated from
non-generated code in the commit history.

## Enabling federation of a type

Implementing support for federation of a Kubernetes type requires
the following steps:

 - add a new template type (as per the [instructions](#adding-a-new-api-type) for adding a new API type)
   - Ensure the spec of the new type has a `Template` field of the target Kubernetes type.
   - e.g. [FederatedSecret](https://github.com/kubernetes-sigs/federation-v2/blob/master/pkg/apis/federation/v1alpha1/federatedsecret_types.go#L49)

 - add a new placement type
   - Ensure the spec of the new type has the `ClusterNames` field of type `[]string`
   - e.g. [FederatedSecretPlacement](https://github.com/kubernetes-sigs/federation-v2/blob/master/pkg/apis/federation/v1alpha1/federatedsecretplacement_types.go)

 - (optionally) add a new override type
   - Ensure the new type contains fields that should be overridable
   - e.g. [FederatedSecretOverride](https://github.com/kubernetes-sigs/federation-v2/blob/master/pkg/apis/federation/v1alpha1/federatedsecretoverride_types.go)

 - Add a new type config resource to configure a propagation controller
   - Ensure the new type contains fields that should be overridable
   - e.g. [secrets](https://github.com/kubernetes-sigs/federation-v2/blob/master/config/federatedtypes/secret.yaml)

 - (optionally) Add yaml test objects for template and override to support integration and e2e testing
   - e.g. [secret-template.yaml](https://github.com/kubernetes-sigs/federation-v2/blob/master/test/common/fixtures/secret-template.yaml)
   - e.g. [secret-override.yaml](https://github.com/kubernetes-sigs/federation-v2/blob/master/test/common/fixtures/secret-override.yaml)

## Testing

### Integration

The integration tests will spin up a federation consisting of kube
api + cluster registry api + federation api + 2 member clusters and
run [CRUD (create-read-update-delete)
checks](https://github.com/kubernetes-sigs/federation-v2/blob/master/test/integration/crud_test.go)
for federated types against that federation.  To run:

 - Download required binaries via [download-binaries.sh](https://github.com/kubernetes-sigs/federation-v2/blob/master/scripts/download-binaries.sh),
   this will generate a `bin` folder under your current directory.
 - Set three environment variables with above `bin` directory.
 ```bash
base_dir=/path/to/your/bin
export TEST_ASSET_PATH=${base_dir}/bin
export TEST_ASSET_ETCD=${TEST_ASSET_PATH}/etcd
export TEST_ASSET_KUBE_APISERVER=${TEST_ASSET_PATH}/kube-apiserver
```
 - `cd test/integration && go test -i && go test -v`

To run tests for a single type:

``
cd test/integration && go test -i && go test -v -run ^TestCrud/FederatedSecret$
``

It may be helpful to use the [delve
debugger](https://github.com/derekparker/delve) to gain insight into
the components involved in the test:

``
cd test/integration && dlv test -- -test.run ^TestCrud$
``

### E2E

The federation-v2 E2E tests can run in an *unmanaged*, *managed*, or *hybrid*
modes. For both unmanaged and hybrid modes, you will need to bring your own
clusters. The managed mode runs similarly to integration as it uses the same
test fixture setup. All of these modes run CRUD operations. CRUD here means
that the tests will run through each of the requested federated types and
verify that:

1. the objects are created in the target clusters.
1. an annotation update is reflected in the objects stored in the target
   clusters.
1. a placement update for the object is reflected in the target clusters.
1. deleted resources are removed from the target clusters.

The read operation is implicit.

#### Managed

The E2E managed tests will spin up the same environment as the
[Integration](development.md#integration) tests described above and run [CRUD
(create-read-update-delete)
checks](https://github.com/kubernetes-sigs/federation-v2/blob/master/test/e2e/crud.go) for
federated types against that federation. To run:

 - ensure the same binaries are available as described in the
   [Integration](development.md#integration) section.

To run tests for all types:

```bash
cd test/e2e
go test -v
```

To run tests for a single type:

```bash
cd test/e2e
go test -args -v=4 -test.v --ginkgo.focus='"FederatedSecret"'
```

It may be helpful to use the [delve
debugger](https://github.com/derekparker/delve) to gain insight into
the components involved in the test:

```bash
cd test/e2e
dlv test -- -v=4 -test.v --ginkgo.focus='"FederatedSecret"'
```

#### Unmanaged and Hybrid Cluster Setup

The difference between unmanaged and hybrid is that with hybrid, you run
the federation-v2 controllers in-process as part of executing the test. This
helps in running a debugger to debug anything in the controllers. On the other
hand with unmanaged, the controllers are already running in the K8s cluster.
Both methods require the clusters to have already been joined.

##### Setup Clusters, Deploy the Cluster Registry and Federation-v2 Control Plane

In order to run E2E tests in an unmanaged or hybrid setup, you first need to
create clusters and deploy the federation-v2 control plane. For the steps to do
that, follow the instructions in the [user
guide](userguide.md#create-clusters). Once completed, return here for
instructions on running tests in an unmanaged or hybrid setup.

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
    --ginkgo.focus='"FederatedSecret" resources'
```

It may be helpful to use the [delve
debugger](https://github.com/derekparker/delve) to gain insight into
the components involved in the test:

```bash
cd test/e2e
dlv test -- -kubeconfig=/path/to/kubeconfig -v=4 -test.v \
    --ginkgo.focus='"FederatedSecret"'
```

##### Hybrid

Since hybrid mode runs the federation-v2 controllers as part of the test
executable to aid in debugging, we need to kill the existing federation
controller container so that they will not step on each other. Follow
these steps:

1. Edit the federation deployment to delete the controller container. The
   federation deployment contains 2 containers: the `apiserver` and the
   `controller`. We'll keep the `apiserver` for now and delete the `controller`.
   This way we can launch the necessary federation-v2 controllers ourselves via the
   test binary. Specifically, we will launch the cluster and federated sync
   controllers.
    ```bash
    kubectl -n federation edit deploy/federation
    ```
    Once you've deleted the controller, you should see the federation
    pod update to show only 1 container running:
    ```bash
    kubectl -n federation get pod -l api=federation
    NAME                          READY     STATUS    RESTARTS   AGE
    federation-5d6bcc8f97-jb5ms   1/1       Running   0          31s
    ```

1. Run tests.

    ```bash
    cd test/e2e
    go test -args -kubeconfig=/path/to/kubeconfig -in-memory-controllers=true \
        --v=4 -test.v --ginkgo.focus='"FederatedSecret" resources'
    ```

   Additionally, you can run delve to debug the test:

    ```bash
    cd test/e2e
    dlv test -- -kubeconfig=/path/to/kubeconfig -in-memory-controllers=true \
        -v=4 -test.v --ginkgo.focus='"FederatedSecret" resources'
    ```

##### Unmanaged and Hybrid Cleanup

Follow the [cleanup instructions in the user guide](userguide.md#cleanup).
