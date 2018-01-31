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
package fake

import (
	v1alpha1 "github.com/marun/fnord/pkg/apis/federation/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFederationPlacements implements FederationPlacementInterface
type FakeFederationPlacements struct {
	Fake *FakeFederationV1alpha1
	ns   string
}

var federationplacementsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "v1alpha1", Resource: "federationplacements"}

var federationplacementsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "v1alpha1", Kind: "FederationPlacement"}

// Get takes name of the federationPlacement, and returns the corresponding federationPlacement object, and an error if there is any.
func (c *FakeFederationPlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederationPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federationplacementsResource, c.ns, name), &v1alpha1.FederationPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederationPlacement), err
}

// List takes label and field selectors, and returns the list of FederationPlacements that match those selectors.
func (c *FakeFederationPlacements) List(opts v1.ListOptions) (result *v1alpha1.FederationPlacementList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federationplacementsResource, federationplacementsKind, c.ns, opts), &v1alpha1.FederationPlacementList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FederationPlacementList{}
	for _, item := range obj.(*v1alpha1.FederationPlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federationPlacements.
func (c *FakeFederationPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federationplacementsResource, c.ns, opts))

}

// Create takes the representation of a federationPlacement and creates it.  Returns the server's representation of the federationPlacement, and an error, if there is any.
func (c *FakeFederationPlacements) Create(federationPlacement *v1alpha1.FederationPlacement) (result *v1alpha1.FederationPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federationplacementsResource, c.ns, federationPlacement), &v1alpha1.FederationPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederationPlacement), err
}

// Update takes the representation of a federationPlacement and updates it. Returns the server's representation of the federationPlacement, and an error, if there is any.
func (c *FakeFederationPlacements) Update(federationPlacement *v1alpha1.FederationPlacement) (result *v1alpha1.FederationPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federationplacementsResource, c.ns, federationPlacement), &v1alpha1.FederationPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederationPlacement), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederationPlacements) UpdateStatus(federationPlacement *v1alpha1.FederationPlacement) (*v1alpha1.FederationPlacement, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federationplacementsResource, "status", c.ns, federationPlacement), &v1alpha1.FederationPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederationPlacement), err
}

// Delete takes name of the federationPlacement and deletes it. Returns an error if one occurs.
func (c *FakeFederationPlacements) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federationplacementsResource, c.ns, name), &v1alpha1.FederationPlacement{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederationPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federationplacementsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FederationPlacementList{})
	return err
}

// Patch applies the patch and returns the patched federationPlacement.
func (c *FakeFederationPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederationPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federationplacementsResource, c.ns, name, data, subresources...), &v1alpha1.FederationPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederationPlacement), err
}
