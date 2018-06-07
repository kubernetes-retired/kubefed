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

// Package options contains flags, common options, and aggregated and
// standalone options for initializing the cluster registry API server.
//
// The cluster registry can be run in two different modes: standalone and
// aggregated. The cluster registry implementation is mostly agnostic to the
// mode in which it is run, except where authentication and authorization is
// concerned. In aggregated mode, the cluster registry delgates its authn/z to
// another API server; in standalone mode, the cluster registry provides a
// suite of authn/z methods such as basic auth, token auth and client certs.
package options
