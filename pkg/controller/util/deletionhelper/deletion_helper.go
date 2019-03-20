/*
Copyright 2016 The Kubernetes Authors.

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

// Package to help federation controllers to delete federated resources from
// underlying clusters when the resource is deleted from federation control
// plane.
package deletionhelper

import (
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	meta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	finalizersutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util/finalizers"
)

const (
	// If this finalizer is present on a federated resource, the sync
	// controller will have the opportunity to perform pre-deletion operations
	// (like deleting managed resources from member clusters).
	FinalizerSyncController = "federation.k8s.io/sync-controller"

	// If this annotation is present on a federated resource, resources in the
	// member clusters managed by the federated resource should be orphaned.
	// If the annotation is not present (the default), resources in member
	// clusters will be deleted before the federated resource is deleted.
	OrphanManagedResources = "federation.k8s.io/orphan"
)

type UpdateObjFunc func(runtime.Object) (runtime.Object, error)
type ObjNameFunc func(runtime.Object) string

type DeletionHelper struct {
	updateObjFunc UpdateObjFunc
	objNameFunc   ObjNameFunc
	informer      util.FederatedInformer
	updater       util.FederatedUpdater
}

func NewDeletionHelper(
	updateObjFunc UpdateObjFunc, objNameFunc ObjNameFunc,
	informer util.FederatedInformer, updater util.FederatedUpdater) *DeletionHelper {
	return &DeletionHelper{
		updateObjFunc: updateObjFunc,
		objNameFunc:   objNameFunc,
		informer:      informer,
		updater:       updater,
	}
}

// Ensures that the given object has the FinalizerSyncController finalizer.
// The finalizer ensures that the controller is always notified when a
// federated resource is deleted so that host and member cluster cleanup can be
// performed.
func (dh *DeletionHelper) EnsureFinalizer(obj runtime.Object) (runtime.Object, error) {
	isUpdated, err := finalizersutil.AddFinalizers(obj, sets.NewString(FinalizerSyncController))
	if err != nil || !isUpdated {
		return nil, err
	}
	glog.V(2).Infof("Adding finalizer %s to %s", FinalizerSyncController, dh.objNameFunc(obj))
	// Send the update to apiserver.
	updatedObj, err := dh.updateObjFunc(obj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add finalizer %s to object %s", FinalizerSyncController, dh.objNameFunc(obj))
	}
	return updatedObj, nil
}

// Deletes the resources in member clusters managed by the given federated
// resource unless the OrphanManagedResources annotation is present and has a
// value of 'true'.  Callers are expected to keep calling this (with
// appropriate backoff) until it succeeds.
func (dh *DeletionHelper) HandleObjectInUnderlyingClusters(obj runtime.Object, targetKey string, skipDelete func(runtime.Object) bool) (
	runtime.Object, error) {

	objName := dh.objNameFunc(obj)
	glog.V(2).Infof("Handling deletion of federated dependents for object: %s", objName)

	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return obj, err
	}

	finalizers := sets.NewString(metaAccessor.GetFinalizers()...)
	if !finalizers.Has(FinalizerSyncController) {
		glog.V(2).Infof("obj does not have the %q finalizer. Nothing to do", FinalizerSyncController)
		return obj, nil
	}

	annotations := metaAccessor.GetAnnotations()
	orphanResources := annotations != nil && annotations[OrphanManagedResources] == "true"
	if orphanResources {
		glog.V(2).Infof("Found %q annotation. Nothing to do, just remove the finalizer", OrphanManagedResources)
		// If the obj has the OrphanManagedResources annotation, then we need to
		// orphan the corresponding objects in underlying clusters.  Just
		// remove the finalizer.
		return dh.removeFinalizers(obj, sets.NewString(FinalizerSyncController))
	}

	glog.V(2).Infof("Deleting obj %s from underlying clusters", objName)
	// Else, we need to delete the obj from all underlying clusters.
	unreadyClusters, err := dh.informer.GetUnreadyClusters()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a list of unready clusters")
	}
	// TODO: Handle the case when cluster resource is watched after this is executed.
	// This can happen if a namespace is deleted before its creation had been
	// observed in all underlying clusters.
	clusterNsObjs, err := dh.informer.GetTargetStore().GetFromAllClusters(targetKey)
	glog.V(3).Infof("Found %d objects in underlying clusters", len(clusterNsObjs))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get object %s from underlying clusters", objName)
	}
	operations := make([]util.FederatedOperation, 0)
	for _, clusterNsObj := range clusterNsObjs {
		clusterObj := clusterNsObj.Object.(runtime.Object)
		if skipDelete(clusterObj) {
			continue
		}
		operations = append(operations, util.FederatedOperation{
			Type:        util.OperationTypeDelete,
			ClusterName: clusterNsObj.ClusterName,
			Obj:         clusterObj,
			Key:         targetKey,
		})
	}
	_, operationalErrors := dh.updater.Update(operations)
	if len(operationalErrors) > 0 {
		return nil, errors.Errorf("failed to execute deletions for obj %s: %v", objName, operationalErrors)
	}
	if len(operations) > 0 {
		// We have deleted a bunch of resources.
		// Wait for the store to observe all the deletions.
		var clusterNames []string
		for _, op := range operations {
			clusterNames = append(clusterNames, op.ClusterName)
		}
		return nil, errors.Errorf("waiting for object %s to be deleted from clusters: %s", objName, strings.Join(clusterNames, ", "))
	}

	// We have now deleted the object from all *ready* clusters.
	// But still need to wait for clusters that are not ready to ensure that
	// the object has been deleted from *all* clusters.
	if len(unreadyClusters) != 0 {
		var clusterNames []string
		for _, cluster := range unreadyClusters {
			clusterNames = append(clusterNames, cluster.Name)
		}
		return nil, errors.Errorf("waiting for clusters %s to become ready to verify that obj %s has been deleted", strings.Join(clusterNames, ", "), objName)
	}

	// All done. Just remove the finalizer.
	return dh.removeFinalizers(obj, sets.NewString(FinalizerSyncController))
}

// Removes the given finalizers from the given objects ObjectMeta.
func (dh *DeletionHelper) removeFinalizers(obj runtime.Object, finalizers sets.String) (runtime.Object, error) {
	isUpdated, err := finalizersutil.RemoveFinalizers(obj, finalizers)
	if err != nil || !isUpdated {
		return obj, err
	}
	// Send the update to apiserver.
	updatedObj, err := dh.updateObjFunc(obj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to remove finalizers %v from object %s", finalizers, dh.objNameFunc(obj))
	}
	return updatedObj, nil
}
