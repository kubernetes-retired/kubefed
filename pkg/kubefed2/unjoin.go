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
	"fmt"
	"io"

	"github.com/golang/glog"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclient "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	controllerutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
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
}

// Bind adds the unjoin specific arguments to the flagset passed in as an
// argument.
func (o *unjoinFederationOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.clusterContext, "cluster-context", "",
		"Name of the cluster's context in the local kubeconfig. Defaults to cluster name if unspecified.")
	flags.BoolVar(&o.removeFromRegistry, "remove-from-registry", false,
		"Remove the cluster from the cluster registry running in the host cluster context.")
}

// NewCmdUnJoin defines the `unjoin` command that unjoins a cluster from a
// federation.
func NewCmdUnJoin(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
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

	glog.V(2).Infof("Args and flags: name %s, host: %s, host-system-namespace: %s, kubeconfig: %s, cluster-context: %s, dry-run: %v",
		j.ClusterName, j.Host, j.FederationNamespace, j.Kubeconfig, j.clusterContext, j.DryRun)

	return nil
}

// Run is the implementation of the `unjoin federation` command.
func (j *unjoinFederation) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.Host, j.Kubeconfig)
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

	err = UnJoinCluster(hostConfig, clusterConfig, j.FederationNamespace,
		j.Host, j.ClusterName, j.removeFromRegistry, j.DryRun)
	if err != nil {
		return err
	}

	return nil
}

// UnJoinCluster performs all the necessary steps to unjoin a cluster from the
// federation provided the required set of parameters are passed in.
func UnJoinCluster(hostConfig, clusterConfig *rest.Config, federationNamespace,
	host, unjoiningClusterName string, removeFromRegistry, dryRun bool) error {

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

	glog.V(2).Infof("Performing preflight checks for unjoin cluster: %s.", unjoiningClusterName)
	err = performPreflightChecksForUnjoin(clusterClientset, unjoiningClusterName, host, federationNamespace)
	if err != nil {
		return err
	}

	if removeFromRegistry {
		err = removeFromClusterRegistry(hostConfig, clusterConfig.Host, unjoiningClusterName, dryRun)
		if err != nil {
			return err
		}
	}

	glog.V(2).Infof("Deleting federated cluster resource from namespace: %s for unjoin cluster: %s",
		federationNamespace, unjoiningClusterName)

	_, err = deleteFederatedCluster(hostClientset, fedClientset, federationNamespace,
		unjoiningClusterName, dryRun)
	if err != nil {
		glog.V(2).Infof("Failed to delete federated cluster resource from namespace: %s for unjoin cluster: %s due to: %v",
			federationNamespace, unjoiningClusterName, err)
		return err
	}

	glog.V(2).Infof("Deleted federated cluster resource from namespace: %s for unjoin cluster: %s",
		federationNamespace, unjoiningClusterName)

	err = deleteRBACSecret(hostClientset, clusterClientset,
		federationNamespace, unjoiningClusterName, host, dryRun)
	if err != nil {
		glog.V(2).Infof("Could not delete cluster credentials secret: %v", err)
		return err
	}

	return nil
}

// performPreflightChecks checks that the host and unjoining clusters are in
// a consistent state.
func performPreflightChecksForUnjoin(clusterClientset client.Interface, name, host,
	federationNamespace string) error {
	// Make sure there is a existing service account in the unjoining cluster.
	saName := util.ClusterServiceAccountName(name, host)
	sa, err := clusterClientset.CoreV1().ServiceAccounts(federationNamespace).Get(saName,
		metav1.GetOptions{})

	if errors.IsNotFound(err) {
		return fmt.Errorf("service account does not exist in unjoining cluster")
	} else if err != nil {
		return err
	} else if sa != nil {
		return nil
	}

	return nil
}

// removeFromClusterRegistry handles removing the cluster from the cluster registry and
// reports progress.
func removeFromClusterRegistry(hostConfig *rest.Config, host, unjoiningClusterName string,
	dryRun bool) error {

	// Get the cluster registry clientset using the host cluster config.
	crClientset, err := util.ClusterRegistryClientset(hostConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get cluster registry clientset: %v", err)
		return err
	}

	glog.V(2).Infof("Removing cluster: %s from the cluster registry.", unjoiningClusterName)

	err = unRegisterCluster(crClientset, host, unjoiningClusterName, dryRun)
	if err != nil {
		glog.V(2).Infof("Could not remove cluster from the cluster registry: %v", err)
		return err
	}

	glog.V(2).Infof("Removed cluster: %s from the cluster registry.", unjoiningClusterName)
	return nil
}

// unRegisterCluster removes a cluster from the cluster registry.
func unRegisterCluster(crClientset *crclient.Clientset, host, unjoiningClusterName string,
	dryRun bool) error {
	if dryRun {
		return nil
	}

	err := crClientset.ClusterregistryV1alpha1().Clusters(controllerutil.MulticlusterPublicNamespace).Delete(unjoiningClusterName,
		&metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

// deleteFederatedCluster deletes a federated cluster resource that associates
// the cluster and secret.
func deleteFederatedCluster(hostClientset client.Interface, fedClientset *fedclient.Clientset,
	federationNamespace, unjoiningClusterName string, dryRun bool) (*fedv1a1.FederatedCluster, error) {
	if dryRun {
		return nil, nil
	}

	fedCluster, err := fedClientset.CoreV1alpha1().FederatedClusters(federationNamespace).Get(
		unjoiningClusterName, metav1.GetOptions{})
	if err != nil {
		return fedCluster, err
	}

	err = hostClientset.CoreV1().Secrets(federationNamespace).Delete(fedCluster.Spec.SecretRef.Name,
		&metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}

	err = fedClientset.CoreV1alpha1().FederatedClusters(federationNamespace).Delete(
		unjoiningClusterName, &metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// deleteRBACSecret deletes a secret in the unjoining cluster using a service
// account, and populate that secret into the host cluster to allow it to
// access the unjoining cluster.
func deleteRBACSecret(hostClusterClientset, unjoiningClusterClientset client.Interface,
	namespace, unjoiningClusterName, hostClusterName string, dryRun bool) error {

	saName := util.ClusterServiceAccountName(unjoiningClusterName, hostClusterName)

	glog.V(2).Infof("Deleting cluster role binding for service account: %s in unjoining cluster: %s",
		saName, unjoiningClusterName)

	err := deleteClusterRoleBinding(unjoiningClusterClientset, saName, namespace,
		unjoiningClusterName, dryRun)
	if err != nil {
		glog.V(2).Infof("Error deleting role binding for service account in unjoining cluster: %v",
			err)
		return err
	}

	glog.V(2).Infof("Deleted cluster role binding for service account: %s in unjoining cluster: %s",
		saName, unjoiningClusterName)

	glog.V(2).Infof("Deleting service account %s in unjoining cluster: %s", saName, unjoiningClusterName)

	err = deleteServiceAccount(unjoiningClusterClientset, saName, namespace, dryRun)
	if err != nil {
		return err
	}

	glog.V(2).Infof("Deleted service account %s in unjoining cluster: %s", saName, unjoiningClusterName)

	return nil
}

// deleteServiceAccount deletes a service account in the cluster associated
// with clusterClientset with credentials that will be used by the host cluster
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

// deleteClusterRoleBinding deletes an RBAC cluster role and binding that
// allows the service account identified by saName to access all resources in
// all namespaces in the cluster associated with clusterClientset.
func deleteClusterRoleBinding(clusterClientset client.Interface, saName, namespace,
	unjoiningClusterName string, dryRun bool) error {

	if dryRun {
		return nil
	}

	roleName := util.ClusterRoleName(saName)

	err := clusterClientset.RbacV1().ClusterRoleBindings().Delete(roleName, &metav1.DeleteOptions{})
	if err != nil {
		glog.V(2).Infof("Could not delete cluster role binding for service account in unjoining cluster: %v",
			err)
		return err
	}

	err = clusterClientset.RbacV1().ClusterRoles().Delete(roleName, &metav1.DeleteOptions{})
	if err != nil {
		glog.V(2).Infof("Could not delete cluster role for service account in unjoining cluster: %v",
			err)
		return err
	}

	return nil
}
