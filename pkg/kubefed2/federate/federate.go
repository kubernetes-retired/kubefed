/*
Copyright 2019 The Kubernetes Authors.

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
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	ctlutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/enable"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
)

var (
	federate_long = `
		Federate creates a federated resource from a kubernetes resource.
		The target resource must exist in the cluster hosting the federation
		control plane. The control plane must have a FederatedTypeConfig
		for the type of the kubernetes resource. The new federated resource
		will be created with the same name and namespace (if namespaced) as
		the kubernetes resource.

		Current context is assumed to be a Kubernetes cluster hosting
		the federation control plane. Please use the --host-cluster-context
		flag otherwise.`

	federate_example = `
		# Federate resource named "my-dep" in namespace "my-ns" of type identified by FederatedTypeConfig "deployment.apps"
		kubefed2 federate deployment.apps my-dep -n "my-ns" --host-cluster-context=cluster1`
	// TODO(irfanurrehman): implement —contents flag applicable to namespaces
)

type federateResource struct {
	options.SubcommandOptions
	typeName          string
	resourceName      string
	resourceNamespace string
}

func (j *federateResource) Bind(flags *pflag.FlagSet) {
	flags.StringVarP(&j.resourceNamespace, "namespace", "n", "default", "The namespace of the resource to federate.")
}

// Complete ensures that options are valid.
func (j *federateResource) Complete(args []string) error {
	if len(args) == 0 {
		return errors.New("FEDERATED-TYPE-NAME is required")
	}
	j.typeName = args[0]

	if len(args) == 1 {
		return errors.New("RESOURCE-NAME is required")
	}
	j.resourceName = args[1]

	if j.typeName == ctlutil.NamespaceName {
		// TODO: irfanurrehman: Can a target namespace be federated into another namespace?
		klog.Infof("Resource to federate is a namespace. Given namespace will itself be the container for the federated namespace")
		j.resourceNamespace = ""
	}

	return nil
}

// NewCmdFederateResource defines the `federate` command that federates a
// Kubernetes resource of the given kubernetes type.
func NewCmdFederateResource(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &federateResource{}

	cmd := &cobra.Command{
		Use:     "federate FEDERATED-TYPE-NAME RESOURCE-NAME",
		Short:   "Federate creates a federated resource from a kubernetes resource",
		Long:    federate_long,
		Example: federate_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.Complete(args)
			if err != nil {
				klog.Fatalf("error: %v", err)
			}

			err = opts.Run(cmdOut, config)
			if err != nil {
				klog.Fatalf("error: %v", err)
			}
		},
	}

	flags := cmd.Flags()
	opts.CommonBind(flags)
	opts.Bind(flags)

	return cmd
}

// Run is the implementation of the `federate resource` command.
func (j *federateResource) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		return errors.Wrap(err, "Failed to get host cluster config")
	}

	qualifiedTypeName := ctlutil.QualifiedName{
		Namespace: j.FederationNamespace,
		Name:      j.typeName,
	}

	qualifiedResourceName := ctlutil.QualifiedName{
		Namespace: j.resourceNamespace,
		Name:      j.resourceName,
	}

	_, err = FederateResource(hostConfig, qualifiedTypeName, qualifiedResourceName, j.DryRun)
	return err
}

func FederateResource(hostConfig *rest.Config, qualifiedTypeName, qualifiedName ctlutil.QualifiedName, dryrun bool) (*unstructured.Unstructured, error) {
	typeConfig, err := lookupTypeDetails(hostConfig, qualifiedTypeName)
	if err != nil {
		return nil, err
	}

	targetResource, err := getTargetResource(hostConfig, typeConfig, qualifiedName)
	if err != nil {
		return nil, err
	}

	fedResource, err := FederatedResourceFromTargetResource(typeConfig, targetResource)
	if err != nil {
		return nil, errors.Wrapf(err, "Error getting %s from %s %q", typeConfig.GetFederatedType().Kind, typeConfig.GetTarget().Kind, qualifiedName)
	}

	return createFedResource(hostConfig, typeConfig, fedResource, qualifiedName, dryrun)
}

func lookupTypeDetails(config *rest.Config, qualifiedTypeName ctlutil.QualifiedName) (*fedv1a1.FederatedTypeConfig, error) {
	client, err := genericclient.New(config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get federation client")
	}

	typeConfig := &fedv1a1.FederatedTypeConfig{}
	err = client.Get(context.TODO(), typeConfig, qualifiedTypeName.Namespace, qualifiedTypeName.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving FederatedTypeConfig %q", qualifiedTypeName)
	}

	_, err = enable.LookupAPIResource(config, typeConfig.Name, typeConfig.APIVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving API resource for FederatedTypeConfig %q", qualifiedTypeName)
	}

	klog.Infof("FederatedTypeConfig: %q found", qualifiedTypeName)
	return typeConfig, nil
}

func getTargetResource(hostConfig *rest.Config, typeConfig *fedv1a1.FederatedTypeConfig, qualifiedName ctlutil.QualifiedName) (*unstructured.Unstructured, error) {
	targetAPIResource := typeConfig.GetTarget()
	targetClient, err := ctlutil.NewResourceClient(hostConfig, &targetAPIResource)
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating client for %s", targetAPIResource.Kind)
	}

	kind := targetAPIResource.Kind
	resource, err := targetClient.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving target %s %q", kind, qualifiedName)
	}

	klog.Infof("Target %s %q found", kind, qualifiedName)
	return resource, nil
}

func FederatedResourceFromTargetResource(typeConfig typeconfig.Interface, targetResource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	fedAPIResource := typeConfig.GetFederatedType()

	// Special handling is needed for some controller set fields.
	if typeConfig.GetTarget().Kind == ctlutil.ServiceAccountKind {
		unstructured.RemoveNestedField(targetResource.Object, ctlutil.SecretsField)
	}

	if typeConfig.GetTarget().Kind == ctlutil.ServiceKind {
		var targetPorts []interface{}
		targetPorts, ok, err := unstructured.NestedSlice(targetResource.Object, "spec", "ports")
		if err != nil {
			return nil, err
		}
		if ok {
			for index := range targetPorts {
				port := targetPorts[index].(map[string]interface{})
				delete(port, "nodePort")
				targetPorts[index] = port
			}
			err := unstructured.SetNestedSlice(targetResource.Object, targetPorts, "spec", "ports")
			if err != nil {
				return nil, err
			}
		}
		unstructured.RemoveNestedField(targetResource.Object, "spec", "clusterIP")
	}

	qualifiedName := ctlutil.NewQualifiedName(targetResource)
	resourceNamespace := getNamespace(typeConfig, qualifiedName)
	fedResource := &unstructured.Unstructured{}
	SetBasicMetaFields(fedResource, fedAPIResource, qualifiedName.Name, resourceNamespace, "")
	RemoveUnwantedFields(targetResource)

	err := unstructured.SetNestedField(fedResource.Object, targetResource.Object, ctlutil.SpecField, ctlutil.TemplateField)
	if err != nil {
		return nil, err
	}
	err = unstructured.SetNestedStringMap(fedResource.Object, map[string]string{}, ctlutil.SpecField, ctlutil.PlacementField, ctlutil.ClusterSelectorField, ctlutil.MatchLabelsField)
	if err != nil {
		return nil, err
	}

	return fedResource, err
}

func getNamespace(typeConfig typeconfig.Interface, qualifiedName ctlutil.QualifiedName) string {
	if typeConfig.GetTarget().Kind == ctlutil.NamespaceKind {
		return qualifiedName.Name
	}
	return qualifiedName.Namespace
}

func createFedResource(hostConfig *rest.Config, typeConfig *fedv1a1.FederatedTypeConfig, fedResource *unstructured.Unstructured, qualifiedName ctlutil.QualifiedName, dryrun bool) (*unstructured.Unstructured, error) {
	fedAPIResource := typeConfig.GetFederatedType()
	fedKind := fedAPIResource.Kind
	fedClient, err := ctlutil.NewResourceClient(hostConfig, &fedAPIResource)
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating client for %s", fedKind)
	}

	qualifiedFedName := ctlutil.NewQualifiedName(fedResource)
	var createdResource *unstructured.Unstructured = nil
	if !dryrun {
		createdResource, err = fedClient.Resources(fedResource.GetNamespace()).Create(fedResource, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "Error creating %s %q", fedKind, qualifiedFedName)
		}
	}

	klog.Infof("Successfully created a %s from %s %q", fedKind, typeConfig.GetTarget().Kind, qualifiedName)
	return createdResource, nil
}
