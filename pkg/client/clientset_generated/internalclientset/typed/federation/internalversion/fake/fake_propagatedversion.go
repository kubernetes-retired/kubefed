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

// FakePropagatedVersions implements PropagatedVersionInterface
type FakePropagatedVersions struct {
	Fake *FakeFederation
	ns   string
}

var propagatedversionsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "", Resource: "propagatedversions"}

var propagatedversionsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "", Kind: "PropagatedVersion"}

// Get takes name of the propagatedVersion, and returns the corresponding propagatedVersion object, and an error if there is any.
func (c *FakePropagatedVersions) Get(name string, options v1.GetOptions) (result *federation.PropagatedVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(propagatedversionsResource, c.ns, name), &federation.PropagatedVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.PropagatedVersion), err
}

// List takes label and field selectors, and returns the list of PropagatedVersions that match those selectors.
func (c *FakePropagatedVersions) List(opts v1.ListOptions) (result *federation.PropagatedVersionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(propagatedversionsResource, propagatedversionsKind, c.ns, opts), &federation.PropagatedVersionList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &federation.PropagatedVersionList{}
	for _, item := range obj.(*federation.PropagatedVersionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested propagatedVersions.
func (c *FakePropagatedVersions) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(propagatedversionsResource, c.ns, opts))

}

// Create takes the representation of a propagatedVersion and creates it.  Returns the server's representation of the propagatedVersion, and an error, if there is any.
func (c *FakePropagatedVersions) Create(propagatedVersion *federation.PropagatedVersion) (result *federation.PropagatedVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(propagatedversionsResource, c.ns, propagatedVersion), &federation.PropagatedVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.PropagatedVersion), err
}

// Update takes the representation of a propagatedVersion and updates it. Returns the server's representation of the propagatedVersion, and an error, if there is any.
func (c *FakePropagatedVersions) Update(propagatedVersion *federation.PropagatedVersion) (result *federation.PropagatedVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(propagatedversionsResource, c.ns, propagatedVersion), &federation.PropagatedVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.PropagatedVersion), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePropagatedVersions) UpdateStatus(propagatedVersion *federation.PropagatedVersion) (*federation.PropagatedVersion, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(propagatedversionsResource, "status", c.ns, propagatedVersion), &federation.PropagatedVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.PropagatedVersion), err
}

// Delete takes name of the propagatedVersion and deletes it. Returns an error if one occurs.
func (c *FakePropagatedVersions) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(propagatedversionsResource, c.ns, name), &federation.PropagatedVersion{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePropagatedVersions) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(propagatedversionsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &federation.PropagatedVersionList{})
	return err
}

// Patch applies the patch and returns the patched propagatedVersion.
func (c *FakePropagatedVersions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.PropagatedVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(propagatedversionsResource, c.ns, name, data, subresources...), &federation.PropagatedVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.PropagatedVersion), err
}
