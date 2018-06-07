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

package aggregated

import (
	"crypto/x509"
	"fmt"
	"io"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	crclient "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
	"k8s.io/cluster-registry/pkg/crinit/common"
	"k8s.io/cluster-registry/pkg/crinit/util"
	apiregv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	apiregclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

var (
	longInitCommandDescription = `
	Init initializes an aggregated cluster registry.

	The aggregated cluster registry is hosted inside a Kubernetes
	cluster and registers its API with the Kubernetes API aggregator.
	The host cluster must be specified using the --host-cluster-context flag.`
	initCommandExample = `
	# Initialize an aggregated cluster registry named foo
	# in the host cluster whose local kubeconfig
	# context is bar.
	crinit aggregated init foo --host-cluster-context=bar`
)

// newSubCmdInit defines the `init` subcommand to bootstrap a cluster registry
// inside a host Kubernetes cluster.
func newSubCmdInit(cmdOut io.Writer, pathOptions *clientcmd.PathOptions,
	defaultServerImage,
	defaultEtcdImage string) *cobra.Command {
	opts := &aggregatedClusterRegistryOptions{}

	initCmd := &cobra.Command{
		Use:     "init CLUSTER_REGISTRY_NAME --host-cluster-context=HOST_CONTEXT",
		Short:   "Initialize an aggregated cluster registry.",
		Long:    longInitCommandDescription,
		Example: initCommandExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.SetName(args)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			err = validateOptions(opts)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			err = opts.MarshalOptions()
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			hostConfig, err := util.GetClientConfig(pathOptions, opts.Host, opts.Kubeconfig).ClientConfig()
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			hostClientset, err := client.NewForConfig(hostConfig)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			apiServiceClientset, err := apiregclient.NewForConfig(hostConfig)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			err = runInit(opts, cmdOut, hostClientset, apiServiceClientset, pathOptions)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}
		},
	}

	flags := initCmd.Flags()
	opts.BindCommon(flags)
	opts.BindCommonInit(flags, defaultServerImage, defaultEtcdImage)
	opts.Bind(flags)
	return initCmd
}

// validateOptions ensures that options are valid.
func validateOptions(opts *aggregatedClusterRegistryOptions) error {
	opts.APIServerServiceType = v1.ServiceType(opts.apiServerServiceTypeString)
	return opts.ValidateCommonOptions()
}

// runInit initializes a cluster registry.
func runInit(opts *aggregatedClusterRegistryOptions, cmdOut io.Writer,
	hostClientset client.Interface, apiSvcClientset apiregclient.Interface,
	pathOptions *clientcmd.PathOptions) error {

	err := opts.CreateNamespace(cmdOut, hostClientset)
	if err != nil {
		return err
	}

	svc, ips, hostnames, err := opts.CreateService(cmdOut, hostClientset)
	if err != nil {
		return err
	}

	credentials, err := opts.GenerateCredentials(cmdOut, svc.Name, ips, hostnames,
		false, false)
	if err != nil {
		return err
	}

	err = opts.CreateAPIServerCredentialsSecret(hostClientset, credentials)
	if err != nil {
		return err
	}

	pvc, err := opts.CreatePVC(cmdOut, hostClientset, svc.Name)
	if err != nil {
		return err
	}

	sa, err := createRBACObjects(cmdOut, hostClientset, opts)
	if err != nil {
		return err
	}

	err = opts.CreateAPIServer(cmdOut, hostClientset, false, false, true, ips, pvc, sa.Name)
	if err != nil {
		return err
	}

	_, err = createAPIService(cmdOut, apiSvcClientset, opts,
		util.GetCAKeyPair(credentials).Cert)
	if err != nil {
		return err
	}

	err = opts.UpdateKubeconfig(cmdOut, pathOptions, svc, ips, hostnames,
		credentials)
	if err != nil {
		return err
	}

	err = opts.WaitForAPIServer(cmdOut, hostClientset, pathOptions, ips,
		hostnames, svc)
	if err != nil {
		return err
	}

	return waitForAggregator(cmdOut, opts.Host, opts.Kubeconfig, pathOptions)
}

// createRBACObjects handles the creation of all the RBAC objects necessary
// to deploy the cluster registry in aggregated mode.
func createRBACObjects(cmdOut io.Writer, clientset client.Interface,
	opts *aggregatedClusterRegistryOptions) (*v1.ServiceAccount, error) {

	fmt.Fprintf(cmdOut, "Creating RBAC objects...")

	// Create a Kubernetes service account in our namespace.
	glog.V(4).Infof(
		"Creating service account %v for cluster registry apiserver in the host cluster",
		serviceAccountName)

	sa, err := createServiceAccount(clientset, opts.ClusterRegistryNamespace, opts.DryRun)

	if err != nil {
		glog.V(4).Infof("Failed to create service account %v: %v", sa, err)
		return nil, err
	}

	glog.V(4).Info("Successfully created service account")

	// Create a Kubernetes cluster role binding from the default service account
	// in our namespace to the system:auth-delegator cluster role.
	glog.V(4).Infof("Creating cluster role binding %v", authDelegatorCRBName)

	_, err = createAuthDelegatorClusterRoleBinding(clientset, authDelegatorCRBName,
		opts.ClusterRegistryNamespace, opts.DryRun)

	if err != nil {
		glog.V(4).Infof("Failed to create cluster role binding %v: %v", authDelegatorCRBName, err)
		return nil, err
	}

	// Create a role binding to allow the cluster registry service account to
	// access the extension-apiserver-authentication configmap.
	glog.V(4).Infof("Creating role %v for accessing extension-apiserver-authentication ConfigMap", extensionAPIServerRBName)

	_, err = createExtensionAPIServerAuthenticationRoleBinding(clientset, extensionAPIServerRBName, opts.ClusterRegistryNamespace, opts.DryRun)

	if err != nil {
		glog.V(4).Infof("Failed to create extension-apiserver-authentication ConfigMap reader role binding")
		return nil, err
	}

	glog.V(4).Info("Successfully created cluster role bindings")
	fmt.Fprintln(cmdOut, " done")
	return sa, nil
}

// createServiceAccount handles the creation of the service account for
// the cluster registry to be used with RBAC.
func createServiceAccount(clientset client.Interface,
	namespace string, dryRun bool) (*v1.ServiceAccount, error) {

	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
			Labels:    common.ComponentLabel,
		},
	}

	if dryRun {
		return sa, nil
	}

	return clientset.CoreV1().ServiceAccounts(namespace).Create(sa)
}

// createAuthDelegatorClusterRoleBinding creates and returns the cluster role
// binding object to allow the cluster registry to delegate auth to the
// kubernetes API server.
func createAuthDelegatorClusterRoleBinding(clientset client.Interface, name, namespace string,
	dryRun bool) (*rbacv1.ClusterRoleBinding, error) {

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: common.ComponentLabel,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "system:auth-delegator",
		},
	}

	if dryRun {
		return crb, nil
	}

	return clientset.RbacV1().ClusterRoleBindings().Create(crb)
}

// createExtensionAPIServerAuthenticationRoleBinding creates and returns a rolebinding
// object to allow the cluster registry to access the extension-apiserver-authentication
// ConfigMap.
func createExtensionAPIServerAuthenticationRoleBinding(clientset client.Interface, name, namespace string, dryRun bool) (*rbacv1.RoleBinding, error) {
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: common.ComponentLabel,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     "extension-apiserver-authentication-reader",
		},
	}

	if dryRun {
		return rb, nil
	}

	return clientset.RbacV1().RoleBindings("kube-system").Create(rb)
}

// createAPIService creates the Kubernetes API Service to handle the cluster
// registry objects.
func createAPIService(cmdOut io.Writer, clientset apiregclient.Interface,
	opts *aggregatedClusterRegistryOptions,
	ca *x509.Certificate) (*apiregv1beta1.APIService, error) {

	fmt.Fprint(cmdOut, "Creating cluster registry Kubernetes API Service...")
	glog.V(4).Infof("Creating cluster registry Kubernetes API Service %v", apiServiceName)

	caBundle := cert.EncodeCertPEM(ca)

	apiSvc, err := createAPIServiceObject(clientset, opts.Name,
		opts.ClusterRegistryNamespace, opts.DryRun, caBundle)

	if err != nil {
		glog.V(4).Infof("Failed to create cluster registry Kubernetes API Service %v: %v",
			apiSvc, err)
		return nil, err
	}

	fmt.Fprintln(cmdOut, " done")
	glog.V(4).Info("Successfully created cluster registry Kubernetes API Service")

	return apiSvc, nil
}

// createAPIServiceObject creates and returns the cluster registry API Service
// object.
func createAPIServiceObject(clientset apiregclient.Interface,
	clusterRegistryName, namespace string, dryRun bool,
	caBundle []byte) (*apiregv1beta1.APIService, error) {

	apiSvc := &apiregv1beta1.APIService{
		ObjectMeta: metav1.ObjectMeta{
			Name:   apiServiceName,
			Labels: common.ComponentLabel,
		},
		Spec: apiregv1beta1.APIServiceSpec{
			Service: &apiregv1beta1.ServiceReference{
				Namespace: namespace,
				Name:      clusterRegistryName,
			},
			Group:                v1alpha1.GroupName,
			Version:              v1alpha1.SchemeGroupVersion.Version,
			CABundle:             caBundle,
			GroupPriorityMinimum: apiServiceGroupPriorityMinimum,
			VersionPriority:      apiServiceVersionPriority,
		},
	}

	if dryRun {
		return apiSvc, nil
	}

	return clientset.ApiregistrationV1beta1().APIServices().Create(apiSvc)
}

// waitForAggregator waits for the aggregated API server that is aggregating the
// cluster registry to be successfully serving clusters. Returns an error if the
// aggregator is not serving clusters after some time.
func waitForAggregator(cmdOut io.Writer, host, kubeconfig string,
	pathOptions *clientcmd.PathOptions) error {
	fmt.Fprint(cmdOut, "Waiting for the cluster registry API to be available via the aggregator...")
	glog.V(4).Info("Waiting for the cluster registry API to be available from the aggregator")

	hostConfig, err := util.GetClientConfig(pathOptions, host, kubeconfig).ClientConfig()
	if err != nil {
		return err
	}

	crClientset, err := crclient.NewForConfig(hostConfig)
	if err != nil {
		return err
	}

	var listErr error
	err = wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
		fmt.Fprint(cmdOut, ".")
		_, listErr = crClientset.ClusterregistryV1alpha1().Clusters().List(metav1.ListOptions{})
		if listErr != nil {
			return false, nil
		}
		return true, nil
	})

	// The last list error received is more relevant to the caller than the fact
	// that the timeout was hit.
	if err != nil {
		return listErr
	}

	fmt.Fprintln(cmdOut, " done")
	glog.V(4).Info("Successfully listed clusters from the aggregated API server.")

	return nil
}
