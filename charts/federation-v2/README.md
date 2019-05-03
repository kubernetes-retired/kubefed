<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Kubernetes Federation V2](#kubernetes-federation-v2)
  - [Prerequisites](#prerequisites)
  - [Installing the Chart](#installing-the-chart)
  - [Uninstalling the Chart](#uninstalling-the-chart)
  - [Configuration](#configuration)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Kubernetes Federation V2

Kubernetes Federation V2 is a Kubernetes Incubator project. It builds on the sync controller
(a.k.a. push reconciler) from [Federation v1](https://github.com/kubernetes/federation/)
to iterate on the API concepts laid down in the [brainstorming
doc](https://docs.google.com/document/d/159cQGlfgXo6O4WxXyWzjZiPoIuiHVl933B43xhmqPEE/edit#)
and further refined in the [architecture
doc](https://docs.google.com/document/d/1ihWETo-zE8U_QNuzw5ECxOWX0Df_2BVfO3lC4OesKRQ/edit#).
Access to both documents is available to members of the
[kubernetes-sig-multicluster google
group](https://groups.google.com/forum/#!forum/kubernetes-sig-multicluster).

## Prerequisites

- Kubernetes 1.11+
- Helm 2.10+

## Configuring RBAC for Helm (Optional)

If your Kubernetes cluster has RBAC enabled, it will be necessary to
ensure that helm is deployed with a service account with the
permissions necessary to deploy federation:

```bash
$ cat << EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system
EOF

$ helm init --service-account tiller
```

## Installing the Chart

First you'll need to create the reserved namespace for registering clusters with the
cluster registry:

```bash
$ kubectl create ns kube-multicluster-public
```

Next, add the fedv2 chart repo to your local repository.
```bash
$ helm repo add fedv2-charts https://raw.githubusercontent.com/kubernetes-sigs/federation-v2/master/charts

$ helm repo list
NAME            URL
fedv2-charts   https://raw.githubusercontent.com/kubernetes-sigs/federation-v2/master/charts
```

With the repo added, available charts and versions can be viewed.
```bash
$ helm search federation-v2
```

Install the chart and specify the version to install with the
`--version` argument. Replace `<x.x.x>` with your desired version.
```bash
$ helm install fedv2-charts/federation-v2 --version=<x.x.x> --namespace federation-system
```

If you already have clusterregistry installed, skip installing it by
providing `--set clusterregistry.enabled=false` to the above command.

## Uninstalling the Chart

Due to this helm [issue](https://github.com/helm/helm/issues/4440), the CRDs cannot be deleted
when delete helm release, so before delete the helm release, we need first delete all
of the CR and CRDs for federation v2 release.

Delete all federation v2 `FederatedTypeConfig`:

```bash
$ kubectl -n federation-system delete FederatedTypeConfig --all
```

Delete all federation v2 CRDs:

```bash
$ kubectl delete crd $(kubectl get crd | grep -E 'federation.k8s.io' | awk '{print $1}')
```

If you want to delete `clusters.clusterregistry.k8s.io` as well, do it as follows:

```bash
$ kubectl delete crd clusters.clusterregistry.k8s.io
```

Then you can uninstall/delete the `federation-v2` release:

```bash
$ helm delete --purge federation-v2
```

The command above removes all the Kubernetes components associated with the chart
and deletes the release.

Delete the reserved namespace for registering clusters:

```bash
$ kubectl delete ns kube-multicluster-public
```

## Configuration

The following tables lists the configurable parameters of the Federation V2
chart and their default values.

| Parameter                             | Description                                                                                                                                                                                                 | Default                                                                                               |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| controllermanager.enabled             | Specifies whether to enable the controller manager in federation v2.                                                                                                                                        | true                                                                                                  |
| controllermanager.replicaCount        | Number of replicas for federation v2 controller manager.                                                                                                                                                     | 2                                                                                                     |
| controllermanager.repository          | Repo of the federation v2 image.                                                                                                                                                                            | quay.io/kubernetes-multicluster                                                                       |
| controllermanager.image               | Name of the federation v2 image.                                                                                                                                                                            | federation-v2                                                                                         |
| controllermanager.tag                 | Tag of the federation v2 image.                                                                                                                                                                             | latest                                                                                                |
| controllermanager.imagePullPolicy     | Image pull policy.                                                                                                                                                                                          | IfNotPresent                                                                                          |
| controllermanager.featureGates.PushReconciler               | Push reconciler feature.                                                                                                                                                              | true                                                                                                  |
| controllermanager.featureGates.SchedulerPreferences         | Scheduler preferences feature.                                                                                                                                                        | true                                                                                                  |
| controllermanager.featureGates.CrossClusterServiceDiscovery | Cross cluster service discovery feature.                                                                                                                                              | true                                                                                                  |
| controllermanager.featureGates.FederatedIngress             | Federated ingress feature.                                                                                                                                                            | true                                                                                                  |
| controllermanager.registryNamespace   | The cluster registry namespace.                                                                                                                                                                             | kube-multicluster-public                                                                              |
| controllermanager.clusterAvailableDelay   | Time to wait before reconciling on a healthy cluster.                                                                                                                                                   | 20s                                                                                                   |
| controllermanager.clusterUnavailableDelay | Time to wait before giving up on an unhealthy cluster.                                                                                                                                                  | 60s                                                                                                   |
| controllermanager.leaderElectLeaseDuration | The maximum duration that a leader can be stopped before it is replaced by another candidate.                                                                                                          | 15s                                                                                                   |
| controllermanager.leaderElectRenewDeadline | The interval between attempts by the acting master to renew a leadership slot before it stops leading. This must be less than or equal to `controllermanager.LeaderElectLeaseDuration.                 | 10s                                                                                                   |
| controllermanager.leaderElectRetryPeriod   | The duration the clients should wait between attempting acquisition and renewal of a leadership.                                                                                                       | 5s                                                                                                    |
| controllermanager.leaderElectResourceLock  | The type of resource object that is used for locking during leader election. Supported options are `configmaps` and `endpoints`.                                                                       | configmaps                                                                                            |
| controllermanager.clusterHealthCheckPeriodSeconds    | How often to monitor the cluster health (in seconds).                                                                                                                                        | 10                                                                                                    |
| controllermanager.clusterHealthCheckFailureThreshold | Minimum consecutive failures for the cluster health to be considered failed after having succeeded.                                                                                          | 3                                                                                                     |
| controllermanager.clusterHealthCheckSuccessThreshold | Minimum consecutive successes for the cluster health to be considered successful after having failed.                                                                                        | 1                                                                                                     |
| controllermanager.clusterHealthCheckTimeoutSeconds   | Number of seconds after which the cluster health check times out.                                                                                                                            | 3                                                                                                     |
| controllermanager.syncController.skipAdoptingResources  | Whether to skip adopting pre-existing resource in member clusters.                                                                                                                        | false                                                                                                 |
| clusterregistry.enabled               | Specifies whether to enable the clusterregistry in federation v2.                                                                                                                                           | true                                                                                                  |
| global.scope                   | Whether the federation namespace will be the only target for federation.                                                                                                                                           | Cluster                                                                                              |

Specify each parameter using the `--set key=value[,key=value]` argument to
`helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be
provided while installing the chart. For example:

```bash
$ helm install charts/federation-v2 --name federation-v2 --namespace federation-system --values values.yaml
```
