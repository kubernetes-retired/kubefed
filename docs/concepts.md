<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [KubeFed Concepts](#kubefed-concepts)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

### KubeFed Concepts

| Concept              | Description                                                                                                                                                                                                                                                                                                         |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Federate             | Federating a set of Kubernetes clusters means, effectively creating a common interface to the pool of these clusters which can be used to deploy Kubernetes applications across those clusters.                                                                                                                     |
| KubeFed           | Kubernetes Cluster Federation enables users to federate multiple Kubernetes clusters for resources distribution, service discovery, high availability etc across multiple clusters.                                                                                                                                 |
| Host Cluster         | A cluster which is used to expose the KubeFed API and run the KubeFed control plane.                                                                                                                                                                                                                          |
| Cluster Registration | A cluster join the Host Cluster via command `kubefedctl join`.                                                                                                                                                                                                                                                        |
| Member Cluster       | A cluster which is registered with the KubeFed API and that KubeFed controllers have authentication credentials for. The Host Cluster can also be a Member Cluster.                                                                                                                                           |
| ServiceDNSRecord     | A resource that associates one or more Kubernetes Service resources and how to access the Service, with a scheme for constructing Domain Name System (DNS) [resource records](https://www.ietf.org/rfc/rfc1035.txt) for the Service.                                                                                |
| IngressDNSRecord     | A resource that associates one or more [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) and how to access the Kubernetes Ingress resources, with a scheme for constructing Domain Name System (DNS) [resource records](https://www.ietf.org/rfc/rfc1035.txt) for the Ingress. |
| DNSEndpoint          | A [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) wrapper for the Endpoint resource.                                                                                                                                                                       |
| Endpoint             | A resource that represents a Domain Name System (DNS) [resource record](https://www.ietf.org/rfc/rfc1035.txt).                                                                                                                                                                                                      |
