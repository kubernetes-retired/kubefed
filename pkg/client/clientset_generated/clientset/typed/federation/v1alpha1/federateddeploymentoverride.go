/*
Copyright 2018 The Federation v2 Authors.

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

// FederatedDeploymentOverridesGetter has a method to return a FederatedDeploymentOverrideInterface.
// A group's client should implement this interface.
type FederatedDeploymentOverridesGetter interface {
	FederatedDeploymentOverrides(namespace string) FederatedDeploymentOverrideInterface
}

// FederatedDeploymentOverrideInterface has methods to work with FederatedDeploymentOverride resources.
type FederatedDeploymentOverrideInterface interface {
	Create(*v1alpha1.FederatedDeploymentOverride) (*v1alpha1.FederatedDeploymentOverride, error)
	Update(*v1alpha1.FederatedDeploymentOverride) (*v1alpha1.FederatedDeploymentOverride, error)
	UpdateStatus(*v1alpha1.FederatedDeploymentOverride) (*v1alpha1.FederatedDeploymentOverride, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedDeploymentOverride, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedDeploymentOverrideList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedDeploymentOverride, err error)
	FederatedDeploymentOverrideExpansion
}

// federatedDeploymentOverrides implements FederatedDeploymentOverrideInterface
type federatedDeploymentOverrides struct {
	client rest.Interface
	ns     string
}

// newFederatedDeploymentOverrides returns a FederatedDeploymentOverrides
func newFederatedDeploymentOverrides(c *FederationV1alpha1Client, namespace string) *federatedDeploymentOverrides {
	return &federatedDeploymentOverrides{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedDeploymentOverride, and returns the corresponding federatedDeploymentOverride object, and an error if there is any.
func (c *federatedDeploymentOverrides) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedDeploymentOverride, err error) {
	result = &v1alpha1.FederatedDeploymentOverride{}
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
func (c *federatedDeploymentOverrides) List(opts v1.ListOptions) (result *v1alpha1.FederatedDeploymentOverrideList, err error) {
	result = &v1alpha1.FederatedDeploymentOverrideList{}
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
func (c *federatedDeploymentOverrides) Create(federatedDeploymentOverride *v1alpha1.FederatedDeploymentOverride) (result *v1alpha1.FederatedDeploymentOverride, err error) {
	result = &v1alpha1.FederatedDeploymentOverride{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federateddeploymentoverrides").
		Body(federatedDeploymentOverride).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedDeploymentOverride and updates it. Returns the server's representation of the federatedDeploymentOverride, and an error, if there is any.
func (c *federatedDeploymentOverrides) Update(federatedDeploymentOverride *v1alpha1.FederatedDeploymentOverride) (result *v1alpha1.FederatedDeploymentOverride, err error) {
	result = &v1alpha1.FederatedDeploymentOverride{}
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

func (c *federatedDeploymentOverrides) UpdateStatus(federatedDeploymentOverride *v1alpha1.FederatedDeploymentOverride) (result *v1alpha1.FederatedDeploymentOverride, err error) {
	result = &v1alpha1.FederatedDeploymentOverride{}
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
func (c *federatedDeploymentOverrides) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedDeploymentOverride, err error) {
	result = &v1alpha1.FederatedDeploymentOverride{}
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
