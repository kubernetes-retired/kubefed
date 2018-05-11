# Kubernetes Federation v2

This repo contains an in-progress prototype of some of the
foundational aspects of V2 of Kubernetes Federation.  The prototype
builds on the sync controller (a.k.a. push reconciler) from
[Federation v1](https://github.com/kubernetes/federation/) to iterate
on the API concepts laid down in the [brainstorming
doc](https://docs.google.com/document/d/159cQGlfgXo6O4WxXyWzjZiPoIuiHVl933B43xhmqPEE/edit#)
and further refined in the [architecture
doc](https://docs.google.com/document/d/1ihWETo-zE8U_QNuzw5ECxOWX0Df_2BVfO3lC4OesKRQ/edit#).
Access to both documents is available to members of the
[kubernetes-sig-multicluster google
group](https://groups.google.com/forum/#!forum/kubernetes-sig-multicluster).

<p align="center"><img src="docs/images/propagation.png" width="711"></p>

# Concepts

The following abstractions support the propagation of a logical
federated type:

- Template: defines the representation of the resource common across clusters
- Placement: defines which clusters the resource is intended to appear in
- Override: optionally defines per-cluster field-level variation to apply to the template

These 3 abstractions provide a concise representation of a resource
intended to appear in multiple clusters.  Since the details encoded by
the abstractions are the minimum required for propagation, they are
well-suited to serve as the glue between any given propagation
mechanism and higher-order behaviors like policy-based placement and
dynamic scheduling.

# Getting started

## Required: `apiserver-builder`
This repo depends on
[apiserver-builder](https://github.com/kubernetes-incubator/apiserver-builder)
to generate code and build binaries.  Download a [recent
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

 - add a new template type (as per the [instructions](#adding-a-new-type) for adding a new type)
   - Ensure the spec of the new type has a `Template` field of the target Kubernetes type.
   - e.g. [FederatedSecret](https://github.com/kubernetes-sigs/federation-v2/blob/master/pkg/apis/federation/v1alpha1/federatedsecret_types.go#L49)

 - add a new placement type
   - Ensure the spec of the new type has the `ClusterNames` field of type `[]string`
   - e.g. [FederatedSecretPlacement](https://github.com/kubernetes-sigs/federation-v2/blob/master/pkg/apis/federation/v1alpha1/federatedsecretplacement_types.go)

 - (optionally) add a new override type
   - Ensure the new type contains fields that should be overridable
   - e.g. [FederatedSecretOverride](https://github.com/kubernetes-sigs/federation-v2/blob/master/pkg/apis/federation/v1alpha1/federatedsecretoverride_types.go)

### TODO(marun) update the docs for this
 - Add a new propagation adapter
   - the [push
     reconciler](https://github.com/kubernetes-sigs/federation-v2/blob/master/pkg/controller/sync/controller.go)
     targets an [adapter
     interface](https://github.com/kubernetes-sigs/federation-v2/blob/master/pkg/federatedtypes/adapter.go),
     and any logical federated type implementing the interface can be
     propagated by the reconciler to member clusters.
   - e.g. [FederatedSecretAdapter](https://github.com/kubernetes-sigs/federation-v2/blob/master/pkg/federatedtypes/secret.go)

## Testing

### Integration

The integration tests will spin up a federation consisting of kube
api + cluster registry api + federation api + 2 member clusters and
run [CRUD (create-read-update-delete)
checks](https://github.com/kubernetes-sigs/federation-v2/blob/master/test/integration/crud_test.go)
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

### E2E

The Federation-v2 E2E tests can run in an *unmanaged*, *managed*, or *hybrid*
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
[Integration](README.md#integration) tests described above and run [CRUD
(create-read-update-delete)
checks](https://github.com/kubernetes-sigs/federation-v2/blob/master/test/e2e/crud.go) for
federated types against that federation. To run:

 - ensure the same binaries are available as described in the
   [Integration](README.md#integration) section.

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

##### Create Clusters

The quickest way to set up clusters for use with unmanaged and hybrid modes is
to use [minikube](https://kubernetes.io/docs/getting-started-guides/minikube/).

**NOTE: You will need to use minikube version
[0.25.2](https://github.com/kubernetes/minikube/releases/tag/v0.25.2) in order
to avoid an issue with profiles, which is required for testing multiple
clusters. See issue https://github.com/kubernetes/minikube/issues/2717 for more
details.**

Once you have minikube installed run:

```bash
minikube start -p clusterA
```

Even though the `minikube` cluster has been started, you'll want to verify all
your `minikube` components are up and ready by examining the state of the
kubernetes components via:

```bash
kubectl get all --all-namespaces
```

Once all pods are running you can move on to deploy the cluster registry.

##### Deploy the Cluster Registry

Make sure the storage provisioner is ready before deploying the Cluster
Registry.

```bash
kubectl -n kube-system get pod storage-provisioner
```

Get the latest version of the [Cluster
Registry](https://github.com/kubernetes/cluster-registry/releases) and run the
following:

```bash
crinit aggregated init mycr --host-cluster-context=clusterA
```

You can also specify your own cluster registry image using the `--image` flag.

##### Deploy Federation

First you'll need to create the namespace to deploy the federation into:

```bash
kubectl create ns federation
```

Then run `apiserver-boot` to deploy. This will build the federated `apiserver`
and `controller`, build an image containing them, push the image to the
requested registry, generate a k8s YAML config `config/apiserver.yaml`
with the name and namespace requested, and deploy the config to the
cluster specified in your kubeconfig current-context.

If intending to use the docker hub as the container registry to push
the federation image to, make sure to login to the local docker daemon
to ensure credentials are available for push:

```bash
docker login --username <username>
```

Once you're ready, run the following command. An example of the `--image`
argument is `docker.io/<username>/federation-v2:test`.

```bash
apiserver-boot run in-cluster --name federation --namespace federation \
    --image <containerregistry>/<username>/<imagename>:<tagname> \
    --controller-args="-logtostderr,-v=4"
```

You will most likely need to increase the limit on the amount of memory
required by the `apiserver` as there isn't a way to request it via
`apiserver-boot` at the moment. Otherwise the federation pod may be terminated
with `OOMKilled`. In order to do that you will need to patch the deployment:

```bash
kubectl -n federation patch deploy federation -p \
    '{"spec":{"template":{"spec":{"containers":[{"name":"apiserver","resources":{"limits":{"memory":"128Mi"},"requests":{"memory":"64Mi"}}}]}}}}'
```

If you also run into `OOMKilled` on either the `etcd` or the `controller`, just
run the same command but replace `apiserver` with `etcd` or `controller`.

Verify that the deployment succeeded and is available to serve its API by
seeing if we can retrieve one of its API resources:

```bash
kubectl get federatedcluster
```

It should successfully report that no resources are found.

##### Enabling Push Propagation

Configuration of push propagation is via the creation of
FederatedTypeConfig resources. To enable propagation for default
supported types, run the following command:

```bash
for tc in ./config/federatedtypes/*.yaml; do kubectl create -f "${tc}"; done
```


##### Join Clusters

First, make sure the federation deployment succeeded by verifying it using the
step above. Next, you'll want to use the `kubefnord` tool to join all your
clusters that you want to test against.


1. Build kubefnord
    ```bash
    go build -o bin/kubefnord cmd/kubefnord/kubefnord.go

    ```
1. Join Cluster(s)
    ```bash
    ./bin/kubefnord join clusterA --host-cluster-context clusterA --add-to-registry --v=2
    ```
You can repeat these steps to join any additional clusters.


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

##### Unmanaged and Hybrid Unjoin

Once the test completes, you will have to manually or script an unjoin
operation by deleting all the relevant objects. The ability to unjoin via
`kubefnord` will be added in the future.  For now you can run the following
commands to take care of it:

```bash
CLUSTER=clusterA
kubectl -n federation delete sa ${CLUSTER}-${CLUSTER}
kubectl delete clusterrolebinding \
    federation-controller-manager:${CLUSTER}-${CLUSTER}
kubectl delete clusterrole \
    federation-controller-manager:${CLUSTER}-${CLUSTER}
kubectl delete clusters ${CLUSTER}
kubectl delete federatedclusters ${CLUSTER}
CLUSTER_SECRET=$(kubectl -n federation get secrets \
    -o go-template='{{range .items}}{{if .metadata.generateName}}{{.metadata.name}}{{end}}{{end}}')
kubectl -n federation delete secret ${CLUSTER_SECRET}
```

## Code of Conduct

Participation in the Kubernetes community is governed by the
[Kubernetes Code of Conduct](./code-of-conduct.md).
