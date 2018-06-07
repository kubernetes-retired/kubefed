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

// Package crinit contains the meat of the implementation for the crinit tool,
// which bootstraps a cluster registry into an existing Kubernetes cluster.
package crinit

import (
	"flag"
	"fmt"
	"io"

	apiserverflag "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/cluster-registry/pkg/crinit/aggregated"
	"k8s.io/cluster-registry/pkg/crinit/standalone"
	"k8s.io/cluster-registry/pkg/version"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewClusterregistryCommand creates the `clusterregistry` command.
func NewClusterregistryCommand(out io.Writer, defaultServerImage, defaultEtcdImage string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "crinit",
		Short: "crinit runs a cluster registry in a Kubernetes cluster",
		Long:  "crinit bootstraps and runs a cluster registry as a Deployment in an existing Kubernetes cluster.",
	}

	// Add the command line flags from other dependencies (e.g., glog), but do not
	// warn if they contain underscores.
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.SetNormalizeFunc(apiserverflag.WordSepNormalizeFunc)
	rootCmd.PersistentFlags().AddFlagSet(pflag.CommandLine)

	// Warn for other flags that contain underscores.
	rootCmd.SetGlobalNormalizationFunc(apiserverflag.WarnWordSepNormalizeFunc)

	var shortVersion bool
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the version",
		Run: func(cmd *cobra.Command, args []string) {
			if shortVersion {
				fmt.Printf("%s\n", version.Get().GitVersion)
			} else {
				fmt.Printf("%#v\n", version.Get())
			}
		},
	}
	versionCmd.Flags().BoolVar(&shortVersion, "short", false, "Print just the version number.")

	rootCmd.AddCommand(standalone.NewCmdStandalone(out, clientcmd.NewDefaultPathOptions(), defaultServerImage, defaultEtcdImage))
	rootCmd.AddCommand(aggregated.NewCmdAggregated(out, clientcmd.NewDefaultPathOptions(), defaultServerImage, defaultEtcdImage))
	rootCmd.AddCommand(versionCmd)

	return rootCmd
}
