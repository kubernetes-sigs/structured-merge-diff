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

package framework

import "sigs.k8s.io/structured-merge-diff/typed"

// State of the current test in terms of live object
type State struct {
	Live           typed.YAMLObject `yaml:"live"`
	Implementation Implementation
}

// Apply the passed in object to the current state
func (s *State) Apply(obj typed.YAMLObject, workflow string, force bool) error {
	new, err := s.Implementation.Apply(s.Live, obj, workflow, force)
	if err == nil {
		s.Live = new
	}
	return err
}

// Update the current state with the passed in object
func (s *State) Update(obj typed.YAMLObject, workflow string) error {
	new, err := s.Implementation.Update(s.Live, obj, workflow)
	if err == nil {
		s.Live = new
	}
	return err
}
