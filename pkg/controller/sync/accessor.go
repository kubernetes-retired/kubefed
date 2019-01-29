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

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/placement"
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
	typeConfig        typeconfig.Interface
	targetIsNamespace bool
	fedNamespace      string

	// Store for the templates of the federated type
	templateStore cache.Store
	// Informer for the templates of the federated type
	templateController cache.Controller

	// Store for the override directives of the federated type
	overrideStore cache.Store
	// Informer controller for override directives of the federated type
	overrideController cache.Controller

	placementPlugin placement.PlacementPlugin

	// Manages propagated versions
	versionManager *version.VersionManager

	// Adds finalizers to resources and performs cleanup of target resources.
	deletionHelper *deletionhelper.DeletionHelper
}

func NewFederatedResourceAccessor(
	controllerConfig *util.ControllerConfig,
	typeConfig typeconfig.Interface,
	namespacePlacementAPIResource *metav1.APIResource,
	fedClient fedclientset.Interface,
	enqueueObj func(pkgruntime.Object),
	informer util.FederatedInformer,
	updater util.FederatedUpdater) (FederatedResourceAccessor, error) {

	a := &resourceAccessor{
		typeConfig:        typeConfig,
		targetIsNamespace: typeConfig.GetTarget().Kind == util.NamespaceKind,
		fedNamespace:      controllerConfig.FederationNamespace,
	}

	targetNamespace := controllerConfig.TargetNamespace

	// Start informers on the resources for the federated type
	templateAPIResource := typeConfig.GetTemplate()
	templateClient, err := util.NewResourceClient(controllerConfig.KubeConfig, &templateAPIResource)
	if err != nil {
		return nil, err
	}
	a.templateStore, a.templateController = util.NewResourceInformer(templateClient, targetNamespace, enqueueObj)

	overrideAPIResource := typeConfig.GetOverride()
	overrideClient, err := util.NewResourceClient(controllerConfig.KubeConfig, &overrideAPIResource)
	if err != nil {
		return nil, err
	}
	a.overrideStore, a.overrideController = util.NewResourceInformer(overrideClient, targetNamespace, enqueueObj)

	placementAPIResource := typeConfig.GetPlacement()
	placementClient, err := util.NewResourceClient(controllerConfig.KubeConfig, &placementAPIResource)
	if err != nil {
		return nil, err
	}
	if typeConfig.GetNamespaced() {
		namespacePlacementClient, err := util.NewResourceClient(controllerConfig.KubeConfig, namespacePlacementAPIResource)
		if err != nil {
			return nil, err
		}
		namespacePlacementEnqueue := func(placementObj pkgruntime.Object) {
			// When namespace placement changes, every resource in the
			// namespace needs to be reconciled.
			placementNamespace := util.NewQualifiedName(placementObj).Namespace
			for _, templateObj := range a.templateStore.List() {
				template := templateObj.(pkgruntime.Object)
				qualifiedName := util.NewQualifiedName(template)
				if qualifiedName.Namespace == placementNamespace {
					enqueueObj(template)
				}
			}
		}

		a.placementPlugin = placement.NewNamespacedPlacementPlugin(placementClient, namespacePlacementClient, targetNamespace, enqueueObj, namespacePlacementEnqueue)
	} else {
		a.placementPlugin = placement.NewResourcePlacementPlugin(placementClient, targetNamespace, enqueueObj)
	}

	a.versionManager = version.NewVersionManager(
		fedClient,
		typeConfig.GetFederatedNamespaced(),
		typeConfig.GetFederatedKind(),
		typeConfig.GetTarget().Kind,
		targetNamespace,
	)

	// Most types apply the finalizer to the template.
	finalizerClient := templateClient
	if a.targetIsNamespace {
		// For namespaces the finalizer is applied to the placement to
		// ensure that deletion of namespaces in member clusters only
		// occurs after a namespace has been configured to be propagated
		// to those clusters.
		finalizerClient = placementClient
	}

	a.deletionHelper = deletionhelper.NewDeletionHelper(
		func(rawObj pkgruntime.Object) (pkgruntime.Object, error) {
			obj := rawObj.(*unstructured.Unstructured)
			return finalizerClient.Resources(obj.GetNamespace()).Update(obj, metav1.UpdateOptions{})
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
	go a.templateController.Run(stopChan)
	go a.overrideController.Run(stopChan)
	go a.placementPlugin.Run(stopChan)
}

func (a *resourceAccessor) HasSynced() bool {
	if !a.versionManager.HasSynced() {
		glog.V(2).Infof("Version manager not synced")
		return false
	}
	if !a.templateController.HasSynced() {
		glog.V(2).Infof("Templates not synced")
		return false
	}
	if !a.overrideController.HasSynced() {
		glog.V(2).Infof("Overrides not synced")
		return false
	}
	if !a.placementPlugin.HasSynced() {
		glog.V(2).Infof("Placements not synced")
		return false
	}
	return true
}

func (a *resourceAccessor) FederatedResource(eventSource util.QualifiedName) (FederatedResource, error) {
	if a.targetIsNamespace && a.isSystemNamespace(eventSource.Name) {
		glog.V(7).Infof("Ignoring system namespace %q", eventSource.Name)
		return nil, nil
	}

	// Most federated resources have the same name as their targets.
	targetName := eventSource
	federatedName := util.QualifiedName{
		Namespace: eventSource.Namespace,
		Name:      eventSource.Name,
	}
	templateKey := federatedName.String()

	// If the target type is namespace, the placement resource must be
	// present and will be used as the finalization target.
	var placement *unstructured.Unstructured

	// A federated primitive for namespace "foo" is namespaced
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
			// Ensure the target name is not namespace qualified.
			targetName.Namespace = ""
		}

		// A namespace is only federated if it has a corresponding
		// placement resource.
		var err error
		placement, err = a.placementPlugin.GetPlacement(federatedName.String())
		if err != nil {
			return nil, err
		}
		if placement == nil {
			// No propagation without placement, and finalization
			// ensures that the absence of placement indicates removal
			// of target resources from member clusters.
			glog.V(7).Infof("%s %q was not found which indicates the namespace is not federated",
				a.typeConfig.GetPlacement().Kind, federatedName)
			return nil, nil
		}

		// The template for a federated namespace is the namespace.
		templateKey = targetName.String()
	}

	templateKind := a.typeConfig.GetTemplate().Kind
	template, err := util.ObjFromCache(a.templateStore, templateKind, templateKey)
	if err != nil {
		return nil, err
	}
	if template == nil {
		// Without a template, a resource is not federated.  The event
		// source may be an override or placement, but is more likely
		// to be a non-federated resource in the target cluster.
		glog.V(7).Infof("%s %q was not found which indicates that the %s is not federated",
			templateKind, templateKey, a.typeConfig.GetTarget().Kind)
		return nil, nil
	}

	return &federatedResource{
		typeConfig:        a.typeConfig,
		targetIsNamespace: a.targetIsNamespace,
		targetName:        targetName,
		federatedName:     federatedName,
		template:          template,
		placement:         placement,
		placementPlugin:   a.placementPlugin,
		versionManager:    a.versionManager,
		deletionHelper:    a.deletionHelper,
		// Overrides are loaded lazily to ensure that deletion can
		// handled by the controller first.
		overrideStore: a.overrideStore,
	}, nil
}

func (a *resourceAccessor) VisitFederatedResources(visitFunc func(obj interface{})) {
	for _, obj := range a.templateStore.List() {
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
