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

package options

import "fmt"

// Validate validates the options in the receiver.
func (options *serverRunOptions) Validate() []error {
	var errors []error
	if errs := options.genericServerRunOptions.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := options.etcd.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := options.secureServing.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := options.audit.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := options.features.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if options.eventTTL <= 0 {
		errors = append(errors, fmt.Errorf("--event-ttl must be greater than 0"))
	}
	return errors
}
