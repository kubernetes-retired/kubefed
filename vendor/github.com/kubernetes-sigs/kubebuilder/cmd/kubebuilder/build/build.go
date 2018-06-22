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

package build

import (
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Command group for building source into artifacts.",
	Long:  `Command group for building source into artifacts.`,
	Example: `# Generate code and build the apiserver and controller-manager binaries into bin/
kubebuilder build executables

# Rebuild generated code
kubebuilder build generated
`,
	Run: RunBuild,
}

func AddBuild(cmd *cobra.Command) {
	cmd.AddCommand(buildCmd)

	AddBuildExecutables(buildCmd)
	//	AddBuildContainer(buildCmd)
	AddDocs(buildCmd)
	AddGenerate(buildCmd)
}

func RunBuild(cmd *cobra.Command, args []string) {
	cmd.Help()
}
