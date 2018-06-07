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

package standalone

import (
	"io"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/cluster-registry/pkg/crinit/options"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type standaloneClusterRegistryOptions struct {
	options.SubcommandOptions
	apiServerServiceTypeString   string
	apiServerEnableHTTPBasicAuth bool
	apiServerEnableTokenAuth     bool
}

func (o *standaloneClusterRegistryOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.apiServerServiceTypeString, options.APIServerServiceTypeFlag,
		string(v1.ServiceTypeLoadBalancer),
		"The type of service to create for the cluster registry. Options: 'LoadBalancer', 'NodePort'.")
	flags.BoolVar(&o.apiServerEnableHTTPBasicAuth, "apiserver-enable-basic-auth", false,
		"Enables HTTP Basic authentication for the cluster registry API server. Defaults to false.")
	flags.BoolVar(&o.apiServerEnableTokenAuth, "apiserver-enable-token-auth", false,
		"Enables token authentication for the cluster registry API server. Defaults to false.")
}

// NewCmdStandalone defines the `standalone` command with `init` and `delete`
// subcommands to bootstrap or remove a cluster registry inside a host
// Kubernetes cluster.
func NewCmdStandalone(cmdOut io.Writer, pathOptions *clientcmd.PathOptions,
	defaultServerImage, defaultEtcdImage string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "standalone",
		Short: "Subcommands to manage a standalone cluster registry",
		Long:  "Commands used to manage a standalone cluster registry. That is, a cluster registry that is not aggregated with another Kubernetes API server.",
	}

	cmd.AddCommand(newSubCmdInit(cmdOut, pathOptions, defaultServerImage, defaultEtcdImage))
	cmd.AddCommand(newSubCmdDelete(cmdOut, pathOptions))
	return cmd
}
