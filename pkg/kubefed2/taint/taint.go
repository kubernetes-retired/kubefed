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

package taint

import (
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	taint_long = `
		Add or remove taints from federatedclusters.

		Federated resources will need to have an appropriate
		toleration to be propagated to the tainted cluster.`

	taint_example = `
		# Add a taint to cluster "foo"
		kubefed2 taint foo example=true:NoSchedule

		# Remove the taint
		kubefed2 taint foo example-

		# Remove a taint with a specific key/effect pair
		kubefed2 taint foo example:NoSchedule-`
)

type taintCluster struct {
	options.SubcommandOptions
	taintClusterOptions
}

type taintClusterOptions struct {
	overwrite bool
	taintsToAdd []corev1.Taint
	taintsToRemove []corev1.Taint
}

func (o *taintClusterOptions) Bind(flags *pflag.FlagSet) {
	flags.BoolVar(&o.overwrite, "overwrite", false,
		"Allow taints to be updated with new values.")
}

func NewCmdTaint(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &taintCluster{}

	cmd := &cobra.Command{
		Use:     "taint CLUSTER_NAME key=value:Effect --host-cluster-context=HOST_CONTEXT",
		Short:   "Add or remove taints from a federatedcluster",
		Long:    taint_long,
		Example: taint_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.Complete(args)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			err = opts.Run(cmdOut, config)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}
		},
	}

	flags := cmd.Flags()
	opts.CommonBind(flags)
	opts.Bind(flags)

	return cmd
}

func (t *taintCluster) Complete(args []string) error {
	err := t.SetName(args)
	if err != nil {
		return err
	}

	if len(args) < 2 {
		return fmt.Errorf("must supply at least one taint to add or remove")
	}

	if t.taintsToAdd, t.taintsToRemove, err = parseTaints(args[1:]); err != nil {
		return fmt.Errorf("error parsing taints: %v", err)
	}

	var conflictTaints []string
	for _, adding := range t.taintsToAdd {
		for _, removing := range t.taintsToRemove {
			if adding.Key != removing.Key {
				continue
			}
			if len(removing.Effect) == 0 || adding.Effect == removing.Effect {
				conflictTaint := fmt.Sprintf("{\"%s\":\"%s\"}", removing.Key, adding.Effect)
				conflictTaints = append(conflictTaints, conflictTaint)
			}
		}
	}

	if len(conflictTaints) > 0 {
		return fmt.Errorf("can not both modify and remove the following taint(s) in the same command: %s", strings.Join(conflictTaints, ", "))
	}

	return nil
}


func (t *taintCluster) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(t.HostClusterContext, t.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get host cluster config: %v", err)
	}

	fedClientset, err := util.FedClientset(hostConfig)
	if err != nil {
		return fmt.Errorf("failed to get federation clientset: %v", err)
	}

	clusterToTaint, err := fedClientset.CoreV1alpha1().FederatedClusters(t.FederationNamespace).Get(t.ClusterName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("could not retrieve cluster: %v", err)
	}

	if !t.overwrite {
		if exists := checkIfTaintsAlreadyExist(clusterToTaint.Spec.Taints, t.taintsToAdd); len(exists) != 0 {
			return fmt.Errorf("cluster already has %s taint(s) with same effect(s) and --overwrite is false", exists)
		}
	}

	updatedTaints, err := applyTaints(clusterToTaint.Spec.Taints, t.taintsToAdd, t.taintsToRemove)
	if err != nil {
		return fmt.Errorf("could not update set of taints: %v", err)
	}
	clusterToTaint.Spec.Taints = updatedTaints

	if !t.DryRun {
		_, err = fedClientset.CoreV1alpha1().FederatedClusters(t.FederationNamespace).Update(clusterToTaint)
		if err != nil {
			return fmt.Errorf("could not update cluster: %v", err)
		}
	}

	return nil
}