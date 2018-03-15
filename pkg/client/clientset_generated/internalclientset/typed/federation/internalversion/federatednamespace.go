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
	federation "github.com/marun/fnord/pkg/apis/federation"
	scheme "github.com/marun/fnord/pkg/client/clientset_generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FederatedNamespacesGetter has a method to return a FederatedNamespaceInterface.
// A group's client should implement this interface.
type FederatedNamespacesGetter interface {
	FederatedNamespaces() FederatedNamespaceInterface
}

// FederatedNamespaceInterface has methods to work with FederatedNamespace resources.
type FederatedNamespaceInterface interface {
	Create(*federation.FederatedNamespace) (*federation.FederatedNamespace, error)
	Update(*federation.FederatedNamespace) (*federation.FederatedNamespace, error)
	UpdateStatus(*federation.FederatedNamespace) (*federation.FederatedNamespace, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederatedNamespace, error)
	List(opts v1.ListOptions) (*federation.FederatedNamespaceList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedNamespace, err error)
	FederatedNamespaceExpansion
}

// federatedNamespaces implements FederatedNamespaceInterface
type federatedNamespaces struct {
	client rest.Interface
}

// newFederatedNamespaces returns a FederatedNamespaces
func newFederatedNamespaces(c *FederationClient) *federatedNamespaces {
	return &federatedNamespaces{
		client: c.RESTClient(),
	}
}

// Get takes name of the federatedNamespace, and returns the corresponding federatedNamespace object, and an error if there is any.
func (c *federatedNamespaces) Get(name string, options v1.GetOptions) (result *federation.FederatedNamespace, err error) {
	result = &federation.FederatedNamespace{}
	err = c.client.Get().
		Resource("federatednamespaces").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedNamespaces that match those selectors.
func (c *federatedNamespaces) List(opts v1.ListOptions) (result *federation.FederatedNamespaceList, err error) {
	result = &federation.FederatedNamespaceList{}
	err = c.client.Get().
		Resource("federatednamespaces").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedNamespaces.
func (c *federatedNamespaces) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("federatednamespaces").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedNamespace and creates it.  Returns the server's representation of the federatedNamespace, and an error, if there is any.
func (c *federatedNamespaces) Create(federatedNamespace *federation.FederatedNamespace) (result *federation.FederatedNamespace, err error) {
	result = &federation.FederatedNamespace{}
	err = c.client.Post().
		Resource("federatednamespaces").
		Body(federatedNamespace).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedNamespace and updates it. Returns the server's representation of the federatedNamespace, and an error, if there is any.
func (c *federatedNamespaces) Update(federatedNamespace *federation.FederatedNamespace) (result *federation.FederatedNamespace, err error) {
	result = &federation.FederatedNamespace{}
	err = c.client.Put().
		Resource("federatednamespaces").
		Name(federatedNamespace.Name).
		Body(federatedNamespace).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedNamespaces) UpdateStatus(federatedNamespace *federation.FederatedNamespace) (result *federation.FederatedNamespace, err error) {
	result = &federation.FederatedNamespace{}
	err = c.client.Put().
		Resource("federatednamespaces").
		Name(federatedNamespace.Name).
		SubResource("status").
		Body(federatedNamespace).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedNamespace and deletes it. Returns an error if one occurs.
func (c *federatedNamespaces) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("federatednamespaces").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedNamespaces) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("federatednamespaces").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedNamespace.
func (c *federatedNamespaces) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedNamespace, err error) {
	result = &federation.FederatedNamespace{}
	err = c.client.Patch(pt).
		Resource("federatednamespaces").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
