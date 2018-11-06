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

package federate

import (
	"io"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
)

var (
	federate_long = `
        Enables/disables propagation of Kubernetes API types (including CRDs)
        to multiple clusters.

        Current context is assumed to be a Kubernetes cluster
        hosting the federation control plane. Please use the
        --host-cluster-context flag otherwise.`
)

// NewCmdFederate creates a command object for the "federate" action,
// and adds all child commands to it.
func NewCmdFederate(cmdOut io.Writer, config util.FedConfig) *cobra.Command {

	cmd := &cobra.Command{
		Use:                   "federate SUBCOMMAND",
		DisableFlagsInUseLine: true,
		Short:                 "Enable/disable propagation of Kubernetes types to multiple clusters",
		Long:                  federate_long,
		Run: func(_ *cobra.Command, args []string) {
			if len(args) < 1 {
				glog.Fatalf("missing subcommand; \"federate\" is not meant to be run on its own")
			} else {
				glog.Fatalf("invalid subcommand: %q", args[0])
			}
		},
	}

	cmd.AddCommand(NewCmdFederateEnable(cmdOut, config))
	cmd.AddCommand(NewCmdFederateDisable(cmdOut, config))

	return cmd
}
