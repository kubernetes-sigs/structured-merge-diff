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

type mergingWalker struct {
	errorFormatter
	lhs     *value.Value
	rhs     *value.Value
	schema  *schema.Schema
	typeRef schema.TypeRef

	// How to merge. Called after schema validation for all leaf fields.
	rule mergeRule

	// output of the merge operation (nil if none)
	out *value.Value

	// internal housekeeping--don't set when constructing.
	inLeaf bool // Set to true if we're in a "big leaf"--atomic map/list
}

// merge rules examine w.lhs and w.rhs (up to one of which may be nil) and
// optionally set w.out. If lhs and rhs are both set, they will be of
// comparable type.
type mergeRule func(w *mergingWalker)

var (
	ruleKeepRHS = mergeRule(func(w *mergingWalker) {
		if w.rhs != nil {
			v := *w.rhs
			w.out = &v
		} else if w.lhs != nil {
			v := *w.lhs
			w.out = &v
		}
	})
	ruleSymmetricDifference = mergeRule(func(w *mergingWalker) {
		// Return everything not the same in both.
		if w.lhs == nil {
			v := *w.rhs
			w.out = &v
		} else if w.rhs == nil {
			v := *w.lhs
			w.out = &v
		} else if !reflect.DeepEqual(w.rhs, w.lhs) {
			// TODO: reflect.DeepEqual is not sufficient for this.
			// Need to implement equality check on the value type.
			v := *w.rhs
			w.out = &v
		}
	})
)

// merge sets w.out.
func (w *mergingWalker) merge() ValidationErrors {
	if w.lhs == nil && w.rhs == nil {
		// check this condidition here instead of everywhere below.
		return w.errorf("at least one of lhs and rhs must be provided")
	}
	return resolveSchema(w.schema, w.typeRef, w)
}

// doLeaf should be called on leaves before descending into children, if there
// will be a descent. It modifies w.inLeaf.
func (w *mergingWalker) doLeaf() {
	if w.inLeaf {
		// We're in a "big leaf", an atomic map or list. Ignore
		// subsequent leaves.
		return
	}
	w.inLeaf = true

	// We don't recurse into leaf fields for merging.
	w.rule(w)
}

func (w *mergingWalker) doScalar(t schema.Scalar) (errs ValidationErrors) {
	if w.lhs != nil {
		if err := validateScalar(t, *w.lhs); err != nil {
			errs = append(errs, w.prefixError("lhs: ", err)...)
		}
	}
	if w.rhs != nil {
		if err := validateScalar(t, *w.rhs); err != nil {
			errs = append(errs, w.prefixError("rhs: ", err)...)
		}
	}
	if len(errs) > 0 {
		return errs
	}

	// All scalars are leaf fields.
	w.doLeaf()

	return nil
}

func (w *mergingWalker) prepareDescent(pe fieldpath.PathElement, tr schema.TypeRef) *mergingWalker {
	w2 := *w
	w2.typeRef = tr
	w2.errorFormatter.descend(pe)
	w2.lhs = nil
	w2.rhs = nil
	w2.out = nil
	return &w2
}

func (w *mergingWalker) visitStructFields(t schema.Struct, lhs, rhs *value.Map) (errs ValidationErrors) {
	out := &value.Map{}

	maybeGet := func(m *value.Map, name string) (*value.Field, bool) {
		if m == nil {
			return nil, false
		}
		return m.Get(name)
	}

	allowedNames := map[string]struct{}{}
	for i := range t.Fields {
		// I don't want to use the loop variable since a reference
		// might outlive the loop iteration (in an error message).
		f := t.Fields[i]
		allowedNames[f.Name] = struct{}{}
		lchild, lok := maybeGet(lhs, f.Name)
		rchild, rok := maybeGet(rhs, f.Name)
		if !lok && !rok {
			// All fields are optional
			continue
		}
		w2 := w.prepareDescent(fieldpath.PathElement{FieldName: &f.Name}, f.Type)
		if lok {
			w2.lhs = &lchild.Value
		}
		if rok {
			w2.rhs = &rchild.Value
		}
		if newErrs := w2.merge(); len(newErrs) > 0 {
			errs = append(errs, newErrs...)
		} else if w2.out != nil {
			out.Set(f.Name, *w2.out)
		}
	}

	// All fields may be optional, but unknown fields are not allowed.
	if lhs != nil {
		for _, f := range lhs.Items {
			if _, allowed := allowedNames[f.Name]; !allowed {
				errs = append(errs, w.errorf("lhs: field %v is not mentioned in the schema", f.Name)...)
			}
		}
	}
	if rhs != nil {
		for _, f := range rhs.Items {
			if _, allowed := allowedNames[f.Name]; !allowed {
				errs = append(errs, w.errorf("rhs: field %v is not mentioned in the schema", f.Name)...)
			}
		}
	}

	if len(out.Items) > 0 {
		w.out = &value.Value{Map: out}
	}

	return errs
}

func (w *mergingWalker) doStruct(t schema.Struct) (errs ValidationErrors) {
	var lhs, rhs *value.Map
	var err error
	if w.lhs != nil {
		lhs, err = structValue(*w.lhs)
		if err != nil {
			errs = append(errs, w.prefixError("lhs: ", err)...)
		}
	}
	if w.rhs != nil {
		rhs, err = structValue(*w.rhs)
		if err != nil {
			errs = append(errs, w.prefixError("rhs: ", err)...)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	// If both lhs and rhs are empty/null, treat it as a
	// leaf: this helps preserve the empty/null
	// distinction.
	emptyPromoteToLeaf := (lhs == nil || len(lhs.Items) == 0) &&
		(rhs == nil || len(rhs.Items) == 0)

	if t.ElementRelationship == schema.Atomic || emptyPromoteToLeaf {
		w.doLeaf()
		return nil
	}

	if lhs == nil && rhs == nil {
		// nil is a valid map!
		return nil
	}

	errs = w.visitStructFields(t, lhs, rhs)

	// TODO: Check unions.

	return errs
}

func (w *mergingWalker) visitListItems(t schema.List, lhs, rhs *value.List) (errs ValidationErrors) {
	out := &value.List{}

	// TODO: ordering is totally wrong.
	// TODO: might as well make the map order work the same way.

	// This is a cheap hack to at least make the output order stable.
	rhsOrder := []string{}

	// First, collect all RHS children.
	observedRHS := map[string]value.Value{}
	if rhs != nil {
		for i, child := range rhs.Items {
			pe, err := listItemToPathElement(t, i, child)
			if err != nil {
				errs = append(errs, w.errorf("rhs: element %v: %v", i, err.Error())...)
				// If we can't construct the path element, we can't
				// even report errors deeper in the schema, so bail on
				// this element.
				continue
			}
			keyStr := pe.String()
			if _, found := observedRHS[keyStr]; found {
				errs = append(errs, w.errorf("rhs: duplicate entries for key %v", keyStr)...)
			}
			observedRHS[keyStr] = child
			rhsOrder = append(rhsOrder, keyStr)
		}
	}

	// Then merge with LHS children.
	observedLHS := map[string]struct{}{}
	if lhs != nil {
		for i, child := range lhs.Items {
			pe, err := listItemToPathElement(t, i, child)
			if err != nil {
				errs = append(errs, w.errorf("lhs: element %v: %v", i, err.Error())...)
				// If we can't construct the path element, we can't
				// even report errors deeper in the schema, so bail on
				// this element.
				continue
			}
			keyStr := pe.String()
			if _, found := observedLHS[keyStr]; found {
				errs = append(errs, w.errorf("lhs: duplicate entries for key %v", keyStr)...)
				continue
			}
			observedLHS[keyStr] = struct{}{}
			rchild, ok := observedRHS[keyStr]
			if !ok {
				// only a left child exists; no need to merge.
				out.Items = append(out.Items, child)
				continue
			}
			w2 := w.prepareDescent(pe, t.ElementType)
			w2.lhs = &child
			w2.rhs = &rchild
			if newErrs := w2.merge(); len(newErrs) > 0 {
				errs = append(errs, newErrs...)
			} else if w2.out != nil {
				out.Items = append(out.Items, *w2.out)
			}
			// Keep track of children that have been handled
			delete(observedRHS, keyStr)
		}
	}

	for _, rhsToCheck := range rhsOrder {
		if unmergedChild, ok := observedRHS[rhsToCheck]; ok {
			out.Items = append(out.Items, unmergedChild)
		}
	}

	if len(out.Items) > 0 {
		w.out = &value.Value{List: out}
	}
	return errs
}

func (w *mergingWalker) doList(t schema.List) (errs ValidationErrors) {
	var lhs, rhs *value.List
	var err error
	if w.lhs != nil {
		lhs, err = listValue(*w.lhs)
		if err != nil {
			errs = append(errs, w.prefixError("lhs: ", err)...)
		}
	}
	if w.rhs != nil {
		rhs, err = listValue(*w.rhs)
		if err != nil {
			errs = append(errs, w.prefixError("rhs: ", err)...)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	// If both lhs and rhs are empty/null, treat it as a
	// leaf: this helps preserve the empty/null
	// distinction.
	emptyPromoteToLeaf := (lhs == nil || len(lhs.Items) == 0) &&
		(rhs == nil || len(rhs.Items) == 0)

	if t.ElementRelationship == schema.Atomic || emptyPromoteToLeaf {
		w.doLeaf()
		return nil
	}

	if lhs == nil && rhs == nil {
		return nil
	}

	errs = w.visitListItems(t, lhs, rhs)

	return errs
}

func (w *mergingWalker) visitMapItems(t schema.Map, lhs, rhs *value.Map) (errs ValidationErrors) {
	out := &value.Map{}

	if lhs != nil {
		for _, litem := range lhs.Items {
			var ritem *value.Field
			var ok bool
			if rhs != nil {
				ritem, ok = rhs.Get(litem.Name)
			}
			if !ok {
				out.Set(litem.Name, litem.Value)
				continue
			}
			name := litem.Name
			w2 := w.prepareDescent(fieldpath.PathElement{FieldName: &name}, t.ElementType)
			w2.lhs = &litem.Value
			w2.rhs = &ritem.Value
			if newErrs := w2.merge(); len(newErrs) > 0 {
				errs = append(errs, newErrs...)
			} else if w2.out != nil {
				out.Set(litem.Name, *w2.out)
			}
		}
	}
	if rhs != nil {
		for _, ritem := range rhs.Items {
			if lhs != nil {
				if _, ok := lhs.Get(ritem.Name); ok {
					continue
				}
			}
			out.Set(ritem.Name, ritem.Value)
		}
	}

	if len(out.Items) > 0 {
		w.out = &value.Value{Map: out}
	}
	return errs
}

func (w *mergingWalker) doMap(t schema.Map) (errs ValidationErrors) {
	var lhs, rhs *value.Map
	var err error
	if w.lhs != nil {
		lhs, err = mapValue(*w.lhs)
		if err != nil {
			errs = append(errs, w.prefixError("lhs: ", err)...)
		}
	}
	if w.rhs != nil {
		rhs, err = mapValue(*w.rhs)
		if err != nil {
			errs = append(errs, w.prefixError("rhs: ", err)...)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	// If both lhs and rhs are empty/null, treat it as a
	// leaf: this helps preserve the empty/null
	// distinction.
	emptyPromoteToLeaf := (lhs == nil || len(lhs.Items) == 0) &&
		(rhs == nil || len(rhs.Items) == 0)

	if t.ElementRelationship == schema.Atomic || emptyPromoteToLeaf {
		w.doLeaf()
		return nil
	}

	if lhs == nil && rhs == nil {
		return nil
	}

	errs = w.visitMapItems(t, lhs, rhs)

	return errs
}

func (w *mergingWalker) doUntyped(t schema.Untyped) (errs ValidationErrors) {
	if t.ElementRelationship == "" || t.ElementRelationship == schema.Atomic {
		// Untyped sections allow anything, and are considered leaf
		// fields.
		w.doLeaf()
	}
	return nil
}
