<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Installing KubeFed](#installing-kubefed)
  - [Prerequisites](#prerequisites)
    - [Required binaries](#required-binaries)
    - [Creating Clusters](#creating-clusters)
    - [Deployment Image](#deployment-image)
  - [Helm Chart Deployment](#helm-chart-deployment)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Installing KubeFed

## Prerequisites

KubeFed requires Kubernetes v1.13 or newer.

### Required binaries
The following binaries are required to install and use KubeFed.
- [`kubebuilder`](https://book.kubebuilder.io/getting_started/installation_and_setup.html) v1.13 or newer.
- [`etcd`](https://github.com/etcd-io/etcd/blob/master/Documentation/dl_build.md) v1.13 or newer.
- `kube-apiserver` v1.13 or newer.

All of the binaries listed can be installed by the `download-binaries.sh` script. Use the following commands to clone the repository and run the script.

   `git clone https://github.com/kubernetes-sigs/kubefed.git`   
   `./scripts/download-binaries.sh`   
   `export PATH=$(pwd)/bin:${PATH}`   

### Creating Clusters

The following is a list of Kubernetes environments that have been tested and are supported by the KubeFed community.

- [kind](./environments/kind.md)
- [Minikube](./environments/minikube.md)
- [Google Kubernetes Engine (GKE)](./environments/gke.md)
- [IBM Cloud Private](./environments/icp.md)

After completing the steps in one of the above guides, return here to continue the deployment.

**IMPORTANT:** You must set the correct context in your cluster(s) using the command below.

```bash
kubectl config use-context cluster1
```
### Deployment Image

If you follow this user guide without any changes you will be using the latest master image tagged as [`canary`](development.md#test-latest-master-changes-canary).

## Helm Chart Deployment

You can refer to [helm chart installation guide](https://github.com/kubernetes-sigs/kubefed/blob/master/charts/kubefed/README.md) for instructions on installing KubeFed.
