<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [KubeFed Concepts](#kubefed-concepts)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

### KubeFed Concepts

<table>
  <tr>
    <th> Concept </th>
    <th> Description </th>
  </tr>
  <tr>
    <td> Federate </td>
    <td> Federating a set of Kubernetes clusters means, effectively creating a
    common interface to the pool of these clusters which can be used to deploy
    Kubernetes applications across those clusters. </td>
  </tr>
  <tr>
    <td> KubeFed </td>
    <td> Kubernetes Cluster Federation enables users to federate multiple
    Kubernetes clusters for resources distribution, service discovery, high
    availability etc across multiple clusters. </td>
  </tr>
  <tr>
    <td> Host Cluster </td>
    <td> A cluster which is used to expose the KubeFed API and run the KubeFed
    control plane. </td>
  </tr>
  <tr>
    <td> Cluster Registration </td>
    <td> A cluster join the Host Cluster via command <code>kubefedctl join
    </code>. </td>
  </tr>
  <tr>
    <td> Member Cluster </td>
    <td> A cluster which is registered with the KubeFed API and that KubeFed
    controllers have authentication credentials for. The Host Cluster can also
    be a Member Cluster. </td>
  </tr>
  <tr>
    <td> ServiceDNSRecord </td>
    <td> A resource that associates one or more Kubernetes Service resources
    and how to access the Service, with a scheme for constructing Domain Name
    System (DNS) <a href="https://www.ietf.org/rfc/rfc1035.txt">resource
    records</a> for the Service. </td>
  </tr>
  <tr>
    <td> IngressDNSRecord </td>
    <td> A resource that associates one or more
     <a href="https://kubernetes.io/docs/concepts/services-networking/ingress/">
    Kubernetes Ingress</a> and how to access the Kubernetes Ingress resources,
    with a scheme for constructing Domain Name System (DNS)
    <a href="https://www.ietf.org/rfc/rfc1035.txt">resource records</a> for the
    Ingress. </td>
  </tr>
  <tr>
    <td> DNSEndpoint </td>
    <td> A
    <a href="https://kubernetes.io/docs/concepts/extend-kubernetes/
api-extension/custom-resources/">Custom Resource</a> wrapper for the
    Endpoint resource. </td>
  </tr>
  <tr>
    <td> Endpoint </td>
    <td> A resource that represents a Domain Name System (DNS)
    <a href="https://www.ietf.org/rfc/rfc1035.txt">resource records</a>. </td>
  </tr>
</table>

