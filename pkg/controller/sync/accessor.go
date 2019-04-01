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
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/version"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/deletionhelper"
)

// FederatedResourceAccessor provides a way to retrieve and visit
// logical federated resources (e.g. FederatedConfigMap)
type FederatedResourceAccessor interface {
	Run(stopChan <-chan struct{})
	HasSynced() bool
	FederatedResource(qualifiedName util.QualifiedName) (FederatedResource, error)
	VisitFederatedResources(visitFunc func(obj interface{}))
}

type resourceAccessor struct {
	limitedScope      bool
	typeConfig        typeconfig.Interface
	targetIsNamespace bool
	fedNamespace      string

	// The informer for the federated type.
	federatedStore      cache.Store
	federatedController cache.Controller

	// The informer used to source namespaces for templates of
	// federated namespaces.  Will only be initialized if
	// targetIsNamespace=true.
	namespaceStore      cache.Store
	namespaceController cache.Controller

	fedNamespaceAPIResource *metav1.APIResource

	// The informer used to source federated namespaces used in
	// determining placement for namespaced resources.  Will only be
	// initialized if the target resource is namespaced.
	fedNamespaceStore      cache.Store
	fedNamespaceController cache.Controller

	// Manages propagated versions
	versionManager *version.VersionManager

	// Adds finalizers to resources and performs cleanup of target resources.
	deletionHelper *deletionhelper.DeletionHelper

	// Records events on the federated resource
	eventRecorder record.EventRecorder
}

func NewFederatedResourceAccessor(
	controllerConfig *util.ControllerConfig,
	typeConfig typeconfig.Interface,
	fedNamespaceAPIResource *metav1.APIResource,
	client genericclient.Client,
	enqueueObj func(pkgruntime.Object),
	informer util.FederatedInformer,
	updater util.FederatedUpdater,
	eventRecorder record.EventRecorder) (FederatedResourceAccessor, error) {

	a := &resourceAccessor{
		limitedScope:            controllerConfig.LimitedScope(),
		typeConfig:              typeConfig,
		targetIsNamespace:       typeConfig.GetTarget().Kind == util.NamespaceKind,
		fedNamespace:            controllerConfig.FederationNamespace,
		fedNamespaceAPIResource: fedNamespaceAPIResource,
		eventRecorder:           eventRecorder,
	}

	targetNamespace := controllerConfig.TargetNamespace

	federatedTypeAPIResource := typeConfig.GetFederatedType()
	federatedTypeClient, err := util.NewResourceClient(controllerConfig.KubeConfig, &federatedTypeAPIResource)
	if err != nil {
		return nil, err
	}
	a.federatedStore, a.federatedController = util.NewResourceInformer(federatedTypeClient, targetNamespace, enqueueObj)

	if a.targetIsNamespace {
		// Initialize an informer for namespaces.  The namespace
		// containing a federated namespace resource is used as the
		// template for target resources in member clusters.
		namespaceAPIResource := typeConfig.GetTarget()
		namespaceTypeClient, err := util.NewResourceClient(controllerConfig.KubeConfig, &namespaceAPIResource)
		if err != nil {
			return nil, err
		}
		a.namespaceStore, a.namespaceController = util.NewResourceInformer(namespaceTypeClient, targetNamespace, enqueueObj)
	}

	if typeConfig.GetNamespaced() {
		fedNamespaceEnqueue := func(fedNamespaceObj pkgruntime.Object) {
			// When a federated namespace changes, every resource in
			// the namespace needs to be reconciled.
			//
			// TODO(marun) Consider optimizing this to only reconcile
			// contained resources in response to a change in
			// placement for the federated namespace.
			namespace := util.NewQualifiedName(fedNamespaceObj).Namespace
			for _, rawObj := range a.federatedStore.List() {
				obj := rawObj.(pkgruntime.Object)
				qualifiedName := util.NewQualifiedName(obj)
				if qualifiedName.Namespace == namespace {
					enqueueObj(obj)
				}
			}
		}
		// Initialize an informer for federated namespaces.  Placement
		// for a resource is computed as the intersection of resource
		// and federated namespace placement.
		fedNamespaceClient, err := util.NewResourceClient(controllerConfig.KubeConfig, fedNamespaceAPIResource)
		if err != nil {
			return nil, err
		}
		a.fedNamespaceStore, a.fedNamespaceController = util.NewResourceInformer(fedNamespaceClient, targetNamespace, fedNamespaceEnqueue)
	}

	a.versionManager = version.NewVersionManager(
		client,
		typeConfig.GetFederatedNamespaced(),
		typeConfig.GetFederatedType().Kind,
		typeConfig.GetTarget().Kind,
		targetNamespace,
	)

	a.deletionHelper = deletionhelper.NewDeletionHelper(
		func(rawObj pkgruntime.Object) (pkgruntime.Object, error) {
			obj := rawObj.(*unstructured.Unstructured)
			return federatedTypeClient.Resources(obj.GetNamespace()).Update(obj, metav1.UpdateOptions{})
		},
		func(obj pkgruntime.Object) string {
			return util.NewQualifiedName(obj).String()
		},
		informer,
		updater,
	)

	return a, nil
}

func (a *resourceAccessor) Run(stopChan <-chan struct{}) {
	go a.versionManager.Sync(stopChan)
	go a.federatedController.Run(stopChan)
	if a.namespaceController != nil {
		go a.namespaceController.Run(stopChan)
	}
	if a.fedNamespaceController != nil {
		go a.fedNamespaceController.Run(stopChan)
	}
}

func (a *resourceAccessor) HasSynced() bool {
	kind := a.typeConfig.GetFederatedType().Kind
	if !a.versionManager.HasSynced() {
		glog.V(2).Infof("Version manager for %s not synced", kind)
		return false
	}
	if !a.federatedController.HasSynced() {
		glog.V(2).Infof("Informer for %s not synced", kind)
		return false
	}
	if a.namespaceController != nil && !a.namespaceController.HasSynced() {
		glog.V(2).Infof("Namespace informer for %s not synced", kind)
		return false
	}
	if a.fedNamespaceController != nil && !a.fedNamespaceController.HasSynced() {
		glog.V(2).Infof("FederatedNamespace informer for %s not synced", kind)
		return false
	}
	return true
}

func (a *resourceAccessor) FederatedResource(eventSource util.QualifiedName) (FederatedResource, error) {
	if a.targetIsNamespace && a.isSystemNamespace(eventSource.Name) {
		glog.V(7).Infof("Ignoring system namespace %q", eventSource.Name)
		return nil, nil
	}

	kind := a.typeConfig.GetFederatedType().Kind

	// Most federated resources have the same name as their targets.
	targetName := eventSource
	federatedName := util.QualifiedName{
		Namespace: eventSource.Namespace,
		Name:      eventSource.Name,
	}

	// A federated type for namespace "foo" is namespaced
	// (e.g. "foo/foo"). An event sourced from a namespace in the host
	// or member clusters will have the name "foo", and an event
	// sourced from a federated resource will have the name "foo/foo".
	// In order to ensure object retrieval from the informers, it is
	// necessary to derive the target name and federated name from the
	// event source.
	if a.targetIsNamespace {
		eventSourceIsTarget := eventSource.Namespace == ""
		if eventSourceIsTarget {
			// Ensure the federated name is namespace qualified.
			federatedName.Namespace = federatedName.Name
		} else {
			if federatedName.Namespace != federatedName.Name {
				// A FederatedNamespace is only valid for propagation
				// if it has the same name as the containing namespace.
				return nil, nil
			}
			// Ensure the target name is not namespace qualified.
			targetName.Namespace = ""
		}
	}

	key := federatedName.String()

	resource, err := util.ObjFromCache(a.federatedStore, kind, key)
	if err != nil {
		return nil, err
	}
	if resource == nil {
		// The event source may be a federated resource that was
		// deleted, but is more likely a non-federated resource in the
		// target cluster.
		glog.V(7).Infof("%s %q was not found which indicates that the %s is not federated",
			kind, key, a.typeConfig.GetTarget().Kind)
		return nil, nil
	}

	var namespace *unstructured.Unstructured
	if a.targetIsNamespace {
		namespace, err = util.ObjFromCache(a.namespaceStore, a.typeConfig.GetTarget().Kind, targetName.String())
		if err != nil {
			return nil, err
		}
		if namespace == nil {
			// The namespace containing the FederatedNamespace was deleted.
			return nil, nil
		}
	}

	var fedNamespace *unstructured.Unstructured
	if a.typeConfig.GetNamespaced() {
		fedNamespaceName := util.QualifiedName{Namespace: targetName.Namespace, Name: targetName.Namespace}
		fedNamespace, err = util.ObjFromCache(a.fedNamespaceStore, a.fedNamespaceAPIResource.Kind, fedNamespaceName.String())
		if err != nil {
			return nil, err
		}
		// If fedNamespace is nil, the resources in member clusters
		// will be removed.
	}

	return &federatedResource{
		limitedScope:      a.limitedScope,
		typeConfig:        a.typeConfig,
		targetIsNamespace: a.targetIsNamespace,
		targetName:        targetName,
		federatedKind:     kind,
		federatedName:     federatedName,
		federatedResource: resource,
		versionManager:    a.versionManager,
		deletionHelper:    a.deletionHelper,
		namespace:         namespace,
		fedNamespace:      fedNamespace,
		eventRecorder:     a.eventRecorder,
	}, nil
}

func (a *resourceAccessor) VisitFederatedResources(visitFunc func(obj interface{})) {
	for _, obj := range a.federatedStore.List() {
		visitFunc(obj)
	}
}

func (a *resourceAccessor) isSystemNamespace(namespace string) bool {
	// TODO(font): Need a configurable or discoverable list of namespaces
	// to not propagate beyond just the default system namespaces e.g.
	switch namespace {
	case "kube-system", "kube-public", "default", a.fedNamespace:
		return true
	default:
		return false
	}
}
