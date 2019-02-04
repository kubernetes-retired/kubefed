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

package options

import (
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

// SubcommandOptions holds the configuration required by the subcommands of
// `kubefed2`.
type SubcommandOptions struct {
	ClusterName         string
	HostClusterContext  string
	FederationNamespace string
	ClusterNamespace    string
	Kubeconfig          string
	DryRun              bool
}

// CommonBind adds the common flags to the flagset passed in.
func (o *SubcommandOptions) CommonBind(flags *pflag.FlagSet) {
	flags.StringVar(&o.Kubeconfig, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests.")
	flags.StringVar(&o.HostClusterContext, "host-cluster-context", "", "Host cluster context")
	flags.StringVar(&o.FederationNamespace, "federation-namespace", util.DefaultFederationSystemNamespace,
		"Namespace in the host cluster where the federation system components are installed.  This namespace will also be the target of propagation if the controller manager is configured with --limited-scope and clusters are joined with --limited-scope.")
	flags.StringVar(&o.ClusterNamespace, "registry-namespace", util.MulticlusterPublicNamespace,
		"Namespace in the host cluster where clusters are registered")
	flags.BoolVar(&o.DryRun, "dry-run", false,
		"Run the command in dry-run mode, without making any server requests.")
}

// SetName sets the name from the args passed in for the required positional
// argument.
func (o *SubcommandOptions) SetName(args []string) error {
	if len(args) == 0 {
		return errors.New("NAME is required")
	}

	o.ClusterName = args[0]
	return nil
}
