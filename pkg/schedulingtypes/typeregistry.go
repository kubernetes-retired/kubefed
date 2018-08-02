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

package schedulingtypes

import (
	"fmt"
)

type SchedulingType struct {
	Kind             string
	SchedulerFactory SchedulerFactory
}

var typeRegistry = make(map[string]SchedulingType)

func RegisterSchedulingType(kind string, factory SchedulerFactory) {
	_, ok := typeRegistry[kind]
	if ok {
		panic(fmt.Sprintf("Scheduler type %q has already been registered", kind))
	}
	typeRegistry[kind] = SchedulingType{
		Kind:             kind,
		SchedulerFactory: factory,
	}
}

func SchedulingTypes() map[string]SchedulingType {
	result := make(map[string]SchedulingType)
	for key, value := range typeRegistry {
		result[key] = value
	}
	return result
}

func GetSchedulerFactory(typ string) SchedulerFactory {
	return typeRegistry[typ].SchedulerFactory
}
