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

// FederatedReplicaSetsGetter has a method to return a FederatedReplicaSetInterface.
// A group's client should implement this interface.
type FederatedReplicaSetsGetter interface {
	FederatedReplicaSets(namespace string) FederatedReplicaSetInterface
}

// FederatedReplicaSetInterface has methods to work with FederatedReplicaSet resources.
type FederatedReplicaSetInterface interface {
	Create(*v1alpha1.FederatedReplicaSet) (*v1alpha1.FederatedReplicaSet, error)
	Update(*v1alpha1.FederatedReplicaSet) (*v1alpha1.FederatedReplicaSet, error)
	UpdateStatus(*v1alpha1.FederatedReplicaSet) (*v1alpha1.FederatedReplicaSet, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedReplicaSet, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedReplicaSetList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedReplicaSet, err error)
	FederatedReplicaSetExpansion
}

// federatedReplicaSets implements FederatedReplicaSetInterface
type federatedReplicaSets struct {
	client rest.Interface
	ns     string
}

// newFederatedReplicaSets returns a FederatedReplicaSets
func newFederatedReplicaSets(c *FederationV1alpha1Client, namespace string) *federatedReplicaSets {
	return &federatedReplicaSets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedReplicaSet, and returns the corresponding federatedReplicaSet object, and an error if there is any.
func (c *federatedReplicaSets) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedReplicaSet, err error) {
	result = &v1alpha1.FederatedReplicaSet{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedreplicasets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedReplicaSets that match those selectors.
func (c *federatedReplicaSets) List(opts v1.ListOptions) (result *v1alpha1.FederatedReplicaSetList, err error) {
	result = &v1alpha1.FederatedReplicaSetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedreplicasets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedReplicaSets.
func (c *federatedReplicaSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedreplicasets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedReplicaSet and creates it.  Returns the server's representation of the federatedReplicaSet, and an error, if there is any.
func (c *federatedReplicaSets) Create(federatedReplicaSet *v1alpha1.FederatedReplicaSet) (result *v1alpha1.FederatedReplicaSet, err error) {
	result = &v1alpha1.FederatedReplicaSet{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedreplicasets").
		Body(federatedReplicaSet).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedReplicaSet and updates it. Returns the server's representation of the federatedReplicaSet, and an error, if there is any.
func (c *federatedReplicaSets) Update(federatedReplicaSet *v1alpha1.FederatedReplicaSet) (result *v1alpha1.FederatedReplicaSet, err error) {
	result = &v1alpha1.FederatedReplicaSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedreplicasets").
		Name(federatedReplicaSet.Name).
		Body(federatedReplicaSet).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedReplicaSets) UpdateStatus(federatedReplicaSet *v1alpha1.FederatedReplicaSet) (result *v1alpha1.FederatedReplicaSet, err error) {
	result = &v1alpha1.FederatedReplicaSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedreplicasets").
		Name(federatedReplicaSet.Name).
		SubResource("status").
		Body(federatedReplicaSet).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedReplicaSet and deletes it. Returns an error if one occurs.
func (c *federatedReplicaSets) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedreplicasets").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedReplicaSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedreplicasets").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedReplicaSet.
func (c *federatedReplicaSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedReplicaSet, err error) {
	result = &v1alpha1.FederatedReplicaSet{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedreplicasets").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
