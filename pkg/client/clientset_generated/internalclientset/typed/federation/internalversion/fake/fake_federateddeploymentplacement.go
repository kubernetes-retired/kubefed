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
	federation "github.com/marun/federation-v2/pkg/apis/federation"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFederatedDeploymentPlacements implements FederatedDeploymentPlacementInterface
type FakeFederatedDeploymentPlacements struct {
	Fake *FakeFederation
	ns   string
}

var federateddeploymentplacementsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "", Resource: "federateddeploymentplacements"}

var federateddeploymentplacementsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "", Kind: "FederatedDeploymentPlacement"}

// Get takes name of the federatedDeploymentPlacement, and returns the corresponding federatedDeploymentPlacement object, and an error if there is any.
func (c *FakeFederatedDeploymentPlacements) Get(name string, options v1.GetOptions) (result *federation.FederatedDeploymentPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federateddeploymentplacementsResource, c.ns, name), &federation.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedDeploymentPlacement), err
}

// List takes label and field selectors, and returns the list of FederatedDeploymentPlacements that match those selectors.
func (c *FakeFederatedDeploymentPlacements) List(opts v1.ListOptions) (result *federation.FederatedDeploymentPlacementList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federateddeploymentplacementsResource, federateddeploymentplacementsKind, c.ns, opts), &federation.FederatedDeploymentPlacementList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &federation.FederatedDeploymentPlacementList{}
	for _, item := range obj.(*federation.FederatedDeploymentPlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedDeploymentPlacements.
func (c *FakeFederatedDeploymentPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federateddeploymentplacementsResource, c.ns, opts))

}

// Create takes the representation of a federatedDeploymentPlacement and creates it.  Returns the server's representation of the federatedDeploymentPlacement, and an error, if there is any.
func (c *FakeFederatedDeploymentPlacements) Create(federatedDeploymentPlacement *federation.FederatedDeploymentPlacement) (result *federation.FederatedDeploymentPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federateddeploymentplacementsResource, c.ns, federatedDeploymentPlacement), &federation.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedDeploymentPlacement), err
}

// Update takes the representation of a federatedDeploymentPlacement and updates it. Returns the server's representation of the federatedDeploymentPlacement, and an error, if there is any.
func (c *FakeFederatedDeploymentPlacements) Update(federatedDeploymentPlacement *federation.FederatedDeploymentPlacement) (result *federation.FederatedDeploymentPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federateddeploymentplacementsResource, c.ns, federatedDeploymentPlacement), &federation.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedDeploymentPlacement), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedDeploymentPlacements) UpdateStatus(federatedDeploymentPlacement *federation.FederatedDeploymentPlacement) (*federation.FederatedDeploymentPlacement, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federateddeploymentplacementsResource, "status", c.ns, federatedDeploymentPlacement), &federation.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedDeploymentPlacement), err
}

// Delete takes name of the federatedDeploymentPlacement and deletes it. Returns an error if one occurs.
func (c *FakeFederatedDeploymentPlacements) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federateddeploymentplacementsResource, c.ns, name), &federation.FederatedDeploymentPlacement{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedDeploymentPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federateddeploymentplacementsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &federation.FederatedDeploymentPlacementList{})
	return err
}

// Patch applies the patch and returns the patched federatedDeploymentPlacement.
func (c *FakeFederatedDeploymentPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedDeploymentPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federateddeploymentplacementsResource, c.ns, name, data, subresources...), &federation.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedDeploymentPlacement), err
}
