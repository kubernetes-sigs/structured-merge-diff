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

import (
	"sigs.k8s.io/structured-merge-diff/merge"
	"sigs.k8s.io/structured-merge-diff/typed"
)

// State of the current test in terms of live object. One can check at
// any time that Live and Owners match the expectations.
type State struct {
	Live    typed.TypedValue
	Owners  merge.Owners
	Updater *merge.Updater
}

// Update the current state with the passed in object
func (s *State) Update(obj typed.TypedValue, owner string) error {
	if s.Owners == nil {
		s.Owners = merge.Owners{}
	}
	owners, err := s.Updater.Update(s.Live, obj, s.Owners, owner)
	if err != nil {
		return err
	}
	s.Live = obj
	s.Owners = owners

	return nil
}

// Apply the passed in object to the current state
func (s *State) Apply(obj typed.TypedValue, owner string, force bool) error {
	if s.Owners == nil {
		s.Owners = merge.Owners{}
	}
	new, owners, err := s.Updater.Apply(s.Live, obj, s.Owners, owner, force)
	if err != nil {
		return err
	}
	s.Live = new
	s.Owners = owners

	return nil
}
