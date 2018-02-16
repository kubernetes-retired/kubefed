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

package kubefnord

import (
	"flag"
	"io"

	"github.com/marun/fnord/pkg/kubefnord/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apiserverflag "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/client-go/tools/clientcmd"
)

// NewKubeFnordCommand creates the `kubefnord` command and its nested children.
func NewKubeFnordCommand(out io.Writer) *cobra.Command {
	// Parent command to which all subcommands are added.
	rootCmd := &cobra.Command{
		Use:   "kubefnord",
		Short: "kubefnord controls a Kubernetes Cluster Federation",
		Long:  "kubefnord controls a Kubernetes Cluster Federation",

		Run: runHelp,
	}

	// Add the command line flags from other dependencies (e.g., glog), but do not
	// warn if they contain underscores.
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.SetNormalizeFunc(apiserverflag.WordSepNormalizeFunc)
	rootCmd.PersistentFlags().AddFlagSet(pflag.CommandLine)

	// From this point and forward we get warnings on flags that contain "_" separators
	rootCmd.SetGlobalNormalizationFunc(apiserverflag.WarnWordSepNormalizeFunc)

	rootCmd.AddCommand(NewCmdJoin(out, util.NewFedConfig(clientcmd.NewDefaultPathOptions())))
	return rootCmd
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}
