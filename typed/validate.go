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
	"errors"
	"fmt"
	"strings"

	"github.com/kubernetes-sigs/structured-merge-diff/fieldpath"
	"github.com/kubernetes-sigs/structured-merge-diff/schema"
	"github.com/kubernetes-sigs/structured-merge-diff/value"
)

// ValidationError reports an error about a particular field
type ValidationError struct {
	Path         fieldpath.Path
	ErrorMessage string
}

// Error returns a human readable error message.
func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %v", ve.Path, ve.ErrorMessage)
}

// ValidationErrors accumulates multiple validation error messages.
type ValidationErrors []ValidationError

// Error returns a human readable error message reporting each error in the
// list.
func (errs ValidationErrors) Error() string {
	if len(errs) == 1 {
		return errs[0].String()
	}
	messages := []string{"errors:"}
	for _, e := range list {
		messages = append(messages, "  "+e.Error())
	}
	return strings.Join(messages, "\n")
}

// Validate returns an error with a list of every spec violation.
func (tv TypedValue) Validate() error {
	v := validation{
		path:    fieldpath.Path{},
		value:   tv.value,
		schema:  tv.schema,
		typeRef: tv.typeRef,
	}
	errs := v.validate()
	if len(errs) == 0 {
		return nil
	}
	return errs
}

type validation struct {
	path    fieldpath.Path
	value   value.Value
	schema  *schema.Schema
	typeRef schema.TypeRef
}

func (v validation) error(format string, args ...interface{}) ValidationError {
	return ValidationError{
		Path:         append(fieldpath.Path{}, v.path...),
		ErrorMessage: fmt.Sprintf(format, args...),
	}
}

func (v validation) validate() ValidationErrors {
	a, ok := v.schema.Resolve(v.typeRef)
	if !ok {
		return ValidationErrors{v.error("no type found matching: %v", *v.typeRef.NamedType)}
	}

	switch {
	case a.Scalar != nil:
		return v.doScalar(*a.Scalar, v.value)
	case a.Struct != nil:
		return v.doStruct(*a.Struct, v.value)
	case a.List != nil:
		return v.doList(*a.List, v.value)
	case a.Map != nil:
		return v.doMap(*a.Map, v.value)
	case a.Untyped != nil:
		// Untyped sections allow anything.
		return nil
	}

	return ValidationErrors{v.error("invalid atom")}
}

func (v validation) doScalar(t schema.Scalar, value value.Value) ValidationErrors {
	switch t {
	case schema.Numeric:
		if value.Float == nil && value.Int == nil {
			// TODO: should the schema separate int and float?
			return ValidationErrors{v.error("expected numeric (int or float)")}
		}
	case schema.String:
		if value.String == nil {
			return ValidationErrors{v.error("expected string")}
		}
	case schema.Boolean:
		if value.Boolean == nil {
			return ValidationErrors{v.error("expected boolean")}
		}
	}
	return nil
}

func (v validation) doStruct(t schema.Struct, value value.Value) (errs ValidationErrors) {
	switch {
	case value.Null:
		// Null is a valid struct.
		return nil
	case value.Map != nil:
		// OK
	default:
		return ValidationErrors{v.error("expected struct")}
	}

	allowedNames := map[string]struct{}{}
	m := *value.Map
	for i := range t.Fields {
		f := &t.Fields[i]
		allowedNames[f.Name] = struct{}{}
		child, ok := m.Get(f.Name)
		if !ok {
			// All fields are optional
			continue
		}
		v2 := v
		name := f.Name
		v2.path = append(v.path, fieldpath.PathElement{FieldName: &name})
		v2.value = child.Value
		v2.typeRef = f.Type
		errs = append(errs, v2.validate()...)
	}

	// All fields may be optional, but unknown fields are not allowed.
	for _, f := range m.Items {
		if _, allowed := allowedNames[f.Name]; !allowed {
			errs = append(errs, v.error("field %v is not mentioned in the schema", f.Name))
		}
	}

	// Check unions

	return errs
}

func listItemToPathElement(list schema.List, index int, child value.Value) (fieldpath.PathElement, error) {
	pe := fieldpath.PathElement{}
	if list.ElementRelationship == schema.Associative {
		if len(list.Keys) > 0 {
			if child.Map != nil {
				for _, fieldName := range list.Keys {
					var fieldValue value.Value
					field, ok := child.Map.Get(fieldName)
					if ok {
						fieldValue = field.Value
					} else {
						fieldValue = value.Value{Null: true}
					}
					pe.Key = append(pe.Key, value.Field{
						Name:  fieldName,
						Value: fieldValue,
					})
				}
				return pe, nil
			}
			if child.Null {
				// Count every key as null?
				for _, fieldName := range list.Keys {
					pe.Key = append(pe.Key, value.Field{
						Name:  fieldName,
						Value: value.Value{Null: true},
					})
				}
				return pe, nil
			}
			return pe, errors.New("associative list with keys has an element that's a non-map type")
		}

		switch {
		case child.Map != nil:
			return pe, errors.New("associative list without keys has an element that's a map type")
		case child.List != nil:
			// Should we support a set of lists? For the moment
			// let's say we don't.
			return pe, errors.New("not supported: associative list with lists as elements")
		case child.Null:
			return pe, errors.New("associative list without keys has an element that's map or list typed (explicit null)")
		default:
			// We are a set type.
			pe.Value = &child
			return pe, nil
		}
	}

	// Use the index as a key for atomic lists.
	pe.Index = &index
	return pe, nil
}

func (v validation) doList(t schema.List, value value.Value) (errs ValidationErrori) {
	switch {
	case value.Null:
		// Null is a valid list.
		return nil
	case value.List != nil:
		// OK
	default:
		return ValidationErrori{v.error("expected list")}
	}

	observedKeys := map[string]struct{}{}

	list := *value.List
	for i, child := range list.Items {
		pe, err := listItemToPathElement(t, i, child)
		if err != nil {
			errs = append(errs, v.error("element %v: %v", i, err.Error()))
			// If we can't construct the path element, we can't
			// even report errors deeper in the schema, so bail on
			// this element.
			continue
		}
		keyStr := pe.String()
		if _, found := observedKeys[keyStr]; found {
			errs = append(errs, v.error("duplicate entries for key %v", keyStr))
		}
		observedKeys[keyStr] = struct{}{}
		v2 := v
		v2.path = append(v.path, pe)
		v2.value = child
		v2.typeRef = t.ElementType
		errs = append(errs, v2.validate()...)
	}

	return errs
}

func (v validation) doMap(t schema.Map, value value.Value) (errs ValidationErrors) {
	switch {
	case value.Null:
		// Null is a valid map.
		return nil
	case value.Map != nil:
		// OK
	default:
		return ValidationErrors{v.error("expected list, found %v", value.HumanReadable())}
	}

	for _, item := range value.Map.Items {
		v2 := v
		name := item.Name
		v2.path = append(v.path, fieldpath.PathElement{FieldName: &name})
		v2.value = item.Value
		v2.typeRef = t.ElementType
		errs = append(errs, v2.validate()...)
	}

	return errs
}
