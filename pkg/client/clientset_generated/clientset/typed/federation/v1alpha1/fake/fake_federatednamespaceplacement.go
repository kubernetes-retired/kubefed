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

// FakeFederatedNamespacePlacements implements FederatedNamespacePlacementInterface
type FakeFederatedNamespacePlacements struct {
	Fake *FakeFederationV1alpha1
}

var federatednamespaceplacementsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "v1alpha1", Resource: "federatednamespaceplacements"}

var federatednamespaceplacementsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "v1alpha1", Kind: "FederatedNamespacePlacement"}

// Get takes name of the federatedNamespacePlacement, and returns the corresponding federatedNamespacePlacement object, and an error if there is any.
func (c *FakeFederatedNamespacePlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedNamespacePlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(federatednamespaceplacementsResource, name), &v1alpha1.FederatedNamespacePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedNamespacePlacement), err
}

// List takes label and field selectors, and returns the list of FederatedNamespacePlacements that match those selectors.
func (c *FakeFederatedNamespacePlacements) List(opts v1.ListOptions) (result *v1alpha1.FederatedNamespacePlacementList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(federatednamespaceplacementsResource, federatednamespaceplacementsKind, opts), &v1alpha1.FederatedNamespacePlacementList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FederatedNamespacePlacementList{}
	for _, item := range obj.(*v1alpha1.FederatedNamespacePlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedNamespacePlacements.
func (c *FakeFederatedNamespacePlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(federatednamespaceplacementsResource, opts))
}

// Create takes the representation of a federatedNamespacePlacement and creates it.  Returns the server's representation of the federatedNamespacePlacement, and an error, if there is any.
func (c *FakeFederatedNamespacePlacements) Create(federatedNamespacePlacement *v1alpha1.FederatedNamespacePlacement) (result *v1alpha1.FederatedNamespacePlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(federatednamespaceplacementsResource, federatedNamespacePlacement), &v1alpha1.FederatedNamespacePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedNamespacePlacement), err
}

// Update takes the representation of a federatedNamespacePlacement and updates it. Returns the server's representation of the federatedNamespacePlacement, and an error, if there is any.
func (c *FakeFederatedNamespacePlacements) Update(federatedNamespacePlacement *v1alpha1.FederatedNamespacePlacement) (result *v1alpha1.FederatedNamespacePlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(federatednamespaceplacementsResource, federatedNamespacePlacement), &v1alpha1.FederatedNamespacePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedNamespacePlacement), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedNamespacePlacements) UpdateStatus(federatedNamespacePlacement *v1alpha1.FederatedNamespacePlacement) (*v1alpha1.FederatedNamespacePlacement, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(federatednamespaceplacementsResource, "status", federatedNamespacePlacement), &v1alpha1.FederatedNamespacePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedNamespacePlacement), err
}

// Delete takes name of the federatedNamespacePlacement and deletes it. Returns an error if one occurs.
func (c *FakeFederatedNamespacePlacements) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(federatednamespaceplacementsResource, name), &v1alpha1.FederatedNamespacePlacement{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedNamespacePlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(federatednamespaceplacementsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FederatedNamespacePlacementList{})
	return err
}

// Patch applies the patch and returns the patched federatedNamespacePlacement.
func (c *FakeFederatedNamespacePlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedNamespacePlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(federatednamespaceplacementsResource, name, data, subresources...), &v1alpha1.FederatedNamespacePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedNamespacePlacement), err
}
