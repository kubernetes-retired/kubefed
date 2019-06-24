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

package schedulingtypes

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

const (
	replicasPath = "/spec/replicas"
)

type Plugin struct {
	targetInformer util.FederatedInformer

	federatedStore      cache.Store
	federatedController cache.Controller

	federatedTypeClient util.ResourceClient

	typeConfig typeconfig.Interface

	stopChannel chan struct{}
}

func NewPlugin(controllerConfig *util.ControllerConfig, eventHandlers SchedulerEventHandlers, typeConfig typeconfig.Interface) (*Plugin, error) {
	targetAPIResource := typeConfig.GetTargetType()
	userAgent := fmt.Sprintf("%s-replica-scheduler", strings.ToLower(targetAPIResource.Kind))
	client := genericclient.NewForConfigOrDieWithUserAgent(controllerConfig.KubeConfig, userAgent)

	targetInformer, err := util.NewFederatedInformer(
		controllerConfig,
		client,
		&targetAPIResource,
		eventHandlers.ClusterEventHandler,
		eventHandlers.ClusterLifecycleHandlers,
	)
	if err != nil {
		return nil, err
	}

	p := &Plugin{
		targetInformer: targetInformer,
		typeConfig:     typeConfig,
		stopChannel:    make(chan struct{}),
	}

	targetNamespace := controllerConfig.TargetNamespace
	kubeFedEventHandler := eventHandlers.KubeFedEventHandler

	federatedTypeAPIResource := typeConfig.GetFederatedType()
	p.federatedTypeClient, err = util.NewResourceClient(controllerConfig.KubeConfig, &federatedTypeAPIResource)
	if err != nil {
		return nil, err
	}
	p.federatedStore, p.federatedController = util.NewResourceInformer(p.federatedTypeClient, targetNamespace, kubeFedEventHandler)

	return p, nil
}

func (p *Plugin) Start() {
	p.targetInformer.Start()

	go p.federatedController.Run(p.stopChannel)
}

func (p *Plugin) Stop() {
	p.targetInformer.Stop()
	close(p.stopChannel)
}

func (p *Plugin) HasSynced() bool {
	if !p.targetInformer.ClustersSynced() {
		klog.V(2).Infof("Cluster list not synced")
		return false
	}

	if !p.federatedController.HasSynced() {
		return false
	}

	clusters, err := p.targetInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(errors.Wrap(err, "Failed to get ready clusters"))
		return false
	}

	if !p.targetInformer.GetTargetStore().ClustersSynced(clusters) {
		return false
	}

	return true
}

func (p *Plugin) FederatedTypeExists(key string) bool {
	_, exist, err := p.federatedStore.GetByKey(key)
	if err != nil {
		klog.Errorf("Failed to query store while reconciling RSP controller for key %q: %v", key, err)
		wrappedErr := errors.Wrapf(err, "Failed to query store while reconciling RSP controller for key %q", key)
		runtime.HandleError(wrappedErr)
		return false
	}
	return exist
}

func (p *Plugin) Reconcile(qualifiedName util.QualifiedName, result map[string]int64) error {
	fedObject, err := p.federatedTypeClient.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		// Federated resource has been deleted - no further action required
		return nil
	}
	if err != nil {
		return err
	}

	isDirty := false

	newClusterNames := []string{}
	for name := range result {
		newClusterNames = append(newClusterNames, name)
	}
	clusterNames, err := util.GetClusterNames(fedObject)
	if err != nil {
		return err
	}
	if PlacementUpdateNeeded(clusterNames, newClusterNames) {
		if err := util.SetClusterNames(fedObject, newClusterNames); err != nil {
			return err
		}

		isDirty = true
	}

	overridesMap, err := util.GetOverrides(fedObject)
	if err != nil {
		return errors.Wrapf(err, "Error reading cluster overrides for %s %q", p.typeConfig.GetFederatedType().Kind, qualifiedName)
	}
	if OverrideUpdateNeeded(overridesMap, result) {
		err := setOverrides(fedObject, overridesMap, result)
		if err != nil {
			return err
		}
		isDirty = true
	}

	if isDirty {
		_, err := p.federatedTypeClient.Resources(qualifiedName.Namespace).Update(fedObject, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// These assume that there would be no duplicate clusternames
func PlacementUpdateNeeded(names, newNames []string) bool {
	sort.Strings(names)
	sort.Strings(newNames)
	return !reflect.DeepEqual(names, newNames)
}

func setOverrides(obj *unstructured.Unstructured, overridesMap util.OverridesMap, replicasMap map[string]int64) error {
	if overridesMap == nil {
		overridesMap = make(util.OverridesMap)
	}
	updateOverridesMap(overridesMap, replicasMap)
	return util.SetOverrides(obj, overridesMap)
}

func updateOverridesMap(overridesMap util.OverridesMap, replicasMap map[string]int64) {
	// Remove replicas override for clusters that are not scheduled
	for clusterName, clusterOverrides := range overridesMap {
		if _, ok := replicasMap[clusterName]; !ok {
			for i, overrideItem := range clusterOverrides {
				if overrideItem.Path == replicasPath {
					clusterOverrides = append(clusterOverrides[:i], clusterOverrides[i+1:]...)
					overridesMap[clusterName] = clusterOverrides
					break
				}
			}
		}
	}
	// Add/update replicas override for clusters that are scheduled
	for clusterName, replicas := range replicasMap {
		replicasOverrideFound := false
		for _, overrideItem := range overridesMap[clusterName] {
			if overrideItem.Path == replicasPath {
				overrideItem.Value = replicas
				replicasOverrideFound = true
				break
			}
		}
		if !replicasOverrideFound {
			clusterOverrides, exist := overridesMap[clusterName]
			if !exist {
				clusterOverrides = util.ClusterOverrides{}
			}
			clusterOverrides = append(clusterOverrides, util.ClusterOverride{Path: replicasPath, Value: replicas})
			overridesMap[clusterName] = clusterOverrides
		}
	}
}

func OverrideUpdateNeeded(overridesMap util.OverridesMap, result map[string]int64) bool {
	resultLen := len(result)
	checkLen := 0
	for clusterName, clusterOverridesMap := range overridesMap {
		for _, overrideItem := range clusterOverridesMap {
			path := overrideItem.Path
			rawValue := overrideItem.Value
			if path != replicasPath {
				continue
			}
			// The type of the value will be float64 due to how json
			// marshalling works for interfaces.
			floatValue, ok := rawValue.(float64)
			if !ok {
				return true
			}
			value := int64(floatValue)
			replicas, ok := result[clusterName]
			if !ok || value != int64(replicas) {
				return true
			}
			checkLen += 1
		}
	}

	return checkLen != resultLen
}
