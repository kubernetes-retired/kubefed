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

// FakeFederatedSecretPlacements implements FederatedSecretPlacementInterface
type FakeFederatedSecretPlacements struct {
	Fake *FakeFederationV1alpha1
	ns   string
}

var federatedsecretplacementsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "v1alpha1", Resource: "federatedsecretplacements"}

var federatedsecretplacementsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "v1alpha1", Kind: "FederatedSecretPlacement"}

// Get takes name of the federatedSecretPlacement, and returns the corresponding federatedSecretPlacement object, and an error if there is any.
func (c *FakeFederatedSecretPlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedSecretPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedsecretplacementsResource, c.ns, name), &v1alpha1.FederatedSecretPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretPlacement), err
}

// List takes label and field selectors, and returns the list of FederatedSecretPlacements that match those selectors.
func (c *FakeFederatedSecretPlacements) List(opts v1.ListOptions) (result *v1alpha1.FederatedSecretPlacementList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedsecretplacementsResource, federatedsecretplacementsKind, c.ns, opts), &v1alpha1.FederatedSecretPlacementList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FederatedSecretPlacementList{}
	for _, item := range obj.(*v1alpha1.FederatedSecretPlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedSecretPlacements.
func (c *FakeFederatedSecretPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedsecretplacementsResource, c.ns, opts))

}

// Create takes the representation of a federatedSecretPlacement and creates it.  Returns the server's representation of the federatedSecretPlacement, and an error, if there is any.
func (c *FakeFederatedSecretPlacements) Create(federatedSecretPlacement *v1alpha1.FederatedSecretPlacement) (result *v1alpha1.FederatedSecretPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedsecretplacementsResource, c.ns, federatedSecretPlacement), &v1alpha1.FederatedSecretPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretPlacement), err
}

// Update takes the representation of a federatedSecretPlacement and updates it. Returns the server's representation of the federatedSecretPlacement, and an error, if there is any.
func (c *FakeFederatedSecretPlacements) Update(federatedSecretPlacement *v1alpha1.FederatedSecretPlacement) (result *v1alpha1.FederatedSecretPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedsecretplacementsResource, c.ns, federatedSecretPlacement), &v1alpha1.FederatedSecretPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretPlacement), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedSecretPlacements) UpdateStatus(federatedSecretPlacement *v1alpha1.FederatedSecretPlacement) (*v1alpha1.FederatedSecretPlacement, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedsecretplacementsResource, "status", c.ns, federatedSecretPlacement), &v1alpha1.FederatedSecretPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretPlacement), err
}

// Delete takes name of the federatedSecretPlacement and deletes it. Returns an error if one occurs.
func (c *FakeFederatedSecretPlacements) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedsecretplacementsResource, c.ns, name), &v1alpha1.FederatedSecretPlacement{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedSecretPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedsecretplacementsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FederatedSecretPlacementList{})
	return err
}

// Patch applies the patch and returns the patched federatedSecretPlacement.
func (c *FakeFederatedSecretPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedSecretPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedsecretplacementsResource, c.ns, name, data, subresources...), &v1alpha1.FederatedSecretPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretPlacement), err
}
