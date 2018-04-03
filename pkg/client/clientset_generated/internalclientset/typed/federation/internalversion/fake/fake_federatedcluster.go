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

// FakeFederatedClusters implements FederatedClusterInterface
type FakeFederatedClusters struct {
	Fake *FakeFederation
}

var federatedclustersResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "", Resource: "federatedclusters"}

var federatedclustersKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "", Kind: "FederatedCluster"}

// Get takes name of the federatedCluster, and returns the corresponding federatedCluster object, and an error if there is any.
func (c *FakeFederatedClusters) Get(name string, options v1.GetOptions) (result *federation.FederatedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(federatedclustersResource, name), &federation.FederatedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedCluster), err
}

// List takes label and field selectors, and returns the list of FederatedClusters that match those selectors.
func (c *FakeFederatedClusters) List(opts v1.ListOptions) (result *federation.FederatedClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(federatedclustersResource, federatedclustersKind, opts), &federation.FederatedClusterList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &federation.FederatedClusterList{}
	for _, item := range obj.(*federation.FederatedClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedClusters.
func (c *FakeFederatedClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(federatedclustersResource, opts))
}

// Create takes the representation of a federatedCluster and creates it.  Returns the server's representation of the federatedCluster, and an error, if there is any.
func (c *FakeFederatedClusters) Create(federatedCluster *federation.FederatedCluster) (result *federation.FederatedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(federatedclustersResource, federatedCluster), &federation.FederatedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedCluster), err
}

// Update takes the representation of a federatedCluster and updates it. Returns the server's representation of the federatedCluster, and an error, if there is any.
func (c *FakeFederatedClusters) Update(federatedCluster *federation.FederatedCluster) (result *federation.FederatedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(federatedclustersResource, federatedCluster), &federation.FederatedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedClusters) UpdateStatus(federatedCluster *federation.FederatedCluster) (*federation.FederatedCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(federatedclustersResource, "status", federatedCluster), &federation.FederatedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedCluster), err
}

// Delete takes name of the federatedCluster and deletes it. Returns an error if one occurs.
func (c *FakeFederatedClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(federatedclustersResource, name), &federation.FederatedCluster{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(federatedclustersResource, listOptions)

	_, err := c.Fake.Invokes(action, &federation.FederatedClusterList{})
	return err
}

// Patch applies the patch and returns the patched federatedCluster.
func (c *FakeFederatedClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(federatedclustersResource, name, data, subresources...), &federation.FederatedCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedCluster), err
}
