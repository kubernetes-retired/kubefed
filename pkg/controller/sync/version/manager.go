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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

type VersionManager struct {
	sync.RWMutex

	targetKind string

	templateKind string

	// Namespace to source propagated versions from
	namespace string

	adapter VersionAdapter

	hasSynced bool

	worker util.ReconcileWorker

	versions map[string]pkgruntime.Object
}

func NewVersionManager(client fedclientset.Interface, namespaced bool, templateKind, targetKind, namespace string) *VersionManager {
	v := &VersionManager{
		targetKind:   targetKind,
		templateKind: templateKind,
		namespace:    namespace,
		adapter:      NewVersionAdapter(client, namespaced),
		versions:     make(map[string]pkgruntime.Object),
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
// template and override.
func (m *VersionManager) Get(template, override *unstructured.Unstructured) map[string]string {
	versionMap := make(map[string]string)

	qualifiedName := m.versionQualifiedName(util.NewQualifiedName(template))
	key := qualifiedName.String()
	m.RLock()
	obj, ok := m.versions[key]
	m.RUnlock()
	if !ok {
		return versionMap
	}
	status := m.adapter.GetStatus(obj)

	templateVersion := template.GetResourceVersion()
	overrideVersion := ""
	if override != nil {
		overrideVersion = override.GetResourceVersion()
	}
	if templateVersion == status.TemplateVersion &&
		overrideVersion == status.OverrideVersion {
		for _, versions := range status.ClusterVersions {
			versionMap[versions.ClusterName] = versions.Version
		}
	}

	return versionMap
}

// Update ensures that the propagated version for the given template
// and override is recorded.
func (m *VersionManager) Update(template, override *unstructured.Unstructured,
	selectedClusters []string, versionMap map[string]string) {

	templateVersion := template.GetResourceVersion()

	overrideVersion := ""
	if override != nil {
		overrideVersion = override.GetResourceVersion()
	}

	templateQualifiedName := util.NewQualifiedName(template)
	qualifiedName := m.versionQualifiedName(templateQualifiedName)
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
		ownerReference := ownerReferenceForUnstructured(template)
		obj = m.adapter.NewVersion(qualifiedName, ownerReference, status)
		m.versions[key] = obj
	} else {
		m.adapter.SetStatus(obj, status)
	}

	m.Unlock()

	m.worker.Enqueue(qualifiedName)
}

// Delete removes the named propagated version from the manager.
// Versions are written to the API with an owner reference to the
// template, and they should be removed by the garbage collector on
// template removal.
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
			return false, fmt.Errorf("")
		default:
		}

		var err error
		versionList, err = m.adapter.List(m.namespace)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to list propagated versions for %q: %v", m.templateKind, err))
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
		runtime.HandleError(fmt.Errorf("Failed to understand list result for %q: %v", m.adapter.TypeName(), err))
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
	glog.V(4).Infof("Version manager for %q synced", m.templateKind)
	return true
}

// versionQualifiedName derives the qualified name of a version
// resource from the qualified name of a template or target resource.
func (m *VersionManager) versionQualifiedName(qualifiedName util.QualifiedName) util.QualifiedName {
	namespace := qualifiedName.Namespace
	if m.targetKind == util.NamespaceKind {
		namespace = qualifiedName.Name
	}
	versionName := common.PropagatedVersionName(m.targetKind, qualifiedName.Name)
	return util.QualifiedName{Name: versionName, Namespace: namespace}
}

// writeVersion serializes the current state of the named propagated version to the API.
func (m *VersionManager) writeVersion(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	key := qualifiedName.String()
	adapterType := m.adapter.TypeName()

	m.RLock()
	obj, ok := m.versions[key]
	m.RUnlock()
	if !ok {
		// Version is no longer tracked
		return util.StatusAllOK
	}

	// Locking shouldn't be required here since there is only a single
	// caller (this function) whose actions will result in metadata
	// changing.
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to retrieve meta accessor for %s %q: %s", adapterType, key, err))
		return util.StatusError
	}

	// Locking shouldn't be required here since there is only a single
	// caller (this function) whose actions will result in metadata
	// changing.
	creationNeeded := len(metaAccessor.GetResourceVersion()) == 0

	if creationNeeded {
		createdObj, err := m.adapter.Create(obj)
		if errors.IsAlreadyExists(err) {
			// Version was written to the API after the version manager loaded
			glog.V(4).Infof("Refreshing %s %q from the API due to already existing", adapterType, key)
			err := m.refreshVersion(obj)
			if err != nil {
				runtime.HandleError(fmt.Errorf("Failed to refresh existing %s %q from the API: %v", adapterType, key, err))
				return util.StatusError
			}
			return util.StatusNeedsRecheck
		}
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to create version %s %q: %s", adapterType, key, err))
			return util.StatusError
		}

		// Status on the created version will have been cleared.  Set
		// it from the in-memory instance.
		status := m.adapter.GetStatus(obj)
		m.adapter.SetStatus(createdObj, status)
		obj = createdObj
	}

	updatedObj, err := m.adapter.UpdateStatus(obj)
	if err == nil {
		m.Lock()
		defer m.Unlock()
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
	runtime.HandleError(fmt.Errorf("Failed to update status of %s %q: %v", adapterType, key, err))
	if !errors.IsConflict(err) {
		return util.StatusError
	}

	// Version has been updated or deleted since the last version
	// manager write.  Attempt to refresh and retry.

	glog.V(4).Infof("Refreshing %s %q from the API due to conflict", adapterType, key)
	err = m.refreshVersion(obj)
	if err == nil {
		return util.StatusNeedsRecheck
	}
	if errors.IsNotFound(err) {
		// Version has been deleted from the API since the last version manager write.
		// Clear the resource version to prompt creation.
		err := m.clearResourceVersion(key)
		if err == nil {
			return util.StatusNeedsRecheck
		}
		runtime.HandleError(fmt.Errorf("Failed to clear resource version for %s %q: %s", adapterType, key, err))
		return util.StatusError
	}
	runtime.HandleError(fmt.Errorf("Failed to refresh conflicted %s %q from the API: %v", adapterType, key, err))
	return util.StatusError
}

func (m *VersionManager) refreshVersion(obj pkgruntime.Object) error {
	// A read lock is not suggested due to the name and namespace of a
	// resource being immutable.
	qualifiedName := util.NewQualifiedName(obj)

	glog.V(4).Infof("Refreshing %s version %q from the API", m.templateKind, qualifiedName)
	refreshedObj, err := m.adapter.Get(qualifiedName)
	if err != nil {
		return err
	}
	key := qualifiedName.String()
	m.Lock()
	defer m.Unlock()
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
	m.Lock()
	defer m.Unlock()
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
