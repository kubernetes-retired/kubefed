# User Guide

If you are looking to use federation v2, you've come to the right place. Below
is a walkthrough tutorial for how to deploy the federation v2 control plane.

## Prerequisites

The federation v2 deployment requires kubernetes version >= 1.11. The following
is a detailed list of binaries required.

### Binaries

The federation deployment depends on `kubebuilder`, `etcd`, `kubectl`, and
`kube-apiserver` >= v1.11 being installed in the path. The `kubebuilder`
([v1.0.3](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v1.0.3)
as of this writing) release packages all of these dependencies together.

These binaries can be installed via the `download-binaries.sh` script, which
downloads them to `./bin`:

```bash
./scripts/download-binaries.sh
export PATH=$(pwd)/bin:${PATH}
```

Or you can install them manually yourself using the guidelines provided below.

#### kubebuilder

This repo depends on
[kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
to generate code and build binaries. Download the [v1.0.3
release](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v1.0.3)
and install it in your `PATH`.

### Deployment Image

If you follow this user guide without any changes you will be using the latest
stable released version of the federation-v2 image tagged as `latest`.
Alternatively, we support the ability to deploy the [latest master image tagged
as `canary`](development.md#test-latest-master-changes-canary) or [your own
custom image](development.md#test-your-changes).

### Create Clusters

The quickest way to set up clusters for use with the federation-v2 control
plane is to use [minikube](https://kubernetes.io/docs/getting-started-guides/minikube/).

**NOTE:** You will need to use a minikube version that supports deploying a
kubernetes cluster >= 1.11. Currently there is no released version of minikube
that supports kube v1.11 with profiles so you'll need to either:

1. Build minikube from master by following these
   [instructions](https://github.com/kubernetes/minikube/blob/master/docs/contributors/build_guide.md).
2. Or use a recent CI build such as [this one from PR
   2943](http://storage.googleapis.com/minikube-builds/2943/minikube-linux-amd64).

Once you have minikube installed run:

```bash
minikube start -p cluster1 --kubernetes-version v1.11.0
minikube start -p cluster2 --kubernetes-version v1.11.0
```
**NOTE:** Please make sure to set the correct context using the command below as this guide depends on it.
```bash
kubectl config use-context cluster1   
```

Even though the `minikube` cluster has been started, you'll want to verify all
your `minikube` components are up and ready by examining the state of the
kubernetes components in the clusters via:

```bash
kubectl get all --all-namespaces
```

Once all pods are running you can move on to deploy the cluster registry and
federation-v2 control plane.

## Automated Deployment

If you would like to have the deployment of the federation-v2 control plane
automated, then invoke the deployment script by running:

```bash
./scripts/deploy-federation-latest.sh cluster2
```

The above script joins the host cluster to the federation control plane it deploys, by default.
The argument(s) used is/are the list of context names of the additional clusters that needs to be
joined to this federation control plane. Clarifying, say the `host-cluster-context` used is `cluster1`,
then on successful completion of the script used in example, both `cluster1` and `cluster2` will be
joined to the deployed federation control plane.

**NOTE:** You can list multiple joining cluster names in the above command.
Also, please make sure the joining cluster name(s) provided matches the joining
cluster context from your kubeconfig. This will already be the case if you used
the minikube instructions above to create your clusters.

## Manual Deployment

If you'd like to understand what the script is automating for you, then proceed
by manually running the commands below.

### Deploy the Cluster Registry CRD

First you'll need to create the reserved namespace for registering clusters
with the cluster registry:

```bash
kubectl create ns kube-multicluster-public
```

Using this repo's vendored version of the [Cluster
Registry](https://github.com/kubernetes/cluster-registry), run the
following:

```bash
kubectl apply --validate=false -f vendor/k8s.io/cluster-registry/cluster-registry-crd.yaml
```

### Deploy Federation

First you'll need to create a permissive rolebinding to allow federation
controllers to run. This will eventually be made more restrictive, but for now
run:

```bash
kubectl create clusterrolebinding federation-admin --clusterrole=cluster-admin \
    --serviceaccount="federation-system:default"
```

Now you're ready to deploy federation v2 using the existing YAML config. This
config creates the `federation-system` namespace, RBAC resources, all the CRDs
supported, along with the service and statefulset for the
federation-controller-manager.

```bash
kubectl apply --validate=false -f hack/install-latest.yaml
```

**NOTE:** The validation fails for harmless reasons on kube >= 1.11 so ignore validation
until `kubebuilder` generation can pass validation.

Verify that the deployment succeeded and is available to serve its API by
seeing if we can retrieve one of its API resources:

```bash
kubectl -n federation-system get federatedcluster
```

It should successfully report that no resources are found.

### Enabling Push Propagation

Configuration of push propagation is via the creation of
FederatedTypeConfig resources. To enable propagation for default
supported types, run the following command:

```bash
for f in ./config/federatedtypes/*.yaml; do
    kubectl -n federation-system apply -f "${f}"
done
```

Once this is complete, you now have a working federation-v2 control plane and
can proceed to join clusters.

### Join Clusters

Next, you'll want to use the `kubefed2` tool to join all your
clusters that you want to test against.

1. Build kubefed2
    ```bash
    go build -o bin/kubefed2 cmd/kubefed2/kubefed2.go

    ```
1. Join Cluster(s)
    ```bash
    ./bin/kubefed2 join cluster1 --cluster-context cluster1 \
        --host-cluster-context cluster1 --add-to-registry --v=2
    ./bin/kubefed2 join cluster2 --cluster-context cluster2 \
        --host-cluster-context cluster1 --add-to-registry --v=2
    ```
You can repeat these steps to join any additional clusters.

**NOTE:** `cluster-context` will default to use the joining cluster name if not
specified.

#### Check Status of Joined Clusters

Check the status of the joined clusters until you verify they are ready:

```bash
kubectl -n federation-system describe federatedclusters
```

## Example

Follow these instructions for running an example to verify your deployment is
working. The example will create a test namespace with a
`federatednamespaceplacement` resource as well as federated template,
override, and placement resources for the following k8s resources: `configmap`,
`secret`, `deployment`, `service` and `serviceaccount`. It will then show how
to update the `federatednamespaceplacement` resource to move resources.

### Create the Test Namespace

First create the `test-namespace` for the test resources:

```bash
kubectl apply -f example/sample1/federatednamespace-template.yaml \
    -f example/sample1/federatednamespace-placement.yaml
```

### Create Test Resources

Create all the test resources by running:

```bash
kubectl apply -R -f example/sample1
```

### Check Status of Resources

Check the status of all the resources in each cluster by running:

```bash
for r in configmaps secrets service deployment serviceaccount job; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

Now make sure `nginx` is running properly in each cluster:

```bash
for c in cluster1 cluster2; do
    NODE_PORT=$(kubectl --context=${c} -n test-namespace get service \
        test-service -o jsonpath='{.spec.ports[0].nodePort}')
    echo; echo ------------ ${c} ------------; echo
    curl $(echo -n $(minikube ip -p ${c})):${NODE_PORT}
    echo; echo
done
```

### Update FederatedNamespacePlacement

Remove `cluster2` via a patch command or manually:

```bash
kubectl -n test-namespace patch federatednamespaceplacement test-namespace \
    --type=merge -p '{"spec":{"clusterNames": ["cluster1"]}}'

kubectl -n test-namespace edit federatednamespaceplacement test-namespace
```

Then wait to verify all resources are removed from `cluster2`:

```bash
for r in configmaps secrets service deployment serviceaccount job; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

We can quickly add back all the resources by simply updating the
`FederatedNamespacePlacement` to add `cluster2` again via a patch command or
manually:

```bash
kubectl -n test-namespace patch federatednamespaceplacement test-namespace \
    --type=merge -p '{"spec":{"clusterNames": ["cluster1", "cluster2"]}}'

kubectl -n test-namespace edit federatednamespaceplacement test-namespace
```

Then wait and verify all resources are added back to `cluster2`:

```bash
for r in configmaps secrets service deployment serviceaccount job; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

Lastly, make sure `nginx` is running properly in each cluster:

```bash
for c in cluster1 cluster2; do
    NODE_PORT=$(kubectl --context=${c} -n test-namespace get service \
        test-service -o jsonpath='{.spec.ports[0].nodePort}')
    echo; echo ------------ ${c} ------------; echo
    curl $(echo -n $(minikube ip -p ${c})):${NODE_PORT}
    echo; echo
done
```

If you were able to verify the resources removed and added back then you have
successfully verified a working federation-v2 deployment.

### Example Cleanup

To cleanup the example simply delete the namespace:

```bash
kubectl delete ns test-namespace
```

## Cleanup

### Deployment Cleanup

Run the following command to perform a cleanup of the cluster registry and
federation deployments:

```bash
./scripts/delete-federation.sh cluster2
```

The above script unjoins the host cluster from the federation control plane it deploys, by default.
The argument(s) used is/are the list of context names of the additional clusters that needs to be
unjoined from this federation control plane. Clarifying, say the `host-cluster-context` used is
`cluster1`, then on successful completion of the script used in example, both `cluster1` and `cluster2`
will be unjoined from the deployed federation control plane.

## Namespaced Federation

All prior instructions referred to the deployment and use of a
cluster-scoped federation control plane.  It is also possible to
deploy a namespace-scoped control plane.  In this mode of operation,
federation controllers will target resources in a single namespace on
both host and member clusters.  This may be desirable when
experimenting with federation on a production cluster.

### Automated Deployment

The only supported method to deploy namespaced federation is via the
deployment script configured with environment variables:

```bash
NAMESPACED=y FEDERATION_NAMESPACE=<namespace> scripts/deploy-federation.sh <image name>
```

- `NAMESPACED` indicates that the control plane should target a
single namespace - the same namespace it is deployed to.
- `FEDERATION_NAMESPACE`indicates the namespace to deploy the control
plane to.  The control plane will only have permission to access this
on both the host and member clusters.

It may be useful to supply `FEDERATION_NAMESPACE=test-namespace` to
allow the examples to work unmodified.

### Joining Clusters

Joining additional clusters to a namespaced federation requires
providing additional arguments to `kubefed2 join`:

- `--federation-namespace=<namespace>` to ensure the cluster is joined
  to the federation running in the specified namespace
- `--registry-namespace=<namespace>` if using `--add-to-registry` to
  ensure the cluster is registered in the correct namespace
- `--limited-scope=true` to ensure that the scope of the service account created in
  the target cluster is limited to the federation namespace

To join `mycluster` when `FEDERATION_NAMESPACE=test-namespace` was used for deployment:

```bash
./bin/kubefed2 join mycluster --cluster-context mycluster \
    --host-cluster-context mycluster --add-to-registry --v=2 \
    --federation-namespace=test-namespace --registry-namespace=test-namespace \
    --limited-scope=true
```

### Deployment Cleanup

Cleanup similarly requires the use of the same environment variables
employed by deployment:

```bash
NAMESPACED=y FEDERATION_NAMESPACE=<namespace> ./scripts/delete-federation.sh
```
