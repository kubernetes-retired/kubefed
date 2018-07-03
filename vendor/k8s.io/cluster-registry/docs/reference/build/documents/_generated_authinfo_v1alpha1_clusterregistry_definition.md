## AuthInfo v1alpha1

Group        | Version     | Kind
------------ | ---------- | -----------
`clusterregistry` | `v1alpha1` | `AuthInfo`



AuthInfo holds information that describes how a client can get credentials to access the cluster. For example, OAuth2 client registration endpoints and supported flows, or Kerberos server locations.

<aside class="notice">
Appears In:

<ul> 
<li><a href="#clusterspec-v1alpha1">ClusterSpec v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
`controller`<br /> *[ObjectReference](#objectreference-v1alpha1)*    | Controller references an object that contains implementation-specific details about how a controller should authenticate. A simple use case for this would be to reference a secret in another namespace that stores a bearer token that can be used to authenticate against this cluster&#39;s API server.
`user`<br /> *[ObjectReference](#objectreference-v1alpha1)*    | User references an object that contains implementation-specific details about how a user should authenticate against this cluster.

