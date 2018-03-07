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

package util

import (
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
)

const (
	// TODO(marun) this should be discovered rather than hard-coded
	FederationSystemNamespace = "federation"
	KubeAPIQPS                = 20.0
	KubeAPIBurst              = 30
	KubeconfigSecretDataKey   = "kubeconfig"
	getSecretTimeout          = 1 * time.Minute
)

func BuildClusterConfig(fedCluster *fedv1a1.FederatedCluster, kubeClient kubeclientset.Interface, crClient crclientset.Interface) (*restclient.Config, error) {
	clusterName := fedCluster.Name

	// Retrieve the associated cluster
	cluster, err := crClient.ClusterregistryV1alpha1().Clusters().Get(clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var serverAddress string
	var clusterConfig *restclient.Config
	hostIP, err := utilnet.ChooseHostInterface()
	if err != nil {
		return nil, err
	}

	// Determine the server address
	for _, item := range cluster.Spec.KubernetesAPIEndpoints.ServerEndpoints {
		_, cidrnet, err := net.ParseCIDR(item.ClientCIDR)
		if err != nil {
			return nil, err
		}
		myaddr := net.ParseIP(hostIP.String())
		if cidrnet.Contains(myaddr) == true {
			serverAddress = item.ServerAddress
			break
		}
	}
	if serverAddress == "" {
		return nil, fmt.Errorf("Unable to find address for cluster %s for host ip %s", clusterName, hostIP.String())
	}

	secretRef := fedCluster.Spec.SecretRef

	if secretRef == nil {
		glog.Infof("didn't find secretRef for cluster %s. Trying insecure access", clusterName)
		clusterConfig, err = clientcmd.BuildConfigFromFlags(serverAddress, "")
		if err != nil {
			return nil, err
		}
	} else {
		if secretRef.Name == "" {
			return nil, fmt.Errorf("found secretRef but no secret name for cluster %s", clusterName)
		}
		secret, err := kubeClient.CoreV1().Secrets(FederationSystemNamespace).Get(secretRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		token, tokenFound := secret.Data["token"]
		ca, caFound := secret.Data["ca.crt"]

		if tokenFound != caFound {
			return nil, fmt.Errorf("secret should have values for either both 'ca.crt' and 'token' in its Data, or neither: %v", secret)
		} else if tokenFound && caFound {
			clusterConfig, err = clientcmd.BuildConfigFromFlags(serverAddress, "")
			clusterConfig.CAData = ca
			clusterConfig.BearerToken = string(token)
		} else {
			kubeconfigGetter := KubeconfigGetterForSecret(secret)
			clusterConfig, err = clientcmd.BuildConfigFromKubeconfigGetter(serverAddress, kubeconfigGetter)
		}

		if err != nil {
			return nil, err
		}
	}

	clusterConfig.QPS = KubeAPIQPS
	clusterConfig.Burst = KubeAPIBurst

	return clusterConfig, nil
}

// KubeconfigGetterForSecret gets the kubeconfig from the given secret.
// This is to inject a different KubeconfigGetter in tests. We don't use
// the standard one which calls NewInCluster in tests to avoid having to
// set up service accounts and mount files with secret tokens.
var KubeconfigGetterForSecret = func(secret *apiv1.Secret) clientcmd.KubeconfigGetter {
	return func() (*clientcmdapi.Config, error) {
		data, ok := secret.Data[KubeconfigSecretDataKey]
		if !ok {
			return nil, fmt.Errorf("secret does not have data with key %s", KubeconfigSecretDataKey)
		}
		return clientcmd.Load(data)
	}
}
