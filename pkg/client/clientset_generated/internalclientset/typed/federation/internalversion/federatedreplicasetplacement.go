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

// FederatedReplicaSetPlacementsGetter has a method to return a FederatedReplicaSetPlacementInterface.
// A group's client should implement this interface.
type FederatedReplicaSetPlacementsGetter interface {
	FederatedReplicaSetPlacements(namespace string) FederatedReplicaSetPlacementInterface
}

// FederatedReplicaSetPlacementInterface has methods to work with FederatedReplicaSetPlacement resources.
type FederatedReplicaSetPlacementInterface interface {
	Create(*federation.FederatedReplicaSetPlacement) (*federation.FederatedReplicaSetPlacement, error)
	Update(*federation.FederatedReplicaSetPlacement) (*federation.FederatedReplicaSetPlacement, error)
	UpdateStatus(*federation.FederatedReplicaSetPlacement) (*federation.FederatedReplicaSetPlacement, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederatedReplicaSetPlacement, error)
	List(opts v1.ListOptions) (*federation.FederatedReplicaSetPlacementList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedReplicaSetPlacement, err error)
	FederatedReplicaSetPlacementExpansion
}

// federatedReplicaSetPlacements implements FederatedReplicaSetPlacementInterface
type federatedReplicaSetPlacements struct {
	client rest.Interface
	ns     string
}

// newFederatedReplicaSetPlacements returns a FederatedReplicaSetPlacements
func newFederatedReplicaSetPlacements(c *FederationClient, namespace string) *federatedReplicaSetPlacements {
	return &federatedReplicaSetPlacements{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedReplicaSetPlacement, and returns the corresponding federatedReplicaSetPlacement object, and an error if there is any.
func (c *federatedReplicaSetPlacements) Get(name string, options v1.GetOptions) (result *federation.FederatedReplicaSetPlacement, err error) {
	result = &federation.FederatedReplicaSetPlacement{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedreplicasetplacements").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedReplicaSetPlacements that match those selectors.
func (c *federatedReplicaSetPlacements) List(opts v1.ListOptions) (result *federation.FederatedReplicaSetPlacementList, err error) {
	result = &federation.FederatedReplicaSetPlacementList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedreplicasetplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedReplicaSetPlacements.
func (c *federatedReplicaSetPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedreplicasetplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedReplicaSetPlacement and creates it.  Returns the server's representation of the federatedReplicaSetPlacement, and an error, if there is any.
func (c *federatedReplicaSetPlacements) Create(federatedReplicaSetPlacement *federation.FederatedReplicaSetPlacement) (result *federation.FederatedReplicaSetPlacement, err error) {
	result = &federation.FederatedReplicaSetPlacement{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedreplicasetplacements").
		Body(federatedReplicaSetPlacement).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedReplicaSetPlacement and updates it. Returns the server's representation of the federatedReplicaSetPlacement, and an error, if there is any.
func (c *federatedReplicaSetPlacements) Update(federatedReplicaSetPlacement *federation.FederatedReplicaSetPlacement) (result *federation.FederatedReplicaSetPlacement, err error) {
	result = &federation.FederatedReplicaSetPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedreplicasetplacements").
		Name(federatedReplicaSetPlacement.Name).
		Body(federatedReplicaSetPlacement).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedReplicaSetPlacements) UpdateStatus(federatedReplicaSetPlacement *federation.FederatedReplicaSetPlacement) (result *federation.FederatedReplicaSetPlacement, err error) {
	result = &federation.FederatedReplicaSetPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedreplicasetplacements").
		Name(federatedReplicaSetPlacement.Name).
		SubResource("status").
		Body(federatedReplicaSetPlacement).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedReplicaSetPlacement and deletes it. Returns an error if one occurs.
func (c *federatedReplicaSetPlacements) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedreplicasetplacements").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedReplicaSetPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedreplicasetplacements").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedReplicaSetPlacement.
func (c *federatedReplicaSetPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedReplicaSetPlacement, err error) {
	result = &federation.FederatedReplicaSetPlacement{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedreplicasetplacements").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
