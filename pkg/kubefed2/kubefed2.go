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
	"flag"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apiserverflag "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/federate"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
)

// NewKubeFed2Command creates the `kubefed2` command and its nested children.
func NewKubeFed2Command(out io.Writer) *cobra.Command {
	// Parent command to which all subcommands are added.
	rootCmd := &cobra.Command{
		Use:   "kubefed2",
		Short: "kubefed2 controls a Kubernetes Cluster Federation",
		Long:  "kubefed2 controls a Kubernetes Cluster Federation. Find more information at https://github.com/kubernetes-sigs/federation-v2.",

		Run: runHelp,
	}

	// Add the command line flags from other dependencies (e.g., glog), but do not
	// warn if they contain underscores.
	pflag.CommandLine.SetNormalizeFunc(apiserverflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	rootCmd.PersistentFlags().AddFlagSet(pflag.CommandLine)

	// From this point and forward we get warnings on flags that contain "_" separators
	rootCmd.SetGlobalNormalizationFunc(apiserverflag.WarnWordSepNormalizeFunc)

	// Prevent glog errors about logging before parsing.
	flag.CommandLine.Parse(nil)

	fedConfig := util.NewFedConfig(clientcmd.NewDefaultPathOptions())
	rootCmd.AddCommand(federate.NewCmdFederate(out, fedConfig))
	rootCmd.AddCommand(NewCmdJoin(out, fedConfig))
	rootCmd.AddCommand(NewCmdUnjoin(out, fedConfig))
	rootCmd.AddCommand(NewCmdVersion(out))

	return rootCmd
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}
