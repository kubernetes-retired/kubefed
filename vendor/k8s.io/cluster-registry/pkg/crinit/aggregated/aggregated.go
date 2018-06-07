/*
Copyright 2017 The Kubernetes Authors.

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
	"io"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/cluster-registry/pkg/crinit/options"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// Set priorities for our API in the APIService object.
	apiServiceGroupPriorityMinimum int32 = 10000
	apiServiceVersionPriority      int32 = 20

	// Name used for our cluster registry APIService object to register with
	// the K8s API aggregator.
	apiServiceName = v1alpha1.SchemeGroupVersion.Version + "." + v1alpha1.GroupName

	// Name used for our cluster registry service account object to be used
	// with our cluster role objects.
	serviceAccountName = strings.Replace(v1alpha1.GroupName, ".", "-", -1) + "-apiserver"

	// Name used for our cluster registry cluster role binding (CRB) object that
	// allows delegated authentication and authorization checks.
	authDelegatorCRBName = v1alpha1.GroupName + ":apiserver-auth-delegator"

	// Name used for the cluster registry role binding that allows the cluster
	// registry service account to access the extension-apiserver-authentication
	// ConfigMap.
	extensionAPIServerRBName = v1alpha1.GroupName + ":extension-apiserver-authentication-reader"
)

type aggregatedClusterRegistryOptions struct {
	options.SubcommandOptions
	apiServerServiceTypeString string
}

func (o *aggregatedClusterRegistryOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.apiServerServiceTypeString, options.APIServerServiceTypeFlag,
		string(v1.ServiceTypeNodePort),
		"The type of service to create for the cluster registry. Options: 'LoadBalancer', 'NodePort'.")
}

// NewCmdAggregated defines the `aggregated` command with `init` and `delete`
// subcommands to bootstrap or remove a cluster registry inside a host
// Kubernetes cluster.
func NewCmdAggregated(cmdOut io.Writer, pathOptions *clientcmd.PathOptions,
	defaultServerImage, defaultEtcdImage string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aggregated",
		Short: "Subcommands to manage an aggregated cluster registry",
		Long:  "Commands used to manage an aggregated cluster registry. That is, a cluster registry that is aggregated with another Kubernetes API server.",
	}

	cmd.AddCommand(newSubCmdInit(cmdOut, pathOptions, defaultServerImage, defaultEtcdImage))
	cmd.AddCommand(newSubCmdDelete(cmdOut, pathOptions))
	return cmd
}
