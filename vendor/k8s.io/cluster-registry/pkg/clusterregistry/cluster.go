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
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/storage"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/install"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	clusteretcd "k8s.io/cluster-registry/pkg/registry/cluster/etcd"
)

func installClusterAPIs(g *genericapiserver.GenericAPIServer, optsGetter generic.RESTOptionsGetter, apiResourceConfigSource storage.APIResourceConfigSource) {
	clusterStorage, err := clusteretcd.NewREST(optsGetter, install.Scheme)
	if err != nil {
		glog.Fatalf("Error in creating cluster storage: %v", err)
	}
	resources := map[string]rest.Storage{
		"clusters": clusterStorage,
	}
	clusterregistryGroupMeta := install.Registry.GroupOrDie(v1alpha1.GroupName)

	apiGroupInfo := genericapiserver.APIGroupInfo{
		GroupMeta: *clusterregistryGroupMeta,
		VersionedResourcesStorageMap: map[string]map[string]rest.Storage{
			"v1alpha1": resources,
		},
		OptionsExternalVersion: &clusterregistryGroupMeta.GroupVersion,
		Scheme:                 install.Scheme,
		ParameterCodec:         metav1.ParameterCodec,
		NegotiatedSerializer:   install.Codecs,
	}
	if err := g.InstallAPIGroup(&apiGroupInfo); err != nil {
		glog.Fatalf("Error in registering group versions: %v", err)
	}
}
