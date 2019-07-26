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

package common

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/kubefedcluster"
)

func ValidKubeFedCluster() *v1beta1.KubeFedCluster {
	lastProbeTime := time.Now()
	clusterReady := kubefedcluster.ClusterReady
	healthzOk := kubefedcluster.HealthzOk
	clusterNotReady := kubefedcluster.ClusterNotReady
	healthzNotOk := kubefedcluster.HealthzNotOk
	clusterNotReachableReason := kubefedcluster.ClusterNotReachableReason
	clusterNotReachableMsg := kubefedcluster.ClusterNotReachableMsg
	clusterReachableReason := kubefedcluster.ClusterReachableReason
	clusterReachableMsg := kubefedcluster.ClusterReachableMsg
	region := "us-west1"
	return &v1beta1.KubeFedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "validation-test-cluster",
		},
		Spec: v1beta1.KubeFedClusterSpec{
			APIEndpoint: "https://my.example.com:80/path/to/endpoint",
			SecretRef: v1beta1.LocalSecretReference{
				Name: "validation-unit-test-cluster-pw97k",
			},
		},
		Status: v1beta1.KubeFedClusterStatus{
			Conditions: []v1beta1.ClusterCondition{
				{
					Type:   common.ClusterReady,
					Status: corev1.ConditionTrue,
					LastProbeTime: metav1.Time{
						Time: lastProbeTime,
					},
					LastTransitionTime: &metav1.Time{
						Time: lastProbeTime,
					},
					Reason:  &clusterReady,
					Message: &healthzOk,
				},
				{
					Type:   common.ClusterReady,
					Status: corev1.ConditionFalse,
					LastProbeTime: metav1.Time{
						Time: lastProbeTime,
					},
					LastTransitionTime: &metav1.Time{
						Time: lastProbeTime,
					},
					Reason:  &clusterNotReady,
					Message: &healthzNotOk,
				},
				{
					Type:   common.ClusterOffline,
					Status: corev1.ConditionTrue,
					LastProbeTime: metav1.Time{
						Time: lastProbeTime,
					},
					LastTransitionTime: &metav1.Time{
						Time: lastProbeTime,
					},
					Reason:  &clusterNotReachableReason,
					Message: &clusterNotReachableMsg,
				},
				{
					Type:   common.ClusterOffline,
					Status: corev1.ConditionFalse,
					LastProbeTime: metav1.Time{
						Time: lastProbeTime,
					},
					LastTransitionTime: &metav1.Time{
						Time: lastProbeTime,
					},
					Reason:  &clusterReachableReason,
					Message: &clusterReachableMsg,
				},
			},
			Zones:  []string{"us-west1-a", "us-west1-b"},
			Region: &region,
		},
	}
}
