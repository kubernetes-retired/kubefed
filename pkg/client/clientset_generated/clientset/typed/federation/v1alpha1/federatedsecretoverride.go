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
package v1alpha1

import (
	v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	scheme "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FederatedSecretOverridesGetter has a method to return a FederatedSecretOverrideInterface.
// A group's client should implement this interface.
type FederatedSecretOverridesGetter interface {
	FederatedSecretOverrides(namespace string) FederatedSecretOverrideInterface
}

// FederatedSecretOverrideInterface has methods to work with FederatedSecretOverride resources.
type FederatedSecretOverrideInterface interface {
	Create(*v1alpha1.FederatedSecretOverride) (*v1alpha1.FederatedSecretOverride, error)
	Update(*v1alpha1.FederatedSecretOverride) (*v1alpha1.FederatedSecretOverride, error)
	UpdateStatus(*v1alpha1.FederatedSecretOverride) (*v1alpha1.FederatedSecretOverride, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedSecretOverride, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedSecretOverrideList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedSecretOverride, err error)
	FederatedSecretOverrideExpansion
}

// federatedSecretOverrides implements FederatedSecretOverrideInterface
type federatedSecretOverrides struct {
	client rest.Interface
	ns     string
}

// newFederatedSecretOverrides returns a FederatedSecretOverrides
func newFederatedSecretOverrides(c *FederationV1alpha1Client, namespace string) *federatedSecretOverrides {
	return &federatedSecretOverrides{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedSecretOverride, and returns the corresponding federatedSecretOverride object, and an error if there is any.
func (c *federatedSecretOverrides) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedSecretOverride, err error) {
	result = &v1alpha1.FederatedSecretOverride{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecretoverrides").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedSecretOverrides that match those selectors.
func (c *federatedSecretOverrides) List(opts v1.ListOptions) (result *v1alpha1.FederatedSecretOverrideList, err error) {
	result = &v1alpha1.FederatedSecretOverrideList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecretoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedSecretOverrides.
func (c *federatedSecretOverrides) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecretoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedSecretOverride and creates it.  Returns the server's representation of the federatedSecretOverride, and an error, if there is any.
func (c *federatedSecretOverrides) Create(federatedSecretOverride *v1alpha1.FederatedSecretOverride) (result *v1alpha1.FederatedSecretOverride, err error) {
	result = &v1alpha1.FederatedSecretOverride{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedsecretoverrides").
		Body(federatedSecretOverride).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedSecretOverride and updates it. Returns the server's representation of the federatedSecretOverride, and an error, if there is any.
func (c *federatedSecretOverrides) Update(federatedSecretOverride *v1alpha1.FederatedSecretOverride) (result *v1alpha1.FederatedSecretOverride, err error) {
	result = &v1alpha1.FederatedSecretOverride{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedsecretoverrides").
		Name(federatedSecretOverride.Name).
		Body(federatedSecretOverride).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedSecretOverrides) UpdateStatus(federatedSecretOverride *v1alpha1.FederatedSecretOverride) (result *v1alpha1.FederatedSecretOverride, err error) {
	result = &v1alpha1.FederatedSecretOverride{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedsecretoverrides").
		Name(federatedSecretOverride.Name).
		SubResource("status").
		Body(federatedSecretOverride).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedSecretOverride and deletes it. Returns an error if one occurs.
func (c *federatedSecretOverrides) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedsecretoverrides").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedSecretOverrides) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedsecretoverrides").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedSecretOverride.
func (c *federatedSecretOverrides) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedSecretOverride, err error) {
	result = &v1alpha1.FederatedSecretOverride{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedsecretoverrides").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
