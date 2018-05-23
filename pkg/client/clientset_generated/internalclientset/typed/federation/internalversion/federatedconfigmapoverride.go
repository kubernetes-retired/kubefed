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

// FederatedConfigMapOverridesGetter has a method to return a FederatedConfigMapOverrideInterface.
// A group's client should implement this interface.
type FederatedConfigMapOverridesGetter interface {
	FederatedConfigMapOverrides(namespace string) FederatedConfigMapOverrideInterface
}

// FederatedConfigMapOverrideInterface has methods to work with FederatedConfigMapOverride resources.
type FederatedConfigMapOverrideInterface interface {
	Create(*federation.FederatedConfigMapOverride) (*federation.FederatedConfigMapOverride, error)
	Update(*federation.FederatedConfigMapOverride) (*federation.FederatedConfigMapOverride, error)
	UpdateStatus(*federation.FederatedConfigMapOverride) (*federation.FederatedConfigMapOverride, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederatedConfigMapOverride, error)
	List(opts v1.ListOptions) (*federation.FederatedConfigMapOverrideList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedConfigMapOverride, err error)
	FederatedConfigMapOverrideExpansion
}

// federatedConfigMapOverrides implements FederatedConfigMapOverrideInterface
type federatedConfigMapOverrides struct {
	client rest.Interface
	ns     string
}

// newFederatedConfigMapOverrides returns a FederatedConfigMapOverrides
func newFederatedConfigMapOverrides(c *FederationClient, namespace string) *federatedConfigMapOverrides {
	return &federatedConfigMapOverrides{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedConfigMapOverride, and returns the corresponding federatedConfigMapOverride object, and an error if there is any.
func (c *federatedConfigMapOverrides) Get(name string, options v1.GetOptions) (result *federation.FederatedConfigMapOverride, err error) {
	result = &federation.FederatedConfigMapOverride{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedconfigmapoverrides").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedConfigMapOverrides that match those selectors.
func (c *federatedConfigMapOverrides) List(opts v1.ListOptions) (result *federation.FederatedConfigMapOverrideList, err error) {
	result = &federation.FederatedConfigMapOverrideList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedconfigmapoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedConfigMapOverrides.
func (c *federatedConfigMapOverrides) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedconfigmapoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedConfigMapOverride and creates it.  Returns the server's representation of the federatedConfigMapOverride, and an error, if there is any.
func (c *federatedConfigMapOverrides) Create(federatedConfigMapOverride *federation.FederatedConfigMapOverride) (result *federation.FederatedConfigMapOverride, err error) {
	result = &federation.FederatedConfigMapOverride{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedconfigmapoverrides").
		Body(federatedConfigMapOverride).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedConfigMapOverride and updates it. Returns the server's representation of the federatedConfigMapOverride, and an error, if there is any.
func (c *federatedConfigMapOverrides) Update(federatedConfigMapOverride *federation.FederatedConfigMapOverride) (result *federation.FederatedConfigMapOverride, err error) {
	result = &federation.FederatedConfigMapOverride{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedconfigmapoverrides").
		Name(federatedConfigMapOverride.Name).
		Body(federatedConfigMapOverride).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedConfigMapOverrides) UpdateStatus(federatedConfigMapOverride *federation.FederatedConfigMapOverride) (result *federation.FederatedConfigMapOverride, err error) {
	result = &federation.FederatedConfigMapOverride{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedconfigmapoverrides").
		Name(federatedConfigMapOverride.Name).
		SubResource("status").
		Body(federatedConfigMapOverride).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedConfigMapOverride and deletes it. Returns an error if one occurs.
func (c *federatedConfigMapOverrides) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedconfigmapoverrides").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedConfigMapOverrides) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedconfigmapoverrides").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedConfigMapOverride.
func (c *federatedConfigMapOverrides) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedConfigMapOverride, err error) {
	result = &federation.FederatedConfigMapOverride{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedconfigmapoverrides").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
