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

// FederatedSecretsGetter has a method to return a FederatedSecretInterface.
// A group's client should implement this interface.
type FederatedSecretsGetter interface {
	FederatedSecrets(namespace string) FederatedSecretInterface
}

// FederatedSecretInterface has methods to work with FederatedSecret resources.
type FederatedSecretInterface interface {
	Create(*federation.FederatedSecret) (*federation.FederatedSecret, error)
	Update(*federation.FederatedSecret) (*federation.FederatedSecret, error)
	UpdateStatus(*federation.FederatedSecret) (*federation.FederatedSecret, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederatedSecret, error)
	List(opts v1.ListOptions) (*federation.FederatedSecretList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedSecret, err error)
	FederatedSecretExpansion
}

// federatedSecrets implements FederatedSecretInterface
type federatedSecrets struct {
	client rest.Interface
	ns     string
}

// newFederatedSecrets returns a FederatedSecrets
func newFederatedSecrets(c *FederationClient, namespace string) *federatedSecrets {
	return &federatedSecrets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedSecret, and returns the corresponding federatedSecret object, and an error if there is any.
func (c *federatedSecrets) Get(name string, options v1.GetOptions) (result *federation.FederatedSecret, err error) {
	result = &federation.FederatedSecret{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecrets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedSecrets that match those selectors.
func (c *federatedSecrets) List(opts v1.ListOptions) (result *federation.FederatedSecretList, err error) {
	result = &federation.FederatedSecretList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecrets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedSecrets.
func (c *federatedSecrets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecrets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedSecret and creates it.  Returns the server's representation of the federatedSecret, and an error, if there is any.
func (c *federatedSecrets) Create(federatedSecret *federation.FederatedSecret) (result *federation.FederatedSecret, err error) {
	result = &federation.FederatedSecret{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedsecrets").
		Body(federatedSecret).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedSecret and updates it. Returns the server's representation of the federatedSecret, and an error, if there is any.
func (c *federatedSecrets) Update(federatedSecret *federation.FederatedSecret) (result *federation.FederatedSecret, err error) {
	result = &federation.FederatedSecret{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedsecrets").
		Name(federatedSecret.Name).
		Body(federatedSecret).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedSecrets) UpdateStatus(federatedSecret *federation.FederatedSecret) (result *federation.FederatedSecret, err error) {
	result = &federation.FederatedSecret{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedsecrets").
		Name(federatedSecret.Name).
		SubResource("status").
		Body(federatedSecret).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedSecret and deletes it. Returns an error if one occurs.
func (c *federatedSecrets) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedsecrets").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedSecrets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedsecrets").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedSecret.
func (c *federatedSecrets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedSecret, err error) {
	result = &federation.FederatedSecret{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedsecrets").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
