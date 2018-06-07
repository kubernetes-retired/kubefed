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
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	apiserverflag "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/install"
	clusterregistryv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	clientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
	informers "k8s.io/cluster-registry/pkg/client/informers_generated/externalversions"
	"k8s.io/cluster-registry/pkg/clusterregistry/options"
	"k8s.io/cluster-registry/pkg/version"
)

// NewClusterRegistryCommand creates the 'clusterregistry' command.
func NewClusterRegistryCommand(out io.Writer) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "clusterregistry",
		Short: "clusterregistry runs the cluster registry API server",
		Long:  "clusterregistry is the executable that runs the cluster registry apiserver.",
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
		Short: "Prints the version information and exits",
		Run: func(cmd *cobra.Command, args []string) {
			if shortVersion {
				fmt.Printf("%s\n", version.Get().GitVersion)
			} else {
				fmt.Printf("%#v\n", version.Get())
			}
		},
	}
	versionCmd.Flags().BoolVar(&shortVersion, "short", false, "Print just the version number.")

	rootCmd.AddCommand(NewCmdAggregated(out, clientcmd.NewDefaultPathOptions()))
	rootCmd.AddCommand(NewCmdStandalone(out, clientcmd.NewDefaultPathOptions()))
	rootCmd.AddCommand(versionCmd)

	return rootCmd
}

// Run runs the cluster registry API server. It only returns if stopCh is closed
// or one of the ports cannot be listened on initially.
func Run(s options.Options, stopCh <-chan struct{}) error {
	err := NonBlockingRun(s, stopCh)
	if err != nil {
		return err
	}
	<-stopCh
	return nil
}

// NonBlockingRun runs the cluster registry API server and configures it to
// stop with the given channel.
func NonBlockingRun(s options.Options, stopCh <-chan struct{}) error {
	server, err := CreateServer(s)
	if err != nil {
		return err
	}

	return server.PrepareRun().NonBlockingRun(stopCh)
}

// CreateServer creates a cluster registry API server.
func CreateServer(s options.Options) (*genericapiserver.GenericAPIServer, error) {
	// set defaults
	if err := s.GenericServerRunOptions().DefaultAdvertiseAddress(s.SecureServing()); err != nil {
		return nil, err
	}

	if err := s.SecureServing().MaybeDefaultWithSelfSignedCerts(s.GenericServerRunOptions().AdvertiseAddress.String(), nil, nil); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	if errs := s.Validate(); len(errs) != 0 {
		return nil, utilerrors.NewAggregate(errs)
	}

	genericConfig := genericapiserver.NewConfig(install.Codecs)
	version := version.Get()
	genericConfig.Version = &version

	genericConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(clusterregistryv1alpha1.GetOpenAPIDefinitions, install.Scheme)
	genericConfig.OpenAPIConfig.Info.Title = "Cluster Registry"
	genericConfig.OpenAPIConfig.Info.Version = strings.Split(genericConfig.Version.String(), "-")[0]
	genericConfig.OpenAPIConfig.Info.License = &spec.License{
		Name: "Apache License, Version 2.0",
		URL:  fmt.Sprintf("https://github.com/kubernetes/cluster-registry/blob/%s/LICENSE", genericConfig.OpenAPIConfig.Info.Version),
	}

	if err := s.GenericServerRunOptions().ApplyTo(genericConfig); err != nil {
		return nil, err
	}
	if err := s.SecureServing().ApplyTo(genericConfig); err != nil {
		return nil, err
	}
	if err := s.ApplyAuthentication(genericConfig); err != nil {
		return nil, err
	}
	if err := s.ApplyAuthorization(genericConfig); err != nil {
		return nil, err
	}
	if err := s.Audit().ApplyTo(genericConfig); err != nil {
		return nil, err
	}
	if err := s.Features().ApplyTo(genericConfig); err != nil {
		return nil, err
	}

	resourceConfig := defaultResourceConfig()

	if s.Etcd().StorageConfig.DeserializationCacheSize == 0 {
		// When size of cache is not explicitly set, set it to 50000
		s.Etcd().StorageConfig.DeserializationCacheSize = 50000
	}

	storageFactory := serverstorage.NewDefaultStorageFactory(
		s.Etcd().StorageConfig, s.Etcd().DefaultStorageMediaType, install.Codecs,
		serverstorage.NewDefaultResourceEncodingConfig(install.Registry),
		resourceConfig, nil,
	)

	for _, override := range s.Etcd().EtcdServersOverrides {
		tokens := strings.Split(override, "#")
		if len(tokens) != 2 {
			glog.Errorf("invalid value of etcd server overrides: %s", override)
			continue
		}

		apiresource := strings.Split(tokens[0], "/")
		if len(apiresource) != 2 {
			glog.Errorf("invalid resource definition: %s", tokens[0])
			continue
		}
		group := apiresource[0]
		resource := apiresource[1]
		groupResource := schema.GroupResource{Group: group, Resource: resource}

		servers := strings.Split(tokens[1], ";")
		storageFactory.SetEtcdLocation(groupResource, servers)
	}
	if err := s.Etcd().ApplyWithStorageFactoryTo(storageFactory, genericConfig); err != nil {
		return nil, err
	}

	genericConfig.LoopbackClientConfig.ContentConfig.ContentType = "application/vnd.kubernetes.protobuf"

	client, err := clientset.NewForConfig(genericConfig.LoopbackClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	m, err := genericConfig.Complete(nil).New("clusterregistry", genericapiserver.EmptyDelegate)
	if err != nil {
		return nil, err
	}

	apiResourceConfigSource := storageFactory.APIResourceConfigSource
	installClusterAPIs(m, genericConfig.RESTOptionsGetter, apiResourceConfigSource)

	sharedInformers := informers.NewSharedInformerFactory(client, genericConfig.LoopbackClientConfig.Timeout)
	m.AddPostStartHook("start-informers", func(context genericapiserver.PostStartHookContext) error {
		sharedInformers.Start(context.StopCh)
		return nil
	})
	return m, nil
}

func defaultResourceConfig() *serverstorage.ResourceConfig {
	rc := serverstorage.NewResourceConfig()
	rc.EnableVersions(clusterregistryv1alpha1.SchemeGroupVersion)
	return rc
}
