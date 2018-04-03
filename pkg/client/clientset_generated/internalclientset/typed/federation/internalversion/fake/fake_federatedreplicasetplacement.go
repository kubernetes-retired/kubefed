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
	federation "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFederatedReplicaSetPlacements implements FederatedReplicaSetPlacementInterface
type FakeFederatedReplicaSetPlacements struct {
	Fake *FakeFederation
	ns   string
}

var federatedreplicasetplacementsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "", Resource: "federatedreplicasetplacements"}

var federatedreplicasetplacementsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "", Kind: "FederatedReplicaSetPlacement"}

// Get takes name of the federatedReplicaSetPlacement, and returns the corresponding federatedReplicaSetPlacement object, and an error if there is any.
func (c *FakeFederatedReplicaSetPlacements) Get(name string, options v1.GetOptions) (result *federation.FederatedReplicaSetPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedreplicasetplacementsResource, c.ns, name), &federation.FederatedReplicaSetPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedReplicaSetPlacement), err
}

// List takes label and field selectors, and returns the list of FederatedReplicaSetPlacements that match those selectors.
func (c *FakeFederatedReplicaSetPlacements) List(opts v1.ListOptions) (result *federation.FederatedReplicaSetPlacementList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedreplicasetplacementsResource, federatedreplicasetplacementsKind, c.ns, opts), &federation.FederatedReplicaSetPlacementList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &federation.FederatedReplicaSetPlacementList{}
	for _, item := range obj.(*federation.FederatedReplicaSetPlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedReplicaSetPlacements.
func (c *FakeFederatedReplicaSetPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedreplicasetplacementsResource, c.ns, opts))

}

// Create takes the representation of a federatedReplicaSetPlacement and creates it.  Returns the server's representation of the federatedReplicaSetPlacement, and an error, if there is any.
func (c *FakeFederatedReplicaSetPlacements) Create(federatedReplicaSetPlacement *federation.FederatedReplicaSetPlacement) (result *federation.FederatedReplicaSetPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedreplicasetplacementsResource, c.ns, federatedReplicaSetPlacement), &federation.FederatedReplicaSetPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedReplicaSetPlacement), err
}

// Update takes the representation of a federatedReplicaSetPlacement and updates it. Returns the server's representation of the federatedReplicaSetPlacement, and an error, if there is any.
func (c *FakeFederatedReplicaSetPlacements) Update(federatedReplicaSetPlacement *federation.FederatedReplicaSetPlacement) (result *federation.FederatedReplicaSetPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedreplicasetplacementsResource, c.ns, federatedReplicaSetPlacement), &federation.FederatedReplicaSetPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedReplicaSetPlacement), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedReplicaSetPlacements) UpdateStatus(federatedReplicaSetPlacement *federation.FederatedReplicaSetPlacement) (*federation.FederatedReplicaSetPlacement, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedreplicasetplacementsResource, "status", c.ns, federatedReplicaSetPlacement), &federation.FederatedReplicaSetPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedReplicaSetPlacement), err
}

// Delete takes name of the federatedReplicaSetPlacement and deletes it. Returns an error if one occurs.
func (c *FakeFederatedReplicaSetPlacements) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedreplicasetplacementsResource, c.ns, name), &federation.FederatedReplicaSetPlacement{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedReplicaSetPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedreplicasetplacementsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &federation.FederatedReplicaSetPlacementList{})
	return err
}

// Patch applies the patch and returns the patched federatedReplicaSetPlacement.
func (c *FakeFederatedReplicaSetPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedReplicaSetPlacement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedreplicasetplacementsResource, c.ns, name, data, subresources...), &federation.FederatedReplicaSetPlacement{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedReplicaSetPlacement), err
}
