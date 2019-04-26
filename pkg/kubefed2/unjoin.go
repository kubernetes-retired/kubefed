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
	"context"
	goerrors "errors"
	"io"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	controllerutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	crv1a1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
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
	options.GlobalSubcommandOptions
	options.CommonSubcommandOptions
	options.FederationConfigOptions
	unjoinFederationOptions
}

type unjoinFederationOptions struct {
	removeFromRegistry bool
	forceDeletion      bool
}

// Bind adds the unjoin specific arguments to the flagset passed in as an
// argument.
func (o *unjoinFederationOptions) Bind(flags *pflag.FlagSet) {
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
				glog.Fatalf("Error: %v", err)
			}

			err = opts.Run(cmdOut, config)
			if err != nil {
				glog.Fatalf("Error: %v", err)
			}
		},
	}

	flags := cmd.Flags()
	opts.GlobalSubcommandBind(flags)
	opts.CommonSubcommandBind(flags)
	opts.Bind(flags)

	return cmd
}

// Complete ensures that options are valid and marshals them if necessary.
func (j *unjoinFederation) Complete(args []string) error {
	err := j.SetName(args)
	if err != nil {
		return err
	}

	if j.ClusterContext == "" {
		glog.V(2).Infof("Defaulting cluster context to unjoining cluster name %s", j.ClusterName)
		j.ClusterContext = j.ClusterName
	}

	if j.HostClusterName != "" && strings.ContainsAny(j.HostClusterName, ":/") {
		return goerrors.New("host-cluster-name may not contain \"/\" or \":\"")
	}

	if j.HostClusterName == "" && strings.ContainsAny(j.HostClusterContext, ":/") {
		return goerrors.New("host-cluster-name must be set if the name of the host cluster context contains one of \":\" or \"/\"")
	}

	glog.V(2).Infof("Args and flags: name %s, host-cluster-context: %s, host-system-namespace: %s, kubeconfig: %s, cluster-context: %s, dry-run: %v",
		j.ClusterName, j.HostClusterContext, j.FederationNamespace, j.Kubeconfig, j.ClusterContext, j.DryRun)

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
	_, j.ClusterNamespace, err = options.GetOptionsFromFederationConfig(hostConfig, j.FederationNamespace)
	if err != nil {
		return err
	}

	clusterConfig, err := config.ClusterConfig(j.ClusterContext, j.Kubeconfig)
	if err != nil {
		glog.V(2).Infof("Failed to get unjoining cluster config: %v", err)

		if !j.forceDeletion {
			return err
		}
		// If configuration for the member cluster cannot be successfully loaded,
		// forceDeletion indicates that resources associated with the member cluster
		// should still be removed from the host cluster.
	}

	hostClusterName := j.HostClusterContext
	if j.HostClusterName != "" {
		hostClusterName = j.HostClusterName
	}

	return UnjoinCluster(hostConfig, clusterConfig, j.FederationNamespace, j.ClusterNamespace,
		hostClusterName, j.HostClusterContext, j.ClusterContext, j.ClusterName, j.removeFromRegistry, j.forceDeletion, j.DryRun)
}

// UnjoinCluster performs all the necessary steps to unjoin a cluster from the
// federation provided the required set of parameters are passed in.
func UnjoinCluster(hostConfig, clusterConfig *rest.Config, federationNamespace, clusterNamespace, hostClusterName, hostClusterContext,
	unjoiningClusterContext, unjoiningClusterName string, removeFromRegistry, forceDeletion, dryRun bool) error {

	hostClientset, err := util.HostClientset(hostConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get host cluster clientset: %v", err)
		return err
	}

	var clusterClientset *kubeclient.Clientset
	if clusterConfig != nil {
		clusterClientset, err = util.ClusterClientset(clusterConfig)
		if err != nil {
			glog.V(2).Infof("Failed to get unjoining cluster clientset: %v", err)
			if !forceDeletion {
				return err
			}
		}
	}

	client, err := genericclient.New(hostConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get federation clientset: %v", err)
		return err
	}

	if removeFromRegistry {
		removeFromClusterRegistry(hostConfig, clusterNamespace, unjoiningClusterName, dryRun)
	}

	var deletionSucceeded bool
	if clusterClientset != nil {
		deletionSucceeded = deleteRBACResources(clusterClientset, federationNamespace, unjoiningClusterName, hostClusterName, dryRun)

		err = deleteFedNSFromUnjoinCluster(hostClientset, clusterClientset, federationNamespace, unjoiningClusterName, dryRun)
		if err != nil {
			glog.Errorf("Error deleting federation namespace from unjoin cluster: %v", err)
			deletionSucceeded = false
		}
	}

	// deletionSucceeded when all operations in deleteRBACResources and deleteFedNSFromUnjoinCluster succeed.
	if deletionSucceeded || forceDeletion {
		deleteFederatedClusterAndSecret(hostClientset, client, federationNamespace, unjoiningClusterName, dryRun)
	}

	return nil
}

// removeFromClusterRegistry handles removing the cluster from the cluster registry and
// reports progress.
func removeFromClusterRegistry(hostConfig *rest.Config, clusterNamespace, unjoiningClusterName string,
	dryRun bool) {

	client, err := util.ClusterRegistryClientset(hostConfig)
	if err != nil {
		glog.Errorf("Failed to get cluster registry clientset: %v", err)
		return
	}

	glog.V(2).Infof("Removing cluster: %s from the cluster registry.", unjoiningClusterName)

	err = unregisterCluster(client, clusterNamespace, unjoiningClusterName, dryRun)
	if err != nil {
		glog.Errorf("Could not remove cluster from the cluster registry: %v", err)
		return
	}

	glog.V(2).Infof("Removed cluster: %s from the cluster registry.", unjoiningClusterName)
}

// unregisterCluster removes a cluster from the cluster registry.
func unregisterCluster(client genericclient.Client, clusterNamespace, unjoiningClusterName string,
	dryRun bool) error {
	if dryRun {
		return nil
	}

	return client.Delete(context.TODO(), &crv1a1.Cluster{}, clusterNamespace, unjoiningClusterName)
}

// deleteFederatedClusterAndSecret deletes a federated cluster resource that associates
// the cluster and secret.
func deleteFederatedClusterAndSecret(hostClientset kubeclient.Interface, client genericclient.Client,
	federationNamespace, unjoiningClusterName string, dryRun bool) {
	if dryRun {
		return
	}

	glog.V(2).Infof("Deleting federated cluster resource from namespace: %s for unjoin cluster: %s",
		federationNamespace, unjoiningClusterName)

	fedCluster := &fedv1a1.FederatedCluster{}
	err := client.Get(context.TODO(), fedCluster, federationNamespace, unjoiningClusterName)
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

	err = client.Delete(context.TODO(), fedCluster, fedCluster.Namespace, fedCluster.Name)
	if err != nil {
		glog.Errorf("Failed to delete FederatedCluster resource from namespace: %s for unjoin cluster: %s due to: %v", federationNamespace, unjoiningClusterName, err)
	} else {
		glog.V(2).Infof("Deleted FederatedCluster resource from namespace: %s for unjoin cluster: %s", federationNamespace, unjoiningClusterName)
	}
}

// deleteRBACResources deletes the cluster role, cluster rolebindings and service account
// from the unjoining cluster.
func deleteRBACResources(unjoiningClusterClientset kubeclient.Interface,
	namespace, unjoiningClusterName, hostClusterName string, dryRun bool) bool {

	saName := util.ClusterServiceAccountName(unjoiningClusterName, hostClusterName)

	glog.V(2).Infof("Deleting cluster role binding for service account: %s in unjoining cluster: %s",
		saName, unjoiningClusterName)

	deletionSucceeded := deleteClusterRoleAndBinding(unjoiningClusterClientset, saName, namespace, dryRun)
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

// deleteFedNSFromUnjoinCluster deletes the federation namespace from
// the unjoining cluster so long as the unjoining cluster is not the
// host cluster.
func deleteFedNSFromUnjoinCluster(hostClientset, unjoiningClusterClientset kubeclient.Interface,
	federationNamespace, unjoiningClusterName string, dryRun bool) error {

	if dryRun {
		return nil
	}

	hostClusterNamespace, err := hostClientset.CoreV1().Namespaces().Get(federationNamespace, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Error retrieving namespace %q from host cluster", federationNamespace)
	}

	unjoiningClusterNamespace, err := unjoiningClusterClientset.CoreV1().Namespaces().Get(federationNamespace, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Error retrieving namespace %q from unjoining cluster %q", federationNamespace, unjoiningClusterName)
	}

	if controllerutil.IsPrimaryCluster(hostClusterNamespace, unjoiningClusterNamespace) {
		glog.V(2).Infof("The federation namespace %q does not need to be deleted from the host cluster by unjoin.", federationNamespace)
		return nil
	}

	glog.V(2).Infof("Deleting federation namespace %q from unjoining cluster %q", federationNamespace, unjoiningClusterName)
	err = unjoiningClusterClientset.CoreV1().Namespaces().Delete(federationNamespace, &metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		glog.V(2).Infof("The federation namespace %q no longer exists in unjoining cluster %q", federationNamespace, unjoiningClusterName)
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "Could not delete federation namespace %q from unjoining cluster %q", federationNamespace, unjoiningClusterName)
	}
	glog.V(2).Infof("Deleted federation namespace %q from unjoining cluster %q", federationNamespace, unjoiningClusterName)
	return nil
}

// deleteServiceAccount deletes a service account in the cluster associated
// with clusterClientset with credentials that are used by the host cluster
// to access its API server.
func deleteServiceAccount(clusterClientset kubeclient.Interface, saName,
	namespace string, dryRun bool) error {
	if dryRun {
		return nil
	}

	// Delete a service account.
	return clusterClientset.CoreV1().ServiceAccounts(namespace).Delete(saName,
		&metav1.DeleteOptions{})
}

// deleteClusterRoleAndBinding deletes an RBAC cluster role and binding that
// allows the service account identified by saName to access all resources in
// all namespaces in the cluster associated with clusterClientset.
func deleteClusterRoleAndBinding(clusterClientset kubeclient.Interface, saName, namespace string, dryRun bool) bool {
	var deletionSucceeded = true

	if dryRun {
		return deletionSucceeded
	}

	roleName := util.RoleName(saName)
	healthCheckRoleName := util.HealthCheckRoleName(saName, namespace)

	// Attempt to delete all role and role bindings created by join
	// and ignore if there is any error

	for _, name := range []string{roleName, healthCheckRoleName} {
		err := clusterClientset.RbacV1().ClusterRoleBindings().Delete(name, &metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			deletionSucceeded = false
			glog.Errorf("Could not delete cluster role binding %q in unjoining cluster: %v", name, err)
		}

		err = clusterClientset.RbacV1().ClusterRoles().Delete(name, &metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			deletionSucceeded = false
			glog.Errorf("Could not delete cluster role %q in unjoining cluster: %v", name, err)
		}
	}

	err := clusterClientset.RbacV1().RoleBindings(namespace).Delete(roleName, &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		deletionSucceeded = false
		glog.Errorf("Could not delete role binding for service account: %s in unjoining cluster: %v",
			saName, err)
	}

	err = clusterClientset.RbacV1().Roles(namespace).Delete(roleName, &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		deletionSucceeded = false
		glog.Errorf("Could not delete role for service account: %s in unjoining cluster: %v",
			saName, err)
	}

	return deletionSucceeded
}
