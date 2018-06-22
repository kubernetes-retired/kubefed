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
	"fmt"

	fedclient "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crclient "k8s.io/cluster-registry/pkg/client/clientset/versioned"
)

// FedConfig provides a rest config based on the filesystem kubeconfig (via
// pathOptions) and context in order to talk to the host kubernetes cluster
// and the joining kubernetes cluster.
type FedConfig interface {
	HostConfig(context, kubeconfigPath string) (*rest.Config, error)
	ClusterConfig(context, kubeconfigPath string) (*rest.Config, error)
}

// fedConfig implements the FedConfig interface.
type fedConfig struct {
	pathOptions *clientcmd.PathOptions
}

// NewFedConfig creates a fedConfig for `kubefnord` commands.
func NewFedConfig(pathOptions *clientcmd.PathOptions) FedConfig {
	return &fedConfig{
		pathOptions: pathOptions,
	}
}

// HostConfig provides a rest config to talk to the host kubernetes cluster
// based on the context and kubeconfig passed in.
func (a *fedConfig) HostConfig(context, kubeconfigPath string) (*rest.Config, error) {
	hostConfig := a.getClientConfig(context, kubeconfigPath)
	hostClientConfig, err := hostConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return hostClientConfig, nil
}

// ClusterConfig provides a rest config to talk to the joining kubernetes
// cluster based on the context and kubeconfig passed in.
func (a *fedConfig) ClusterConfig(context, kubeconfigPath string) (*rest.Config, error) {
	clusterConfig := a.getClientConfig(context, kubeconfigPath)
	clusterClientConfig, err := clusterConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return clusterClientConfig, nil
}

// getClientConfig is a helper method to create a client config from the
// context and kubeconfig passed as arguments.
func (a *fedConfig) getClientConfig(context, kubeconfigPath string) clientcmd.ClientConfig {
	loadingRules := *a.pathOptions.LoadingRules
	loadingRules.Precedence = a.pathOptions.GetLoadingPrecedence()
	loadingRules.ExplicitPath = kubeconfigPath
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: context,
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&loadingRules, overrides)
}

// HostClientset provides a kubernetes API compliant clientset to
// communicate with the host cluster's kubernetes API server.
func HostClientset(config *rest.Config) (*client.Clientset, error) {
	return client.NewForConfigOrDie(config), nil
}

// ClusterClientset provides a kubernetes API compliant clientset to
// communicate with the joining cluster's kubernetes API server.
func ClusterClientset(config *rest.Config) (*client.Clientset, error) {
	return client.NewForConfigOrDie(config), nil
}

// ClusterRegistryClientset provides a cluster registry API compliant
// clientset to communicate with the cluster registry.
func ClusterRegistryClientset(config *rest.Config) (*crclient.Clientset, error) {
	return crclient.NewForConfigOrDie(config), nil
}

// FedClientset provides a federation API compliant clientset
// to communicate with the federation API server.
func FedClientset(config *rest.Config) (*fedclient.Clientset, error) {
	return fedclient.NewForConfigOrDie(config), nil
}

// ClusterServiceAccountName returns the name of a service account whose
// credentials are used by the host cluster to access the client cluster.
func ClusterServiceAccountName(joiningClusterName, hostClusterName string) string {
	return fmt.Sprintf("%s-%s", joiningClusterName, hostClusterName)
}

// ClusterRoleName returns the name of a ClusterRole and its associated
// ClusterRoleBinding that are used to allow the service account to
// access necessary resources on the cluster.
func ClusterRoleName(serviceAccountName string) string {
	return fmt.Sprintf("federation-controller-manager:%s", serviceAccountName)
}
