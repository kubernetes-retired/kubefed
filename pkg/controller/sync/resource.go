/*
Copyright 2019 The Kubernetes Authors.

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

package sync

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/placement"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/version"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/deletionhelper"
)

// FederatedResource encapsulates the behavior of a logical federated
// resource which may be implemented by one or more kubernetes
// resources in the cluster hosting the federation control plane.
type FederatedResource interface {
	FederatedName() util.QualifiedName
	TargetName() util.QualifiedName
	Object() *unstructured.Unstructured
	GetVersions() (map[string]string, error)
	UpdateVersions(selectedClusters []string, versionMap map[string]string) error
	DeleteVersions()
	ComputePlacement(clusters []*fedv1a1.FederatedCluster) (selectedClusters, unselectedClusters []string, err error)
	SkipClusterChange(clusterObj pkgruntime.Object) bool
	ObjectForCluster(clusterName string) (*unstructured.Unstructured, error)
	FinalizationKind() string
	MarkedForDeletion() bool
	EnsureDeletion() error
	EnsureFinalizers() error
}

type federatedResource struct {
	typeConfig        typeconfig.Interface
	targetIsNamespace bool
	targetName        util.QualifiedName
	federatedName     util.QualifiedName
	template          *unstructured.Unstructured
	placement         *unstructured.Unstructured
	placementPlugin   placement.PlacementPlugin
	versionManager    *version.VersionManager
	deletionHelper    *deletionhelper.DeletionHelper
	overrideStore     cache.Store
	override          *unstructured.Unstructured
	overridesMap      util.OverridesMap
}

func (r *federatedResource) FederatedName() util.QualifiedName {
	return r.federatedName
}

func (r *federatedResource) TargetName() util.QualifiedName {
	return r.targetName
}

func (r *federatedResource) Object() *unstructured.Unstructured {
	if r.targetIsNamespace {
		return r.placement
	}
	return r.template
}

func (r *federatedResource) TemplateVersion() (string, error) {
	return GetTemplateHash(r.template)
}

func (r *federatedResource) OverrideVersion() (string, error) {
	override, err := r.getOverride()
	if err != nil {
		return "", err
	}
	// Overrides are optional
	if override == nil {
		return "", nil
	}
	// TODO(marun) Hash the overrides
	return override.GetResourceVersion(), nil
}

func (r *federatedResource) GetVersions() (map[string]string, error) {
	return r.versionManager.Get(r)
}

func (r *federatedResource) UpdateVersions(selectedClusters []string, versionMap map[string]string) error {
	return r.versionManager.Update(r, selectedClusters, versionMap)
}

func (r *federatedResource) DeleteVersions() {
	r.versionManager.Delete(r.federatedName)
}

func (r *federatedResource) ComputePlacement(clusters []*fedv1a1.FederatedCluster) ([]string, []string, error) {
	return r.placementPlugin.ComputePlacement(r.federatedName, clusters)
}

func (r *federatedResource) SkipClusterChange(clusterObj pkgruntime.Object) bool {
	// Updates should not be performed on namespaces in the host
	// cluster.  Such operations need to be performed via the Kube
	// API.
	//
	// The Namespace type is a special case because it is the only
	// container in the Kubernetes API.  Federation presumes a
	// separation between the template and target resources, but a
	// namespace in the host cluster is necessarily both template and
	// target.
	return r.targetIsNamespace && util.IsPrimaryCluster(r.template, clusterObj)
}

// TODO(marun) Marshall the template once per reconcile, not per-cluster
func (r *federatedResource) ObjectForCluster(clusterName string) (*unstructured.Unstructured, error) {
	// Federation of namespaces uses Namespace resources as the
	// template for resource creation in member clusters. All other
	// federated types rely on a template type distinct from the
	// target type.
	//
	// Namespace is the only type that can contain other resources,
	// and adding a federation-specific container type would be
	// difficult or impossible. This implies that federation
	// primitives need to exist in regular namespaces.
	//
	// TODO(marun) Ensure this is reflected in documentation
	obj := &unstructured.Unstructured{}
	if r.targetIsNamespace {
		metadata, ok, err := unstructured.NestedMap(r.template.Object, "metadata")
		if err != nil {
			return nil, fmt.Errorf("Error retrieving namespace metadata: %s", err)
		}
		if !ok {
			return nil, fmt.Errorf("Unable to retrieve namespace metadata")
		}
		// Retain only the target fields from the template
		targetFields := sets.NewString("name", "namespace", "labels", "annotations")
		for key := range metadata {
			if !targetFields.Has(key) {
				delete(metadata, key)
			}
		}
		obj.Object = make(map[string]interface{})
		obj.Object["metadata"] = metadata
	} else {
		var ok bool
		var err error
		obj.Object, ok, err = unstructured.NestedMap(r.template.Object, "spec", "template")
		if err != nil {
			return nil, fmt.Errorf("Error retrieving template body: %v", err)
		}
		if !ok {
			return nil, fmt.Errorf("Unable to retrieve template body")
		}
		// Avoid having to duplicate these details in the template or have
		// the name/namespace vary between the federation api and member
		// clusters.
		//
		// TODO(marun) this should be documented
		obj.SetName(r.template.GetName())
		obj.SetNamespace(r.template.GetNamespace())
		targetApiResource := r.typeConfig.GetTarget()
		obj.SetKind(targetApiResource.Kind)
		obj.SetAPIVersion(fmt.Sprintf("%s/%s", targetApiResource.Group, targetApiResource.Version))
	}

	overrides, err := r.overridesForCluster(clusterName)
	if err != nil {
		return nil, err
	}
	if overrides != nil {
		for path, value := range overrides {
			pathEntries := strings.Split(path, ".")
			unstructured.SetNestedField(obj.Object, value, pathEntries...)
		}
	}

	return obj, nil
}

func (r *federatedResource) FinalizationKind() string {
	return r.finalizationTarget().GetKind()
}

func (r *federatedResource) MarkedForDeletion() bool {
	return r.finalizationTarget().GetDeletionTimestamp() != nil
}

func (r *federatedResource) EnsureDeletion() error {
	r.DeleteVersions()
	_, err := r.deletionHelper.HandleObjectInUnderlyingClusters(
		r.finalizationTarget(),
		func(clusterObj pkgruntime.Object) bool {
			// Skip deletion of a namespace in the host cluster as it will be
			// removed by the garbage collector once its contents are removed.
			return r.targetIsNamespace && util.IsPrimaryCluster(r.template, clusterObj)
		},
	)
	return err
}

func (r *federatedResource) EnsureFinalizers() error {
	updatedObj, err := r.deletionHelper.EnsureFinalizers(r.finalizationTarget())
	if updatedObj != nil && !r.targetIsNamespace {
		// Retain the updated template for use in future API calls.
		// If the target is namespace, the placement resource is used
		// for finalization and no updates are expected.
		r.template = updatedObj.(*unstructured.Unstructured)
	}
	return err
}

func (r *federatedResource) finalizationTarget() *unstructured.Unstructured {
	// A placement resource determines whether a resource should exist
	// in member clusters. If a template exists but the associated
	// placement does not, then the resource represented by the
	// template should be removed from all member clusters.  That
	// suggests that finalization could as well be performed on the
	// placement as the template.
	//
	// Adding a finalizer to a namespace resource is problematic
	// because it has the potential to try to delete namespaces in
	// member clusters when a namespace in the host cluster is
	// deleted.  This could occur even if a namespace was never
	// intended to be federated.  Adding the finalizer to the
	// namespace placement resource will still ensure cleanup of
	// resources in member clusters (since deletion of the namespace
	// will trigger deletion of its placement resource) but ensure
	// that cleanup is only attempted for namespaces that have been
	// explicitly targeted for propagation.
	//
	// TODO(marun) Consider performing finalization on the placement
	// resource for other types.
	//
	if r.targetIsNamespace {
		return r.placement
	}
	return r.template
}

func (r *federatedResource) overridesForCluster(clusterName string) (util.ClusterOverridesMap, error) {
	if r.overridesMap == nil {
		override, err := r.getOverride()
		if err != nil {
			return nil, err
		}
		r.overridesMap, err = util.GetOverrides(override)
		if err != nil {
			overrideKind := r.typeConfig.GetOverride().Kind
			return nil, fmt.Errorf("Error reading cluster overrides for %s %q: %v", overrideKind, r.federatedName, err)
		}
	}
	return r.overridesMap[clusterName], nil
}

func (r *federatedResource) getOverride() (*unstructured.Unstructured, error) {
	if r.override == nil {
		overrideKind := r.typeConfig.GetOverride().Kind
		var err error
		r.override, err = util.ObjFromCache(r.overrideStore, overrideKind, r.federatedName.String())
		if err != nil {
			return nil, err
		}
	}
	return r.override, nil
}

func GetTemplateHash(template *unstructured.Unstructured) (string, error) {
	// A namespace resource is the template and the lack of status
	// updates to namespaces means the resource version is a good
	// indicator of changes the sync controller needs to consider.
	if template.GetKind() == util.NamespaceKind {
		return template.GetResourceVersion(), nil
	}

	obj := &unstructured.Unstructured{}
	templateMap, ok, err := unstructured.NestedMap(template.Object, "spec", "template")
	if err != nil {
		return "", fmt.Errorf("Error retrieving template body: %s", err)
	}
	if !ok {
		return "", nil
	}
	obj.Object = templateMap

	jsonBytes, err := obj.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("Failed to marshal template body to json: %v", err)
	}
	hash := md5.New()
	hash.Write(jsonBytes)
	return hex.EncodeToString(hash.Sum(nil)), nil
}
