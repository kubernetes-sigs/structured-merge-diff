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

	"sigs.k8s.io/structured-merge-diff/fieldpath"
	"sigs.k8s.io/structured-merge-diff/schema"
	"sigs.k8s.io/structured-merge-diff/value"
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
		return errs[0].Error()
	}
	messages := []string{"errors:"}
	for _, e := range errs {
		messages = append(messages, "  "+e.Error())
	}
	return strings.Join(messages, "\n")
}

func (tv TypedValue) walker() *validatingObjectWalker {
	return &validatingObjectWalker{
		path:    fieldpath.Path{},
		value:   tv.value,
		schema:  tv.schema,
		typeRef: tv.typeRef,
	}
}

type validatingObjectWalker struct {
	path    fieldpath.Path
	value   value.Value
	schema  *schema.Schema
	typeRef schema.TypeRef

	// If set, this is called on "leaf fields":
	//  * scalars: int/string/float/bool
	//  * atomic maps and lists
	//  * untyped fields
	leafFieldCallback func(fieldpath.Path)

	// internal housekeeping--don't set when constructing.
	inLeaf bool // Set to true if we're in a "big leaf"--atomic map/list
}

func (v validatingObjectWalker) errorf(format string, args ...interface{}) ValidationErrors {
	return ValidationErrors{{
		Path:         append(fieldpath.Path{}, v.path...),
		ErrorMessage: fmt.Sprintf(format, args...),
	}}
}

func (v validatingObjectWalker) error(err error) ValidationErrors {
	return ValidationErrors{{
		Path:         append(fieldpath.Path{}, v.path...),
		ErrorMessage: err.Error(),
	}}
}

func (v validatingObjectWalker) validate() ValidationErrors {
	a, ok := v.schema.Resolve(v.typeRef)
	if !ok {
		return v.errorf("schema error: no type found matching: %v", *v.typeRef.NamedType)
	}

	switch {
	case a.Scalar != nil:
		return v.doScalar(*a.Scalar)
	case a.Struct != nil:
		return v.doStruct(*a.Struct)
	case a.List != nil:
		return v.doList(*a.List)
	case a.Map != nil:
		return v.doMap(*a.Map)
	case a.Untyped != nil:
		return v.doUntyped(*a.Untyped)
	}

	return v.errorf("schema error: invalid atom")
}

// doLeaf should be called on leaves before descending into children, if there
// will be a descent. It modifies v.inLeaf.
func (v *validatingObjectWalker) doLeaf() {
	if v.inLeaf {
		// We're in a "big leaf", an atomic map or list. Ignore
		// subsequent leaves.
		return
	}
	v.inLeaf = true

	if v.leafFieldCallback != nil {
		// At the moment, this is only used to build fieldsets; we can
		// add more than the path in here if needed.
		v.leafFieldCallback(v.path)
	}
}

func (v validatingObjectWalker) doScalar(t schema.Scalar) ValidationErrors {
	switch t {
	case schema.Numeric:
		if v.value.Float == nil && v.value.Int == nil {
			// TODO: should the schema separate int and float?
			return v.errorf("expected numeric (int or float), got %v", v.value.HumanReadable())
		}
	case schema.String:
		if v.value.String == nil {
			return v.errorf("expected string, got %v", v.value.HumanReadable())
		}
	case schema.Boolean:
		if v.value.Boolean == nil {
			return v.errorf("expected boolean, got %v", v.value.HumanReadable())
		}
	}

	// All scalars are leaf fields.
	v.doLeaf()

	return nil
}

// Returns the map, or an error. Reminder: nil is a valid map and might be returned.
// Same as mapValue except for the error message.
func structValue(val value.Value) (*value.Map, error) {
	switch {
	case val.Null:
		return nil, nil
	case val.Map != nil:
		return val.Map, nil
	default:
		return nil, fmt.Errorf("expected struct, got %v", val.HumanReadable())
	}
}

func (v validatingObjectWalker) visitStructFields(t schema.Struct, m *value.Map) (errs ValidationErrors) {
	allowedNames := map[string]struct{}{}
	for i := range t.Fields {
		// I don't want to use the loop variable since a reference
		// might outlive the loop iteration (in an error message).
		f := t.Fields[i]
		allowedNames[f.Name] = struct{}{}
		child, ok := m.Get(f.Name)
		if !ok {
			// All fields are optional
			continue
		}
		v2 := v
		v2.path = append(v.path, fieldpath.PathElement{FieldName: &f.Name})
		v2.value = child.Value
		v2.typeRef = f.Type
		errs = append(errs, v2.validate()...)
	}

	// All fields may be optional, but unknown fields are not allowed.
	for _, f := range m.Items {
		if _, allowed := allowedNames[f.Name]; !allowed {
			errs = append(errs, v.errorf("field %v is not mentioned in the schema", f.Name)...)
		}
	}

	return errs
}

func (v validatingObjectWalker) doStruct(t schema.Struct) (errs ValidationErrors) {
	m, err := structValue(v.value)
	if err != nil {
		return v.error(err)
	}

	if t.ElementRelationship == schema.Atomic {
		v.doLeaf()
	}

	if m == nil {
		// nil is a valid map!
		return nil
	}

	errs = v.visitStructFields(t, m)

	// TODO: Check unions.

	return errs
}

func keyedAssociativeListItemToPathElement(list schema.List, index int, child value.Value) (fieldpath.PathElement, error) {
	pe := fieldpath.PathElement{}
	if child.Null {
		// For now, the keys are required which means that null entries
		// are illegal.
		return pe, errors.New("associative list with keys may not have a null element")
	}
	if child.Map == nil {
		return pe, errors.New("associative list with keys may not have non-map elements")
	}
	for _, fieldName := range list.Keys {
		var fieldValue value.Value
		field, ok := child.Map.Get(fieldName)
		if ok {
			fieldValue = field.Value
		} else {
			// Treat keys as required.
			return pe, errors.New("associative list with keys has an element that omits key field " + fieldName)
		}
		pe.Key = append(pe.Key, value.Field{
			Name:  fieldName,
			Value: fieldValue,
		})
	}
	return pe, nil
}

func setItemToPathElement(list schema.List, index int, child value.Value) (fieldpath.PathElement, error) {
	pe := fieldpath.PathElement{}
	switch {
	case child.Map != nil:
		// TODO: atomic maps should be acceptable.
		return pe, errors.New("associative list without keys has an element that's a map type")
	case child.List != nil:
		// Should we support a set of lists? For the moment
		// let's say we don't.
		// TODO: atomic lists should be acceptable.
		return pe, errors.New("not supported: associative list with lists as elements")
	case child.Null:
		return pe, errors.New("associative list without keys has an element that's an explicit null")
	default:
		// We are a set type.
		pe.Value = &child
		return pe, nil
	}
}

func listItemToPathElement(list schema.List, index int, child value.Value) (fieldpath.PathElement, error) {
	if list.ElementRelationship == schema.Associative {
		if len(list.Keys) > 0 {
			return keyedAssociativeListItemToPathElement(list, index, child)
		}

		// If there's no keys, then we must be a set of primitives.
		return setItemToPathElement(list, index, child)
	}

	// Use the index as a key for atomic lists.
	return fieldpath.PathElement{Index: &index}, nil
}

// Returns the list, or an error. Reminder: nil is a valid list and might be returned.
func listValue(val value.Value) (*value.List, error) {
	switch {
	case val.Null:
		// Null is a valid list.
		return nil, nil
	case val.List != nil:
		return val.List, nil
	default:
		return nil, fmt.Errorf("expected list, got %v", val.HumanReadable())
	}
}

func (v validatingObjectWalker) visitListItems(t schema.List, list *value.List) (errs ValidationErrors) {
	observedKeys := map[string]struct{}{}
	for i, child := range list.Items {
		pe, err := listItemToPathElement(t, i, child)
		if err != nil {
			errs = append(errs, v.errorf("element %v: %v", i, err.Error())...)
			// If we can't construct the path element, we can't
			// even report errors deeper in the schema, so bail on
			// this element.
			continue
		}
		keyStr := pe.String()
		if _, found := observedKeys[keyStr]; found {
			errs = append(errs, v.errorf("duplicate entries for key %v", keyStr)...)
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

func (v validatingObjectWalker) doList(t schema.List) (errs ValidationErrors) {
	list, err := listValue(v.value)
	if err != nil {
		return v.error(err)
	}

	if t.ElementRelationship == schema.Atomic {
		v.doLeaf()
	}

	if list == nil {
		return nil
	}

	errs = v.visitListItems(t, list)

	return errs
}

// Returns the map, or an error. Reminder: nil is a valid map and might be returned.
func mapValue(val value.Value) (*value.Map, error) {
	switch {
	case val.Null:
		return nil, nil
	case val.Map != nil:
		return val.Map, nil
	default:
		return nil, fmt.Errorf("expected map, got %v", val.HumanReadable())
	}
}

func (v validatingObjectWalker) visitMapItems(t schema.Map, m *value.Map) (errs ValidationErrors) {
	for _, item := range m.Items {
		v2 := v
		name := item.Name
		v2.path = append(v.path, fieldpath.PathElement{FieldName: &name})
		v2.value = item.Value
		v2.typeRef = t.ElementType
		errs = append(errs, v2.validate()...)
	}
	return errs
}

func (v validatingObjectWalker) doMap(t schema.Map) (errs ValidationErrors) {
	m, err := mapValue(v.value)
	if err != nil {
		return v.error(err)
	}

	if t.ElementRelationship == schema.Atomic {
		v.doLeaf()
	}

	if m == nil {
		return nil
	}

	errs = v.visitMapItems(t, m)

	return errs
}

func (v validatingObjectWalker) doUntyped(t schema.Untyped) (errs ValidationErrors) {
	if t.ElementRelationship == "" || t.ElementRelationship == schema.Atomic {
		// Untyped sections allow anything, and are considered leaf
		// fields.
		v.doLeaf()
	}
	return nil
}
