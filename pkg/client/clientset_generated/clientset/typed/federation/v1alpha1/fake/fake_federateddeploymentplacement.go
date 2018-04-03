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

// FakeFederatedDeploymentPlacements implements FederatedDeploymentPlacementInterface
type FakeFederatedDeploymentPlacements struct {
	Fake *FakeFederationV1alpha1
	ns   string
}

var federateddeploymentplacementsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "v1alpha1", Resource: "federateddeploymentplacements"}

var federateddeploymentplacementsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "v1alpha1", Kind: "FederatedDeploymentPlacement"}

// Get takes name of the federatedDeploymentPlacement, and returns the corresponding federatedDeploymentPlacement object, and an error if there is any.
func (c *FakeFederatedDeploymentPlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedDeploymentPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federateddeploymentplacementsResource, c.ns, name), &v1alpha1.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentPlacement), err
}

// List takes label and field selectors, and returns the list of FederatedDeploymentPlacements that match those selectors.
func (c *FakeFederatedDeploymentPlacements) List(opts v1.ListOptions) (result *v1alpha1.FederatedDeploymentPlacementList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federateddeploymentplacementsResource, federateddeploymentplacementsKind, c.ns, opts), &v1alpha1.FederatedDeploymentPlacementList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FederatedDeploymentPlacementList{}
	for _, item := range obj.(*v1alpha1.FederatedDeploymentPlacementList).Items {
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
func (c *FakeFederatedDeploymentPlacements) Create(federatedDeploymentPlacement *v1alpha1.FederatedDeploymentPlacement) (result *v1alpha1.FederatedDeploymentPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federateddeploymentplacementsResource, c.ns, federatedDeploymentPlacement), &v1alpha1.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentPlacement), err
}

// Update takes the representation of a federatedDeploymentPlacement and updates it. Returns the server's representation of the federatedDeploymentPlacement, and an error, if there is any.
func (c *FakeFederatedDeploymentPlacements) Update(federatedDeploymentPlacement *v1alpha1.FederatedDeploymentPlacement) (result *v1alpha1.FederatedDeploymentPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federateddeploymentplacementsResource, c.ns, federatedDeploymentPlacement), &v1alpha1.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentPlacement), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedDeploymentPlacements) UpdateStatus(federatedDeploymentPlacement *v1alpha1.FederatedDeploymentPlacement) (*v1alpha1.FederatedDeploymentPlacement, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federateddeploymentplacementsResource, "status", c.ns, federatedDeploymentPlacement), &v1alpha1.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentPlacement), err
}

// Delete takes name of the federatedDeploymentPlacement and deletes it. Returns an error if one occurs.
func (c *FakeFederatedDeploymentPlacements) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federateddeploymentplacementsResource, c.ns, name), &v1alpha1.FederatedDeploymentPlacement{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedDeploymentPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federateddeploymentplacementsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FederatedDeploymentPlacementList{})
	return err
}

// Patch applies the patch and returns the patched federatedDeploymentPlacement.
func (c *FakeFederatedDeploymentPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedDeploymentPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federateddeploymentplacementsResource, c.ns, name, data, subresources...), &v1alpha1.FederatedDeploymentPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentPlacement), err
}
