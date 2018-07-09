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
	"fmt"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/spf13/pflag"
)

// SubcommandOptions holds the configuration required by the subcommands of
// `kubefed2`.
type SubcommandOptions struct {
	ClusterName         string
	Host                string
	FederationNamespace string
	Kubeconfig          string
	DryRun              bool
}

// CommonBind adds the common flags to the flagset passed in.
func (o *SubcommandOptions) CommonBind(flags *pflag.FlagSet) {
	flags.StringVar(&o.Kubeconfig, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests.")
	flags.StringVar(&o.Host, "host-cluster-context", "", "Host cluster context")
	flags.BoolVar(&o.DryRun, "dry-run", false,
		"Run the command in dry-run mode, without making any server requests.")
}

// SetName sets the name from the args passed in for the required positional
// argument.
func (o *SubcommandOptions) SetName(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("NAME is required")
	}

	// Hard-code the federation namespace until a canonical way of
	// configuring federation (e.g. via configmap) exists.  In the
	// absence of canonical configuration, allowing override of the
	// system namespace for a kubefed2 command is a likely source of
	// problems since only the hard-coded namespace is valid.
	o.FederationNamespace = util.FederationSystemNamespace
	o.ClusterName = args[0]
	return nil
}
