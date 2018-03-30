## `fnord`

This repo contains an in-progress prototype of some of the
foundational aspects of V2 of Kubernetes Federation.  fnord builds on the
sync controller (a.k.a. push reconciler) from [Federation
V1](https://github.com/kubernetes/federation/) to iterate on the API
concepts laid down in the [brainstorming
doc](https://docs.google.com/document/d/159cQGlfgXo6O4WxXyWzjZiPoIuiHVl933B43xhmqPEE/edit#)
and further refined in the [architecture
doc](https://docs.google.com/document/d/1ihWETo-zE8U_QNuzw5ECxOWX0Df_2BVfO3lC4OesKRQ/edit#).
Access to both documents is available to members of the
[kubernetes-sig-multicluster google group](https://groups.google.com/forum/#!forum/kubernetes-sig-multicluster).

<p align="center"><img src="docs/images/propagation.png" width="711"></p>

## Concepts

fnord uses the following abstractions to support the propagation of a
logical federated type:

- Template: defines the representation of the resource common across clusters
- Placement: defines which clusters the resource is intended to appear in
- Override: optionally defines per-cluster field-level variation to apply to the template

These 3 abstractions provide a concise representation of a resource
intended to appear in multiple clusters.  Since the details encoded by
the abstractions are the minimum required for propagation, they are
well-suited to serve as the glue between any given propagation
mechanism and higher-order behaviors like policy-based placement and
dynamic scheduling.

## Working with fnord

### Required: `apiserver-builder`
fnord depends on
[apiserver-builder](https://github.com/kubernetes-incubator/apiserver-builder)
to generate code and build binaries.  Download a [recent
release](https://github.com/kubernetes-incubator/apiserver-builder/releases)
and install it in your `PATH`.

### Adding a new API type

As per the
[docs](https://github.com/kubernetes-incubator/apiserver-builder/blob/master/docs/tools_user_guide.md#create-an-api-resource)
for apiserver-builder, bootstrapping a new fnord API type can be
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

### Enabling federation of a type

Implementing support for federation of a Kubernetes type requires
the following steps:

 - add a new template type (as per the [instructions](#adding-a-new-type) for adding a new type)
   - Ensure the spec of the new type has a `Template` field of the target Kubernetes type.
   - e.g. [FederatedSecret](https://github.com/marun/fnord/blob/master/pkg/apis/federation/v1alpha1/federatedsecret_types.go#L49)

 - add a new placement type
   - Ensure the spec of the new type has the `ClusterNames` field of type `[]string`
   - e.g. [FederatedSecretPlacement](https://github.com/marun/fnord/blob/master/pkg/apis/federation/v1alpha1/federatedsecretplacement_types.go)

 - (optionally) add a new override type
   - Ensure the new type contains fields that should be overridable
   - e.g. [FederatedSecretOverride](https://github.com/marun/fnord/blob/master/pkg/apis/federation/v1alpha1/federatedsecretoverride_types.go)

 - Add a new propagation adapter
   - fnord's [push
     reconciler](https://github.com/marun/fnord/blob/master/pkg/controller/sync/controller.go)
     targets an [adapter
     interface](https://github.com/marun/fnord/blob/master/pkg/federatedtypes/adapter.go),
     and any logical federated type implementing the interface can be
     propagated by the reconciler to member clusters.
   - e.g. [FederatedSecretAdapter](https://github.com/marun/fnord/blob/master/pkg/federatedtypes/secret.go)

### Testing

#### Integration

The fnord integration tests will spin up a federation consisting of
kube api + cluster registry api + federation api + 2 member clusters
and run [CRUD (create-read-update-delete)
checks](https://github.com/marun/fnord/blob/master/test/integration/crud_test.go)
for federated types against that federation.  To run:

 - ensure binaries for `etcd`, `kube-apiserver` and `clusterregistry` are in the path
   - https://github.com/coreos/etcd/releases
   - https://storage.googleapis.com/kubernetes-release/release/v1.9.6/bin/linux/amd64/kube-apiserver
   - https://github.com/kubernetes/cluster-registry/releases
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

#### E2E

The federation-v2 (aka fnord) E2E tests can run in an unmanaged, managed, or
hybrid modes. For both unmanaged and hybrid modes, you will need to bring your
own clusters (BYOC).  The managed mode runs similarly to integration as it uses
the same test fixture setup. All of these modes run CRUD operations. CRUD here
means that the tests will run through each of the requested federated types and
verify that:

1. the objects are created in the target clusters
1. an annotation update is reflected in the objects stored in the underlying
   target clusters.
1. deleted resources are removed from the target clusters.


##### Managed

The E2E managed tests will spin up the same environment as the
[Integration](README.md#integration) tests described above and run [CRUD
(create-read-update-delete)
checks](https://github.com/marun/federation-v2/blob/master/test/e2e/crud.go) for
federated types against that federation. To run:

 - ensure the same binaries are available as described in the
   [Integration](README.md#integration) section.
 - `cd test/e2e && go test -i && go test -v`

To run tests for a single type:

```bash
cd test/e2e && go test -i && go test -v --ginkgo.focus='"FederatedSecret"'
```

It may be helpful to use the [delve
debugger](https://github.com/derekparker/delve) to gain insight into
the components involved in the test:

```bash
cd test/e2e && dlv test -- -v=4 -test.v --ginkgo.focus='"FederatedSecret"'
```

##### Unmanaged and Hybrid Cluster Setup

The difference between unmanaged and hybrid is that with hybrid, you run
the federation-v2 controllers in-process as part of executing the test. This
helps in running a debugger to debug anything in the controllers. On the other
hand with unmanaged, the controllers are already running in the K8s cluster and
the clusters have already been joined.

###### Create Clusters

The quickest way to set up clusters for use with unmanaged and hybrid modes is
to use [minikube](https://kubernetes.io/docs/getting-started-guides/minikube/).
Once you have minikube installed run:

```bash
minikube start -p clusterA --vm-driver=kvm
```

###### Deploy the Cluster Registry

Get the latest version of the [Cluster
Registry](https://github.com/kubernetes/cluster-registry/releases) and run the
following:

```bash
crinit aggregated init mycr --host-cluster-context=clusterA
```

You can also specify your own image using the `--image` flag.

###### Deploy Federation

First you'll need to create the namespace to deploy the federation into:

```bash
kubectl create ns federation
```

Then run `apiserver-boot` to deploy:

```bash
apiserver-boot run in-cluster --name federation --namespace federation \
    --image <containerregistry>/<username>/<imagename>:<tagname> \
    --controller-args="-logtostderr,-v=4"
```

You will most likely need to increase the limit on the amount of memory
required by the API server as there isn't a way to request it via
`apiserver-boot`. In order to do that you will need to modify the
deployment:

```bash
kubectl -n federation edit deploy/federation
```

Update the `apiserver` container memory resources so that they read as follows:

```yaml
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 100m
            memory: 128Mi
```

From here, the Unmanaged and Hybrid setups differ slightly. Proceed to the
corresponding subsection depending on what you're interested in. If you are unsure,
the Hybrid setup is best for debugging as you can run the controllers
in-process with delve.

###### Unmanaged

1. Build kubefnord
    ```bash
    go build -o bin/kubefnord cmd/kubefnord/kubefnord.go

    ```
1. Join Cluster(s)
    ```bash
    ./bin/kubefnord join clusterA --host-cluster-context clusterA --add-to-registry --v=2
    ```
1. Run tests
    ```bash
    cd test/e2e
    go test -c -v
    ./e2e.test -v=4 -kubeconfig=/path/to/kubeconfig -test.v --ginkgo.focus='"FederatedSecret" resources'
    ```

You can repeat these steps to join any additional clusters.

###### Hybrid

Since hybrid mode runs the federation-v2 controllers as part of the test
executable to aid in debugging, we need to kill the existing federation
controller container so that they will not step on each other's toes. Follow
these steps:

1. Edit the federation deployment to delete the controller container. The
   federation deployment contains 2 containers: the API server and the
   controller. We'll keep the API server for now and delete the controller.
   This way we can launch the necessary federation-v2 controllers ourselves via the
   test binary. Specifically, we will launch the cluster and federated sync
   controllers.
    ```bash
    kubectl -n federation edit deploy/federation
    ```
    Once you've deleted the controller, you should see the federation
    pod update to show only 1 container running:
    ```bash
    → kubectl get pod federation-5768dbfbf5-dwkfb -n federation
    NAME                          READY     STATUS    RESTARTS   AGE
    federation-5768dbfbf5-dwkfb   1/1       Running   0          31s
    ```

1. Run tests. Launching these tests will perform the join operation for you.
   Currently it only supports joining 1 cluster but the ability to join more
   will be added in the future. Once the test completes, you will have to
   manually or script an unjoin operation by deleting all the relevant objects.
   The ability to automatically unjoin after the test completes will be added
   in the future.
   ```bash
	cd test/e2e
	go test -c -v
   ./e2e.test -v=4 -kubeconfig=/path/to/kubeconfig -in-memory-controllers=true \
	-test.v --ginkgo.focus='"FederatedSecret" resources'
   ```
   Additionally, you can run delve to debug the test:
   ```bash
    dlv test -- -v=4 -kubeconfig=/path/to/kubeconfig -in-memory-controllers=true \
	-test.v --ginkgo.focus='"FederatedSecret" resources'
   ```

1. Once the test completes, you will have to manually or script an unjoin
   operation by deleting all the relevant objects. The ability to
   automatically unjoin after the test completes will be added in the future.
   For now you can run the following commands to take care of most of it:
   ```bash
   kubectl delete sa clusterA-clusterA -n federation
   kubectl delete clusterrolebinding
   federation-controller-manager:federation-clusterA-clusterA
   kubectl delete clusterrole
   federation-controller-manager:federation-clusterA-clusterA
   kubectl delete clusters clusterA
   kubectl delete federatedclusters clusterA
   ```
