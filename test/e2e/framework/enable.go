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

package framework

import (
	kfenable "sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
	"sigs.k8s.io/kubefed/test/common"
)

func LoadEnableTypeDirectives(tl common.TestLogger) []*kfenable.EnableTypeDirective {
	enableTypeDirectives := []*kfenable.EnableTypeDirective{}
	for _, file := range common.AssetNames() {
		obj := kfenable.NewEnableTypeDirective()
		filename, err := common.DecodeYamlFromBindata(file, "config/enabletypedirectives/", obj)
		if err != nil {
			tl.Fatalf("Error loading EnableTypeDirective from this file %q: %v", filename, err)
		}
		if len(filename) != 0 {
			enableTypeDirectives = append(enableTypeDirectives, obj)
		}

	}
	return enableTypeDirectives
}
