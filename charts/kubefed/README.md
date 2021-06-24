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

- Kubernetes 1.16+
- Helm 3.2+

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
$ helm search repo kubefed
```

Install the chart and specify the version to install with the
`--version` argument. Replace `<x.x.x>` with your desired version.
If you don't want to install CRDs, add a `--skip-crds` at the end of the line:

```bash
$ helm --namespace kube-federation-system upgrade -i kubefed kubefed-charts/kubefed --version=<x.x.x> --create-namespace

Release "kubefed" does not exist. Installing it now.
NAME: kubefed
LAST DEPLOYED: Wed Aug  5 16:03:46 2020
NAMESPACE: kube-federation-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
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

Delete all KubeFed `FederatedTypeConfig`:

```bash
$ kubectl -n kube-federation-system delete FederatedTypeConfig --all
```

Then you can uninstall/delete the `kubefed` release:

```bash
$ helm --namespace kube-federation-system uninstall kubefed
```

The command above removes all the Kubernetes components associated with the chart
and deletes the release.

## Configuration

The following tables lists the configurable parameters of the KubeFed
chart and their default values.

| Parameter                             | Description                                                                                                                                                                                 | Default                         |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------|
| controllermanager.enabled             | Specifies whether to enable the controller manager in KubeFed.                                                                                                                              | true                            |
| controllermanager.imagePullSecrets    | Image pull secrets.                                                                                                                                                                         | []
| controllermanager.commonTolerations   | Tolerations for all the pods.                                                                                                                                                               | []                              |
| controllermanager.commonNodeSelector     | Node selector for all the pods.                                                                                                                                                          | {}                              |
| controllermanager.controller.repository        | Repo of the KubeFed image.                                                                                                                                                         | quay.io/kubernetes-multicluster |
| controllermanager.controller.image             | Name of the KubeFed image.                                                                                                                                                         | kubefed                         |
| controllermanager.controller.tag               | Tag of the KubeFed image.                                                                                                                                                          | canary                          |
| controllermanager.controller.replicaCount      | Number of replicas for KubeFed controller manager.                                                                                                                                 | 2                          |
| controllermanager.controller.imagePullPolicy   | Image pull policy.                                                                                                                                                                 | IfNotPresent                          |
| controllermanager.webhook.repository           | Repo of the KubeFed image.                                                                                                                                                         | quay.io/kubernetes-multicluster |
| controllermanager.webhook.image                | Name of the KubeFed image.                                                                                                                                                         | kubefed                         |
| controllermanager.webhook.tag                  | Tag of the KubeFed image.                                                                                                                                                          | canary                          |
| controllermanager.webhook.imagePullPolicy   | Image pull policy.                                                                                                                                                                 | IfNotPresent                          |
| controllermanager.featureGates.PushReconciler               | Push reconciler feature.                                                                                                                                              | true                            |
| controllermanager.featureGates.RawResourceStatusCollection               | Raw collection of resource status on target clusters feature.                                                                                                                                              | false                            |
| controllermanager.featureGates.SchedulerPreferences         | Scheduler preferences feature.                                                                                                                                        | true                            |
| controllermanager.clusterAvailableDelay   | Time to wait before reconciling on a healthy cluster.                                                                                                                                   | 20s                             |
| controllermanager.clusterUnavailableDelay | Time to wait before giving up on an unhealthy cluster.                                                                                                                                  | 60s                             |
| controllermanager.leaderElectLeaseDuration | The maximum duration that a leader can be stopped before it is replaced by another candidate.                                                                                          | 15s                             |
| controllermanager.leaderElectRenewDeadline | The interval between attempts by the acting master to renew a leadership slot before it stops leading. This must be less than or equal to `controllermanager.LeaderElectLeaseDuration. | 10s                             |
| controllermanager.leaderElectRetryPeriod   | The duration the clients should wait between attempting acquisition and renewal of a leadership.                                                                                       | 5s                              |
| controllermanager.leaderElectResourceLock  | The type of resource object that is used for locking during leader election. Supported options are `configmaps` and `endpoints`.                                                       | configmaps                      |
| controllermanager.clusterHealthCheckPeriod           | How often to monitor the cluster health.                                                                                                                                     | 10s                             |
| controllermanager.clusterHealthCheckFailureThreshold | Minimum consecutive failures for the cluster health to be considered failed after having succeeded.                                                                          | 3                               |
| controllermanager.clusterHealthCheckSuccessThreshold | Minimum consecutive successes for the cluster health to be considered successful after having failed.                                                                        | 1                               |
| controllermanager.clusterHealthCheckTimeout          | Duration after which the cluster health check times out.                                                                                                                     | 3s                              |
| controllermanager.syncController.maxConcurrentReconciles | The maximum number of concurrent Reconciles of sync controller which can be run.                                                                                         | 1                               |
| controllermanager.syncController.adoptResources          | Whether to adopt pre-existing resource in member clusters.                                                                                                        		  | Enabled                         |
| controllermanager.statusController.maxConcurrentReconciles | The maximum number of concurrent Reconciles of status controller which can be run.                                                                                     | 1                               |
| controllermanager.service.labels                     | Kubernetes labels attached to the controller manager's services                                                                                                       		    | {}                              |
| controllermanager.certManager.enabled             | Specifies whether to enable the usage of the cert-manager for the certificates generation.                                                                                      | false                           |
| controllermanager.certManager.rootCertificate.organizations       | Specifies the list of organizations to include in the cert-manager generated root certificate.                                                                  | []                              |
| controllermanager.certManager.rootCertificate.commonName     | Specifies the CN value for the cert-manager generated root certificate.                                                                                              | ca.webhook.kubefed              |
| controllermanager.certManager.rootCertificate.dnsNames       | Specifies the list of subject alternative names for the cert-manager generated root certificate.                                                                     | ["ca.webhook.kubefed"]          |
| controllermanager.postInstallJob.repository        | Repo of the kubectl image for the post-install job                                                                                                                             | bitnami                         |
| controllermanager.postInstallJob.image             | Name of the kubectl image for the post-install job                                                                                                                             | kubectl                         |
| controllermanager.postInstallJob.tag               | Tag of the kubectl image for the post-install                                                                                                                                  | 1.17.16                         |
| controllermanager.postInstallJob.imagePullPolicy   | Image pull policy of the kubectl post-install job                                                                                                                              | IfNotPresent                    |
| global.scope                   | Whether the KubeFed namespace will be the only target for the control plane.                                                                                                                       | Cluster                         |

Specify each parameter using the `--set key=value[,key=value]` argument to
`helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be
provided while installing the chart. For example:

```bash
$ helm install kubefed-charts/kubefed --name kubefed --namespace kube-federation-system --values values.yaml --devel
```

## Migration from Helm v2 to v3

Helm v3 has a built-in migration feature which can easy move your current Helm v2 installation to Helm v3.

Download Helm v3 CLI from [Release Page](https://github.com/helm/helm/releases).

Convert your kubefed installation to Helm v3:


```bash
$ helm 2to3 convert kubefed
2020/08/06 18:50:57 Release "kubefed" will be converted from Helm v2 to Helm v3.
2020/08/06 18:50:57 [Helm 3] Release "kubefed" will be created.
2020/08/06 18:50:57 [Helm 3] ReleaseVersion "kubefed.v1" will be created.
2020/08/06 18:50:58 [Helm 3] ReleaseVersion "kubefed.v1" created.
2020/08/06 18:50:58 [Helm 3] Release "kubefed" created.
2020/08/06 18:50:58 Release "kubefed" was converted successfully from Helm v2 to Helm v3.
2020/08/06 18:50:58 Note: The v2 release information still remains and should be removed to avoid conflicts with the migrated v3 release.
2020/08/06 18:50:58 v2 release information should only be removed using `helm 2to3` cleanup and when all releases have been migrated over.
```

Check your successful migration:

```bash
$ helm -n kube-federation-system list
NAME    NAMESPACE               REVISION        UPDATED                                 STATUS          CHART           APP VERSION
kubefed kube-federation-system  1               2020-08-06 16:49:41.593438079 +0000 UTC deployed        kubefed-0.3.1
```
