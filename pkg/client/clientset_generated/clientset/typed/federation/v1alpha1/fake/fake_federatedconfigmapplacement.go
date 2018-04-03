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
	v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFederatedConfigMapPlacements implements FederatedConfigMapPlacementInterface
type FakeFederatedConfigMapPlacements struct {
	Fake *FakeFederationV1alpha1
	ns   string
}

var federatedconfigmapplacementsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "v1alpha1", Resource: "federatedconfigmapplacements"}

var federatedconfigmapplacementsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "v1alpha1", Kind: "FederatedConfigMapPlacement"}

// Get takes name of the federatedConfigMapPlacement, and returns the corresponding federatedConfigMapPlacement object, and an error if there is any.
func (c *FakeFederatedConfigMapPlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedConfigMapPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedconfigmapplacementsResource, c.ns, name), &v1alpha1.FederatedConfigMapPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedConfigMapPlacement), err
}

// List takes label and field selectors, and returns the list of FederatedConfigMapPlacements that match those selectors.
func (c *FakeFederatedConfigMapPlacements) List(opts v1.ListOptions) (result *v1alpha1.FederatedConfigMapPlacementList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedconfigmapplacementsResource, federatedconfigmapplacementsKind, c.ns, opts), &v1alpha1.FederatedConfigMapPlacementList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FederatedConfigMapPlacementList{}
	for _, item := range obj.(*v1alpha1.FederatedConfigMapPlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedConfigMapPlacements.
func (c *FakeFederatedConfigMapPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedconfigmapplacementsResource, c.ns, opts))

}

// Create takes the representation of a federatedConfigMapPlacement and creates it.  Returns the server's representation of the federatedConfigMapPlacement, and an error, if there is any.
func (c *FakeFederatedConfigMapPlacements) Create(federatedConfigMapPlacement *v1alpha1.FederatedConfigMapPlacement) (result *v1alpha1.FederatedConfigMapPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedconfigmapplacementsResource, c.ns, federatedConfigMapPlacement), &v1alpha1.FederatedConfigMapPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedConfigMapPlacement), err
}

// Update takes the representation of a federatedConfigMapPlacement and updates it. Returns the server's representation of the federatedConfigMapPlacement, and an error, if there is any.
func (c *FakeFederatedConfigMapPlacements) Update(federatedConfigMapPlacement *v1alpha1.FederatedConfigMapPlacement) (result *v1alpha1.FederatedConfigMapPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedconfigmapplacementsResource, c.ns, federatedConfigMapPlacement), &v1alpha1.FederatedConfigMapPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedConfigMapPlacement), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedConfigMapPlacements) UpdateStatus(federatedConfigMapPlacement *v1alpha1.FederatedConfigMapPlacement) (*v1alpha1.FederatedConfigMapPlacement, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedconfigmapplacementsResource, "status", c.ns, federatedConfigMapPlacement), &v1alpha1.FederatedConfigMapPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedConfigMapPlacement), err
}

// Delete takes name of the federatedConfigMapPlacement and deletes it. Returns an error if one occurs.
func (c *FakeFederatedConfigMapPlacements) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedconfigmapplacementsResource, c.ns, name), &v1alpha1.FederatedConfigMapPlacement{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedConfigMapPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedconfigmapplacementsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FederatedConfigMapPlacementList{})
	return err
}

// Patch applies the patch and returns the patched federatedConfigMapPlacement.
func (c *FakeFederatedConfigMapPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedConfigMapPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedconfigmapplacementsResource, c.ns, name, data, subresources...), &v1alpha1.FederatedConfigMapPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedConfigMapPlacement), err
}
