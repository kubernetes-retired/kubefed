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
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/version"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/deletionhelper"
)

// FederatedResource encapsulates the behavior of a logical federated
// resource which may be implemented by one or more kubernetes
// resources in the cluster hosting the federation control plane.
type FederatedResource interface {
	FederatedName() util.QualifiedName
	FederatedKind() string
	TargetName() util.QualifiedName
	TargetKind() string
	Object() *unstructured.Unstructured
	VersionForCluster(clusterName string) (string, error)
	UpdateVersions(selectedClusters []string, versionMap map[string]string) error
	DeleteVersions()
	ComputePlacement(clusters []*fedv1a1.FederatedCluster) (selectedClusters sets.String, err error)
	SkipClusterChange(clusterObj pkgruntime.Object) bool
	ObjectForCluster(clusterName string) (*unstructured.Unstructured, error)
	MarkedForDeletion() bool
	EnsureDeletion() error
	EnsureFinalizer() error
	RecordError(errorCode string, err error)
	RecordEvent(reason, messageFmt string, args ...interface{})
	NewUpdater() FederatedUpdater
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
	deletionHelper    *deletionhelper.DeletionHelper
	overridesMap      util.OverridesMap
	versionMap        map[string]string
	namespace         *unstructured.Unstructured
	fedNamespace      *unstructured.Unstructured
	eventRecorder     record.EventRecorder
	informer          util.FederatedInformer
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
	if r.targetIsNamespace {
		obj = r.namespace
	}
	return GetTemplateHash(obj.Object, r.targetIsNamespace)
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
	return r.targetIsNamespace && util.IsPrimaryCluster(r.namespace, clusterObj)
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
	// difficult or impossible. This implies that federated types need
	// to exist in regular namespaces.
	//
	// TODO(marun) Ensure this is reflected in documentation
	obj := &unstructured.Unstructured{}
	if r.targetIsNamespace {
		var err error
		obj, err = namespaceFromTemplate(r.namespace.Object)
		if err != nil {
			return nil, err
		}
	} else {
		var ok bool
		var err error
		obj.Object, ok, err = unstructured.NestedMap(r.federatedResource.Object, util.SpecField, util.TemplateField)
		if err != nil {
			return nil, errors.Wrap(err, "Error retrieving template body")
		}
		if !ok {
			return nil, errors.New("Unable to retrieve template body")
		}
		// Avoid having to duplicate these details in the template or have
		// the name/namespace vary between the federation api and member
		// clusters.
		//
		// TODO(marun) this should be documented
		obj.SetName(r.federatedResource.GetName())
		obj.SetNamespace(r.federatedResource.GetNamespace())
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

func (r *federatedResource) MarkedForDeletion() bool {
	return r.federatedResource.GetDeletionTimestamp() != nil
}

func (r *federatedResource) EnsureDeletion() error {
	r.DeleteVersions()
	_, err := r.deletionHelper.HandleObjectInUnderlyingClusters(
		r.federatedResource,
		r.TargetName().String(),
		func(clusterObj pkgruntime.Object) bool {
			// Skip deletion of a namespace in the host cluster as it will be
			// removed by the garbage collector once its contents are removed.
			return r.targetIsNamespace && util.IsPrimaryCluster(r.namespace, clusterObj)
		},
	)
	return err
}

func (r *federatedResource) EnsureFinalizer() error {
	updatedObj, err := r.deletionHelper.EnsureFinalizer(r.federatedResource)
	if updatedObj != nil {
		// Retain the updated template for use in future API calls.
		r.federatedResource = updatedObj.(*unstructured.Unstructured)
	}
	return err
}

// TODO(marun) Use an enumeration for errorCode.
func (r *federatedResource) RecordError(errorCode string, err error) {
	r.eventRecorder.Eventf(r.Object(), corev1.EventTypeWarning, errorCode, err.Error())
}

func (r *federatedResource) RecordEvent(reason, messageFmt string, args ...interface{}) {
	r.eventRecorder.Eventf(r.Object(), corev1.EventTypeNormal, reason, messageFmt, args...)
}

func (r *federatedResource) NewUpdater() FederatedUpdater {
	return NewFederatedUpdater(r.informer, r)
}

func (r *federatedResource) overridesForCluster(clusterName string) (util.ClusterOverridesMap, error) {
	r.Lock()
	defer r.Unlock()
	if r.overridesMap == nil {
		overridesMap, err := util.GetOverrides(r.federatedResource)
		if err != nil {
			return nil, errors.Errorf("Error reading cluster overrides for %s %q", r.federatedKind, r.federatedName)
		}
		r.overridesMap = overridesMap
	}
	return r.overridesMap[clusterName], nil
}

func namespaceFromTemplate(fieldMap map[string]interface{}) (*unstructured.Unstructured, error) {
	metadata, ok, err := unstructured.NestedMap(fieldMap, "metadata")
	if err != nil {
		return nil, errors.Wrap(err, "Error retrieving namespace metadata")
	}
	if !ok {
		return nil, errors.New("Unable to retrieve namespace metadata")
	}
	// Retain only the target fields from the template
	targetFields := sets.NewString("name", "labels", "annotations")
	for key := range metadata {
		if !targetFields.Has(key) {
			delete(metadata, key)
		}
	}
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": metadata,
		},
	}
	return obj, nil
}

func GetTemplateHash(fieldMap map[string]interface{}, namespaceIsTarget bool) (string, error) {
	var obj *unstructured.Unstructured
	var description string
	if namespaceIsTarget {
		var err error
		obj, err = namespaceFromTemplate(fieldMap)
		if err != nil {
			return "", err
		}
		description = "namespace"
	} else {
		fields := []string{util.SpecField, util.TemplateField}
		fieldMap, ok, err := unstructured.NestedMap(fieldMap, fields...)
		if err != nil {
			return "", errors.Wrapf(err, "Error retrieving %q", strings.Join(fields, "."))
		}
		if !ok {
			return "", nil
		}
		obj = &unstructured.Unstructured{Object: fieldMap}
		description = strings.Join(fields, ".")
	}

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
	hash.Write(jsonBytes)
	return hex.EncodeToString(hash.Sum(nil)), nil
}
