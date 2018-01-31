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

// FederatedSecretOverridesesGetter has a method to return a FederatedSecretOverridesInterface.
// A group's client should implement this interface.
type FederatedSecretOverridesesGetter interface {
	FederatedSecretOverrideses(namespace string) FederatedSecretOverridesInterface
}

// FederatedSecretOverridesInterface has methods to work with FederatedSecretOverrides resources.
type FederatedSecretOverridesInterface interface {
	Create(*federation.FederatedSecretOverrides) (*federation.FederatedSecretOverrides, error)
	Update(*federation.FederatedSecretOverrides) (*federation.FederatedSecretOverrides, error)
	UpdateStatus(*federation.FederatedSecretOverrides) (*federation.FederatedSecretOverrides, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederatedSecretOverrides, error)
	List(opts v1.ListOptions) (*federation.FederatedSecretOverridesList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedSecretOverrides, err error)
	FederatedSecretOverridesExpansion
}

// federatedSecretOverrideses implements FederatedSecretOverridesInterface
type federatedSecretOverrideses struct {
	client rest.Interface
	ns     string
}

// newFederatedSecretOverrideses returns a FederatedSecretOverrideses
func newFederatedSecretOverrideses(c *FederationClient, namespace string) *federatedSecretOverrideses {
	return &federatedSecretOverrideses{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedSecretOverrides, and returns the corresponding federatedSecretOverrides object, and an error if there is any.
func (c *federatedSecretOverrideses) Get(name string, options v1.GetOptions) (result *federation.FederatedSecretOverrides, err error) {
	result = &federation.FederatedSecretOverrides{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecretoverrideses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedSecretOverrideses that match those selectors.
func (c *federatedSecretOverrideses) List(opts v1.ListOptions) (result *federation.FederatedSecretOverridesList, err error) {
	result = &federation.FederatedSecretOverridesList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecretoverrideses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedSecretOverrideses.
func (c *federatedSecretOverrideses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecretoverrideses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedSecretOverrides and creates it.  Returns the server's representation of the federatedSecretOverrides, and an error, if there is any.
func (c *federatedSecretOverrideses) Create(federatedSecretOverrides *federation.FederatedSecretOverrides) (result *federation.FederatedSecretOverrides, err error) {
	result = &federation.FederatedSecretOverrides{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedsecretoverrideses").
		Body(federatedSecretOverrides).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedSecretOverrides and updates it. Returns the server's representation of the federatedSecretOverrides, and an error, if there is any.
func (c *federatedSecretOverrideses) Update(federatedSecretOverrides *federation.FederatedSecretOverrides) (result *federation.FederatedSecretOverrides, err error) {
	result = &federation.FederatedSecretOverrides{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedsecretoverrideses").
		Name(federatedSecretOverrides.Name).
		Body(federatedSecretOverrides).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedSecretOverrideses) UpdateStatus(federatedSecretOverrides *federation.FederatedSecretOverrides) (result *federation.FederatedSecretOverrides, err error) {
	result = &federation.FederatedSecretOverrides{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedsecretoverrideses").
		Name(federatedSecretOverrides.Name).
		SubResource("status").
		Body(federatedSecretOverrides).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedSecretOverrides and deletes it. Returns an error if one occurs.
func (c *federatedSecretOverrideses) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedsecretoverrideses").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedSecretOverrideses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedsecretoverrideses").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedSecretOverrides.
func (c *federatedSecretOverrideses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedSecretOverrides, err error) {
	result = &federation.FederatedSecretOverrides{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedsecretoverrideses").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
