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

// FakeFederatedServicePlacements implements FederatedServicePlacementInterface
type FakeFederatedServicePlacements struct {
	Fake *FakeFederationV1alpha1
	ns   string
}

var federatedserviceplacementsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "v1alpha1", Resource: "federatedserviceplacements"}

var federatedserviceplacementsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "v1alpha1", Kind: "FederatedServicePlacement"}

// Get takes name of the federatedServicePlacement, and returns the corresponding federatedServicePlacement object, and an error if there is any.
func (c *FakeFederatedServicePlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedServicePlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedserviceplacementsResource, c.ns, name), &v1alpha1.FederatedServicePlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedServicePlacement), err
}

// List takes label and field selectors, and returns the list of FederatedServicePlacements that match those selectors.
func (c *FakeFederatedServicePlacements) List(opts v1.ListOptions) (result *v1alpha1.FederatedServicePlacementList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedserviceplacementsResource, federatedserviceplacementsKind, c.ns, opts), &v1alpha1.FederatedServicePlacementList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FederatedServicePlacementList{}
	for _, item := range obj.(*v1alpha1.FederatedServicePlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedServicePlacements.
func (c *FakeFederatedServicePlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedserviceplacementsResource, c.ns, opts))

}

// Create takes the representation of a federatedServicePlacement and creates it.  Returns the server's representation of the federatedServicePlacement, and an error, if there is any.
func (c *FakeFederatedServicePlacements) Create(federatedServicePlacement *v1alpha1.FederatedServicePlacement) (result *v1alpha1.FederatedServicePlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedserviceplacementsResource, c.ns, federatedServicePlacement), &v1alpha1.FederatedServicePlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedServicePlacement), err
}

// Update takes the representation of a federatedServicePlacement and updates it. Returns the server's representation of the federatedServicePlacement, and an error, if there is any.
func (c *FakeFederatedServicePlacements) Update(federatedServicePlacement *v1alpha1.FederatedServicePlacement) (result *v1alpha1.FederatedServicePlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedserviceplacementsResource, c.ns, federatedServicePlacement), &v1alpha1.FederatedServicePlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedServicePlacement), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedServicePlacements) UpdateStatus(federatedServicePlacement *v1alpha1.FederatedServicePlacement) (*v1alpha1.FederatedServicePlacement, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedserviceplacementsResource, "status", c.ns, federatedServicePlacement), &v1alpha1.FederatedServicePlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedServicePlacement), err
}

// Delete takes name of the federatedServicePlacement and deletes it. Returns an error if one occurs.
func (c *FakeFederatedServicePlacements) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedserviceplacementsResource, c.ns, name), &v1alpha1.FederatedServicePlacement{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedServicePlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedserviceplacementsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FederatedServicePlacementList{})
	return err
}

// Patch applies the patch and returns the patched federatedServicePlacement.
func (c *FakeFederatedServicePlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedServicePlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedserviceplacementsResource, c.ns, name, data, subresources...), &v1alpha1.FederatedServicePlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedServicePlacement), err
}
