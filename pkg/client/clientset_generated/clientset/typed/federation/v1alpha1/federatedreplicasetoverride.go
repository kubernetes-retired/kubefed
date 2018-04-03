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

// FederatedReplicaSetOverridesGetter has a method to return a FederatedReplicaSetOverrideInterface.
// A group's client should implement this interface.
type FederatedReplicaSetOverridesGetter interface {
	FederatedReplicaSetOverrides(namespace string) FederatedReplicaSetOverrideInterface
}

// FederatedReplicaSetOverrideInterface has methods to work with FederatedReplicaSetOverride resources.
type FederatedReplicaSetOverrideInterface interface {
	Create(*v1alpha1.FederatedReplicaSetOverride) (*v1alpha1.FederatedReplicaSetOverride, error)
	Update(*v1alpha1.FederatedReplicaSetOverride) (*v1alpha1.FederatedReplicaSetOverride, error)
	UpdateStatus(*v1alpha1.FederatedReplicaSetOverride) (*v1alpha1.FederatedReplicaSetOverride, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedReplicaSetOverride, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedReplicaSetOverrideList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedReplicaSetOverride, err error)
	FederatedReplicaSetOverrideExpansion
}

// federatedReplicaSetOverrides implements FederatedReplicaSetOverrideInterface
type federatedReplicaSetOverrides struct {
	client rest.Interface
	ns     string
}

// newFederatedReplicaSetOverrides returns a FederatedReplicaSetOverrides
func newFederatedReplicaSetOverrides(c *FederationV1alpha1Client, namespace string) *federatedReplicaSetOverrides {
	return &federatedReplicaSetOverrides{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedReplicaSetOverride, and returns the corresponding federatedReplicaSetOverride object, and an error if there is any.
func (c *federatedReplicaSetOverrides) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedReplicaSetOverride, err error) {
	result = &v1alpha1.FederatedReplicaSetOverride{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedreplicasetoverrides").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedReplicaSetOverrides that match those selectors.
func (c *federatedReplicaSetOverrides) List(opts v1.ListOptions) (result *v1alpha1.FederatedReplicaSetOverrideList, err error) {
	result = &v1alpha1.FederatedReplicaSetOverrideList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedreplicasetoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedReplicaSetOverrides.
func (c *federatedReplicaSetOverrides) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedreplicasetoverrides").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedReplicaSetOverride and creates it.  Returns the server's representation of the federatedReplicaSetOverride, and an error, if there is any.
func (c *federatedReplicaSetOverrides) Create(federatedReplicaSetOverride *v1alpha1.FederatedReplicaSetOverride) (result *v1alpha1.FederatedReplicaSetOverride, err error) {
	result = &v1alpha1.FederatedReplicaSetOverride{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedreplicasetoverrides").
		Body(federatedReplicaSetOverride).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedReplicaSetOverride and updates it. Returns the server's representation of the federatedReplicaSetOverride, and an error, if there is any.
func (c *federatedReplicaSetOverrides) Update(federatedReplicaSetOverride *v1alpha1.FederatedReplicaSetOverride) (result *v1alpha1.FederatedReplicaSetOverride, err error) {
	result = &v1alpha1.FederatedReplicaSetOverride{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedreplicasetoverrides").
		Name(federatedReplicaSetOverride.Name).
		Body(federatedReplicaSetOverride).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedReplicaSetOverrides) UpdateStatus(federatedReplicaSetOverride *v1alpha1.FederatedReplicaSetOverride) (result *v1alpha1.FederatedReplicaSetOverride, err error) {
	result = &v1alpha1.FederatedReplicaSetOverride{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedreplicasetoverrides").
		Name(federatedReplicaSetOverride.Name).
		SubResource("status").
		Body(federatedReplicaSetOverride).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedReplicaSetOverride and deletes it. Returns an error if one occurs.
func (c *federatedReplicaSetOverrides) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedreplicasetoverrides").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedReplicaSetOverrides) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedreplicasetoverrides").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedReplicaSetOverride.
func (c *federatedReplicaSetOverrides) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedReplicaSetOverride, err error) {
	result = &v1alpha1.FederatedReplicaSetOverride{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedreplicasetoverrides").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
