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
	"context"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	apiv1 "k8s.io/api/core/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	crcv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

const (
	DefaultFederationSystemNamespace = "federation-system"
	MulticlusterPublicNamespace      = "kube-multicluster-public"
	DefaultClusterAvailableDelay     = 20 * time.Second
	DefaultClusterUnavailableDelay   = 60 * time.Second

	KubeAPIQPS              = 20.0
	KubeAPIBurst            = 30
	KubeconfigSecretDataKey = "kubeconfig"
	getSecretTimeout        = 1 * time.Minute

	DefaultLeaderElectionLeaseDuration = 15 * time.Second
	DefaultLeaderElectionRenewDeadline = 10 * time.Second
	DefaultLeaderElectionRetryPeriod   = 5 * time.Second

	FederationConfigName = "federation-v2"
)

// BuildClusterConfig returns a restclient.Config that can be used to configure
// a client for the given FederatedCluster or an error. The client is used to
// access kubernetes secrets in the federation namespace and cluster-registry
// records in the clusterNamespace.
func BuildClusterConfig(fedCluster *fedv1a1.FederatedCluster, client generic.Client, fedNamespace string, clusterNamespace string) (*restclient.Config, error) {
	clusterName := fedCluster.Name

	// Retrieve the associated cluster
	cluster := &crcv1alpha1.Cluster{}
	err := client.Get(context.TODO(), cluster, clusterNamespace, clusterName)
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
		if cidrnet.Contains(myaddr) {
			serverAddress = item.ServerAddress
			break
		}
	}
	if serverAddress == "" {
		return nil, errors.Errorf("Unable to find address for cluster %s for host ip %s", clusterName, hostIP.String())
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
			return nil, errors.Errorf("found secretRef but no secret name for cluster %s", clusterName)
		}
		secret := &apiv1.Secret{}
		err := client.Get(context.TODO(), secret, fedNamespace, secretRef.Name)
		if err != nil {
			return nil, err
		}

		token, tokenFound := secret.Data["token"]
		ca, caFound := secret.Data["ca.crt"]

		// TODO(font): These changes support both integration (legacy mode) and
		// E2E tests (using service accounts). We cannot use JoinCluster in
		// integration until we have the required RBAC controller(s) e.g. the
		// token controller which observes the service account creation and
		// creates the corresponding secret to allow API access. Until then, we
		// have to rely on the legacy method to allow integration tests to
		// pass.
		if tokenFound != caFound {
			return nil, errors.Errorf("secret should have values for either both 'ca.crt' and 'token' in its Data, or neither: %v", secret)
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
			return nil, errors.Errorf("secret does not have data with key %s", KubeconfigSecretDataKey)
		}
		return clientcmd.Load(data)
	}
}

// IsPrimaryCluster checks if the caller is working with objects for the
// primary cluster by checking if the UIDs match for both ObjectMetas passed
// in.
// TODO (font): Need to revisit this when cluster ID is available.
func IsPrimaryCluster(obj, clusterObj pkgruntime.Object) bool {
	meta := MetaAccessor(obj)
	clusterMeta := MetaAccessor(clusterObj)
	return meta.GetUID() == clusterMeta.GetUID()
}
