/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses\/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// kubefed2 is a tool for managing clusters in a federation.
package main

import (
	"fmt"
	"os"

	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2"
	"k8s.io/apiserver/pkg/util/logs"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Load all client auth plugins for GCP, Azure, etc
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	err := kubefed2.NewKubeFed2Command(os.Stdout).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
