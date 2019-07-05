<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Google Kubernetes Engine (GKE) Deployment Guide](#google-kubernetes-engine-gke-deployment-guide)
  - [RBAC Workaround](#rbac-workaround)
  - [Additional firewall rule for kubefed-admission-webhook in GKE private clusters](#additional-firewall-rule-for-kubefed-admission-webhook-in-gke-private-clusters)

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

Before proceeding with the KubeFed deployment, you must complete the steps outlined in this document.

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

## Additional firewall rule for kubefed-admission-webhook in GKE private clusters

Admission control in [GKE private clusters](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters)
requires an additional VPC network firewall rule.

### Issue

The `kubefed-controller-manager` pod fails to start up when attempting to create
a `KubeFedConfig` resource. A timeout error is observed.

```
Error updating KubeFedConfig "kube-federation-system/kubefed": Timeout: request did not complete within requested timeout 30s
```

### Background

The GKE Kubernetes control plane is deployed in a Google-managed account.

During the provisioning of a private cluster, GKE automatically creates:
1. A network [peering connection](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#features)
to allow the master to communicate to the node pools.
1. A [firewall rule](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#add_firewall_rules)
to allow ports 443 and 10250 from the master to the node pools.

The GKE API Server operates with [`--enable-aggregator-routing=true`](https://github.com/kubernetes/kubernetes/issues/79739#issuecomment-509813068).
This means that the Service IPs are internally translated to endpoint IPs and
routed to the relevant pods.

A number of [webhook configurations are deployed](../../charts/kubefed/charts/controllermanager/templates/webhook.yaml)
during the kubfed helm installation, alongside the `kubefed-admission-webhook`
application. Every time a validated kubefed object is created or updated,
the webhook causes the API server to query the `kubefed-admission-webhook` service.

Because of the `enable-aggregator-routing` being turned on, the request is
directed from the API server to the pod running the `kubefed-admission-webhook`
application directly, on port 8443. This traffic needs to be allowed through the
GCP firewall in order for the webhook query to succeed.

### Resolution

Create a VPC network firewall rule to allow traffic from your master's source IP
range to the node pools. An example gcloud command is provided below.

```bash
CLUSTER_NAME=my_gke_private_cluster
CLUSTER_REGION=europe-west3-a
VPC_NETWORK=$(gcloud container clusters describe $CLUSTER_NAME --region $CLUSTER_REGION --format='value(network)')
MASTER_IPV4_CIDR_BLOCK=$(gcloud container clusters describe $CLUSTER_NAME --region $CLUSTER_REGION --format='value(privateClusterConfig.masterIpv4CidrBlock)')
NODE_POOLS_TARGET_TAGS=$(gcloud container clusters describe $CLUSTER_NAME --region $CLUSTER_REGION --format='value[terminator=","](nodePools.config.tags)' --flatten='nodePools[].config.tags[]')

gcloud compute firewall-rules create "allow-apiserver-to-admission-webhook-8443" \
      --allow tcp:8443 \
      --network="$VPC_NETWORK" \
      --source-ranges="$MASTER_IPV4_CIDR_BLOCK" \
      --target-tags="$NODE_POOLS_TARGET_TAGS" \
      --description="Allow apiserver access to admission webhook pod on port 8443" \
      --direction INGRESS
```

### See also

- [Adding firewall rules for specific use cases for GKE](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#add_firewall_rules)
- [Admission control webhook configuration](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#webhook-configuration)
- [Related kubefed issue](https://github.com/kubernetes-sigs/kubefed/issues/1024)
- [Related kubernetes issue](https://github.com/kubernetes/kubernetes/issues/79739)
