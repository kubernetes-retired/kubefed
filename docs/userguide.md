# User Guide

If you are looking to use federation v2, you've come to the right place. Below
is a walkthrough tutorial for how to deploy the federation v2 control plane.

Please refer to [Federation V2 Concepts](./concepts.md) first before you go through this user guide.

## Prerequisites

The federation v2 deployment requires kubernetes version >= 1.11. The following
is a detailed list of binaries required.

### Binaries

The federation deployment depends on `kubebuilder`, `etcd`, `kubectl`, and
`kube-apiserver` >= v1.11 being installed in the path. The `kubebuilder`
([v1.0.4](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v1.0.4)
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
to generate code and build binaries. Download the [v1.0.4
release](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v1.0.4)
and install it in your `PATH`.

### Deployment Image

If you follow this user guide without any changes you will be using the latest
stable released version of the federation-v2 image tagged as `latest`.
Alternatively, we support the ability to deploy the [latest master image tagged
as `canary`](development.md#test-latest-master-changes-canary) or [your own
custom image](development.md#test-your-changes).

### Create Clusters

The federation v2 control plane can run on any v1.11 or greater Kubernetes clusters. The following is a list of
Kubernetes environments that have been tested and are supported by the Federation v2 community:

- [kind](./environments/kind.md)

- [Minikube](./environments/minikube.md)

- [Google Kubernetes Engine (GKE)](./environments/gke.md)

After completing the steps in one of the above guides, return here to continue the Federation v2 deployment.

### Set Cluster Environment Variables

Please set the following variables in order to continue through this guide.
Replace the name of the clusters with the respective name of the clusters
you're using if they differ.

```bash
HOST_CLUSTER=cluster1
MEMBER_CLUSTER=cluster2
CLUSTERS="${HOST_CLUSTER} ${MEMBER_CLUSTER}"
```

### Set Host Cluster Context
**NOTE:** You must set the correct context using the command below as this guide depends on it.
```bash
kubectl config use-context ${HOST_CLUSTER}
```

## Helm Chart Deployment

You can refer to [helm chart installation guide](https://github.com/kubernetes-sigs/federation-v2/blob/master/charts/federation-v2/README.md)
to install federation-v2.

## Automated Deployment

If you would like to have the deployment of the federation-v2 control plane
automated, then invoke the deployment script by running:

```bash
./scripts/deploy-federation-latest.sh ${MEMBER_CLUSTER}
```

The above script joins the host cluster to the federation control plane it
deploys, by default.  The argument(s) used is/are the list of context names of
the additional clusters that needs to be joined to this federation control
plane. Clarifying, say the `host-cluster-context` used is `HOST_CLUSTER`, then
on successful completion of the script used in example, both `HOST_CLUSTER` and
`MEMBER_CLUSTER` will be joined to the deployed federation control plane.

**NOTE:** You can list multiple joining cluster names in the above command.
Also, please make sure the joining cluster name(s) provided matches the joining
cluster context from your kubeconfig. This will already be the case if you used
the kind instructions above to create your clusters.

## Operations

### Join Clusters

If you want to join your clusters manually, you'll want to use the `kubefed2`
tool to join all your clusters that you want to test against.

```bash
for c in ${CLUSTERS}: do
./bin/kubefed2 join ${c} --cluster-context ${c} \
    --host-cluster-context ${HOST_CLUSTER} --add-to-registry --v=2
done
```

You can repeat these steps to join any additional clusters.

**NOTE:** `cluster-context` will default to use the joining cluster name if not
specified.

### Check Status of Joined Clusters

Check the status of the joined clusters until you verify they are ready:

```bash
kubectl -n federation-system describe federatedclusters
```

The output of `Status.Conditions` field should include cluster ready info as follows:

```yaml
Status:
  Conditions:
    ...
    Reason:                ClusterReady
    Status:                True
    Type:                  Ready
```

## Example

Follow these instructions for running an example to verify your deployment is
working. The example will create a test namespace with a
`federatednamespaceplacement` resource as well as federated template, override,
and placement resources for the below k8s resources. It will then show how to
update the `federatednamespaceplacement` resource to move resources. So let's
set a variable with all the resource types for convenience:

```bash
RESOURCE_TYPES="deployment ingress job secret service"
```

### Create the Test Namespace

First create the `test-namespace` for the test resources:

```bash
kubectl apply -f example/propagation/federatednamespace-template.yaml
```

### Create the Test Namespace Placement

Now we create the `federatednamespaceplacement` resource for `test-namespace`.

```bash
kubectl apply -f example/propagation/federatednamespace-placement.yaml
```

### Update FederatedNamespacePlacement

Add both clusters via a patch command:

```bash
kubectl -n test-namespace patch federatednamespaceplacement test-namespace \
    --type=merge -p "{\"spec\":{\"clusterNames\": [\"${HOST_CLUSTER}\", \"${MEMBER_CLUSTER}\"]}}"
```

Or add both clusters manually:

```bash
kubectl -n test-namespace edit federatednamespaceplacement test-namespace
```

### Create Test Resources

Create all the test resources by running:

```bash
for r in ${RESOURCE_TYPES}; do
    for f in $(ls example/propagation/federated${r}-*.yaml); do
        kubectl apply -f ${f}
    done
done
```

**NOTE:** If you get the following error while creating a test resource i.e.

```
unable to recognize "example/propagation/Federated<type>-placement.yaml": no matches for kind "Federated<type>Placement" in version "primitives.federation.k8s.io/v1alpha1",
```
then it indicates that a given type may need to be federated by `kubefed2 federate enable <type>`

### Update Placement for Test Resources

Update the placement resource for each type to contain both clusters:

```bash
for r in ${RESOURCE_TYPES}; do
    kubectl -n test-namespace patch federated${r}placement test-${r} \
        --type=merge -p "{\"spec\":{\"clusterNames\": [\"${HOST_CLUSTER}\", \"${MEMBER_CLUSTER}\"]}}"
done
```

### Check Status of Resources

Check the status of all the resources in each cluster by running:

```bash
for r in ${RESOURCE_TYPES}; do
    for c in ${CLUSTERS}; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

Now make sure `nginx` is running properly in each cluster. To do this you'll
need the `NODE_IP` and `NODE_PORT` for each cluster. The command to retrieve
the `NODE_IP` below is specific to clusters set up with `kind`. Update the
command if you set up your clusters using some other mechanism.

```bash
for c in ${CLUSTERS}; do
    NODE_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' \
        kind-${c//[!0-9]/}-control-plane)
    NODE_PORT=$(kubectl --context=${c} -n test-namespace get service \
        test-service -o jsonpath='{.spec.ports[0].nodePort}')
    echo; echo "------------ ${c} (${NODE_IP}:${NODE_PORT}) ------------"; echo
    curl ${NODE_IP}:${NODE_PORT}
    echo; echo
done
```

### Update FederatedNamespacePlacement

Remove `MEMBER_CLUSTER` via a patch command:

```bash
kubectl -n test-namespace patch federatednamespaceplacement test-namespace \
    --type=merge -p "{\"spec\":{\"clusterNames\": [\"${HOST_CLUSTER}\"]}}"
```

Or remove `MEMBER_CLUSTER` manually:

```bash
kubectl -n test-namespace edit federatednamespaceplacement test-namespace
```

Then wait to verify all resources are removed from `MEMBER_CLUSTER`:

```bash
for r in ${RESOURCE_TYPES}; do
    for c in ${CLUSTERS}; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

We can quickly add back all the resources by simply updating the
`FederatedNamespacePlacement` to add `MEMBER_CLUSTER` again via a patch
command:

```bash
kubectl -n test-namespace patch federatednamespaceplacement test-namespace \
    --type=merge -p "{\"spec\":{\"clusterNames\": [\"${HOST_CLUSTER}\", \"${MEMBER_CLUSTER}\"]}}"
```

Or add `MEMBER_CLUSTER` again manually:

```bash
kubectl -n test-namespace edit federatednamespaceplacement test-namespace
```

Then wait and verify all resources are added back to `MEMBER_CLUSTER`:

```bash
for r in ${RESOURCE_TYPES}; do
    for c in ${CLUSTERS}; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

Lastly, make sure `nginx` is running properly in each cluster:

```bash
for c in ${CLUSTERS}; do
    NODE_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' \
        kind-${c//[!0-9]/}-control-plane)
    NODE_PORT=$(kubectl --context=${c} -n test-namespace get service \
        test-service -o jsonpath='{.spec.ports[0].nodePort}')
    echo; echo "------------ ${c} (${NODE_IP}:${NODE_PORT}) ------------"; echo
    curl ${NODE_IP}:${NODE_PORT}
    echo; echo
done
```

If you were able to verify the resources removed and added back then you have
successfully verified a working federation-v2 deployment.

### Example: federating a CRD
Please use this [guide](./federating_crds.md) to federate a CRD in the target cluster.

### Example Cleanup

To cleanup the example simply delete the namespace:

```bash
kubectl delete ns test-namespace
```

### Troubleshooting

If federated resources are not propagated as expected to the member clusters, you can
use the following command to view `Events` which may aid in diagnosing the problem.

```bash
kubectl describe <federated CRD> <CR name> -n test-namespace
```

An example for CRD of `federateddeployments` is as follows:

```bash
kubectl describe federateddeployments test-deployment -n test-namespace
```

It may also be useful to inspect the federation controller log as follows:

```bash
kubectl logs -f federation-controller-manager-0 -n federation-system
```

## Cleanup

### Deployment Cleanup

Run the following command to perform a cleanup of the cluster registry and
federation deployments:

```bash
./scripts/delete-federation.sh
```

The above script unjoins the all of the clusters from the federation control
plane it deploys, by default. On successful completion of the script used in
example, both `HOST_CLUSTER` and `MEMBER_CLUSTER` will be unjoined from the
deployed federation control plane.

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
NAMESPACED=y FEDERATION_NAMESPACE=<namespace> scripts/deploy-federation.sh <image name> <joining cluster list>
```

- `NAMESPACED` indicates that the control plane should target a
single namespace - the same namespace it is deployed to.
- `FEDERATION_NAMESPACE`indicates the namespace to deploy the control
plane to.  The control plane will only have permission to access this
on both the host and member clusters.

It may be useful to supply `FEDERATION_NAMESPACE=test-namespace` to
allow the examples to work unmodified. You can run following command
to set up the test environment with `HOST_CLUSTER` and `MEMBER_CLUSTER`.

```bash
NAMESPACED=y FEDERATION_NAMESPACE=test-namespace scripts/deploy-federation.sh \
    <containerregistry>/<username>/federation-v2:test ${MEMBER_CLUSTER}
```

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
### Multi-Cluster Ingress DNS

Multi-Cluster Ingress DNS provides the ability to programmatically manage DNS resource records of Ingress objects
through [ExternalDNS](https://github.com/kubernetes-incubator/external-dns) integration. Review the
[Multi-Cluster Ingress DNS with ExternalDNS Guide](./ingressdns-with-externaldns.md) to learn more.

### Multi-Cluster Service DNS

Multi-Cluster Service DNS provides the ability to programmatically manage DNS resource records of Service objects
through [ExternalDNS](https://github.com/kubernetes-incubator/external-dns) integration. Review the
[Multi-Cluster Service DNS with ExternalDNS Guide](./servicedns-with-externaldns.md) to learn more.

### Deployment Cleanup

Cleanup similarly requires the use of the same environment variables
employed by deployment:

```bash
NAMESPACED=y FEDERATION_NAMESPACE=<namespace> ./scripts/delete-federation.sh
```

## Higher order behaviour

The architecture of federation v2 API allows higher level APIs to be 
constructed using the mechanics provided by the core API types (`template`, 
`placement` and `override`) and associated controllers for a given resource.
Further sections describe few of higher level APIs implemented as part of 
Federation V2.

###  ReplicaSchedulingPreference

ReplicaSchedulingPreference provides an automated mechanism of distributing 
and maintaining total number of replicas for `deployment` or `replicaset` based 
federated workloads into federated clusters. This is based on high level 
user preferences given by the user. These preferences include the semantics 
of weighted distribution and limits (min and max) for distributing the replicas. 
These also include semantics to allow redistribution of replicas dynamically 
in case some replica pods remain unscheduled in some clusters, for example 
due to insufficient resources in that cluster. 

RSP is used in place of ReplicaSchedulingPreference for brevity in text further on.

The RSP controller works in a sync loop observing the RSP resource and the 
matching `namespace/name` pair `FederatedDeployment` or `FederatedReplicaset` 
resource. 
If it finds that both RSP and its target template resource, the type of which 
is specified using `spec.targetKind`, exists, it goes ahead to list currently 
healthy clusters and distributes the `spec.totalReplicas` using the associated 
per cluster user preferences. If the per cluster preferences are absent, it 
distributes the `spec.totalReplicas` evenly among all clusters. It updates (or 
creates if missing) the same `namespace/name` `placement` and `overrides` for the 
`targetKind` with the replica values calculated, leveraging the sync controller 
to actually propagate the k8s resource to federated clusters. Its noteworthy that 
if an RSP is present, the `spec.replicas` from the `template` resource are unused. 
RSP also provides a further more useful feature using `spec.rebalance`. If this is 
set to `true`, the RSP controller monitors the replica pods for target replica 
workload from each federated cluster and if it finds that some clusters are not 
able to schedule those pods for long, it moves (rebalances) the replicas to 
clusters where all the pods are running and healthy. This in other words helps 
moving the replica workloads to those clusters where there is enough capacity 
and away from those clusters which are currently running out of capacity. The 
`rebalance` feature might cause initial shuffle of replicas to  reach an eventually 
balanced state of distribution. The controller might further keep trying to move 
few replicas back into the cluster(s) which ran out of capacity, to check if it can 
be scheduled again to reach the normalised state (even distribution or the state 
desired by user preferences), which apparently is the only mechanism to check if 
this cluster has capacity now. The `spec.rebalance` should not be used if this 
behaviour is unacceptable.

The RSP can be considered as more user friendly mechanism to distribute the 
replicas, where the inputs needed from the user at federated control plane are 
reduced. The user only needs to create the RSP resource and the mapping template 
resource, to distribute the replicas. It can also be considered as a more 
automated approach at distribution and further reconciliation of the workload 
replicas.

The usage of the RSP semantics is illustrated using some examples below. The 
examples considers 3 federated clusters `A`, `B` and `C`.

#### Distribute total replicas evenly in all available clusters

```bash
apiVersion: scheduling.federation.k8s.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 9
```
or 
```bash
apiVersion: scheduling.federation.k8s.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 9
  clusters:
    "*":
      weight: 1
```
A, B and C get 3 replicas each.

#### Distribute total replicas in weighted proportions

```bash
apiVersion: scheduling.federation.k8s.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 9
  clusters:
    A:
      weight: 1
    B:
      weight: 2
```
A gets 3 and B gets 6 replicas in the proportion of 1:2. C does not get 
any replica as missing weight preference is considered as weight=0.

#### Distribute replicas in weighted proportions, also enforcing replica limits per cluster

```bash
apiVersion: scheduling.federation.k8s.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 9
  clusters:
    A:
      minReplicas: 4
      maxReplicas: 6
      weight: 1
    B:
      minReplicas: 4
      maxReplicas: 8
      weight: 2
```
A gets 4 and B get 5 as weighted distribution is capped by cluster A minReplicas=4. 

#### Distribute replicas evenly in all clusters, however not more then 20 in C

```bash
apiVersion: scheduling.federation.k8s.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 50
  clusters:
    "*":
      weight: 1
    "C":
      maxReplicas: 20
      weight: 1
```
Possible scenarios

All have capacity. 
```
Replica layout: A=16 B=17 C=17.
```
B is offline/has no capacity
```
Replica layout: A=30 B=0 C=20
```
A and B are offline: 
```
Replica layout: C=20 
```
