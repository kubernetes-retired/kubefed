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

package standalone

import (
	"io"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/cluster-registry/pkg/crinit/common"
	"k8s.io/cluster-registry/pkg/crinit/util"
)

var (
	longDeleteCommandDescription = `
	Delete deletes a standalone cluster registry.

	The standalone cluster registry is hosted inside a Kubernetes
	cluster but handles its own authentication and authorization.
	The host cluster must be specified using the
        --host-cluster-context flag.`
	deleteCommandExample = `
	# Delete a standalone cluster registry named foo
	# in the host cluster whose local kubeconfig
	# context is bar.
	crinit standalone delete foo --host-cluster-context=bar`
)

// newSubCmdDelete defines the `delete` subcommand to remove a cluster registry
// inside a host Kubernetes cluster.
func newSubCmdDelete(cmdOut io.Writer, pathOptions *clientcmd.PathOptions) *cobra.Command {
	opts := &standaloneClusterRegistryOptions{}

	delCmd := &cobra.Command{
		Use:     "delete CLUSTER_REGISTRY_NAME --host-cluster-context=HOST_CONTEXT",
		Short:   "Delete a standalone cluster registry",
		Long:    longDeleteCommandDescription,
		Example: deleteCommandExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.SetName(args)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			hostConfig, err := util.GetClientConfig(pathOptions, opts.Host, opts.Kubeconfig).ClientConfig()
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			hostClientset, err := client.NewForConfig(hostConfig)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			err = runDelete(opts, cmdOut, hostClientset, pathOptions)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}
		},
	}

	flags := delCmd.Flags()
	opts.BindCommon(flags)
	opts.BindCommonDelete(flags)
	return delCmd
}

// runDelete deletes a cluster registry.
func runDelete(opts *standaloneClusterRegistryOptions, cmdOut io.Writer,
	hostClientset client.Interface, pathOptions *clientcmd.PathOptions) error {

	// Only necessary to delete the cluster registry namespace and the
	// kubeconfig entry.

	err := common.DeleteKubeconfigEntry(cmdOut, pathOptions, opts.Name,
		opts.Kubeconfig, opts.DryRun, opts.IgnoreErrors)
	if err != nil {
		if !opts.IgnoreErrors {
			return err
		} else {
			glog.Infof("error: %v", err)
		}
	}

	err = common.DeleteNamespace(cmdOut, hostClientset,
		opts.ClusterRegistryNamespace, opts.DryRun)
	if err != nil {
		if !opts.IgnoreErrors {
			return err
		} else {
			glog.Infof("error: %v", err)
		}
	}

	err = common.WaitForClusterRegistryDeletion(cmdOut, hostClientset,
		opts.ClusterRegistryNamespace, opts.DryRun)
	if err != nil {
		if !opts.IgnoreErrors {
			return err
		} else {
			glog.Infof("error: %v", err)
		}
	}

	return nil
}
