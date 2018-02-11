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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry"
)

// FakeClusters implements ClusterInterface
type FakeClusters struct {
	Fake *FakeClusterregistry
}

var clustersResource = schema.GroupVersionResource{Group: "clusterregistry.k8s.io", Version: "", Resource: "clusters"}

var clustersKind = schema.GroupVersionKind{Group: "clusterregistry.k8s.io", Version: "", Kind: "Cluster"}

// Get takes name of the cluster, and returns the corresponding cluster object, and an error if there is any.
func (c *FakeClusters) Get(name string, options v1.GetOptions) (result *clusterregistry.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clustersResource, name), &clusterregistry.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*clusterregistry.Cluster), err
}

// List takes label and field selectors, and returns the list of Clusters that match those selectors.
func (c *FakeClusters) List(opts v1.ListOptions) (result *clusterregistry.ClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clustersResource, clustersKind, opts), &clusterregistry.ClusterList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &clusterregistry.ClusterList{}
	for _, item := range obj.(*clusterregistry.ClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusters.
func (c *FakeClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clustersResource, opts))
}

// Create takes the representation of a cluster and creates it.  Returns the server's representation of the cluster, and an error, if there is any.
func (c *FakeClusters) Create(cluster *clusterregistry.Cluster) (result *clusterregistry.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clustersResource, cluster), &clusterregistry.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*clusterregistry.Cluster), err
}

// Update takes the representation of a cluster and updates it. Returns the server's representation of the cluster, and an error, if there is any.
func (c *FakeClusters) Update(cluster *clusterregistry.Cluster) (result *clusterregistry.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clustersResource, cluster), &clusterregistry.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*clusterregistry.Cluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeClusters) UpdateStatus(cluster *clusterregistry.Cluster) (*clusterregistry.Cluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(clustersResource, "status", cluster), &clusterregistry.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*clusterregistry.Cluster), err
}

// Delete takes name of the cluster and deletes it. Returns an error if one occurs.
func (c *FakeClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clustersResource, name), &clusterregistry.Cluster{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clustersResource, listOptions)

	_, err := c.Fake.Invokes(action, &clusterregistry.ClusterList{})
	return err
}

// Patch applies the patch and returns the patched cluster.
func (c *FakeClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *clusterregistry.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clustersResource, name, data, subresources...), &clusterregistry.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*clusterregistry.Cluster), err
}
