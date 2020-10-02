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
	"context"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	client "sigs.k8s.io/kubefed/pkg/client/generic"
)

func GetTypeConfig(genericClient client.Client, name, namespace string) (typeconfig.Interface, error) {
	typeConfig := &fedv1b1.FederatedTypeConfig{}
	err := genericClient.Get(context.Background(), typeConfig, namespace, name)
	if err != nil {
		return nil, err
	}

	return typeConfig, nil
}

func UpdateTypeConfig(genericClient client.Client, typeConfig *fedv1b1.FederatedTypeConfig, namespace string) error {
	err := genericClient.Update(context.Background(), typeConfig)
	if err != nil {
		return err
	}

	return nil
}
