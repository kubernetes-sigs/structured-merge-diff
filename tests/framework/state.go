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

// YAMLObject is an object encoded in YAML.
type YAMLObject string

// State of the current test in terms of live object
type State struct {
	Live           YAMLObject `yaml:"live"`
	Implementation Implementation
}

// Apply the passed in object to the current state
func (s *State) Apply(obj YAMLObject, workflow string, force bool) (err error) {
	s.Live, err = s.Implementation.Apply(s.Live, obj, workflow, force)
	return err
}

// Update the current state with the passed in object
func (s *State) Update(obj YAMLObject, workflow string) (err error) {
	s.Live, err = s.Implementation.NonApply(s.Live, obj, workflow)
	return err
}

// Patch the current state with the passed in object
func (s *State) Patch(obj YAMLObject, workflow string) (err error) {
	s.Live, err = s.Implementation.NonApply(s.Live, obj, workflow)
	return err
}
