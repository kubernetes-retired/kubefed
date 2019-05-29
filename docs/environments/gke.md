<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Google Kubernetes Engine (GKE) Deployment Guide](#google-kubernetes-engine-gke-deployment-guide)
  - [RBAC Workaround](#rbac-workaround)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Google Kubernetes Engine (GKE) Deployment Guide

KubeFed can be deployed to and manage [GKE](https://cloud.google.com/kubernetes-engine/) clusters running
Kubernetes v1.13 or greater. The following example deploys two GKE clusters named `cluster1` and `cluster2`.

```bash
export GKE_VERSION=1.11.2-gke.15
export ZONE=$(gcloud config get-value compute/zone)
gcloud container clusters create cluster1 --zone $ZONE --cluster-version $GKE_VERSION
gcloud container clusters create cluster2 --zone $ZONE --cluster-version $GKE_VERSION
```

If you are following along with the KubeFed [User Guide](../userguide.md), change the cluster context names:

```bash
export GCP_PROJECT=$(gcloud config list --format='value(core.project)')
kubectl config rename-context gke_${GCP_PROJECT}_${ZONE}_cluster1 cluster1
kubectl config rename-context gke_${GCP_PROJECT}_${ZONE}_cluster2 cluster2
```

Before proceeding with the KubeFed deployment, you must complete the steps in the RBAC Workaround section of this
document.

## RBAC Workaround

You can expect the following error when deploying KubeFed to Google Kubernetes Engine (GKE)
v1.6 or later:

```
<....>
Error from server (Forbidden): error when creating "hack/install-latest.yaml": clusterroles.rbac.authorization.k8s.io
"kubefed-role" is forbidden: attempt to grant extra privileges:
<....>
```

This is due to how GKE verifies permissions. From
[Google Kubernetes Engine docs](https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control):

> Because of the way GKE checks permissions when you create a Role or ClusterRole, you must first create a RoleBinding
> that grants you all of the permissions included in the role you want to create.
> An example workaround is to create a RoleBinding that gives your Google identity a cluster-admin role before
> attempting to create additional Role or ClusterRole permissions.
> This is a known issue in the Beta release of Role-Based Access Control in Kubernetes and Container Engine version 1.6.

To workaround this issue, you must grant your current Google Cloud Identity the `cluster-admin` role for each cluster
registered with KubeFed:

```bash
kubectl create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin \
  --user=$(gcloud config get-value core/account) --context cluster1
kubectl create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin \
  --user=$(gcloud config get-value core/account) --context cluster2
```

Once all pods are running you can return to the [User Guide](../userguide.md) to deploy the
KubeFed control-plane.
