/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package internalversion

import (
	federation "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation"
	scheme "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FederatedClustersGetter has a method to return a FederatedClusterInterface.
// A group's client should implement this interface.
type FederatedClustersGetter interface {
	FederatedClusters() FederatedClusterInterface
}

// FederatedClusterInterface has methods to work with FederatedCluster resources.
type FederatedClusterInterface interface {
	Create(*federation.FederatedCluster) (*federation.FederatedCluster, error)
	Update(*federation.FederatedCluster) (*federation.FederatedCluster, error)
	UpdateStatus(*federation.FederatedCluster) (*federation.FederatedCluster, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederatedCluster, error)
	List(opts v1.ListOptions) (*federation.FederatedClusterList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedCluster, err error)
	FederatedClusterExpansion
}

// federatedClusters implements FederatedClusterInterface
type federatedClusters struct {
	client rest.Interface
}

// newFederatedClusters returns a FederatedClusters
func newFederatedClusters(c *FederationClient) *federatedClusters {
	return &federatedClusters{
		client: c.RESTClient(),
	}
}

// Get takes name of the federatedCluster, and returns the corresponding federatedCluster object, and an error if there is any.
func (c *federatedClusters) Get(name string, options v1.GetOptions) (result *federation.FederatedCluster, err error) {
	result = &federation.FederatedCluster{}
	err = c.client.Get().
		Resource("federatedclusters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedClusters that match those selectors.
func (c *federatedClusters) List(opts v1.ListOptions) (result *federation.FederatedClusterList, err error) {
	result = &federation.FederatedClusterList{}
	err = c.client.Get().
		Resource("federatedclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedClusters.
func (c *federatedClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("federatedclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedCluster and creates it.  Returns the server's representation of the federatedCluster, and an error, if there is any.
func (c *federatedClusters) Create(federatedCluster *federation.FederatedCluster) (result *federation.FederatedCluster, err error) {
	result = &federation.FederatedCluster{}
	err = c.client.Post().
		Resource("federatedclusters").
		Body(federatedCluster).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedCluster and updates it. Returns the server's representation of the federatedCluster, and an error, if there is any.
func (c *federatedClusters) Update(federatedCluster *federation.FederatedCluster) (result *federation.FederatedCluster, err error) {
	result = &federation.FederatedCluster{}
	err = c.client.Put().
		Resource("federatedclusters").
		Name(federatedCluster.Name).
		Body(federatedCluster).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedClusters) UpdateStatus(federatedCluster *federation.FederatedCluster) (result *federation.FederatedCluster, err error) {
	result = &federation.FederatedCluster{}
	err = c.client.Put().
		Resource("federatedclusters").
		Name(federatedCluster.Name).
		SubResource("status").
		Body(federatedCluster).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedCluster and deletes it. Returns an error if one occurs.
func (c *federatedClusters) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("federatedclusters").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("federatedclusters").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedCluster.
func (c *federatedClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedCluster, err error) {
	result = &federation.FederatedCluster{}
	err = c.client.Patch(pt).
		Resource("federatedclusters").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
