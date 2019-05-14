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

package common

import (
	"fmt"

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	restclient "k8s.io/client-go/rest"

	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func CreateResource(kubeconfig *restclient.Config, apiResource metav1.APIResource, desiredObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	namespace := desiredObj.GetNamespace()
	kind := apiResource.Kind
	resourceMsg := kind
	if len(namespace) > 0 {
		resourceMsg = fmt.Sprintf("%s in namespace %q", resourceMsg, namespace)
	}

	client, err := util.NewResourceClient(kubeconfig, &apiResource)
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating resource client")
	}
	obj, err := client.Resources(namespace).Create(desiredObj, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating %s", resourceMsg)
	}

	return obj, nil
}
