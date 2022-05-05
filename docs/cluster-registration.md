<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Joining Clusters](#joining-clusters)
- [Checking status of joined clusters](#checking-status-of-joined-clusters)
- [Joining kind clusters on MacOS](#joining-kind-clusters-on-macos)
- [Unjoining clusters](#unjoining-clusters)
- [Joining additional clusters in a namespace scoped deployment](#joining-additional-clusters-in-a-namespace-scoped-deployment)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Joining Clusters

You can use the `kubefedctl` tool to join clusters as follows.

```bash
kubefedctl join cluster1 --cluster-context cluster1 \
    --host-cluster-context cluster1 --v=2
kubefedctl join cluster2 --cluster-context cluster2 \
    --host-cluster-context cluster1 --v=2
```

Repeat this step to join any additional clusters.

**NOTE:** `cluster-context` will default to use the joining cluster name if not
specified.
**NOTE:** Before the [PR](https://github.com/kubernetes-sigs/kubefed/pull/1361), `kubefed` automatically fetches apiserver's `certificate-authority-data` from member cluster, after that kubefed will use `certificate-authority-data` in joining cluster's kubeconfig file.

# Checking status of joined clusters

Check the status of the joined clusters by using the following command.

```bash
kubectl -n kube-federation-system get kubefedclusters

NAME       AGE   READY   KUBERNETES-VERSION
cluster1   1m    True    v1.21.2
cluster2   1m    True    v1.22.0

```

The Kubernetes version is checked periodically along with the cluster health check so that it would be automatically updated within the cluster health check period after a Kubernetes upgrade/downgrade of the cluster.

# Joining kind clusters on MacOS

A Kubernetes cluster deployed with [kind](https://sigs.k8s.io/kind) on Docker
for MacOS will have an API endpoint of `https://localhost:<random-port>` in its
kubeconfig context. Such an endpoint will be compatible with local invocations
of cli tools like `kubectl`. The same endpoint will not be reachable from a
KubeFed control plane, and the endpoints of kind clusters joined to a KubeFed
control plane will need to be updated to `https://<kind pod ip>:6443`. This can
be accomplished by executing the following script after a cluster is registered
to the control plane with `kubefedctl join`.

```bash
./scripts/fix-joined-kind-clusters.sh
```

# Unjoining clusters

You can unjoin clusters using `kubefedctl` tool as follows.

```bash
kubefedctl unjoin cluster2 --cluster-context cluster2 --host-cluster-context cluster1 --v=2
```
Repeat this step to unjoin any additional clusters.

# Joining additional clusters in a namespace scoped deployment

Joining additional clusters to a namespaced control plane requires
providing additional arguments to `kubefedctl join`:

- `--kubefed-namespace=<namespace>` to ensure the cluster has been joined
  with the KubeFed control plane running in the specified namespace

You can join `mycluster` to a control plane deployed in namespace `test-namespace` as follows.

```bash
kubefedctl join mycluster --cluster-context mycluster \
    --host-cluster-context mycluster --v=2 \
    --kubefed-namespace=test-namespace
```
