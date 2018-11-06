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

package util

import (
	"time"

	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"

	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
)

// FederationNamespaces defines the namespace configuration shared by
// most federation controllers.
type FederationNamespaces struct {
	FederationNamespace string
	ClusterNamespace    string
	TargetNamespace     string
}

// ControllerConfig defines the configuration common to federation
// controllers.
type ControllerConfig struct {
	FederationNamespaces
	KubeConfig              *restclient.Config
	ClusterAvailableDelay   time.Duration
	ClusterUnavailableDelay time.Duration
	MinimizeLatency         bool
}

func (c *ControllerConfig) AllClients(userAgent string) (fedclientset.Interface, kubeclientset.Interface, crclientset.Interface) {
	restclient.AddUserAgent(c.KubeConfig, userAgent)
	fedClient := fedclientset.NewForConfigOrDie(c.KubeConfig)
	kubeClient := kubeclientset.NewForConfigOrDie(c.KubeConfig)
	crClient := crclientset.NewForConfigOrDie(c.KubeConfig)
	return fedClient, kubeClient, crClient
}
