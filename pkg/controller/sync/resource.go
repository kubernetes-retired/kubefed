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
	"sync"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/dispatch"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/version"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

// FederatedResource encapsulates the behavior of a logical federated
// resource which may be implemented by one or more kubernetes
// resources in the cluster hosting the federation control plane.
type FederatedResource interface {
	dispatch.FederatedResourceForDispatch

	FederatedName() util.QualifiedName
	FederatedKind() string
	UpdateVersions(selectedClusters []string, versionMap map[string]string) error
	DeleteVersions()
	ComputePlacement(clusters []*fedv1a1.FederatedCluster) (selectedClusters sets.String, err error)
	IsNamespaceInHostCluster(clusterObj pkgruntime.Object) bool
}

type federatedResource struct {
	sync.RWMutex

	limitedScope      bool
	typeConfig        typeconfig.Interface
	targetIsNamespace bool
	targetName        util.QualifiedName
	federatedKind     string
	federatedName     util.QualifiedName
	federatedResource *unstructured.Unstructured
	versionManager    *version.VersionManager
	overridesMap      util.OverridesMap
	versionMap        map[string]string
	namespace         *unstructured.Unstructured
	fedNamespace      *unstructured.Unstructured
	eventRecorder     record.EventRecorder
}

func (r *federatedResource) FederatedName() util.QualifiedName {
	return r.federatedName
}

func (r *federatedResource) FederatedKind() string {
	return r.typeConfig.GetFederatedType().Kind
}

func (r *federatedResource) TargetName() util.QualifiedName {
	return r.targetName
}

func (r *federatedResource) TargetKind() string {
	return r.typeConfig.GetTarget().Kind
}

func (r *federatedResource) Object() *unstructured.Unstructured {
	return r.federatedResource
}

func (r *federatedResource) TemplateVersion() (string, error) {
	obj := r.federatedResource
	return GetTemplateHash(obj.Object)
}

func (r *federatedResource) OverrideVersion() (string, error) {
	// TODO(marun) Consider hashing overrides per cluster to minimize
	// unnecessary updates.
	return GetOverrideHash(r.federatedResource)
}

func (r *federatedResource) VersionForCluster(clusterName string) (string, error) {
	r.Lock()
	defer r.Unlock()
	if r.versionMap == nil {
		var err error
		r.versionMap, err = r.versionManager.Get(r)
		if err != nil {
			return "", err
		}
	}
	return r.versionMap[clusterName], nil
}

func (r *federatedResource) UpdateVersions(selectedClusters []string, versionMap map[string]string) error {
	return r.versionManager.Update(r, selectedClusters, versionMap)
}

func (r *federatedResource) DeleteVersions() {
	r.versionManager.Delete(r.federatedName)
}

func (r *federatedResource) ComputePlacement(clusters []*fedv1a1.FederatedCluster) (sets.String, error) {
	if r.typeConfig.GetNamespaced() {
		return computeNamespacedPlacement(r.federatedResource, r.fedNamespace, clusters, r.limitedScope)
	}
	return computePlacement(r.federatedResource, clusters)
}

func (r *federatedResource) IsNamespaceInHostCluster(clusterObj pkgruntime.Object) bool {
	// TODO(marun) This comment should be added to the documentation
	// and removed from this function (where it is no longer
	// relevant).
	//
	// `Namespace` is the only Kubernetes type that can contain other
	// types, and adding a federation-specific container type would be
	// difficult or impossible. This requires that namespaced
	// federated resources exist in regular namespaces.
	//
	// An implication of using regular namespaces is that the sync
	// controller cannot delete a namespace in the host cluster in
	// response to a contained federated namespace not selecting the
	// host cluster for placement.  Doing so would remove the
	// federated namespace and result in the removal of the namespace
	// from all clusters (not just from the host cluster).
	//
	// Deletion of a federated namespace should also not result in
	// deletion of its containing namespace, since that could result
	// in the deletion of a namespaced federation control plane.
	return r.targetIsNamespace && util.IsPrimaryCluster(r.namespace, clusterObj)
}

// TODO(marun) Marshall the template once per reconcile, not per-cluster
func (r *federatedResource) ObjectForCluster(clusterName string) (*unstructured.Unstructured, error) {
	templateBody, ok, err := unstructured.NestedMap(r.federatedResource.Object, util.SpecField, util.TemplateField)
	if err != nil {
		return nil, errors.Wrap(err, "Error retrieving template body")
	}
	if !ok {
		// Some resources (like namespaces) can be created from an
		// empty template.
		templateBody = make(map[string]interface{})
	}
	obj := &unstructured.Unstructured{Object: templateBody}

	// Avoid having to duplicate these details in the template or have
	// the name/namespace vary between the federation api and member
	// clusters.
	//
	// TODO(marun) this should be documented
	obj.SetName(r.federatedResource.GetName())
	if !r.targetIsNamespace {
		obj.SetNamespace(r.federatedResource.GetNamespace())
	}
	targetApiResource := r.typeConfig.GetTarget()
	obj.SetKind(targetApiResource.Kind)
	obj.SetAPIVersion(fmt.Sprintf("%s/%s", targetApiResource.Group, targetApiResource.Version))

	overrides, err := r.overridesForCluster(clusterName)
	if err != nil {
		return nil, err
	}
	if overrides != nil {
		for path, value := range overrides {
			pathEntries := strings.Split(path, ".")
			if err := unstructured.SetNestedField(obj.Object, value, pathEntries...); err != nil {
				return nil, err
			}
		}
	}

	// Ensure that resources managed by federation always have the
	// managed label.  The label is intended to be targeted by all the
	// federation controllers.
	util.AddManagedLabel(obj)

	return obj, nil
}

// TODO(marun) Use an enumeration for errorCode.
func (r *federatedResource) RecordError(errorCode string, err error) {
	r.eventRecorder.Eventf(r.Object(), corev1.EventTypeWarning, errorCode, err.Error())
}

func (r *federatedResource) RecordEvent(reason, messageFmt string, args ...interface{}) {
	r.eventRecorder.Eventf(r.Object(), corev1.EventTypeNormal, reason, messageFmt, args...)
}

func (r *federatedResource) overridesForCluster(clusterName string) (util.ClusterOverridesMap, error) {
	r.Lock()
	defer r.Unlock()
	if r.overridesMap == nil {
		overridesMap, err := util.GetOverrides(r.federatedResource)
		if err != nil {
			return nil, errors.Wrapf(err, "Error reading cluster overrides")
		}
		r.overridesMap = overridesMap
	}
	return r.overridesMap[clusterName], nil
}

func GetTemplateHash(fieldMap map[string]interface{}) (string, error) {
	fields := []string{util.SpecField, util.TemplateField}
	fieldMap, ok, err := unstructured.NestedMap(fieldMap, fields...)
	if err != nil {
		return "", errors.Wrapf(err, "Error retrieving %q", strings.Join(fields, "."))
	}
	if !ok {
		return "", nil
	}
	obj := &unstructured.Unstructured{Object: fieldMap}
	description := strings.Join(fields, ".")
	return hashUnstructured(obj, description)
}

func GetOverrideHash(rawObj *unstructured.Unstructured) (string, error) {
	override := util.GenericOverride{}
	err := util.UnstructuredToInterface(rawObj, &override)
	if err != nil {
		return "", errors.Wrap(err, "Error retrieving overrides")
	}
	// Only hash the overrides
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"overrides": override.Spec.Overrides,
		},
	}

	return hashUnstructured(obj, "overrides")
}

// TODO(marun) Investigate alternate ways of computing the hash of a field map.
func hashUnstructured(obj *unstructured.Unstructured, description string) (string, error) {
	jsonBytes, err := obj.MarshalJSON()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to marshal %q to json", description)
	}
	hash := md5.New()
	if _, err := hash.Write(jsonBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
