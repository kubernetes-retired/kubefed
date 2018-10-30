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
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	apicommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
)

var (
	enable_long = `
		Enables a Kubernetes API type (including a CRD) to be propagated
		to members of a federation.  Federation primitives will be
		generated as CRDs and a FederatedTypeConfig will be created to
		configure a sync controller.

		Current context is assumed to be a Kubernetes cluster hosting
		the federation control plane. Please use the
		--host-cluster-context flag otherwise.`

	enable_example = `
		# Enable federation of Services with service type overrideable
		kubefed2 federate enable Service --override-paths=spec.type --host-cluster-context=cluster1`
)

type enableType struct {
	options.SubcommandOptions
	enableTypeOptions
}

type enableTypeOptions struct {
	targetName         string
	rawComparisonField string
	comparisonField    apicommon.VersionComparisonField
	rawOverridePaths   string
	overridePaths      []string
	templateVersion    string
	templateGroup      string
	useExistingCRDs    bool
}

// Bind adds the join specific arguments to the flagset passed in as an
// argument.
func (o *enableTypeOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.rawComparisonField, "comparison-field", string(apicommon.ResourceVersionField),
		fmt.Sprintf("The field in the target type to compare for equality. Valid values are %q (default) and %q.",
			apicommon.ResourceVersionField, apicommon.GenerationField,
		),
	)
	flags.StringVar(&o.rawOverridePaths, "override-paths", "", "A common-separated list of dot-separated paths (e.g. spec.completions,spec.parallelism).")
	flags.StringVar(&o.templateGroup, "template-group", "generated.federation.k8s.io", "The name of the API group of the target API type.")
	flags.StringVar(&o.templateVersion, "template-version", "v1alpha1", "The API version of the target API type.")
	flags.BoolVar(&o.useExistingCRDs, "use-existing-crds", false, "Whether to reuse existing primitive CRDs.")
}

// NewCmdFederateEnable defines the `federate enable` command that
// enables federation of a Kubernetes API type.
func NewCmdFederateEnable(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &enableType{}

	cmd := &cobra.Command{
		Use:     "enable NAME",
		Short:   "Enables propagation of a Kubernetes API type",
		Long:    enable_long,
		Example: enable_example,
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

// Complete ensures that options are valid and marshals them if necessary.
func (j *enableType) Complete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("NAME is required")
	}
	j.targetName = args[0]

	if j.rawComparisonField == string(apicommon.ResourceVersionField) ||
		j.rawComparisonField == string(apicommon.GenerationField) {
		j.comparisonField = apicommon.VersionComparisonField(j.rawComparisonField)
	} else {
		return fmt.Errorf("--comparison-field must be %q or %q",
			apicommon.ResourceVersionField, apicommon.GenerationField,
		)
	}
	if len(j.rawOverridePaths) > 0 {
		j.overridePaths = strings.Split(j.rawOverridePaths, ",")
	}
	if len(j.templateGroup) == 0 {
		return fmt.Errorf("--template-group is a mandatory parameter ")
	}
	if len(j.templateVersion) == 0 {
		return fmt.Errorf("--template-version is a mandatory parameter")
	}

	return nil
}

// Run is the implementation of the `federate enable` command.
func (j *enableType) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		return fmt.Errorf("Failed to get host cluster config: %v", err)
	}

	_, err = EnableFederation(hostConfig, j.FederationNamespace, j.targetName, j.templateGroup,
		j.templateVersion, j.comparisonField, j.overridePaths, j.useExistingCRDs, j.DryRun)
	if err != nil {
		return err
	}

	return nil
}

// TODO(marun) Allow updates to the configuration for a type that has
// already been enabled for federation.  This would likely involve
// updating the version of the target type and the validation of the schema.
func EnableFederation(config *rest.Config, federationNamespace, key, templateGroup,
	templateVersion string, comparisonField apicommon.VersionComparisonField,
	overridePaths []string, useExisting, dryRun bool) (typeconfig.Interface, error) {

	apiResource, err := LookupAPIResource(config, key)
	if err != nil {
		return nil, err
	}
	glog.V(2).Infof("Found resource %q", resourceKey(*apiResource))

	typeConfig := typeConfigForTarget(*apiResource, comparisonField, overridePaths, templateGroup, templateVersion)

	if dryRun {
		// Avoid mutating the API
		return nil, nil
	}

	crdClient, err := apiextv1b1client.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create crd clientset: %v", err)
	}
	// TODO(marun) Retrieve the validation schema of the target and
	// use it in constructing the schema for the template.
	err = createPrimitives(crdClient, typeConfig, useExisting)
	if err != nil {
		return nil, err
	}

	fedClient, err := util.FedClientset(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to get federation clientset: %v", err)
	}
	concreteTypeConfig := typeConfig.(*fedv1a1.FederatedTypeConfig)
	createdTypeConfig, err := fedClient.CoreV1alpha1().FederatedTypeConfigs(federationNamespace).Create(concreteTypeConfig)
	if err != nil {
		return nil, fmt.Errorf("Error creating FederatedTypeConfig %q: %v", concreteTypeConfig.Name, err)
	}

	return createdTypeConfig, nil
}

func typeConfigForTarget(apiResource metav1.APIResource, comparisonField apicommon.VersionComparisonField, overridePaths []string, templateGroup, templateVersion string) typeconfig.Interface {
	kind := apiResource.Kind
	typeConfig := &fedv1a1.FederatedTypeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: groupQualifiedName(apiResource),
		},
		Spec: fedv1a1.FederatedTypeConfigSpec{
			Target: fedv1a1.APIResource{
				Version: apiResource.Version,
				Kind:    kind,
			},
			Namespaced:         apiResource.Namespaced,
			ComparisonField:    comparisonField,
			PropagationEnabled: true,
			Template: fedv1a1.APIResource{
				Group:   templateGroup,
				Version: templateVersion,
				Kind:    fmt.Sprintf("Federated%s", kind),
			},
			Placement: fedv1a1.APIResource{
				Kind: fmt.Sprintf("Federated%sPlacement", kind),
			},
		},
	}
	if len(overridePaths) > 0 {
		typeConfig.Spec.Override = &fedv1a1.APIResource{
			Kind: fmt.Sprintf("Federated%sOverride", kind),
		}
		specPaths := []fedv1a1.OverridePath{}
		for _, overridePath := range overridePaths {
			specPaths = append(specPaths, fedv1a1.OverridePath{Path: overridePath})
		}
		typeConfig.Spec.OverridePaths = specPaths
	}
	// Set defaults that would normally be set by the api
	fedv1a1.SetFederatedTypeConfigDefaults(typeConfig)
	return typeConfig
}

func createPrimitives(client *apiextv1b1client.ApiextensionsV1beta1Client, typeConfig typeconfig.Interface, useExisting bool) error {
	err := createCrdFromResource(client, typeConfig.GetTemplate(), useExisting)
	if err != nil {
		return err
	}
	err = createCrdFromResource(client, typeConfig.GetPlacement(), useExisting)
	if err != nil {
		return err
	}
	overrideAPIResource := typeConfig.GetOverride()
	if overrideAPIResource == nil {
		return nil
	}
	return createCrdFromResource(client, *overrideAPIResource, useExisting)
}

func createCrdFromResource(client *apiextv1b1client.ApiextensionsV1beta1Client, apiResource metav1.APIResource, useExisting bool) error {
	crd := CrdForAPIResource(apiResource)
	_, err := client.CustomResourceDefinitions().Create(crd)
	// TODO(marun) Update the crd to ensure the validation schema can be updated to the latest target type
	if err == nil || useExisting && errors.IsAlreadyExists(err) {
		return nil
	}
	return fmt.Errorf("Error creating CRD %q: %v", crd.Name, err)
}
