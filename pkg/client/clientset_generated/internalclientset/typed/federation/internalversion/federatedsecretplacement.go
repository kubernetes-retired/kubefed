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

// FederatedSecretPlacementsGetter has a method to return a FederatedSecretPlacementInterface.
// A group's client should implement this interface.
type FederatedSecretPlacementsGetter interface {
	FederatedSecretPlacements(namespace string) FederatedSecretPlacementInterface
}

// FederatedSecretPlacementInterface has methods to work with FederatedSecretPlacement resources.
type FederatedSecretPlacementInterface interface {
	Create(*federation.FederatedSecretPlacement) (*federation.FederatedSecretPlacement, error)
	Update(*federation.FederatedSecretPlacement) (*federation.FederatedSecretPlacement, error)
	UpdateStatus(*federation.FederatedSecretPlacement) (*federation.FederatedSecretPlacement, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederatedSecretPlacement, error)
	List(opts v1.ListOptions) (*federation.FederatedSecretPlacementList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedSecretPlacement, err error)
	FederatedSecretPlacementExpansion
}

// federatedSecretPlacements implements FederatedSecretPlacementInterface
type federatedSecretPlacements struct {
	client rest.Interface
	ns     string
}

// newFederatedSecretPlacements returns a FederatedSecretPlacements
func newFederatedSecretPlacements(c *FederationClient, namespace string) *federatedSecretPlacements {
	return &federatedSecretPlacements{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedSecretPlacement, and returns the corresponding federatedSecretPlacement object, and an error if there is any.
func (c *federatedSecretPlacements) Get(name string, options v1.GetOptions) (result *federation.FederatedSecretPlacement, err error) {
	result = &federation.FederatedSecretPlacement{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecretplacements").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedSecretPlacements that match those selectors.
func (c *federatedSecretPlacements) List(opts v1.ListOptions) (result *federation.FederatedSecretPlacementList, err error) {
	result = &federation.FederatedSecretPlacementList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecretplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedSecretPlacements.
func (c *federatedSecretPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedsecretplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedSecretPlacement and creates it.  Returns the server's representation of the federatedSecretPlacement, and an error, if there is any.
func (c *federatedSecretPlacements) Create(federatedSecretPlacement *federation.FederatedSecretPlacement) (result *federation.FederatedSecretPlacement, err error) {
	result = &federation.FederatedSecretPlacement{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedsecretplacements").
		Body(federatedSecretPlacement).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedSecretPlacement and updates it. Returns the server's representation of the federatedSecretPlacement, and an error, if there is any.
func (c *federatedSecretPlacements) Update(federatedSecretPlacement *federation.FederatedSecretPlacement) (result *federation.FederatedSecretPlacement, err error) {
	result = &federation.FederatedSecretPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedsecretplacements").
		Name(federatedSecretPlacement.Name).
		Body(federatedSecretPlacement).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedSecretPlacements) UpdateStatus(federatedSecretPlacement *federation.FederatedSecretPlacement) (result *federation.FederatedSecretPlacement, err error) {
	result = &federation.FederatedSecretPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedsecretplacements").
		Name(federatedSecretPlacement.Name).
		SubResource("status").
		Body(federatedSecretPlacement).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedSecretPlacement and deletes it. Returns an error if one occurs.
func (c *federatedSecretPlacements) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedsecretplacements").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedSecretPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedsecretplacements").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedSecretPlacement.
func (c *federatedSecretPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedSecretPlacement, err error) {
	result = &federation.FederatedSecretPlacement{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedsecretplacements").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
