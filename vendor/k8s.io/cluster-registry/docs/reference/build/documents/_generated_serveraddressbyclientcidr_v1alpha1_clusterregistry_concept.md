

-----------
# ServerAddressByClientCIDR v1alpha1 clusterregistry



Group        | Version     | Kind
------------ | ---------- | -----------
clusterregistry | v1alpha1 | ServerAddressByClientCIDR




<aside class="notice">Other api versions of this object exist: <a href="#serveraddressbyclientcidr-v1-meta">v1</a> </aside>


ServerAddressByClientCIDR helps clients determine the server address that they should use, depending on the ClientCIDR that they match.

<aside class="notice">
Appears In:

<ul> 
<li><a href="#kubernetesapiendpoints-v1alpha1-clusterregistry">KubernetesAPIEndpoints clusterregistry/v1alpha1</a></li>
</ul> </aside>

Field        | Description
------------ | -----------
clientCIDR <br /> *string*    | The CIDR with which clients can match their IP to figure out if they should use the corresponding server address.
serverAddress <br /> *string*    | Address of this server, suitable for a client that matches the above CIDR. This can be a hostname, hostname:port, IP or IP:port.






