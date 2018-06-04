## AuthProviderType v1alpha1 clusterregistry

Group        | Version     | Kind
------------ | ---------- | -----------
clusterregistry | v1alpha1 | AuthProviderType



AuthProviderType contains metadata about the auth provider. It should be used by clients to differentiate between different kinds of auth providers, and to select a relevant provider for the client's configuration. For example, a controller would look for a provider type that denotes a service account that it should use to access the cluster, whereas a user would look for a provider type that denotes an authentication system from which they should request a token.

<aside class="notice">
Appears In:

<ul> 
<li><a href="#authproviderconfig-v1alpha1-clusterregistry">AuthProviderConfig clusterregistry/v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
name <br /> *string*    | Name is the name of the auth provider.

