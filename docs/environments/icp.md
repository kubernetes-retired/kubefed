<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [IBM Cloud Private Deployment Guide](#ibm-cloud-private-deployment-guide)
  - [Install IBM Cloud Private](#install-ibm-cloud-private)
  - [Pre KubeFed Install Configuration](#pre-kubefed-install-configuration)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# IBM Cloud Private Deployment Guide

KubeFed can be deployed to and manage [IBM Cloud Private](https://www.ibm.com/cloud/private) clusters.
As KubeFed requires Kubernetes v1.13 or greater, please make sure to deploy IBM Cloud Private 3.1.1
or higher.

The following example deploys two IBM Cloud Private 3.1.1 clusters named `cluster1` and `cluster2`.

## Install IBM Cloud Private

Please follow the [guide in IBM Cloud Private 3.1.1 Knowledge Center](https://www.ibm.com/support/knowledgecenter/SSBS6K_3.1.1/installing/install.html)
to install.

**NOTE:** We need to install two clusters named `cluster1` and `cluster2`, so after `cluster/config.yaml`
is generated, update the names of the 2 clusters to 'cluster1' and 'cluster2' before installing KubeFed.

For the first cluster, set the following value in `cluster/config.yaml` as follows:

```yaml
cluster_name: cluster1
```

For the second cluster, set the following value in `cluster/config.yaml` as follows:

```yaml
cluster_name: cluster2
```

## Pre KubeFed Install Configuration

As IBM Cloud Private is [enforcing container image security](https://www.ibm.com/support/knowledgecenter/SSBS6K_3.1.1/manage_images/image_security.html)
policy by default, and the default image security policy does not allow pulling the KubeFed
image from `quay.io/kubernetes-multicluster/kubefed:*`, we need to update the image security
policy as follows:

```bash
$ kubectl edit clusterimagepolicies ibmcloud-default-cluster-image-policy
```

Update `spec.repositories` by adding `quay.io/kubernetes-multicluster/kubefed:*`:

```yaml
spec:
  repositories:
    - name: "quay.io/kubernetes-multicluster/kubefed:*"
```

IBM Cloud Private supports [pod isolation](https://www.ibm.com/support/knowledgecenter/SSBS6K_3.2.0/user_management/iso_pod.html) 
with `ibm-restricted-psp` as the default pod security policy. This policy requires pods to run with a non-root user ID, 
and prevents pods from accessing the host. The kubefed pods try to run as root, so the Pod Security Policy prevents their start.
The simplest way to configure the Pod security policy for the `kube-federation-system` namespace, is to create and configure 
the namespace before kubefed installation.

1. Log in to your IBM Cloud Private cluster as a cluster administrator.
2. From the navigation menu, click Manage > Namespaces.
3. Click the Create Namespace button.
4. In the Create Namespace dialog box, enter `kube-federation-system` as the name of the new namespace.
5. Click the Pod Security drop-down menu and select `ibm-anyuid-psp`as pod security policy for your namespace.

When you finish these operations, you can return to the [User Guide](../userguide.md) to deploy the KubeFed control-plane.

