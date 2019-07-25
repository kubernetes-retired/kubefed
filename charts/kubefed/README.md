<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Kubernetes Cluster Federation](#kubernetes-cluster-federation)
  - [Prerequisites](#prerequisites)
  - [Installing the Chart](#installing-the-chart)
  - [Uninstalling the Chart](#uninstalling-the-chart)
  - [Configuration](#configuration)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Kubernetes Cluster Federation

Kubernetes Cluster Federation is a Kubernetes Incubator project. It builds on the sync controller
(a.k.a. push reconciler) from [Federation v1](https://github.com/kubernetes/federation/)
to iterate on the API concepts laid down in the [brainstorming
doc](https://docs.google.com/document/d/159cQGlfgXo6O4WxXyWzjZiPoIuiHVl933B43xhmqPEE/edit#)
and further refined in the [architecture
doc](https://docs.google.com/document/d/1ihWETo-zE8U_QNuzw5ECxOWX0Df_2BVfO3lC4OesKRQ/edit#).
Access to both documents is available to members of the
[kubernetes-sig-multicluster google
group](https://groups.google.com/forum/#!forum/kubernetes-sig-multicluster).

## Prerequisites

- Kubernetes 1.13+
- Helm 2.10+

## Configuring RBAC for Helm (Optional)

If your Kubernetes cluster has RBAC enabled, it will be necessary to
ensure that helm is deployed with a service account with the
permissions necessary to deploy KubeFed:

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

First, add the KubeFed chart repo to your local repository.
```bash
$ helm repo add kubefed-charts https://raw.githubusercontent.com/kubernetes-sigs/kubefed/master/charts

$ helm repo list
NAME            URL
kubefed-charts   https://raw.githubusercontent.com/kubernetes-sigs/kubefed/master/charts
```

With the repo added, available charts and versions can be viewed.
```bash
$ helm search kubefed
```

Install the chart and specify the version to install with the
`--version` argument. Replace `<x.x.x>` with your desired version.
```bash
$ helm install kubefed-charts/kubefed --name kubefed --version=<x.x.x> --namespace kube-federation-system
```

**NOTE:** For **namespace-scoped deployments** (configured with the `--set
global.scope=Namespaced` option in the `helm install` command): if you created
your namespace prior to installing the chart, make sure to **add a `name:
<namespace>` label to the namespace** using the following command:

```bash
kubectl label namespaces <namespace> name=<namespace>
```

This label is necessary to get proper validation for KubeFed core APIs. If the
namespace does not already exist, the `helm install` command will create the
namespace with this label by default.

## Uninstalling the Chart

Due to this helm [issue](https://github.com/helm/helm/issues/4440), the CRDs cannot be deleted
when delete helm release, so before delete the helm release, we need first delete all
of the CR and CRDs for KubeFed release.

Delete all KubeFed `FederatedTypeConfig`:

```bash
$ kubectl -n kube-federation-system delete FederatedTypeConfig --all
```

Delete all KubeFed CRDs:

```bash
$ kubectl delete crd $(kubectl get crd | grep -E 'kubefed.io' | awk '{print $1}')
```

Then you can uninstall/delete the `kubefed` release:

```bash
$ helm delete --purge kubefed
```

The command above removes all the Kubernetes components associated with the chart
and deletes the release.

## Configuration

The following tables lists the configurable parameters of the KubeFed
chart and their default values.

| Parameter                             | Description                                                                                                                                                                                 | Default                         |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------|
| controllermanager.enabled             | Specifies whether to enable the controller manager in KubeFed.                                                                                                                              | true                            |
| controllermanager.replicaCount        | Number of replicas for KubeFed controller manager.                                                                                                                                          | 2                               |
| controllermanager.repository          | Repo of the KubeFed image.                                                                                                                                                                  | quay.io/kubernetes-multicluster |
| controllermanager.image               | Name of the KubeFed image.                                                                                                                                                                  | kubefed                         |
| controllermanager.tag                 | Tag of the KubeFed image.                                                                                                                                                                   | latest                          |
| controllermanager.imagePullPolicy     | Image pull policy.                                                                                                                                                                          | IfNotPresent                    |
| controllermanager.featureGates.PushReconciler               | Push reconciler feature.                                                                                                                                              | true                            |
| controllermanager.featureGates.SchedulerPreferences         | Scheduler preferences feature.                                                                                                                                        | true                            |
| controllermanager.featureGates.CrossClusterServiceDiscovery | Cross cluster service discovery feature.                                                                                                                              | true                            |
| controllermanager.featureGates.FederatedIngress             | Federated ingress feature.                                                                                                                                            | true                            |
| controllermanager.clusterAvailableDelay   | Time to wait before reconciling on a healthy cluster.                                                                                                                                   | 20s                             |
| controllermanager.clusterUnavailableDelay | Time to wait before giving up on an unhealthy cluster.                                                                                                                                  | 60s                             |
| controllermanager.leaderElectLeaseDuration | The maximum duration that a leader can be stopped before it is replaced by another candidate.                                                                                          | 15s                             |
| controllermanager.leaderElectRenewDeadline | The interval between attempts by the acting master to renew a leadership slot before it stops leading. This must be less than or equal to `controllermanager.LeaderElectLeaseDuration. | 10s                             |
| controllermanager.leaderElectRetryPeriod   | The duration the clients should wait between attempting acquisition and renewal of a leadership.                                                                                       | 5s                              |
| controllermanager.leaderElectResourceLock  | The type of resource object that is used for locking during leader election. Supported options are `configmaps` and `endpoints`.                                                       | configmaps                      |
| controllermanager.clusterHealthCheckPeriod           | How often to monitor the cluster health.                                                                                                                                     | 10s                              |
| controllermanager.clusterHealthCheckFailureThreshold | Minimum consecutive failures for the cluster health to be considered failed after having succeeded.                                                                          | 3                               |
| controllermanager.clusterHealthCheckSuccessThreshold | Minimum consecutive successes for the cluster health to be considered successful after having failed.                                                                        | 1                               |
| controllermanager.clusterHealthCheckTimeout          | Duration after which the cluster health check times out.                                                                                                                     | 3s                               |
| controllermanager.syncController.adoptResources  | Whether to adopt pre-existing resource in member clusters.                                                                                                        		          | Enabled                         |
| global.scope                   | Whether the KubeFed namespace will be the only target for the control plane.                                                                                                                           | Cluster                         |

Specify each parameter using the `--set key=value[,key=value]` argument to
`helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be
provided while installing the chart. For example:

```bash
$ helm install kubefed-charts/kubefed --name kubefed --namespace kube-federation-system --values values.yaml
```
