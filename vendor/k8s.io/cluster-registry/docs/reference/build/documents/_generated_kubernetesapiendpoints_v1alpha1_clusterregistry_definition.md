## KubernetesAPIEndpoints v1alpha1

Group        | Version     | Kind
------------ | ---------- | -----------
`clusterregistry` | `v1alpha1` | `KubernetesAPIEndpoints`



KubernetesAPIEndpoints represents the endpoints for one and only one Kubernetes API server.

<aside class="notice">
Appears In:

<ul> 
<li><a href="#clusterspec-v1alpha1">ClusterSpec v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
`caBundle`<br /> *string*    | CABundle contains the certificate authority information.
`serverEndpoints`<br /> *[ServerAddressByClientCIDR](#serveraddressbyclientcidr-v1alpha1) array*    | ServerEndpoints specifies the address(es) of the Kubernetes API server’s network identity or identities.

