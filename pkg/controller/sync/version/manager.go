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

	worker util.ReconcileWorker

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

	v.worker = util.NewReconcileWorker(v.writeVersion, util.WorkerTiming{
		Interval:       time.Millisecond * 50,
		RetryDelay:     time.Nanosecond * 1, // Effectively 0 delay
		InitialBackoff: time.Second * 1,
		MaxBackoff:     time.Second * 10,
	})

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

	m.worker.Run(stopChan)
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

	m.worker.Enqueue(qualifiedName)

	return nil
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

// writeVersion serializes the current state of the named propagated version to the API.
func (m *VersionManager) writeVersion(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	key := qualifiedName.String()
	adapterType := m.adapter.TypeName()

	m.Lock()
	obj, ok := m.versions[key]
	defer m.Unlock()
	if !ok {
		// Version is no longer tracked
		return util.StatusAllOK
	}

	// Locking shouldn't be required here since there is only a single
	// caller (this function) whose actions will result in metadata
	// changing.
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to retrieve meta accessor for %s %q", adapterType, key))
		return util.StatusError
	}

	// Locking shouldn't be required here since there is only a single
	// caller (this function) whose actions will result in metadata
	// changing.
	creationNeeded := len(metaAccessor.GetResourceVersion()) == 0

	if creationNeeded {
		// Make a copy to avoid mutating the map outside of a lock.
		createdObj := obj.DeepCopyObject()
		err := m.client.Create(context.TODO(), createdObj)
		if apierrors.IsAlreadyExists(err) {
			// Version was written to the API after the version manager loaded
			glog.V(4).Infof("Refreshing %s %q from the API due to already existing", adapterType, key)
			err := m.refreshVersion(obj)
			if err != nil {
				runtime.HandleError(errors.Wrapf(err, "Failed to refresh existing %s %q from the API", adapterType, key))
				return util.StatusError
			}
			return util.StatusNeedsRecheck
		}
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to create version %s %q", adapterType, key))
			return util.StatusError
		}

		// Status on the created version will have been cleared.  Set
		// it from the in-memory instance.  Since status is referred
		// to via a pointer and it is desirable to have the latest
		// status, it should be ok to do this outside of a lock.
		status := m.adapter.GetStatus(obj)
		m.adapter.SetStatus(createdObj, status)
		obj = createdObj
	}

	// Make a copy to avoid mutating the map outside of a lock.
	updatedObj := obj.DeepCopyObject()
	err = m.client.UpdateStatus(context.TODO(), updatedObj)
	if err == nil {
		_, ok := m.versions[key]
		if ok {
			// Update the version since it is still being tracked.
			m.versions[key] = updatedObj
			return util.StatusAllOK
		}

		// Version was deleted from memory and may need to be deleted
		// from the API
		return util.StatusNeedsRecheck
	}
	if !apierrors.IsConflict(err) {
		runtime.HandleError(errors.Wrapf(err, "Failed to update status of %s %q", adapterType, key))
		return util.StatusError
	}
	glog.Warningf("Error indicating conflict occurred on status update of %s %q: %v", adapterType, key, err)

	// Version has been updated or deleted since the last version
	// manager write.  Attempt to refresh and retry.

	glog.V(4).Infof("Refreshing %s %q from the API", adapterType, key)
	err = m.refreshVersion(obj)
	if err == nil {
		return util.StatusNeedsRecheck
	}
	if apierrors.IsNotFound(err) {
		// Version has been deleted from the API since the last version manager write.
		// Clear the resource version to prompt creation.
		err := m.clearResourceVersion(key)
		if err == nil {
			return util.StatusNeedsRecheck
		}
		runtime.HandleError(errors.Wrapf(err, "Failed to clear resource version for %s %q", adapterType, key))
		return util.StatusError
	}
	runtime.HandleError(errors.Wrapf(err, "Failed to refresh conflicted %s %q from the API", adapterType, key))
	return util.StatusError
}

func (m *VersionManager) refreshVersion(obj pkgruntime.Object) error {
	// A read lock is not suggested due to the name and namespace of a
	// resource being immutable.
	qualifiedName := util.NewQualifiedName(obj)

	glog.V(4).Infof("Refreshing %s version %q from the API", m.federatedKind, qualifiedName)
	refreshedObj := m.adapter.NewObject()
	err := m.client.Get(context.TODO(), refreshedObj, qualifiedName.Namespace, qualifiedName.Name)
	if err != nil {
		return err
	}
	key := qualifiedName.String()
	// Should be Locked in caller function
	_, ok := m.versions[key]
	if !ok {
		// Version has been deleted, no further action required
		return nil
	}

	// Retain the status from the in-memory copy.
	status := m.adapter.GetStatus(obj)
	m.adapter.SetStatus(refreshedObj, status)
	m.versions[key] = refreshedObj
	return nil
}

func (m *VersionManager) clearResourceVersion(key string) error {
	// Should be locked in calller function
	obj, ok := m.versions[key]
	if !ok {
		// Version is deleted
		return nil
	}
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	metaAccessor.SetResourceVersion("")
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
