## AuthProviderConfig v1alpha1 clusterregistry

Group        | Version     | Kind
------------ | ---------- | -----------
clusterregistry | v1alpha1 | AuthProviderConfig



AuthProviderConfig contains the information necessary for a client to authenticate to a Kubernetes API server. It is modeled after k8s.io/client-go/tools/clientcmd/api/v1.AuthProviderConfig.

<aside class="notice">
Appears In:

<ul> 
<li><a href="#authinfo-v1alpha1-clusterregistry">AuthInfo clusterregistry/v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
config <br /> *object*    | Config is a map of values that contains the information necessary for a client to determine how to authenticate to a Kubernetes API server.
name <br /> *string*    | Name is the name of this configuration.
type <br /> *[AuthProviderType](#authprovidertype-v1alpha1-clusterregistry)*    | Type contains type information about this auth provider. Clients of the cluster registry should use this field to differentiate between different kinds of authentication providers.

