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

// FederationPlacementsGetter has a method to return a FederationPlacementInterface.
// A group's client should implement this interface.
type FederationPlacementsGetter interface {
	FederationPlacements(namespace string) FederationPlacementInterface
}

// FederationPlacementInterface has methods to work with FederationPlacement resources.
type FederationPlacementInterface interface {
	Create(*federation.FederationPlacement) (*federation.FederationPlacement, error)
	Update(*federation.FederationPlacement) (*federation.FederationPlacement, error)
	UpdateStatus(*federation.FederationPlacement) (*federation.FederationPlacement, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederationPlacement, error)
	List(opts v1.ListOptions) (*federation.FederationPlacementList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederationPlacement, err error)
	FederationPlacementExpansion
}

// federationPlacements implements FederationPlacementInterface
type federationPlacements struct {
	client rest.Interface
	ns     string
}

// newFederationPlacements returns a FederationPlacements
func newFederationPlacements(c *FederationClient, namespace string) *federationPlacements {
	return &federationPlacements{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federationPlacement, and returns the corresponding federationPlacement object, and an error if there is any.
func (c *federationPlacements) Get(name string, options v1.GetOptions) (result *federation.FederationPlacement, err error) {
	result = &federation.FederationPlacement{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federationplacements").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederationPlacements that match those selectors.
func (c *federationPlacements) List(opts v1.ListOptions) (result *federation.FederationPlacementList, err error) {
	result = &federation.FederationPlacementList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federationplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federationPlacements.
func (c *federationPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federationplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federationPlacement and creates it.  Returns the server's representation of the federationPlacement, and an error, if there is any.
func (c *federationPlacements) Create(federationPlacement *federation.FederationPlacement) (result *federation.FederationPlacement, err error) {
	result = &federation.FederationPlacement{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federationplacements").
		Body(federationPlacement).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federationPlacement and updates it. Returns the server's representation of the federationPlacement, and an error, if there is any.
func (c *federationPlacements) Update(federationPlacement *federation.FederationPlacement) (result *federation.FederationPlacement, err error) {
	result = &federation.FederationPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federationplacements").
		Name(federationPlacement.Name).
		Body(federationPlacement).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federationPlacements) UpdateStatus(federationPlacement *federation.FederationPlacement) (result *federation.FederationPlacement, err error) {
	result = &federation.FederationPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federationplacements").
		Name(federationPlacement.Name).
		SubResource("status").
		Body(federationPlacement).
		Do().
		Into(result)
	return
}

// Delete takes name of the federationPlacement and deletes it. Returns an error if one occurs.
func (c *federationPlacements) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federationplacements").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federationPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federationplacements").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federationPlacement.
func (c *federationPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederationPlacement, err error) {
	result = &federation.FederationPlacement{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federationplacements").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
