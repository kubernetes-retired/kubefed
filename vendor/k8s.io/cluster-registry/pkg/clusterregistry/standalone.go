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

package clusterregistry

import (
	"io"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/cluster-registry/pkg/clusterregistry/options"
)

// NewCmdStandalone defines the 'standalone' subcommand that runs the cluster
// registry using the Kubernetes aggregator.
func NewCmdStandalone(cmdOut io.Writer, pathOptions *clientcmd.PathOptions) *cobra.Command {
	opts := options.NewStandaloneServerRunOptions()

	cmd := &cobra.Command{
		Use:   "standalone",
		Short: "Subcommand to run a standalone cluster registry",
		Long:  "Subcommand to run the cluster registry as a standalone Kubernetes API server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(opts, wait.NeverStop)
		},
	}

	flags := cmd.Flags()
	opts.AddFlags(flags)
	return cmd
}
