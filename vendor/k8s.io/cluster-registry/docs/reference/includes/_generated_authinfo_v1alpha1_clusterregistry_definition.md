## AuthInfo v1alpha1 clusterregistry

Group        | Version     | Kind
------------ | ---------- | -----------
clusterregistry | v1alpha1 | AuthInfo



AuthInfo holds public information that describes how a client can get credentials to access the cluster. For example, OAuth2 client registration endpoints and supported flows, or Kerberos servers locations.

It should not hold any private or sensitive information.

<aside class="notice">
Appears In:

<ul> 
<li><a href="#clusterspec-v1alpha1-clusterregistry">ClusterSpec clusterregistry/v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
providers <br /> *[AuthProviderConfig](#authproviderconfig-v1alpha1-clusterregistry) array*    | AuthProviders is a list of configurations for auth providers.

