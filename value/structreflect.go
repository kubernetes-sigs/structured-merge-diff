/*
Copyright 2019 The Kubernetes Authors.

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

package value

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// reflectStructCache keeps track of json tag related data for structs and fields to speed up reflection.
// TODO: This overlaps in functionality with the fieldCache in
// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/runtime/converter.go#L57 but
// is more efficient at lookup by json field name. The logic should be consolidated. Only one copy of the cache needs
// to be kept for each running process.
var (
	reflectStructCache = newStructCache()
)

type structCache struct {
	// use an atomic and copy-on-write since there are a fixed (typically very small) number of structs compiled into any
	// go program using this cache
	value atomic.Value
	// mu is held by writers when performing load/modify/store operations on the cache, readers do not need to hold a
	// read-lock since the atomic value is always read-only
	mu sync.Mutex
}

type structCacheMap map[reflect.Type]structCacheEntry

// structCacheEntry contains information about each struct field, keyed by json field name, that is expensive to
// compute using reflection.
type structCacheEntry map[string]*fieldCacheEntry

// Get returns true and fieldCacheEntry for the given type if the type is in the cache. Otherwise Get returns false.
func (c *structCache) Get(t reflect.Type) (map[string]*fieldCacheEntry, bool) {
	entry, ok := c.value.Load().(structCacheMap)[t]
	return entry, ok
}

// Update sets the fieldCacheEntry for the given type via a copy-on-write update to the struct cache.
func (c *structCache) Update(t reflect.Type, m map[string]*fieldCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldCacheMap := c.value.Load().(structCacheMap)
	newCacheMap := make(structCacheMap, len(oldCacheMap)+1)
	for k, v := range oldCacheMap {
		newCacheMap[k] = v
	}
	newCacheMap[t] = m
	c.value.Store(newCacheMap)
}

func newStructCache() *structCache {
	cache := &structCache{}
	cache.value.Store(make(structCacheMap))
	return cache
}

type fieldCacheEntry struct {
	// isOmitEmpty is true if the field has the json 'omitempty' tag.
	isOmitEmpty bool
	// fieldPath is the field indices (see FieldByIndex) to lookup the value of
	// a field in a reflect.Value struct. A path of field indices is used
	// to support traversing to a field nested in struct fields that have the 'inline'
	// json tag.
	fieldPath [][]int
}

func (f *fieldCacheEntry) getFieldFromStruct(structVal reflect.Value) reflect.Value {
	// field might be nested within 'inline' structs
	for _, elem := range f.fieldPath {
		structVal = structVal.FieldByIndex(elem)
	}
	return structVal
}

func getStructCacheEntry(t reflect.Type) structCacheEntry {
	if hints, ok := reflectStructCache.Get(t); ok {
		return hints
	}

	hints := map[string]*fieldCacheEntry{}
	buildStructCacheEntry(t, hints, nil)

	reflectStructCache.Update(t, hints)
	return hints
}

func buildStructCacheEntry(t reflect.Type, infos map[string]*fieldCacheEntry, fieldPath [][]int) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonName, isInline, isOmitempty := lookupJsonTags(field)
		if isInline {
			buildStructCacheEntry(field.Type, infos, append(fieldPath, field.Index))
			continue
		}
		info := &fieldCacheEntry{isOmitEmpty: isOmitempty, fieldPath: append(fieldPath, field.Index)}
		infos[jsonName] = info

	}
}

type structReflect struct {
	Value reflect.Value
}

func (r structReflect) Length() int {
	i := 0
	eachStructField(r.Value, func(s string, value reflect.Value) bool {
		i++
		return true
	})
	return i
}

func (r structReflect) Get(key string) (Value, bool) {
	if val, ok := r.findJsonNameField(key); ok {
		return mustWrapValueReflect(val), true
	}
	return nil, false
}

func (r structReflect) Has(key string) bool {
	_, ok := r.findJsonNameField(key)
	return ok
}

func (r structReflect) Set(key string, val Value) {
	fieldVal, ok := r.findJsonNameField(key)
	if !ok {
		panic(fmt.Sprintf("key %s may not be set on struct %T: field does not exist", key, r.Value.Interface()))
	}
	if !fieldVal.CanSet() {
		// See https://blog.golang.org/laws-of-reflection for details on why a struct may not be settable
		panic(fmt.Sprintf("key %s may not be set on struct: %T: struct is not settable", key, r.Value.Interface()))
	}
	fieldVal.Set(reflect.ValueOf(val.Unstructured()))
}

func (r structReflect) Delete(key string) {
	fieldVal, ok := r.findJsonNameField(key)
	if !ok {
		panic(fmt.Sprintf("key %s may not be deleted on struct %T: field does not exist", key, r.Value.Interface()))
	}
	if !fieldVal.CanSet() {
		// See https://blog.golang.org/laws-of-reflection for details on why a struct may not be settable
		panic(fmt.Sprintf("key %s may not be deleted on struct: %T: struct is not settable", key, r.Value.Interface()))
	}
	fieldVal.Set(reflect.Zero(fieldVal.Type()))
}

func (r structReflect) Iterate(fn func(string, Value) bool) bool {
	return eachStructField(r.Value, func(s string, value reflect.Value) bool {
		v := mustWrapValueReflect(value)
		defer v.Recycle()
		return fn(s, v)
	})
}

func eachStructField(structVal reflect.Value, fn func(string, reflect.Value) bool) bool {
	for jsonName, fieldCacheEntry := range getStructCacheEntry(structVal.Type()) {
		fieldVal := fieldCacheEntry.getFieldFromStruct(structVal)
		if fieldCacheEntry.isOmitEmpty && (safeIsNil(fieldVal) || isZero(fieldVal)) {
			// omit it
			continue
		}
		ok := fn(jsonName, fieldVal)
		if !ok {
			return false
		}
	}
	return true
}

func (r structReflect) Unstructured() interface{} {
	// Use number of struct fields as a cheap way to rough estimate map size
	result := make(map[string]interface{}, r.Value.NumField())
	r.Iterate(func(s string, value Value) bool {
		result[s] = value.Unstructured()
		return true
	})
	return result
}

func (r structReflect) Equals(m Map) bool {
	if rhsStruct, ok := m.(structReflect); ok {
		return reflect.DeepEqual(r.Value.Interface(), rhsStruct.Value.Interface())
	}
	if r.Length() != m.Length() {
		return false
	}
	structCacheEntry := getStructCacheEntry(r.Value.Type())

	return m.Iterate(func(s string, value Value) bool {
		fieldCacheEntry, ok := structCacheEntry[s]
		if !ok {
			return false
		}
		lhsVal := fieldCacheEntry.getFieldFromStruct(r.Value)
		return Equals(mustWrapValueReflect(lhsVal), value)
	})
}

func (r structReflect) findJsonNameFieldAndNotEmpty(jsonName string) (reflect.Value, bool) {
	structCacheEntry, ok := getStructCacheEntry(r.Value.Type())[jsonName]
	if !ok {
		return reflect.Value{}, false
	}
	fieldVal := structCacheEntry.getFieldFromStruct(r.Value)
	omit := structCacheEntry.isOmitEmpty && (safeIsNil(fieldVal) || isZero(fieldVal))
	return fieldVal, !omit
}

func (r structReflect) findJsonNameField(jsonName string) (reflect.Value, bool) {
	structCacheEntry, ok := getStructCacheEntry(r.Value.Type())[jsonName]
	if !ok {
		return reflect.Value{}, false
	}
	fieldVal := structCacheEntry.getFieldFromStruct(r.Value)
	return fieldVal, true
}