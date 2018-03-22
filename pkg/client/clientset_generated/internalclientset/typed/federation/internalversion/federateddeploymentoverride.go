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

// FederatedDeploymentOverridesGetter has a method to return a FederatedDeploymentOverrideInterface.
// A group's client should implement this interface.
type FederatedDeploymentOverridesGetter interface {
	FederatedDeploymentOverrides(namespace string) FederatedDeploymentOverrideInterface
}

// FederatedDeploymentOverrideInterface has methods to work with FederatedDeploymentOverride resources.
type FederatedDeploymentOverrideInterface interface {
	Create(*federation.FederatedDeploymentOverride) (*federation.FederatedDeploymentOverride, error)
	Update(*federation.FederatedDeploymentOverride) (*federation.FederatedDeploymentOverride, error)
	UpdateStatus(*federation.FederatedDeploymentOverride) (*federation.FederatedDeploymentOverride, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederatedDeploymentOverride, error)
	List(opts v1.ListOptions) (*federation.FederatedDeploymentOverrideList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedDeploymentOverride, err error)
	FederatedDeploymentOverrideExpansion
}

// federatedDeploymentOverrides implements FederatedDeploymentOverrideInterface
type federatedDeploymentOverrides struct {
	client rest.Interface
	ns     string
}

// newFederatedDeploymentOverrides returns a FederatedDeploymentOverrides
func newFederatedDeploymentOverrides(c *FederationClient, namespace string) *federatedDeploymentOverrides {
	return &federatedDeploymentOverrides{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedDeploymentOverride, and returns the corresponding federatedDeploymentOverride object, and an error if there is any.
func (c *federatedDeploymentOverrides) Get(name string, options v1.GetOptions) (result *federation.FederatedDeploymentOverride, err error) {
	result = &federation.FederatedDeploymentOverride{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedDeploymentOverrides that match those selectors.
func (c *federatedDeploymentOverrides) List(opts v1.ListOptions) (result *federation.FederatedDeploymentOverrideList, err error) {
	result = &federation.FederatedDeploymentOverrideList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedDeploymentOverrides.
func (c *federatedDeploymentOverrides) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedDeploymentOverride and creates it.  Returns the server's representation of the federatedDeploymentOverride, and an error, if there is any.
func (c *federatedDeploymentOverrides) Create(federatedDeploymentOverride *federation.FederatedDeploymentOverride) (result *federation.FederatedDeploymentOverride, err error) {
	result = &federation.FederatedDeploymentOverride{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		Body(federatedDeploymentOverride).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedDeploymentOverride and updates it. Returns the server's representation of the federatedDeploymentOverride, and an error, if there is any.
func (c *federatedDeploymentOverrides) Update(federatedDeploymentOverride *federation.FederatedDeploymentOverride) (result *federation.FederatedDeploymentOverride, err error) {
	result = &federation.FederatedDeploymentOverride{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		Name(federatedDeploymentOverride.Name).
		Body(federatedDeploymentOverride).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedDeploymentOverrides) UpdateStatus(federatedDeploymentOverride *federation.FederatedDeploymentOverride) (result *federation.FederatedDeploymentOverride, err error) {
	result = &federation.FederatedDeploymentOverride{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		Name(federatedDeploymentOverride.Name).
		SubResource("status").
		Body(federatedDeploymentOverride).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedDeploymentOverride and deletes it. Returns an error if one occurs.
func (c *federatedDeploymentOverrides) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedDeploymentOverrides) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedDeploymentOverride.
func (c *federatedDeploymentOverrides) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedDeploymentOverride, err error) {
	result = &federation.FederatedDeploymentOverride{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
