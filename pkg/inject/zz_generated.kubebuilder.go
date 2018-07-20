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
package inject

import (
	corev1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	multiclusterdnsv1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	schedulingv1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
	rscheme "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/scheme"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/dnsendpoint"
	"github.com/kubernetes-sigs/federation-v2/pkg/inject/args"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	rscheme.AddToScheme(scheme.Scheme)

	// Inject Informers
	Inject = append(Inject, func(arguments args.InjectArgs) error {
		Injector.ControllerManager = arguments.ControllerManager

		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedCluster{}, arguments.Informers.Core().V1alpha1().FederatedClusters()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedConfigMap{}, arguments.Informers.Core().V1alpha1().FederatedConfigMaps()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedConfigMapOverride{}, arguments.Informers.Core().V1alpha1().FederatedConfigMapOverrides()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedConfigMapPlacement{}, arguments.Informers.Core().V1alpha1().FederatedConfigMapPlacements()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedDeployment{}, arguments.Informers.Core().V1alpha1().FederatedDeployments()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedDeploymentOverride{}, arguments.Informers.Core().V1alpha1().FederatedDeploymentOverrides()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedDeploymentPlacement{}, arguments.Informers.Core().V1alpha1().FederatedDeploymentPlacements()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedJob{}, arguments.Informers.Core().V1alpha1().FederatedJobs()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedJobOverride{}, arguments.Informers.Core().V1alpha1().FederatedJobOverrides()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedJobPlacement{}, arguments.Informers.Core().V1alpha1().FederatedJobPlacements()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedNamespacePlacement{}, arguments.Informers.Core().V1alpha1().FederatedNamespacePlacements()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedReplicaSet{}, arguments.Informers.Core().V1alpha1().FederatedReplicaSets()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedReplicaSetOverride{}, arguments.Informers.Core().V1alpha1().FederatedReplicaSetOverrides()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedReplicaSetPlacement{}, arguments.Informers.Core().V1alpha1().FederatedReplicaSetPlacements()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedSecret{}, arguments.Informers.Core().V1alpha1().FederatedSecrets()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedSecretOverride{}, arguments.Informers.Core().V1alpha1().FederatedSecretOverrides()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedSecretPlacement{}, arguments.Informers.Core().V1alpha1().FederatedSecretPlacements()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedService{}, arguments.Informers.Core().V1alpha1().FederatedServices()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedServicePlacement{}, arguments.Informers.Core().V1alpha1().FederatedServicePlacements()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.FederatedTypeConfig{}, arguments.Informers.Core().V1alpha1().FederatedTypeConfigs()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1alpha1.PropagatedVersion{}, arguments.Informers.Core().V1alpha1().PropagatedVersions()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&multiclusterdnsv1alpha1.DNSEndpoint{}, arguments.Informers.Multiclusterdns().V1alpha1().DNSEndpoints()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&multiclusterdnsv1alpha1.MultiClusterServiceDNSRecord{}, arguments.Informers.Multiclusterdns().V1alpha1().MultiClusterServiceDNSRecords()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&schedulingv1alpha1.ReplicaSchedulingPreference{}, arguments.Informers.Scheduling().V1alpha1().ReplicaSchedulingPreferences()); err != nil {
			return err
		}

		// Add Kubernetes informers

		if c, err := dnsendpoint.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		return nil
	})

	// Inject CRDs
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedClusterCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedConfigMapCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedConfigMapOverrideCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedConfigMapPlacementCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedDeploymentCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedDeploymentOverrideCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedDeploymentPlacementCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedJobCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedJobOverrideCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedJobPlacementCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedNamespacePlacementCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedReplicaSetCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedReplicaSetOverrideCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedReplicaSetPlacementCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedSecretCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedSecretOverrideCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedSecretPlacementCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedServiceCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedServicePlacementCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.FederatedTypeConfigCRD)
	Injector.CRDs = append(Injector.CRDs, &corev1alpha1.PropagatedVersionCRD)
	Injector.CRDs = append(Injector.CRDs, &multiclusterdnsv1alpha1.DNSEndpointCRD)
	Injector.CRDs = append(Injector.CRDs, &multiclusterdnsv1alpha1.MultiClusterServiceDNSRecordCRD)
	Injector.CRDs = append(Injector.CRDs, &schedulingv1alpha1.ReplicaSchedulingPreferenceCRD)
	// Inject PolicyRules
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{"core.federation.k8s.io"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{"multiclusterdns.federation.k8s.io"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{"scheduling.federation.k8s.io"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	})
	// Inject GroupVersions
	Injector.GroupVersions = append(Injector.GroupVersions, schema.GroupVersion{
		Group:   "core.federation.k8s.io",
		Version: "v1alpha1",
	})
	Injector.GroupVersions = append(Injector.GroupVersions, schema.GroupVersion{
		Group:   "multiclusterdns.federation.k8s.io",
		Version: "v1alpha1",
	})
	Injector.GroupVersions = append(Injector.GroupVersions, schema.GroupVersion{
		Group:   "scheduling.federation.k8s.io",
		Version: "v1alpha1",
	})
	Injector.RunFns = append(Injector.RunFns, func(arguments run.RunArguments) error {
		Injector.ControllerManager.RunInformersAndControllers(arguments)
		return nil
	})
}
