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
	"reflect"

	"sigs.k8s.io/structured-merge-diff/fieldpath"
	"sigs.k8s.io/structured-merge-diff/schema"
	"sigs.k8s.io/structured-merge-diff/value"
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

// Merge returns the result of merging tv and pso ("partially specified
// object") together. Of note:
//  * No fields can be removed by this operation.
//  * If both tv and pso specify a given leaf field, the result will keep pso's
//    value.
//  * Container typed elements will have their items ordered:
//    * like tv, if pso doesn't change anything in the container
//    * like pso, if pso does change something in the container.
// tv and pso must both be of the same type (their Schema and TypeRef must
// match), or an error will be returned. Validation errors will be returned if
// the objects don't conform to the schema.
func (tv TypedValue) Merge(pso TypedValue) (TypedValue, error) {
	if tv.schema != pso.schema {
		return TypedValue{}, errorFormatter{}.
			errorf("expected objects with types from the same schema")
	}
	if !reflect.DeepEqual(tv.typeRef, pso.typeRef) {
		return TypedValue{}, errorFormatter{}.
			errorf("expected objects of the same type, but got %v and %v", tv.typeRef, pso.typeRef)
	}

	mw := mergingWalker{
		lhs:     &tv.value,
		rhs:     &pso.value,
		schema:  tv.schema,
		typeRef: tv.typeRef,

		rule: ruleKeepRHS,
	}
	errs := mw.merge()
	if len(errs) > 0 {
		return TypedValue{}, errs
	}

	out := TypedValue{
		schema:  tv.schema,
		typeRef: tv.typeRef,
	}
	if mw.out == nil {
		out.value = value.Value{Null: true}
	} else {
		out.value = *mw.out
	}
	return out, nil
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
