<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [KubeFed DNS for Ingress and Service](#kubefed-dns-for-ingress-and-service)
  - [Creating KubeFed cluster](#creating-kubefed-cluster)
  - [Installing ExternalDNS](#installing-externaldns)
  - [Enable DNS for KubeFed resources](#enable-dns-for-kubefed-resources)
    - [Installing MetalLB for LoadBalancer Service](#installing-metallb-for-loadbalancer-service)
    - [Creating service resources](#creating-service-resources)
    - [Enable the ingress controller](#enable-the-ingress-controller)
    - [Creating ingress resources](#creating-ingress-resources)
  - [DNS Example](#dns-example)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# KubeFed DNS for Ingress and Service

This tutorial describes how to set up a KubeFed cluster DNS with [ExternalDNS](https://github.com/kubernetes-incubator/external-dns/) based on [CoreDNS](https://github.com/coredns/coredns) in [minikube](https://github.com/kubernetes/minikube) clusters. It provides guidance for the following steps:

- Install ExternalDNS with etcd enabled CoreDNS as a provider
- Install [ingress controller](https://github.com/kubernetes/ingress-nginx) for your minikube clusters to enable Ingress resource
- Install [metallb](https://github.com/google/metallb) for your minikube clusters to enable LoadBalancer Service

You can use either Loadbalancer Service or Ingress resource or both in your environment, this tutorial includes guidance for both Loadbalancer Service and Ingress resource.
For related conceptions of Muilti-cluster Ingress and Service, you can refer to [ingressdns-with-externaldns.md](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/ingressdns-with-externaldns.md) and [servicedns-with-externaldns.md](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/servicedns-with-externaldns.md).

## Creating KubeFed cluster

Install KubeFed with minikube in [User Guide](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/userguide.md).

## Installing ExternalDNS

Install ExternalDNS with CoreDNS as backend in your host cluster. You can follow the [tutorial](https://github.com/kubernetes-incubator/external-dns/blob/master/docs/tutorials/coredns.md).  
**Note**: You should replace `parameters: example.org` with `parameters: example.com` when [Installing CoreDNS](https://github.com/kubernetes-incubator/external-dns/blob/master/docs/tutorials/coredns.md#installing-coredns)

To make it work for KubeFed resources, you need to use below ExternalDNS deployment instead of the one in the tutorial.
**Note**: You should replace value of `ETCD_URLS` with your own etcd client service IP address.

```bash
$ kubectl get svc example-etcd-cluster-client -o jsonpath={.spec.clusterIP} && echo
10.102.147.224
$ cat <<EOF | kubectl create -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns
  namespace: kube-system
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: external-dns
  template:
    metadata:
      labels:
        app: external-dns
    spec:
      containers:
      - name: external-dns
        image: registry.opensource.zalan.do/teapot/external-dns:latest
        args:
        - --source=crd
        - --crd-source-apiversion=multiclusterdns.kubefed.io/v1alpha1
        - --crd-source-kind=DNSEndpoint
        - --registry=txt
        - --provider=coredns
        - --log-level=debug # debug only
        env:
        - name: ETCD_URLS
          value: http://10.102.147.224:2379
EOF
```

## Enable DNS for KubeFed resources

### Installing MetalLB for LoadBalancer Service

Install metallb in each member cluster to make LoadBalancer type Service work.
For related conceptions of metallb, you can refer to [BGP on Minikube](https://metallb.universe.tf/tutorial/minikube/).

```bash
$ helm --kube-context cluster1 install --name metallb stable/metallb
$ helm --kube-context cluster2 install --name metallb stable/metallb
```

Apply configmap to configure metallb in each cluster.

```bash
$ cat <<EOF | kubectl create --context cluster1 -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: metallb-config
data:
  config: |
    peers:
    - peer-address: 10.0.0.1
      peer-asn: 64501
      my-asn: 64500
    address-pools:
    - name: default
      protocol: bgp
      addresses:
      - 192.168.20.0/24
EOF
$ cat <<EOF | kubectl create --context cluster2 -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: metallb-config
data:
  config: |
    peers:
    - peer-address: 10.0.0.2
      peer-asn: 64500
      my-asn: 64501
    address-pools:
    - name: default
      protocol: bgp
      addresses:
      - 172.168.20.0/24
EOF
```

### Creating service resources

After metallb works, create a sample deployment and service from [sample](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/ingressdns-with-externaldns.md). Make service as LoadBalancer type.

```bash
sed -i 's/NodePort/LoadBalancer/' example/sample1/federatedservice.yaml
```

Create `ServiceDNSRecord` to make DNS work for service.

```bash
$ cat <<EOF | kubectl create -f -
apiVersion: multiclusterdns.kubefed.io/v1alpha1
kind: Domain
metadata:
  # Corresponds to <federation> in the resource records.
  name: test-domain
  # The namespace running kubefed-controller-manager.
  namespace: kube-federation-system
# The domain/subdomain that is setup in your externl-dns provider.
domain: example.com
---
apiVersion: multiclusterdns.kubefed.io/v1alpha1
kind: ServiceDNSRecord
metadata:
  # The name of the sample service.
  name: test-service
  # The namespace of the sample deployment/service.
  namespace: test-namespace
spec:
  # The name of the corresponding Domain.
  domainRef: test-domain
  recordTTL: 300
EOF
```

### Enable the ingress controller

```bash
$ kubectl --context cluster1 apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/mandatory.yaml
$ kubectl --context cluster1 apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/provider/baremetal/service-nodeport.yaml
$ kubectl --context cluster1 patch svc ingress-nginx -n ingress-nginx -p '{"spec": {"type": "LoadBalancer"}}'

$ kubectl --context cluster2 apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/mandatory.yaml
$ kubectl --context cluster2 apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/provider/baremetal/service-nodeport.yaml
$ kubectl --context cluster2 patch svc ingress-nginx -n ingress-nginx -p '{"spec": {"type": "LoadBalancer"}}'
```

After ingress controller enabled, create a sample deployment, service and ingress from [sample](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/ingressdns-with-externaldns.md).

### Creating ingress resources

Create `IngressDNSRecord` to make DNS work for ingress.

```bash
$ cat <<EOF | kubectl create -f -
apiVersion: multiclusterdns.kubefed.io/v1alpha1
kind: IngressDNSRecord
metadata:
  name: test-ingress
  namespace: test-namespace
spec:
  hosts:
  - ingress.example.com
  recordTTL: 300
EOF
```

## DNS Example

Wait a moment until DNS has the ingress/service IP. The DNS service IP is from CoreDNS service. It is `my-coredns-coredns` in this example.

```bash
$ kubectl get svc my-coredns-coredns
NAME                 TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
my-coredns-coredns   ClusterIP   10.100.4.143   <none>        53/UDP    12m

$ kubectl run -it --rm --restart=Never --image=infoblox/dnstools:latest dnstools
dnstools# dig @10.100.4.143 test-service.test-namespace.test-domain.svc.example.com +short
192.168.20.0
172.168.20.0
dnstools# dig @10.100.4.143 ingress.example.com +short
172.168.20.1
192.168.20.1
```
