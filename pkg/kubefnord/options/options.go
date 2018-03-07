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

	"github.com/marun/fnord/pkg/controller/util"
	"github.com/spf13/pflag"
)

// SubcommandOptions holds the configuration required by the subcommands of
// `kubefnord`.
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
	flags.StringVar(&o.FederationNamespace, "federation-namespace", util.FederationSystemNamespace, "Namespace in the host cluster where the federation system components are installed")
	flags.BoolVar(&o.DryRun, "dry-run", false,
		"Run the command in dry-run mode, without making any server requests.")
}

// SetName sets the name from the args passed in for the required positional
// argument.
func (o *SubcommandOptions) SetName(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("NAME is required")
	}

	o.ClusterName = args[0]
	return nil
}
