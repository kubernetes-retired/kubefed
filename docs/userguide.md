# User Guide

If you are looking to use federation v2, you've come to the right place. Below
is a walkthrough tutorial for how to deploy the federation v2 control plane.

## Prerequisites

### Binaries

The federation deployment depends on `apiserver-boot`, `crinit`, and `kubectl`
being installed in the path. These binaries can be installed via the
`download-binaries.sh` script, which downloads them to `./bin`:

```bash
./scripts/download-binaries.sh
export PATH=$(pwd)/bin:${PATH}
```

Or you can install them manually yourself using the guidelines provided below.

#### apiserver-builder

This repo depends on
[apiserver-builder](https://github.com/kubernetes-incubator/apiserver-builder)
to generate code and build binaries. Download a [recent
release](https://github.com/kubernetes-incubator/apiserver-builder/releases)
and install it in your `PATH`.

#### cluster-registry

Get version 0.0.4 of the [Cluster
Registry](https://github.com/kubernetes/cluster-registry/releases/tag/v0.0.4)
and install it in your `PATH`.

#### kubectl

Download a version of
[`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and
install it in your `PATH`.

### Create Clusters

The quickest way to set up clusters for use with the federation-v2 control
plane is to use [minikube](https://kubernetes.io/docs/getting-started-guides/minikube/).

**NOTE: You will need to use minikube version
[0.25.2](https://github.com/kubernetes/minikube/releases/tag/v0.25.2) in order
to avoid an issue with profiles, which is required for testing multiple
clusters. See issue https://github.com/kubernetes/minikube/issues/2717 for more
details.**

Once you have minikube installed run:

```bash
minikube start -p cluster1
minikube start -p cluster2
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

If you would like to have the deployment of the cluster registry and
federation-v2 control plane automated, then invoke the deployment script by
running:

```bash
./scripts/deploy-federation.sh <image> cluster2
```

Where `<image>` is in the form
`<containerregistry>/<username>/<imagename>:<tagname>` e.g.
`docker.io/<username>/federation-v2:test`.

## Manual Deployment

If you'd like to understand what the script is automating for you, then proceed
by manually running the commands below.

### Deploy the Cluster Registry

Make sure the storage provisioner is ready before deploying the Cluster
Registry.

```bash
kubectl -n kube-system get pod storage-provisioner
```

Get version 0.0.4 of the [Cluster
Registry](https://github.com/kubernetes/cluster-registry/releases/tag/v0.0.4) and run the
following:

```bash
crinit aggregated init mycr --host-cluster-context=cluster1
```

You can also specify your own cluster registry image using the `--image` flag.

### Deploy Federation

First you'll need to create the namespace to deploy the federation into:

```bash
kubectl create ns federation
```

Then run `apiserver-boot` to deploy. This will build the federated `apiserver`
and `controller`, build an image containing them, push the image to the
requested registry, generate a k8s YAML config `config/apiserver.yaml`
with the name and namespace requested, and deploy the config to the
cluster specified in your `kubeconfig config current-context`.

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

### Enabling Push Propagation

Configuration of push propagation is via the creation of
FederatedTypeConfig resources. To enable propagation for default
supported types, run the following command:

```bash
for tc in ./config/federatedtypes/*.yaml; do kubectl create -f "${tc}"; done
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
    ./bin/kubefed2 join cluster1 --host-cluster-context cluster1 --add-to-registry --v=2
    ./bin/kubefed2 join cluster2 --host-cluster-context cluster1 --add-to-registry --v=2
    ```
You can repeat these steps to join any additional clusters.

#### Check Status of Joined Clusters

Check the status of the joined clusters until you verify they are ready:

```bash
kubectl -n federation describe federatedclusters
```

## Example

Follow these instructions for running an example to verify your deployment is
working. The example will create a test namespace with a
`federatednamespaceplacement` resource as well as federated template,
override, and placement resources for the following k8s resources: `configmap`,
`secret`, and `deployment`. It will then show how to update the
`federatednamespaceplacement` resource to move resources.

### Create Test Resources

Create all the test resources by running:

```bash
kubectl create -f example/sample1
```

### Check Status of Resources

Check the status of all the resources in each cluster by running:

```bash
for r in configmaps secrets deploy; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

### Update FederatedNamespacePlacement

Remove `cluster2` via a patch command or manually:

```bash
kubectl -n test-namespace patch federatednamespaceplacement test-namespace -p \
    '{"spec":{"clusternames": ["cluster1"]}}'

kubectl -n test-namespace edit federatednamespaceplacement test-namespace
```

Then wait to verify all resources are removed from `cluster2`:

```bash
for r in configmaps secrets deploy; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

We can quickly add back all the resources by simply updating the
`FederatedNamespacePlacement` to add `cluster2` again via a patch command or
manually:

```bash
kubectl -n test-namespace patch federatednamespaceplacement test-namespace -p \
    '{"spec":{"clusternames": ["cluster1", "cluster2"]}}'

kubectl -n test-namespace edit federatednamespaceplacement test-namespace
```

Then wait and verify all resources are added back to `cluster2`:

```bash
for r in configmaps secrets deploy; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

If you were able to verify the resources removed and added back then you have
successfully verified a working federation-v2 deployment.

### Example Cleanup

To cleanup the example simply delete the namespace and its placement:

```bash
kubectl delete federatednamespaceplacement test-namespace
kubectl delete ns test-namespace
```

## Cleanup

### Unjoin Clusters

In order to unjoin, you will have to manually or script an unjoin
operation by deleting all the relevant objects. The ability to unjoin via
`kubefed2` will be added in the future.  For now you can run the following
commands to take care of it:

```bash
HOST_CLUSTER=cluster1
for c in cluster1 cluster2; do
    if [[ "${c}" != "${HOST_CLUSTER}" ]]; then
        kubectl --context=${c} delete ns federation
    else
        kubectl --context=${c} -n federation delete sa ${c}-${HOST_CLUSTER}
    fi
    kubectl --context=${c} delete clusterrolebinding \
        federation-controller-manager:${c}-${HOST_CLUSTER}
    kubectl --context=${c} delete clusterrole \
        federation-controller-manager:${c}-${HOST_CLUSTER}
    kubectl delete clusters ${c}
    kubectl delete federatedclusters ${c}
    CLUSTER_SECRET=$(kubectl -n federation get secrets \
        -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}' | grep "${c}-*")
    kubectl -n federation delete secret ${CLUSTER_SECRET}
done
```

### Deployment Cleanup

Run the following command to perform a cleanup of the cluster registry and
federation deployments:

```bash
./scripts/delete-federation.sh
```
