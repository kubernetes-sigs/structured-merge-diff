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
	"sigs.k8s.io/structured-merge-diff/fieldpath"
	"sigs.k8s.io/structured-merge-diff/schema"
	"sigs.k8s.io/structured-merge-diff/value"
)

func (tv TypedValue) walker() *validatingObjectWalker {
	return &validatingObjectWalker{
		value:   tv.value,
		schema:  tv.schema,
		typeRef: tv.typeRef,
	}
}

type validatingObjectWalker struct {
	errorFormatter
	value   value.Value
	schema  *schema.Schema
	typeRef schema.TypeRef

	// If set, this is called on "leaf fields":
	//  * scalars: int/string/float/bool
	//  * atomic maps and lists
	//  * untyped fields
	leafFieldCallback func(fieldpath.Path)

	// If set, this is called on "node fields":
	//  * list items
	//  * map items
	nodeFieldCallback func(fieldpath.Path)

	// internal housekeeping--don't set when constructing.
	inLeaf bool // Set to true if we're in a "big leaf"--atomic map/list
}

func (v validatingObjectWalker) validate() ValidationErrors {
	return resolveSchema(v.schema, v.typeRef, &v.value, v)
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

// doNode should be called on nodes after descending into children
func (v *validatingObjectWalker) doNode() {
	if v.inLeaf {
		// We're in a "big leaf", an atomic map or list. Ignore
		// subsequent leaves.
		return
	}

	if v.nodeFieldCallback != nil {
		// At the moment, this is only used to build fieldsets; we can
		// add more than the path in here if needed.
		v.nodeFieldCallback(v.path)
	}
}

func (v validatingObjectWalker) doScalar(t schema.Scalar) ValidationErrors {
	if errs := v.validateScalar(t, &v.value, ""); len(errs) > 0 {
		return errs
	}

	// All scalars are leaf fields.
	v.doLeaf()

	return nil
}

func (v validatingObjectWalker) visitListItems(t schema.List, list *value.List) (errs ValidationErrors) {
	observedKeys := map[string]struct{}{}
	for i, child := range list.SortedItems() {
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
		v2.errorFormatter.descend(pe)
		v2.value = child
		v2.typeRef = t.ElementType
		v2.doNode()
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

func (v validatingObjectWalker) visitMapItems(t schema.Map, m *value.Map) (errs ValidationErrors) {
	fieldTypes := map[string]schema.TypeRef{}
	for i := range t.Fields {
		// I don't want to use the loop variable since a reference
		// might outlive the loop iteration (in an error message).
		f := t.Fields[i]
		fieldTypes[f.Name] = f.Type
	}

	for _, item := range m.SortedItems() {
		v2 := v
		name := item.Name
		v2.errorFormatter.descend(fieldpath.PathElement{FieldName: &name})
		v2.value = item.Value

		var ok bool
		if v2.typeRef, ok = fieldTypes[name]; ok {
			errs = append(errs, v2.validate()...)
		} else {
			v2.doNode()
			v2.typeRef = t.ElementType
			errs = append(errs, v2.validate()...)
		}
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
