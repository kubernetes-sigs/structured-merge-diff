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

// Vector describes an individual test case. Test cases are exported for ease
// in applying them to various implementations.
type Vector interface {
	Name() string
	Valid() bool
	// Run the Vector on cur with schema to return the resulting object and
	// failing the test on unexpected conflicts
	Run(t *testing.T, cur YAMLObject, schema SchemaDefinition) YAMLObject
}

// TestVector is an implementation of Vector which just returns the defined ReturnObject
type TestVector struct {
	ValidVectorHelper
	VectorName   string     `yaml:"name"`
	ReturnObject YAMLObject `yaml:"returnObject"`
}

// Name of the current vector
func (v *TestVector) Name() string { return v.VectorName }

// Run the TestVector
func (v *TestVector) Run(t *testing.T, cur YAMLObject, schema SchemaDefinition) YAMLObject {
	return v.ReturnObject
}

// FailVector is an implementation of Vector which just returns the defined ReturnObject
type FailVector struct {
	ValidVectorHelper
	VectorName string `yaml:"name"`
}

// Name of the current vector
func (v *FailVector) Name() string { return v.VectorName }

// Run the FailVector
func (v *FailVector) Run(t *testing.T, cur YAMLObject, schema SchemaDefinition) YAMLObject {
	t.Errorf("Intentionally failing the test")
	return cur
}

// ValidVectorHelper is a helper for building test vectors to implement the Valid() function through embedding
type ValidVectorHelper struct{}

// Valid fakes the Valid implementation and always returns true
func (v *ValidVectorHelper) Valid() bool { return true }

// InvalidVectorHelper is a helper for building test vectors to implement the Valid() function through embedding
type InvalidVectorHelper struct{}

// Valid fakes the Valid implementation and always returns false
func (v *InvalidVectorHelper) Valid() bool { return false }
