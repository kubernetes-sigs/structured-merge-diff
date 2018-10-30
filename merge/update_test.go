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

package merge_test

import (
	"fmt"
	"testing"

	"sigs.k8s.io/structured-merge-diff/merge"
	"sigs.k8s.io/structured-merge-diff/typed"
)

// State of the current test in terms of live object. One can check at
// any time that Live and Managers match the expectations.
type State struct {
	Live   *typed.TypedValue
	Parser *typed.Parser
	// Typename is the typename used to create objects in the
	// schema.
	Typename string
	Managers merge.ManagedFields
	Updater  *merge.Updater
}

func (s *State) ObjectFactory() *typed.ParseableType {
	return s.Parser.Type(s.Typename)
}

func (s *State) checkInit() error {
	if s.Live == nil {
		obj, err := s.ObjectFactory().New()
		if err != nil {
			return fmt.Errorf("failed to create new empty object: %v", err)
		}
		s.Live = &obj
	}
	return nil
}

// Update the current state with the passed in object
func (s *State) Update(obj typed.YAMLObject, manager string) error {
	if err := s.checkInit(); err != nil {
		return err
	}
	tv, err := s.ObjectFactory().FromYAML(obj)
	managers, err := s.Updater.Update(*s.Live, tv, s.Managers, manager)
	if err != nil {
		return err
	}
	s.Live = &tv
	s.Managers = managers

	return nil
}

// Apply the passed in object to the current state
func (s *State) Apply(obj typed.YAMLObject, manager string, force bool) error {
	if err := s.checkInit(); err != nil {
		return err
	}
	tv, err := s.ObjectFactory().FromYAML(obj)
	if err != nil {
		return err
	}
	new, managers, err := s.Updater.Apply(*s.Live, tv, s.Managers, manager, force)
	if err != nil {
		return err
	}
	s.Live = &new
	s.Managers = managers

	return nil
}

// CompareLive takes a YAML string and returns the comparison with the
// current live object or an error.
func (s *State) CompareLive(obj typed.YAMLObject) (*typed.Comparison, error) {
	if err := s.checkInit(); err != nil {
		return nil, err
	}
	tv, err := s.ObjectFactory().FromYAML(obj)
	if err != nil {
		return nil, err
	}
	return s.Live.Compare(tv)
}

// dummyConverter doesn't convert, it just returns the same exact object no matter what.
type dummyConverter struct{}

// Convert returns the object given in input, not doing any conversion.
func (dummyConverter) Convert(v typed.TypedValue, version merge.APIVersion) (typed.TypedValue, error) {
	return v, nil
}

// TestExample shows how to use the test framework
func TestExample(t *testing.T) {
	parser, err := typed.NewParser(`types:
- name: lists
  struct:
    fields:
    - name: list
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative`)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	state := &State{
		Updater:  &merge.Updater{},
		Parser:   parser,
		Typename: "lists",
	}

	config := typed.YAMLObject(`
list:
- a
- b
- c
`)
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- a
- b
- c
- d`)
	err = state.Apply(config, "default", false)

	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	// The following is wrong because the code doesn't work yet.
	_, err = state.CompareLive(config)
	if err != nil {
		t.Fatalf("Failed to compare live with config: %v", err)
	}
}
