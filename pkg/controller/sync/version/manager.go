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

package version

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

// VersionedResource defines the methods a federated resource must
// implement to allow versions to be tracked by the VersionManager.
type VersionedResource interface {
	FederatedName() util.QualifiedName
	Object() *unstructured.Unstructured
	TemplateVersion() (string, error)
	OverrideVersion() (string, error)
}

type VersionManager struct {
	sync.RWMutex

	targetKind string

	federatedKind string

	// Namespace to source propagated versions from
	namespace string

	adapter VersionAdapter

	hasSynced bool

	versions map[string]pkgruntime.Object

	client generic.Client
}

func NewVersionManager(client generic.Client, namespaced bool, federatedKind, targetKind, namespace string) *VersionManager {
	v := &VersionManager{
		targetKind:    targetKind,
		federatedKind: federatedKind,
		namespace:     namespace,
		adapter:       NewVersionAdapter(namespaced),
		versions:      make(map[string]pkgruntime.Object),
		client:        client,
	}

	return v
}

// Sync retrieves propagated versions from the api and loads it into
// memory.
func (m *VersionManager) Sync(stopChan <-chan struct{}) {
	versionList, ok := m.list(stopChan)
	if !ok {
		return
	}
	ok = m.load(versionList, stopChan)
	if !ok {
		return
	}
}

// HasSynced indicates whether the manager's in-memory state has been
// synced with the api.
func (m *VersionManager) HasSynced() bool {
	m.RLock()
	defer m.RUnlock()
	return m.hasSynced
}

// Get retrieves a mapping of cluster names to versions for the given
// versioned resource.
func (m *VersionManager) Get(resource VersionedResource) (map[string]string, error) {
	versionMap := make(map[string]string)

	qualifiedName := m.versionQualifiedName(resource.FederatedName())
	key := qualifiedName.String()
	m.RLock()
	obj, ok := m.versions[key]
	m.RUnlock()
	if !ok {
		return versionMap, nil
	}
	status := m.adapter.GetStatus(obj)

	templateVersion, err := resource.TemplateVersion()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to determine template version")
	}
	overrideVersion, err := resource.OverrideVersion()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to determine override version")
	}
	if templateVersion == status.TemplateVersion &&
		overrideVersion == status.OverrideVersion {
		for _, versions := range status.ClusterVersions {
			versionMap[versions.ClusterName] = versions.Version
		}
	}

	return versionMap, nil
}

// Update ensures that the propagated version for the given versioned
// resource is recorded.
func (m *VersionManager) Update(resource VersionedResource,
	selectedClusters []string, versionMap map[string]string) error {

	templateVersion, err := resource.TemplateVersion()
	if err != nil {
		return errors.Wrap(err, "Failed to determine template version")
	}
	overrideVersion, err := resource.OverrideVersion()
	if err != nil {
		return errors.Wrap(err, "Failed to determine override version")
	}
	qualifiedName := m.versionQualifiedName(resource.FederatedName())
	key := qualifiedName.String()

	m.Lock()

	obj, ok := m.versions[key]

	var oldStatus *fedv1a1.PropagatedVersionStatus
	var clusterVersions []fedv1a1.ClusterObjectVersion
	if ok {
		oldStatus = m.adapter.GetStatus(obj)
		// The existing versions are still valid if the template and override versions match.
		if oldStatus.TemplateVersion == templateVersion && oldStatus.OverrideVersion == overrideVersion {
			clusterVersions = oldStatus.ClusterVersions
		}
		clusterVersions = updateClusterVersions(clusterVersions, versionMap, selectedClusters)
	} else {
		clusterVersions = VersionMapToClusterVersions(versionMap)
	}

	status := &fedv1a1.PropagatedVersionStatus{
		TemplateVersion: templateVersion,
		OverrideVersion: overrideVersion,
		ClusterVersions: clusterVersions,
	}

	if oldStatus != nil && util.PropagatedVersionStatusEquivalent(oldStatus, status) {
		glog.V(4).Infof("No update necessary for %s %q", m.adapter.TypeName(), qualifiedName)
	} else if obj == nil {
		ownerReference := ownerReferenceForUnstructured(resource.Object())
		obj = m.adapter.NewVersion(qualifiedName, ownerReference, status)
		m.versions[key] = obj
	} else {
		m.adapter.SetStatus(obj, status)
	}

	m.Unlock()

	return m.writeVersion(obj, qualifiedName)
}

// Delete removes the named propagated version from the manager.
// Versions are written to the API with an owner reference to the
// versioned resource, and they should be removed by the garbage
// collector when the resource is removed.
func (m *VersionManager) Delete(qualifiedName util.QualifiedName) {
	versionQualifiedName := m.versionQualifiedName(qualifiedName)
	m.Lock()
	delete(m.versions, versionQualifiedName.String())
	m.Unlock()
}

func (m *VersionManager) list(stopChan <-chan struct{}) (pkgruntime.Object, bool) {
	// Attempt retrieval of list of versions until success or the channel is closed.
	var versionList pkgruntime.Object
	err := wait.PollImmediateInfinite(1*time.Second, func() (bool, error) {
		select {
		case <-stopChan:
			glog.V(4).Infof("Halting version manager list due to closed stop channel")
			return false, errors.New("")
		default:
		}
		versionList = m.adapter.NewListObject()
		err := m.client.List(context.TODO(), versionList, m.namespace)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to list propagated versions for %q", m.federatedKind))
			// Do not return the error to allow the operation to be retried.
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, false
	}
	return versionList, true
}

// load processes a list of versions into in-memory cache.  Since the
// version manager should not be used in advance of HasSynced
// returning true, locking is assumed to be unnecessary.
func (m *VersionManager) load(versionList pkgruntime.Object, stopChan <-chan struct{}) bool {
	typePrefix := common.PropagatedVersionPrefix(m.targetKind)
	items, err := meta.ExtractList(versionList)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to understand list result for %q", m.adapter.TypeName()))
		return false
	}
	for _, obj := range items {
		select {
		case <-stopChan:
			glog.V(4).Infof("Halting version manager load due to closed stop channel")
			return false
		default:
		}

		qualifiedName := util.NewQualifiedName(obj)
		// Ignore propagated version for other types
		if strings.HasPrefix(qualifiedName.Name, typePrefix) {
			m.versions[qualifiedName.String()] = obj
		}
	}
	m.Lock()
	m.hasSynced = true
	m.Unlock()
	glog.V(4).Infof("Version manager for %q synced", m.federatedKind)
	return true
}

// versionQualifiedName derives the qualified name of a version
// resource from the qualified name of a template or target resource.
func (m *VersionManager) versionQualifiedName(qualifiedName util.QualifiedName) util.QualifiedName {
	versionName := common.PropagatedVersionName(m.targetKind, qualifiedName.Name)
	return util.QualifiedName{Name: versionName, Namespace: qualifiedName.Namespace}
}

// writeVersion serializes the current state of the named propagated
// version to the API.
//
// The manager is expected to be called synchronously by the sync
// controller which should ensure that the version object for a given
// resource is updated by at most one thread at a time.  This should
// guarantee safe manipulation of an object retrieved from the
// version map.
func (m *VersionManager) writeVersion(obj pkgruntime.Object, qualifiedName util.QualifiedName) error {
	key := qualifiedName.String()
	adapterType := m.adapter.TypeName()

	resourceVersion, err := getResourceVersion(obj)
	if err != nil {
		return errors.Wrapf(err, "Failed to retrieve the resourceVersion from %s %q", adapterType, key)
	}
	refreshVersion := false
	// TODO(marun) Centralize polling interval and duration
	waitDuration := 30 * time.Second
	err = wait.PollImmediate(100*time.Millisecond, waitDuration, func() (bool, error) {
		if refreshVersion {
			// Version was written to the API by another process after the last manager write.
			var err error
			resourceVersion, err = m.getResourceVersionFromAPI(qualifiedName)
			if err != nil {
				runtime.HandleError(errors.Wrapf(err, "Failed to refresh the resourceVersion for %s %q from the API", adapterType, key))
				return false, nil
			}
			refreshVersion = false
		}

		if resourceVersion == "" {
			// Version resource needs to be created

			createdObj := obj.DeepCopyObject()
			err := setResourceVersion(createdObj, "")
			if err != nil {
				runtime.HandleError(errors.Wrapf(err, "Failed to clear the resourceVersion for %s %q", adapterType, key))
				return false, nil
			}

			glog.V(4).Infof("Creating %s %q", adapterType, qualifiedName)
			err = m.client.Create(context.TODO(), createdObj)
			if apierrors.IsAlreadyExists(err) {
				glog.V(4).Infof("%s %q was created by another process. Will refresh the resourceVersion and attempt to update.", adapterType, qualifiedName)
				refreshVersion = true
				return false, nil
			}
			// Forbidden is likely to be a permanent failure and
			// likely the result of the containing namespace being
			// deleted.
			if apierrors.IsForbidden(err) {
				return false, err
			}
			if err != nil {
				runtime.HandleError(errors.Wrapf(err, "Failed to create %s %q", adapterType, key))
				return false, nil
			}

			// Update the resource version that will be used for update.
			resourceVersion, err = getResourceVersion(createdObj)
			if err != nil {
				runtime.HandleError(errors.Wrapf(err, "Failed to retrieve the resourceVersion for %s %q", adapterType, key))
				return false, nil
			}
		}

		// Update the status of an existing object

		updatedObj := obj.DeepCopyObject()
		err := setResourceVersion(updatedObj, resourceVersion)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to set the resourceVeresion for %s %q", adapterType, key))
			return false, nil
		}

		glog.V(4).Infof("Updating the status of %s %q", adapterType, qualifiedName)
		err = m.client.UpdateStatus(context.TODO(), updatedObj)
		if apierrors.IsConflict(err) {
			glog.V(4).Infof("%s %q was updated by another process. Will refresh the resourceVersion and retry the update.", adapterType, qualifiedName)
			refreshVersion = true
			return false, nil
		}
		if apierrors.IsNotFound(err) {
			glog.V(4).Infof("%s %q was deleted by another process. Will clear the resourceVersion and retry the update.", adapterType, qualifiedName)
			resourceVersion = ""
			return false, nil
		}
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to update the status of %s %q", adapterType, key))
			return false, nil
		}

		// Update was successful. All returns should be true even in
		// the event of an error since the next reconcile can also
		// refresh the resource version if necessary.

		// Update the version resource
		resourceVersion, err = getResourceVersion(updatedObj)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to retrieve the resourceVersion for %s %q", adapterType, key))
			return true, nil
		}
		err = setResourceVersion(obj, resourceVersion)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to set the resourceVersion for %s %q", adapterType, key))
		}

		return true, nil
	})
	if err != nil {
		return errors.Wrapf(err, "Failed to write the version map for %s %q to the API", adapterType, key)
	}
	return nil
}

func (m *VersionManager) getResourceVersionFromAPI(qualifiedName util.QualifiedName) (string, error) {
	glog.V(4).Infof("Retrieving resourceVersion for %s %q from the API", m.federatedKind, qualifiedName)
	obj := m.adapter.NewObject()
	err := m.client.Get(context.TODO(), obj, qualifiedName.Namespace, qualifiedName.Name)
	if err != nil {
		return "", err
	}
	return getResourceVersion(obj)
}

func getResourceVersion(obj pkgruntime.Object) (string, error) {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return "", err
	}
	return metaAccessor.GetResourceVersion(), nil
}

func setResourceVersion(obj pkgruntime.Object, resourceVersion string) error {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	metaAccessor.SetResourceVersion(resourceVersion)
	return nil
}

func ownerReferenceForUnstructured(obj *unstructured.Unstructured) metav1.OwnerReference {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       obj.GetName(),
		UID:        obj.GetUID(),
	}
}

func updateClusterVersions(oldVersions []fedv1a1.ClusterObjectVersion,
	newVersions map[string]string, selectedClusters []string) []fedv1a1.ClusterObjectVersion {

	// Retain versions for selected clusters that were not changed
	selectedClusterSet := sets.NewString(selectedClusters...)
	for _, oldVersion := range oldVersions {
		if !selectedClusterSet.Has(oldVersion.ClusterName) {
			continue
		}
		if _, ok := newVersions[oldVersion.ClusterName]; !ok {
			newVersions[oldVersion.ClusterName] = oldVersion.Version
		}
	}

	return VersionMapToClusterVersions(newVersions)
}

func VersionMapToClusterVersions(versionMap map[string]string) []fedv1a1.ClusterObjectVersion {
	clusterVersions := []fedv1a1.ClusterObjectVersion{}
	for clusterName, version := range versionMap {
		// Lack of version indicates deletion
		if version == "" {
			continue
		}
		clusterVersions = append(clusterVersions, fedv1a1.ClusterObjectVersion{
			ClusterName: clusterName,
			Version:     version,
		})
	}
	util.SortClusterVersions(clusterVersions)
	return clusterVersions
}
