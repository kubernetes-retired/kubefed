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
	"fmt"
	"io"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/cluster-registry/pkg/crinit/common"
	"k8s.io/cluster-registry/pkg/crinit/util"
	apiregclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

var (
	longDeleteCommandDescription = `
	Delete deletes an aggregated cluster registry.

	The aggregated cluster registry is hosted inside a Kubernetes
	cluster and has its API registered with the Kubernetes API aggregator.
	The host cluster must be specified using the --host-cluster-context flag.`
	deleteCommandExample = `
	# Delete an aggregated cluster registry named foo
	# in the host cluster whose local kubeconfig
	# context is bar.
	crinit aggregated delete foo --host-cluster-context=bar`
)

// newSubCmdDelete defines the `delete` subcommand to remove a cluster registry
// inside a host Kubernetes cluster.
func newSubCmdDelete(cmdOut io.Writer, pathOptions *clientcmd.PathOptions) *cobra.Command {
	opts := &aggregatedClusterRegistryOptions{}

	delCmd := &cobra.Command{
		Use:     "delete CLUSTER_REGISTRY_NAME --host-cluster-context=HOST_CONTEXT",
		Short:   "Delete an aggregated cluster registry.",
		Long:    longDeleteCommandDescription,
		Example: deleteCommandExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.SetName(args)
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

			err = runDelete(opts, cmdOut, hostClientset, apiServiceClientset, pathOptions)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}
		},
	}

	flags := delCmd.Flags()
	opts.BindCommon(flags)
	opts.BindCommonDelete(flags)
	return delCmd
}

// runDelete deletes a cluster registry.
func runDelete(opts *aggregatedClusterRegistryOptions, cmdOut io.Writer,
	hostClientset client.Interface, apiSvcClientset apiregclient.Interface,
	pathOptions *clientcmd.PathOptions) error {

	// Only necessary to delete the cluster registry namespace, the cluster
	// registry API service, the RBAC objects that are not in the cluster
	// registry namespace (those that are not removed by removing the
	// namespace), and the kubeconfig entry.

	err := common.DeleteKubeconfigEntry(cmdOut, pathOptions, opts.Name,
		opts.Kubeconfig, opts.DryRun, opts.IgnoreErrors)
	if err != nil {
		if !opts.IgnoreErrors {
			return err
		} else {
			glog.Infof("error: %v", err)
		}
	}

	err = deleteAPIService(cmdOut, apiSvcClientset, opts)
	if err != nil {
		if !opts.IgnoreErrors {
			return err
		} else {
			glog.Infof("error: %v", err)
		}
	}

	err = deleteRBACObjects(cmdOut, hostClientset, opts)
	if err != nil {
		if !opts.IgnoreErrors {
			return err
		} else {
			glog.Infof("error: %v", err)
		}
	}

	err = common.DeleteNamespace(cmdOut, hostClientset,
		opts.ClusterRegistryNamespace, opts.DryRun)
	if err != nil {
		if !opts.IgnoreErrors {
			return err
		} else {
			glog.Infof("error: %v", err)
		}
	}

	err = common.WaitForClusterRegistryDeletion(cmdOut, hostClientset,
		opts.ClusterRegistryNamespace, opts.DryRun)
	if err != nil {
		if !opts.IgnoreErrors {
			return err
		} else {
			glog.Infof("error: %v", err)
		}
	}

	return nil
}

// deleteAPIService deletes the Kubernetes API Service for the cluster
// registry cluster objects.
func deleteAPIService(cmdOut io.Writer, clientset apiregclient.Interface,
	opts *aggregatedClusterRegistryOptions) error {

	fmt.Fprint(cmdOut, "Deleting cluster registry Kubernetes API Service...")
	glog.V(4).Infof("Deleting cluster registry Kubernetes API Service %v", apiServiceName)

	err := deleteAPIServiceObject(clientset, apiServiceName, opts.DryRun)

	if err != nil {
		glog.V(4).Infof("Failed to delete cluster registry Kubernetes API Service %v: %v",
			apiServiceName, err)
		return err
	}

	fmt.Fprintln(cmdOut, " done")
	glog.V(4).Info("Successfully deleted cluster registry Kubernetes API Service")

	return nil
}

// deleteAPIServiceObject deletes the cluster registry API Service object.
func deleteAPIServiceObject(clientset apiregclient.Interface, name string,
	dryRun bool) error {

	if dryRun {
		return nil
	}

	return clientset.ApiregistrationV1beta1().APIServices().Delete(name,
		&metav1.DeleteOptions{})
}

// deleteRBACObjects handles the deletion of the RBAC objects not in the
// cluster registry namespace necessary to remove the aggregated cluster
// registry.
func deleteRBACObjects(cmdOut io.Writer, clientset client.Interface,
	opts *aggregatedClusterRegistryOptions) error {

	fmt.Fprintf(cmdOut, "Deleting RBAC objects...")

	// Delete the role binding that allows the cluster registry service account to
	// access the extension-apiserver-authentication configmap.
	glog.V(4).Infof("Deleting role %v for accessing extension-apiserver-authentication ConfigMap",
		extensionAPIServerRBName)

	err := deleteExtensionAPIServerAuthenticationRoleBinding(clientset,
		extensionAPIServerRBName, opts.DryRun)

	if err != nil {
		glog.V(4).Infof("Failed to delete extension-apiserver-authentication ConfigMap reader role binding")
		return err
	}

	// Delete Kubernetes cluster role binding that allows the cluster registry
	// service account to delegate auth to the kubernetes API server.
	glog.V(4).Infof("Deleting cluster role binding %v", authDelegatorCRBName)

	err = deleteAuthDelegatorClusterRoleBinding(clientset, authDelegatorCRBName, opts.DryRun)

	if err != nil {
		glog.V(4).Infof("Failed to delete cluster role binding %v: %v",
			authDelegatorCRBName, err)
		return err
	}

	glog.V(4).Info("Successfully deleted cluster role binding")

	fmt.Fprintln(cmdOut, " done")
	return nil
}

// deleteExtensionAPIServerAuthenticationRoleBinding deletes the rolebinding
// object that allows the cluster registry to access the extension-apiserver-authentication
// ConfigMap.
func deleteExtensionAPIServerAuthenticationRoleBinding(clientset client.Interface,
	name string, dryRun bool) error {
	if dryRun {
		return nil
	}

	return clientset.RbacV1().RoleBindings("kube-system").Delete(name,
		&metav1.DeleteOptions{})
}

// deleteAuthDelegatorClusterRoleBinding deletes the system:auth-delegator cluster role
// binding object.
func deleteAuthDelegatorClusterRoleBinding(clientset client.Interface, name string,
	dryRun bool) error {

	if dryRun {
		return nil
	}

	return clientset.RbacV1().ClusterRoleBindings().Delete(name,
		&metav1.DeleteOptions{})
}
