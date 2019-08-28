<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Installing KubeFed](#installing-kubefed)
  - [Prerequisites](#prerequisites)
    - [kubefedctl CLI](#kubefedctl-cli)
    - [Creating Clusters](#creating-clusters)
    - [Deployment Image](#deployment-image)
  - [Helm Chart Deployment](#helm-chart-deployment)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Installing KubeFed

## Prerequisites

### kubefedctl CLI

`kubefedctl` is the KubeFed command line utility. You can download
the latest binary from the [release page](https://github.com/kubernetes-sigs/kubefed/releases).

```bash
VERSION=<latest-version, e.g. 0.1.0-rc3>
OS=<darwin/linux>
ARCH=amd64
curl -LO https://github.com/kubernetes-sigs/kubefed/releases/download/v${VERSION}/kubefedctl-${VERSION}-${OS}-${ARCH}.tgz
tar -zxvf kubefedctl-*.tgz
chmod u+x kubefedctl
sudo mv kubefedctl /usr/local/bin/ # make sure the location is in the PATH
```

**NOTE:** `kubefedctl` is built for Linux and OSX only in the release package.

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
