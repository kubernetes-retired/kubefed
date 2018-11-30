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

package kubefed2

import (
	"io"

	"github.com/golang/glog"

	fedclient "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	crclient "k8s.io/cluster-registry/pkg/client/clientset/versioned"
)

var (
	unjoin_long = `
		Unjoin removes a cluster from a federation.
		Current context is assumed to be a Kubernetes cluster
		hosting the federation control plane. Please use the
		--host-cluster-context flag otherwise.`
	unjoin_example = `
		# Unjoin a cluster from a federation by specifying the
		# cluster name and the context name of the federation
		# control plane's host cluster. Cluster name must be
		# a valid RFC 1123 subdomain name. Cluster context
		# must be specified if the cluster name is different
		# than the cluster's context in the local kubeconfig.
		kubefed2 unjoin foo --host-cluster-context=bar`
)

type unjoinFederation struct {
	options.SubcommandOptions
	unjoinFederationOptions
}

type unjoinFederationOptions struct {
	clusterContext     string
	removeFromRegistry bool
	forceDeletion      bool
}

// Bind adds the unjoin specific arguments to the flagset passed in as an
// argument.
func (o *unjoinFederationOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.clusterContext, "cluster-context", "",
		"Name of the cluster's context in the local kubeconfig. Defaults to cluster name if unspecified.")
	flags.BoolVar(&o.removeFromRegistry, "remove-from-registry", false,
		"Remove the cluster from the cluster registry running in the host cluster context.")
	flags.BoolVar(&o.forceDeletion, "force", false,
		"Delete federated cluster and secret resources even if resources in the cluster targeted for unjoin are not removed successfully.")
}

// NewCmdUnjoin defines the `unjoin` command that unjoins a cluster from a
// federation.
func NewCmdUnjoin(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &unjoinFederation{}

	cmd := &cobra.Command{
		Use:     "unjoin CLUSTER_NAME --host-cluster-context=HOST_CONTEXT",
		Short:   "Unjoin a cluster from a federation",
		Long:    unjoin_long,
		Example: unjoin_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.Complete(args)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			err = opts.Run(cmdOut, config)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}
		},
	}

	flags := cmd.Flags()
	opts.CommonBind(flags)
	opts.Bind(flags)

	return cmd
}

// Complete ensures that options are valid and marshals them if necessary.
func (j *unjoinFederation) Complete(args []string) error {
	err := j.SetName(args)
	if err != nil {
		return err
	}

	if j.clusterContext == "" {
		glog.V(2).Infof("Defaulting cluster context to unjoining cluster name %s", j.ClusterName)
		j.clusterContext = j.ClusterName
	}

	glog.V(2).Infof("Args and flags: name %s, host-cluster-context: %s, host-system-namespace: %s, kubeconfig: %s, cluster-context: %s, dry-run: %v",
		j.ClusterName, j.HostClusterContext, j.FederationNamespace, j.Kubeconfig, j.clusterContext, j.DryRun)

	return nil
}

// Run is the implementation of the `unjoin federation` command.
func (j *unjoinFederation) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		// TODO(font): Return new error with this same text so it can be output
		// by caller.
		glog.V(2).Infof("Failed to get host cluster config: %v", err)
		return err
	}

	clusterConfig, err := config.ClusterConfig(j.clusterContext, j.Kubeconfig)
	if err != nil {
		glog.V(2).Infof("Failed to get unjoining cluster config: %v", err)
		return err
	}

	err = UnjoinCluster(hostConfig, clusterConfig, j.FederationNamespace, j.ClusterNamespace,
		j.HostClusterContext, j.clusterContext, j.ClusterName, j.removeFromRegistry, j.forceDeletion, j.DryRun)
	if err != nil {
		return err
	}

	return nil
}

// UnjoinCluster performs all the necessary steps to unjoin a cluster from the
// federation provided the required set of parameters are passed in.
func UnjoinCluster(hostConfig, clusterConfig *rest.Config, federationNamespace, clusterNamespace, hostClusterContext,
	unjoiningClusterContext, unjoiningClusterName string, removeFromRegistry, forceDeletion, dryRun bool) error {

	hostClientset, err := util.HostClientset(hostConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get host cluster clientset: %v", err)
		return err
	}

	clusterClientset, err := util.ClusterClientset(clusterConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get unjoining cluster clientset: %v", err)
		return err
	}

	fedClientset, err := util.FedClientset(hostConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get federation clientset: %v", err)
		return err
	}

	if removeFromRegistry {
		removeFromClusterRegistry(hostConfig, clusterNamespace, clusterConfig.Host, unjoiningClusterName, dryRun)
	}

	deletionSucceeded := deleteRBACResources(clusterClientset, federationNamespace, unjoiningClusterName, hostClusterContext, dryRun)

	if unjoiningClusterContext == hostClusterContext {
		glog.V(2).Infof("Host cluster: %s and unjoin cluster: %s are same, no need to delete federation namespace: %s from unjoin cluster.",
			hostClusterContext, unjoiningClusterContext, federationNamespace)
	} else {
		err = deleteFedNSFromUnjoinCluster(clusterClientset, federationNamespace, unjoiningClusterName, dryRun)
		if err != nil {
			deletionSucceeded = false
		}
	}

	// deletionSucceeded when all operations in deleteRBACResources and deleteFedNSFromUnjoinCluster succeed.
	if deletionSucceeded || forceDeletion {
		deleteFederatedClusterAndSecret(hostClientset, fedClientset, federationNamespace, unjoiningClusterName, dryRun)
	}

	return nil
}

// removeFromClusterRegistry handles removing the cluster from the cluster registry and
// reports progress.
func removeFromClusterRegistry(hostConfig *rest.Config, clusterNamespace, host, unjoiningClusterName string,
	dryRun bool) {

	// Get the cluster registry clientset using the host cluster config.
	crClientset, err := util.ClusterRegistryClientset(hostConfig)
	if err != nil {
		glog.Errorf("Failed to get cluster registry clientset: %v", err)
		return
	}

	glog.V(2).Infof("Removing cluster: %s from the cluster registry.", unjoiningClusterName)

	err = unregisterCluster(crClientset, clusterNamespace, host, unjoiningClusterName, dryRun)
	if err != nil {
		glog.Errorf("Could not remove cluster from the cluster registry: %v", err)
		return
	}

	glog.V(2).Infof("Removed cluster: %s from the cluster registry.", unjoiningClusterName)
}

// unregisterCluster removes a cluster from the cluster registry.
func unregisterCluster(crClientset *crclient.Clientset, clusterNamespace, host, unjoiningClusterName string,
	dryRun bool) error {
	if dryRun {
		return nil
	}

	err := crClientset.ClusterregistryV1alpha1().Clusters(clusterNamespace).Delete(unjoiningClusterName,
		&metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

// deleteFederatedClusterAndSecret deletes a federated cluster resource that associates
// the cluster and secret.
func deleteFederatedClusterAndSecret(hostClientset client.Interface, fedClientset *fedclient.Clientset,
	federationNamespace, unjoiningClusterName string, dryRun bool) {
	if dryRun {
		return
	}

	glog.V(2).Infof("Deleting federated cluster resource from namespace: %s for unjoin cluster: %s",
		federationNamespace, unjoiningClusterName)

	fedCluster, err := fedClientset.CoreV1alpha1().FederatedClusters(federationNamespace).Get(
		unjoiningClusterName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("Failed to get FederatedCluster resource from namespace: %s for unjoin cluster: %s due to: %v", federationNamespace, unjoiningClusterName, err)
		return
	}

	err = hostClientset.CoreV1().Secrets(federationNamespace).Delete(fedCluster.Spec.SecretRef.Name,
		&metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("Failed to delete Secret resource from namespace: %s for unjoin cluster: %s due to: %v", federationNamespace, unjoiningClusterName, err)
	} else {
		glog.V(2).Infof("Deleted Secret resource from namespace: %s for unjoin cluster: %s", federationNamespace, unjoiningClusterName)
	}

	err = fedClientset.CoreV1alpha1().FederatedClusters(federationNamespace).Delete(
		unjoiningClusterName, &metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("Failed to delete FederatedCluster resource from namespace: %s for unjoin cluster: %s due to: %v", federationNamespace, unjoiningClusterName, err)
	} else {
		glog.V(2).Infof("Deleted FederatedCluster resource from namespace: %s for unjoin cluster: %s", federationNamespace, unjoiningClusterName)
	}
}

// deleteRBACResources deletes the cluster role, cluster rolebindings and service account
// from the unjoining cluster.
func deleteRBACResources(unjoiningClusterClientset client.Interface,
	namespace, unjoiningClusterName, hostClusterName string, dryRun bool) bool {

	var deletionSucceeded = true

	saName := util.ClusterServiceAccountName(unjoiningClusterName, hostClusterName)

	glog.V(2).Infof("Deleting cluster role binding for service account: %s in unjoining cluster: %s",
		saName, unjoiningClusterName)

	deletionSucceeded = deleteClusterRoleAndBinding(unjoiningClusterClientset, saName, namespace, dryRun)
	if deletionSucceeded {
		glog.V(2).Infof("Deleted cluster role binding for service account: %s in unjoining cluster: %s",
			saName, unjoiningClusterName)
	}

	glog.V(2).Infof("Deleting service account %s in unjoining cluster: %s", saName, unjoiningClusterName)

	err := deleteServiceAccount(unjoiningClusterClientset, saName, namespace, dryRun)
	if err != nil {
		deletionSucceeded = false
		glog.Errorf("Error deleting service account: %s in unjoining cluster. %v", saName, err)
	} else {
		glog.V(2).Infof("Deleted service account %s in unjoining cluster: %s", saName, unjoiningClusterName)
	}

	return deletionSucceeded
}

// deleteFedNSFromUnjoinCluster deletes the federation namespace and multi cluster
// public namespace in the unjoining cluster.
func deleteFedNSFromUnjoinCluster(unjoiningClusterClientset client.Interface,
	federationNamespace, unjoiningClusterName string, dryRun bool) error {

	if dryRun {
		return nil
	}

	glog.V(2).Infof("Deleting federation namespace: %s from unjoin cluster: %s",
		federationNamespace, unjoiningClusterName)
	err := unjoiningClusterClientset.CoreV1().Namespaces().Delete(federationNamespace,
		&metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("Could not delete federation namespace: %s due to: %v", federationNamespace, err)
	} else {
		glog.V(2).Infof("Deleted federation namespace: %s from unjoin cluster: %s", federationNamespace, unjoiningClusterName)
	}

	return err
}

// deleteServiceAccount deletes a service account in the cluster associated
// with clusterClientset with credentials that are used by the host cluster
// to access its API server.
func deleteServiceAccount(clusterClientset client.Interface, saName,
	namespace string, dryRun bool) error {
	if dryRun {
		return nil
	}

	// Delete a service account.
	err := clusterClientset.CoreV1().ServiceAccounts(namespace).Delete(saName,
		&metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

// deleteClusterRoleAndBinding deletes an RBAC cluster role and binding that
// allows the service account identified by saName to access all resources in
// all namespaces in the cluster associated with clusterClientset.
func deleteClusterRoleAndBinding(clusterClientset client.Interface, saName, namespace string, dryRun bool) bool {
	var deletionSucceeded = true

	if dryRun {
		return deletionSucceeded
	}

	roleName := util.RoleName(saName)
	healthCheckRoleName := util.HealthCheckRoleName(saName)

	// Attempt to delete all role and role bindings created by join
	// and ignore if there is any error

	for _, name := range []string{roleName, healthCheckRoleName} {
		err := clusterClientset.RbacV1().ClusterRoleBindings().Delete(name, &metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			deletionSucceeded = false
			glog.Errorf("Could not delete cluster role binding %q in unjoining cluster: %v", name, err)
		}

		err = clusterClientset.RbacV1().ClusterRoles().Delete(name, &metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			deletionSucceeded = false
			glog.Errorf("Could not delete cluster role %q in unjoining cluster: %v", name, err)
		}
	}

	err := clusterClientset.RbacV1().RoleBindings(namespace).Delete(roleName, &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		deletionSucceeded = false
		glog.Errorf("Could not delete role binding for service account: %s in unjoining cluster: %v",
			saName, err)
	}

	err = clusterClientset.RbacV1().Roles(namespace).Delete(roleName, &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		deletionSucceeded = false
		glog.Errorf("Could not delete role for service account: %s in unjoining cluster: %v",
			saName, err)
	}

	return deletionSucceeded
}
