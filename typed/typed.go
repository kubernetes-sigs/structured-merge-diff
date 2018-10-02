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

package typed

import (
	"github.com/kubernetes-sigs/structured-merge-diff/fieldpath"
	"github.com/kubernetes-sigs/structured-merge-diff/schema"
	"github.com/kubernetes-sigs/structured-merge-diff/value"
)

// TypedValue is a value of some specific type.
type TypedValue struct {
	value   value.Value
	typeRef schema.TypeRef
	schema  *schema.Schema
}

// AsTyped accepts a value and a type and returns a TypedValue. 'v' must have
// type 'typeName' in the schema. An error is returned if the v doesn't conform
// to the schema.
func AsTyped(v value.Value, s *schema.Schema, typeName string) (TypedValue, error) {
	tv := TypedValue{
		value:   v,
		typeRef: schema.TypeRef{NamedType: &typeName},
		schema:  s,
	}
	if err := tv.Validate(); err != nil {
		return TypedValue{}, err
	}
	return tv, nil
}

// Validate returns an error with a list of every spec violation.
func (tv TypedValue) Validate() error {
	if errs := tv.walker().validate(); len(errs) != 0 {
		return errs
	}
	return nil
}

// ToFieldSet creates a set containing every leaf field mentioned in tv, or
// validation errors, if any were encountered.
func (tv TypedValue) ToFieldSet() (*fieldpath.Set, error) {
	s := fieldpath.NewSet()
	w := tv.walker()
	w.leafFieldCallback = func(p fieldpath.Path) { s.Insert(p) }
	if errs := w.validate(); len(errs) != 0 {
		return nil, errs
	}
	return s, nil
}

// AsTypeUnvalidated is just like WithType, but doesn't validate that the type
// conforms to the schema, for cases where that has already been checked or
// where you're going to call a method that validates as a side-effect (like
// ToFieldSet).
func AsTypedUnvalidated(v value.Value, s *schema.Schema, typeName string) TypedValue {
	tv := TypedValue{
		value:   v,
		typeRef: schema.TypeRef{NamedType: &typeName},
		schema:  s,
	}
	return tv
}
