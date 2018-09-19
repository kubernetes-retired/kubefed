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

package sync

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

type PropagatedVersionManager interface {
	// Sync retrieves propagated versions from the api and loads it into memory
	Sync(stopChan <-chan struct{})
	// HasSynced indicates whether the manager's in-memory state has been synced with the api
	HasSynced() bool

	// Get retrieves a mapping of cluster names to versions for the given template and override
	Get(template, override *unstructured.Unstructured) map[string]string
	// Update ensures that the propagated version for the given template and override is represented in-memory and in the api
	Update(template, override *unstructured.Unstructured, selectedClusters []string, clusterVersions map[string]string) error
	// Delete removes the named propagated version from in-memory and the api
	Delete(qualifiedName util.QualifiedName) error
}

type propagatedVersionManager struct {
	sync.RWMutex

	client fedclientset.Interface

	typeConfig typeconfig.Interface

	// Namespace to read propagated versions for
	namespace string

	versions map[string]*fedv1a1.PropagatedVersion

	hasSynced bool
}

func NewPropagatedVersionManager(typeConfig typeconfig.Interface, client fedclientset.Interface, namespace string) PropagatedVersionManager {
	return &propagatedVersionManager{
		client:     client,
		typeConfig: typeConfig,
		namespace:  namespace,
		versions:   make(map[string]*fedv1a1.PropagatedVersion),
	}
}

func (m *propagatedVersionManager) Sync(stopChan <-chan struct{}) {
	targetKind := m.typeConfig.GetTarget().Kind

	// Attempt retrieval of propagated version until success or the channel is closed.
	var versionList *fedv1a1.PropagatedVersionList
	err := wait.PollImmediateInfinite(1*time.Second, func() (bool, error) {
		select {
		case <-stopChan:
			return false, fmt.Errorf("Halting version manager sync due to closed stop channel")
		default:
		}

		var err error
		versionList, err = m.client.CoreV1alpha1().PropagatedVersions(m.namespace).List(metav1.ListOptions{})
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to list propagated versions for %q: %v", targetKind, err))
			// Do not return the error to allow the operation to be retried.
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		runtime.HandleError(err)
		return
	}

	m.Lock()
	defer m.Unlock()
	typePrefix := common.PropagatedVersionPrefix(targetKind)
	for _, version := range versionList.Items {
		select {
		case <-stopChan:
			return
		default:
		}

		// Ignore propagated version for other types
		if !strings.HasPrefix(version.Name, typePrefix) {
			continue
		}
		name := util.NewQualifiedName(&version)
		m.versions[name.String()] = &version
	}
	m.hasSynced = true
	glog.V(4).Infof("Version manager for %q synced", targetKind)
}

func (m *propagatedVersionManager) HasSynced() bool {
	m.RLock()
	defer m.RUnlock()
	return m.hasSynced
}

func (m *propagatedVersionManager) Get(template, override *unstructured.Unstructured) map[string]string {
	clusterVersions := make(map[string]string)

	key := util.QualifiedName{
		Namespace: template.GetNamespace(),
		Name:      m.versionName(template.GetName()),
	}.String()
	propagatedVersion, ok := m.getVersion(key)
	if !ok {
		return clusterVersions
	}

	templateVersion := template.GetResourceVersion()
	overrideVersion := ""
	if override != nil {
		overrideVersion = override.GetResourceVersion()
	}
	if templateVersion == propagatedVersion.Status.TemplateVersion &&
		overrideVersion == propagatedVersion.Status.OverrideVersion {
		for _, versions := range propagatedVersion.Status.ClusterVersions {
			clusterVersions[versions.ClusterName] = versions.Version
		}
	}

	return clusterVersions
}

func (m *propagatedVersionManager) Update(template, override *unstructured.Unstructured,
	selectedClusters []string, clusterVersions map[string]string) error {

	overrideVersion := ""
	if override != nil {
		overrideVersion = override.GetResourceVersion()
	}

	key := util.QualifiedName{
		Namespace: template.GetNamespace(),
		Name:      m.versionName(template.GetName()),
	}.String()

	version, ok := m.getVersion(key)
	if !ok {
		version := newVersion(clusterVersions, template, m.typeConfig.GetTarget().Kind, overrideVersion)
		createdVersion, err := m.client.CoreV1alpha1().PropagatedVersions(version.Namespace).Create(version)
		if errors.IsAlreadyExists(err) {
			createdVersion, err = m.client.CoreV1alpha1().PropagatedVersions(version.Namespace).Get(version.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
		m.setVersion(key, createdVersion)

		createdVersion.Status = version.Status
		updatedVersion, err := m.updateStatus(createdVersion)
		if err != nil {
			return err
		}
		m.setVersion(key, updatedVersion)

		return nil
	}

	oldVersionStatus := version.Status
	templateVersion := template.GetResourceVersion()
	var existingVersions []fedv1a1.ClusterObjectVersion
	if version.Status.TemplateVersion == templateVersion && version.Status.OverrideVersion == overrideVersion {
		existingVersions = version.Status.ClusterVersions
	} else {
		version.Status.TemplateVersion = templateVersion
		version.Status.OverrideVersion = overrideVersion
		existingVersions = []fedv1a1.ClusterObjectVersion{}
	}
	version.Status.ClusterVersions = updateClusterVersions(existingVersions, clusterVersions, selectedClusters)

	if util.PropagatedVersionStatusEquivalent(&oldVersionStatus, &version.Status) {
		glog.V(4).Infof("No PropagatedVersion update necessary for %s %q",
			m.typeConfig.GetTemplate().Kind, util.NewQualifiedName(template).String())
		return nil
	}

	updatedVersion, err := m.updateStatus(version)
	if err != nil {
		return err
	}

	m.setVersion(key, updatedVersion)
	return nil
}

func (m *propagatedVersionManager) Delete(qualifiedName util.QualifiedName) error {
	versionName := m.versionName(qualifiedName.Name)
	m.delVersion(util.QualifiedName{Name: versionName, Namespace: qualifiedName.Namespace}.String())
	err := m.client.CoreV1alpha1().PropagatedVersions(qualifiedName.Namespace).Delete(versionName, nil)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (m *propagatedVersionManager) updateStatus(version *fedv1a1.PropagatedVersion) (*fedv1a1.PropagatedVersion, error) {
	var updatedVersion *fedv1a1.PropagatedVersion
	err := wait.PollImmediate(100*time.Millisecond, 5*time.Second, func() (bool, error) {
		var err error
		updatedVersion, err = m.client.CoreV1alpha1().PropagatedVersions(version.Namespace).UpdateStatus(version)
		if err == nil {
			return true, nil
		}
		// Resource was updated in the api
		if errors.IsConflict(err) {
			retrievedVersion, err := m.client.CoreV1alpha1().PropagatedVersions(version.Namespace).Get(version.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			retrievedVersion.Status = version.Status
			version = retrievedVersion
			return false, nil
		}
		// Resource was deleted from the api
		if errors.IsNotFound(err) {
			createdVersion, err := m.client.CoreV1alpha1().PropagatedVersions(version.Namespace).Create(version)
			if err != nil {
				return false, err
			}
			createdVersion.Status = version.Status
			version = createdVersion
			return false, nil
		}
		return false, err
	})
	if err != nil {
		return nil, err
	}
	return updatedVersion, nil
}

func (m *propagatedVersionManager) delVersion(key string) {
	m.Lock()
	delete(m.versions, key)
	m.Unlock()
}

func (m *propagatedVersionManager) setVersion(key string, version *fedv1a1.PropagatedVersion) {
	m.Lock()
	m.versions[key] = version
	m.Unlock()
}

func (m *propagatedVersionManager) getVersion(key string) (*fedv1a1.PropagatedVersion, bool) {
	m.RLock()
	version, ok := m.versions[key]
	m.RUnlock()
	return version, ok
}

func (m *propagatedVersionManager) versionName(name string) string {
	kind := m.typeConfig.GetTarget().Kind
	return common.PropagatedVersionName(kind, name)
}

// newVersion initializes a new propagated version resource for the given
// cluster versions and template and override.
func newVersion(clusterVersions map[string]string, templateMeta metav1.Object, targetKind,
	overrideVersion string) *fedv1a1.PropagatedVersion {
	versions := []fedv1a1.ClusterObjectVersion{}
	for clusterName, version := range clusterVersions {
		versions = append(versions, fedv1a1.ClusterObjectVersion{
			ClusterName: clusterName,
			Version:     version,
		})
	}

	util.SortClusterVersions(versions)
	var namespace string
	if targetKind == util.NamespaceKind {
		namespace = templateMeta.GetName()
	} else {
		namespace = templateMeta.GetNamespace()
	}

	return &fedv1a1.PropagatedVersion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      common.PropagatedVersionName(targetKind, templateMeta.GetName()),
		},
		Status: fedv1a1.PropagatedVersionStatus{
			TemplateVersion: templateMeta.GetResourceVersion(),
			OverrideVersion: overrideVersion,
			ClusterVersions: versions,
		},
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

	// Convert map to slice
	versions := []fedv1a1.ClusterObjectVersion{}
	for clusterName, version := range newVersions {
		// Lack of version indicates deletion
		if version == "" {
			continue
		}
		versions = append(versions, fedv1a1.ClusterObjectVersion{
			ClusterName: clusterName,
			Version:     version,
		})
	}

	util.SortClusterVersions(versions)
	return versions
}
