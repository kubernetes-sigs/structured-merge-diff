/*
Copyright 2025 The Kubernetes Authors.

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
	"fmt"
	"sync"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/schema"
	"sigs.k8s.io/structured-merge-diff/v4/value"
)

const (
	// markerKey is the key used to store marker values in the object.
	markerKey = "k8s_io__value"

	// unsetMarkerValue is the value used to indicate that a marker is unset.
	unsetMarkerValue = "unset"
)

var mPool = sync.Pool{
	New: func() interface{} { return &markerExtractorWalker{} },
}

func (tv TypedValue) markerExtractorWalker() *markerExtractorWalker {
	v := mPool.Get().(*markerExtractorWalker)
	v.value = tv.value
	v.schema = tv.schema
	v.typeRef = tv.typeRef
	v.unsetMarkers = &fieldpath.Set{}
	v.orphanedFields = &fieldpath.Set{}
	if v.allocator == nil {
		v.allocator = value.NewFreelistAllocator()
	}
	return v
}

func (v *markerExtractorWalker) finished() {
	v.schema = nil
	v.typeRef = schema.TypeRef{}
	v.path = nil
	v.unsetMarkers = nil
	v.orphanedFields = nil
	mPool.Put(v)
}

type markerExtractorWalker struct {
	value   value.Value
	schema  *schema.Schema
	typeRef schema.TypeRef

	unsetMarkers   *fieldpath.Set
	orphanedFields *fieldpath.Set // Map or list fields that only contain marker children.

	path                      fieldpath.Path
	parentElementRelationship []*schema.ElementRelationship // Track the element relationships of parents for marker validation.

	// Allocate only as many walkers as needed for the depth by storing them here.
	spareWalkers *[]*markerExtractorWalker
	allocator    value.Allocator
}

func (v *markerExtractorWalker) prepareDescent(pe fieldpath.PathElement, tr schema.TypeRef) *markerExtractorWalker {
	if v.spareWalkers == nil {
		// first descent.
		v.spareWalkers = &[]*markerExtractorWalker{}
	}
	var v2 *markerExtractorWalker
	if n := len(*v.spareWalkers); n > 0 {
		v2, *v.spareWalkers = (*v.spareWalkers)[n-1], (*v.spareWalkers)[:n-1]
	} else {
		v2 = &markerExtractorWalker{}
	}
	*v2 = *v
	v2.typeRef = tr
	v2.path = append(v2.path, pe)

	// Track the element relationships of parents
	v2.parentElementRelationship = append(v2.parentElementRelationship, v.resolveElementRelationship())
	return v2
}

func (v *markerExtractorWalker) finishDescent(v2 *markerExtractorWalker) {
	// if the descent caused a realloc, ensure that we reuse the buffer
	// for the next sibling.
	v.path = v2.path[:len(v2.path)-1]
	v.parentElementRelationship = v2.parentElementRelationship[:len(v2.parentElementRelationship)-1]
	*v.spareWalkers = append(*v.spareWalkers, v2)
}

func (v *markerExtractorWalker) extractMarkers() ValidationErrors {
	return resolveSchema(v.schema, v.typeRef, v.value, v)
}

func (v *markerExtractorWalker) resolveElementRelationship() *schema.ElementRelationship {
	if v.typeRef.ElementRelationship != nil {
		return v.typeRef.ElementRelationship
	} else if resolvedType, ok := v.schema.Resolve(v.typeRef); ok {
		if resolvedType.Map != nil {
			return &resolvedType.Map.ElementRelationship
		} else if resolvedType.List != nil {
			return &resolvedType.List.ElementRelationship
		}
	}
	return nil
}

func (v *markerExtractorWalker) nearestElementRelationship() *schema.ElementRelationship {
	for i := len(v.parentElementRelationship) - 1; i >= 0; i-- {
		if v.parentElementRelationship[i] != nil {
			return v.parentElementRelationship[i]
		}
	}
	return nil
}

func (v *markerExtractorWalker) nearestElementRelationshipIs(relationship schema.ElementRelationship) bool {
	rel := v.nearestElementRelationship()
	return rel != nil && *rel == relationship
}

// validateMarkerLocation returns true if the current value is a marker and false otherwise.
// It the value is a marker, it returns a list of errors if the marker is not in a valid location.
func (v *markerExtractorWalker) validateMarkerLocation(t interface{}) (bool, ValidationErrors) {
	isMarker, errs := v.validateUnsetMarker()
	if !isMarker {
		return false, errs
	}

	// Check if we're inside an atomic structure first
	if v.nearestElementRelationshipIs(schema.Atomic) {
		return false, ValidationErrors{{ErrorMessage: "markers are not allowed in the contents of atomics"}}.WithPath(v.path[:len(v.path)-1].String())
	}

	switch t := t.(type) {
	case *schema.Map:
		if v.nearestElementRelationshipIs(schema.Associative) {
			return true, nil
		}
		if t.ElementRelationship != schema.Atomic {
			return false, ValidationErrors{{ErrorMessage: "markers are only allowed on atomic maps and associative list entries"}}.WithPath(v.path.String())
		}
	case *schema.List:
		if t.ElementRelationship != schema.Atomic {
			return false, ValidationErrors{{ErrorMessage: "markers are only allowed on atomic lists"}}.WithPath(v.path.String())
		}
	case *schema.Scalar:
		// No additional checks needed since we already checked for atomic above
	default:
		return false, ValidationErrors{{ErrorMessage: fmt.Sprintf("markers are only allowed on unrecognized type: %T", t)}}.WithPath(v.path.String())
	}
	return true, nil
}

func (v *markerExtractorWalker) doScalar(t *schema.Scalar) ValidationErrors {
	isMarker, errs := v.validateMarkerLocation(t)
	if errs != nil {
		return errs
	}
	if isMarker {
		v.unsetMarkers.Insert(v.path)
	}
	return nil
}

func (v *markerExtractorWalker) visitListItems(t *schema.List, list value.List) (errs ValidationErrors) {
	if list.Length() == 0 {
		return nil
	}

	markerCount := 0
	for i := 0; i < list.Length(); i++ {
		child := list.At(i)
		pe, _ := listItemToPathElement(v.allocator, v.schema, t, child)

		v2 := v.prepareDescent(pe, t.ElementType)
		v2.value = child
		errs = append(errs, v2.extractMarkers()...)
		v.finishDescent(v2)

		if v2.unsetMarkers.Has(v2.path) {
			markerCount++
		}
	}
	if markerCount == list.Length() {
		v.orphanedFields.Insert(v.path)
	}
	return errs
}

func (v *markerExtractorWalker) validateUnsetMarker() (bool, ValidationErrors) {
	marker, ok := getMarker(v.value)
	if !ok {
		return false, nil
	}
	if marker != unsetMarkerValue {
		// Should never happen since validation already checks for allowed marker values.
		return false, ValidationErrors{{ErrorMessage: fmt.Sprintf("Invalid marker: %v", marker)}}.WithPath(v.path.String())
	}
	return true, nil
}

func isUnsetMarker(v value.Value) bool {
	marker, ok := getMarker(v)
	if !ok {
		return false
	}
	return marker == unsetMarkerValue
}

func getMarker(v value.Value) (string, bool) {
	if v.IsMap() {
		m := v.AsMap()
		if val, ok := m.Get(markerKey); ok && val.IsString() {
			return val.AsString(), true
		}
	}
	return "", false
}

func (v *markerExtractorWalker) doList(t *schema.List) (errs ValidationErrors) {
	list, _ := listValue(v.allocator, v.value)
	if list != nil {
		defer v.allocator.Free(list)
	}

	isMarker, errs := v.validateMarkerLocation(t)
	if errs != nil {
		return errs
	}
	if isMarker {
		v.unsetMarkers.Insert(v.path)
		return nil
	}

	if list == nil {
		return nil
	}

	errs = v.visitListItems(t, list)
	return errs
}

func (v *markerExtractorWalker) visitMapItems(t *schema.Map, m value.Map) (errs ValidationErrors) {
	size := m.Length()
	if size == 0 {
		return nil
	}
	markerCount := 0
	m.Iterate(func(key string, val value.Value) bool {
		pe := fieldpath.PathElement{FieldName: &key}

		tr := t.ElementType
		if sf, ok := t.FindField(key); ok {
			tr = sf.Type
		}
		v2 := v.prepareDescent(pe, tr)
		v2.value = val
		errs = append(errs, v2.extractMarkers()...)
		v.finishDescent(v2)

		if v2.unsetMarkers.Has(v2.path) {
			markerCount++
		}

		return true
	})
	if markerCount == m.Length() {
		v.orphanedFields.Insert(v.path)
	}
	return errs
}

func (v *markerExtractorWalker) doMap(t *schema.Map) (errs ValidationErrors) {
	isMarker, errs := v.validateMarkerLocation(t)
	if errs != nil {
		return errs
	}
	if isMarker {
		v.unsetMarkers.Insert(v.path)
		return nil
	}

	m, _ := mapValue(v.allocator, v.value)
	if m != nil {
		defer v.allocator.Free(m)
	}

	if m == nil {
		return nil
	}

	if t.ElementRelationship == schema.Atomic {
		// If the map is atomic, we need to check for unset markers before we descend further.
		m.Iterate(func(key string, val value.Value) bool {
			if isUnsetMarker(val) {
				errs = append(errs, ValidationErrors{{ErrorMessage: "markers are not allowed in the contents of atomics"}}.WithPath(v.path.String())...)
			}
			return true
		})
		return errs
	}

	errs = v.visitMapItems(t, m)
	return errs
}
