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

package testvectors

import (
	"testing"
)

// Action describes an individual test case. Test cases are exported for ease
// in applying them to various implementations.
type Action interface {
	Name() string
	Valid() bool
	// Run the Action on cur with schema to return the resulting object and
	// failing the test on unexpected conflicts
	Run(t *testing.T, cur YAMLObject, schema SchemaDefinition) YAMLObject
}

// TestAction is an implementation of Action which just returns the defined ReturnObject
type TestAction struct {
	ValidActionHelper
	ActionName   string     `yaml:"name"`
	ReturnObject YAMLObject `yaml:"returnObject"`
}

// Name of the current action
func (a *TestAction) Name() string { return a.ActionName }

// Run the TestAction
func (a *TestAction) Run(t *testing.T, cur YAMLObject, schema SchemaDefinition) YAMLObject {
	return a.ReturnObject
}

// FailAction is an implementation of Action which just returns the defined ReturnObject
type FailAction struct {
	ValidActionHelper
	ActionName string `yaml:"name"`
}

// Name of the current action
func (a *FailAction) Name() string { return a.ActionName }

// Run the FailAction
func (a *FailAction) Run(t *testing.T, cur YAMLObject, schema SchemaDefinition) YAMLObject {
	t.Errorf("Intentionally failing the test")
	return cur
}

// ValidActionHelper is a helper for building test actions to implement the Valid() function through embedding
type ValidActionHelper struct{}

// Valid fakes the Valid implementation and always returns true
func (a *ValidActionHelper) Valid() bool { return true }

// InvalidActionHelper is a helper for building test actions to implement the Valid() function through embedding
type InvalidActionHelper struct{}

// Valid fakes the Valid implementation and always returns false
func (a *InvalidActionHelper) Valid() bool { return false }
