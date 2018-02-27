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

	fedclient "github.com/marun/fnord/pkg/client/clientset_generated/clientset"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crclient "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
)

// FedConfig provides a rest config based on the filesystem kubeconfig (via
// `PathOptions()`) and context in order to talk to the host kubernetes cluster
// and the joining kubernetes cluster
type FedConfig interface {
	PathOptions() *clientcmd.PathOptions
	NewHostConfig(context, kubeconfigPath string) (*rest.Config, error)
	HostConfig() *rest.Config
	NewClusterConfig(context, kubeconfigPath string) (*rest.Config, error)
	ClusterConfig() *rest.Config
	NewFedConfig(context, kubeconfigPath string) (*rest.Config, error)
	FedConfig() *rest.Config
}

// fedConfig implements the FedConfig interface.
type fedConfig struct {
	pathOptions   *clientcmd.PathOptions
	hostConfig    *rest.Config
	clusterConfig *rest.Config
	fedConfig     *rest.Config
}

// NewFedConfig creates a fedConfig for `kubefnord` commands.
func NewFedConfig(pathOptions *clientcmd.PathOptions) FedConfig {
	return &fedConfig{
		pathOptions: pathOptions,
	}
}

// PathOptions provides filesystem based kubeconfig access.
func (a *fedConfig) PathOptions() *clientcmd.PathOptions {
	return a.pathOptions
}

// NewHostConfig provides a rest config to talk to the host kubernetes cluster
// based on the context and kubeconfig passed in.
func (a *fedConfig) NewHostConfig(context, kubeconfigPath string) (*rest.Config, error) {
	hostConfig := a.getClientConfig(context, kubeconfigPath)
	hostClientConfig, err := hostConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	a.hostConfig = hostClientConfig
	return a.hostConfig, nil
}

// HostConfig retrieves the host kubernetes cluster config.
func (a *fedConfig) HostConfig() *rest.Config {
	return a.hostConfig
}

// NewFedConfig provides a rest config to talk to the federation cluster
// based on the context and kubeconfig passed in.
func (a *fedConfig) NewFedConfig(context, kubeconfigPath string) (*rest.Config, error) {
	fedConfig := a.getClientConfig(context, kubeconfigPath)
	fedClientConfig, err := fedConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	a.fedConfig = fedClientConfig
	return a.fedConfig, nil
}

// FedConfig retrieves the host kubernetes cluster config.
func (a *fedConfig) FedConfig() *rest.Config {
	return a.fedConfig
}

// NewClusterConfig provides a rest config to talk to the joining kubernetes
// cluster based on the context and kubeconfig passed in.
func (a *fedConfig) NewClusterConfig(context, kubeconfigPath string) (*rest.Config, error) {
	clusterConfig := a.getClientConfig(context, kubeconfigPath)
	clusterClientConfig, err := clusterConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	a.clusterConfig = clusterClientConfig
	return a.clusterConfig, nil
}

// ClientConfig retrieves the joining kubernetes cluster config.
func (a *fedConfig) ClusterConfig() *rest.Config {
	return a.clusterConfig
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

// FedClientset provides the clients to talk to the kubernetes
// API server, the cluster registry API server, and the federation API server.
type FedClientset interface {
	NewHostClientset(config *rest.Config) (*client.Clientset, error)
	HostClientset() *client.Clientset
	NewClusterClientset(config *rest.Config) (*client.Clientset, error)
	ClusterClientset() *client.Clientset
	NewClusterRegistryClientset(config *rest.Config) (*crclient.Clientset, error)
	ClusterRegistryClientset() *crclient.Clientset
	NewFedClientset(config *rest.Config) (*fedclient.Clientset, error)
	FedClientset() *fedclient.Clientset
}

// fedClientset implements the NewFedClientset interface.
type fedClientset struct {
	hostClientset    *client.Clientset
	clusterClientset *client.Clientset
	crClientset      *crclient.Clientset
	fedClientset     *fedclient.Clientset
}

// NewFedClientset creates a fedClientset for `kubefnord` operations.
func NewFedClientset() FedClientset {
	return &fedClientset{}
}

// NewHostClientset provides a kubernetes API compliant clientset to
// communicate with the host cluster's kubernetes API server.
func (a *fedClientset) NewHostClientset(config *rest.Config) (*client.Clientset, error) {
	a.hostClientset = client.NewForConfigOrDie(config)
	return a.hostClientset, nil
}

// HostClientset retrieves the host cluster's kubernetes API compliant
// clientset.
func (a *fedClientset) HostClientset() *client.Clientset {
	return a.hostClientset
}

// NewClusterClientset provides a kubernetes API compliant clientset to
// communicate with the joining cluster's kubernetes API server.
func (a *fedClientset) NewClusterClientset(config *rest.Config) (*client.Clientset, error) {
	a.clusterClientset = client.NewForConfigOrDie(config)
	return a.clusterClientset, nil
}

// ClusterClientset retrieves the joining cluster's kubernetes API compliant
// clientset.
func (a *fedClientset) ClusterClientset() *client.Clientset {
	return a.clusterClientset
}

// NewClusterRegistryClientset provides a cluster registry API compliant
// clientset to communicate with the cluster registry.
func (a *fedClientset) NewClusterRegistryClientset(config *rest.Config) (*crclient.Clientset, error) {
	a.crClientset = crclient.NewForConfigOrDie(config)
	return a.crClientset, nil
}

// ClusterRegistryClientset retrieves the cluster registry clientset.
func (a *fedClientset) ClusterRegistryClientset() *crclient.Clientset {
	return a.crClientset
}

// NewFedClientset provides a federation API compliant clientset
// to communicate with the federation API server.
func (a *fedClientset) NewFedClientset(config *rest.Config) (*fedclient.Clientset, error) {
	a.fedClientset = fedclient.NewForConfigOrDie(config)
	return a.fedClientset, nil
}

// FedClientset retrieves the federation API compliant clientset.
func (a *fedClientset) FedClientset() *fedclient.Clientset {
	return a.fedClientset
}

// ClusterServiceAccountName returns the name of a service account whose
// credentials are used by the host cluster to access the client cluster.
func ClusterServiceAccountName(joiningClusterName, hostClusterName string) string {
	return fmt.Sprintf("%s-%s", joiningClusterName, hostClusterName)
}

// ClusterRoleName returns the name of a ClusterRole and its associated
// ClusterRoleBinding that are used to allow the service account to
// access necessary resources on the cluster.
func ClusterRoleName(federationName, serviceAccountName string) string {
	return fmt.Sprintf("federation-controller-manager:%s-%s", federationName, serviceAccountName)
}
