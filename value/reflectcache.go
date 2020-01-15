/*
Copyright 2020 The Kubernetes Authors.

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
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// UnstructuredConverter defines how a type can be converted directly to unstructured.
// Types that implement json.Marshaler may also optionally implement this interface to provide a more
// direct and more efficient conversion. All types that choose to implement this interface must still
// implement this same conversion via json.Marshaler.
type UnstructuredConverter interface {
	json.Marshaler // require that json.Marshaler is implemented

	// ToUnstructured returns the unstructured representation.
	ToUnstructured() interface{}
}

// TypeReflectCacheEntry keeps data gathered using reflection about how a type is converted to/from unstructured.
type TypeReflectCacheEntry struct {
	isJsonMarshaler        bool
	ptrIsJsonMarshaler     bool
	isJsonUnmarshaler      bool
	ptrIsJsonUnmarshaler   bool
	isStringConvertable    bool
	ptrIsStringConvertable bool

	structFields map[string]*FieldCacheEntry
}

// FieldCacheEntry keeps data gathered using reflection about how the field of a struct is converted to/from
// unstructured.
type FieldCacheEntry struct {
	// isOmitEmpty is true if the field has the json 'omitempty' tag.
	isOmitEmpty bool
	// fieldPath is a list of field indices (see FieldByIndex) to lookup the value of
	// a field in a reflect.Value struct. The field indices in the list form a path used
	// to traverse through intermediary 'inline' fields.
	fieldPath [][]int
}

// GetFrom returns the field identified by this FieldCacheEntry from the provided struct.
func (f *FieldCacheEntry) GetFrom(structVal reflect.Value) reflect.Value {
	// field might be nested within 'inline' structs
	for _, elem := range f.fieldPath {
		structVal = structVal.FieldByIndex(elem)
	}
	return structVal
}

var marshalerType = reflect.TypeOf(new(json.Marshaler)).Elem()
var unmarshalerType = reflect.TypeOf(new(json.Unmarshaler)).Elem()
var unstructuredConvertableType = reflect.TypeOf(new(UnstructuredConverter)).Elem()
var defaultReflectCache = newReflectCache()

// TypeReflectEntryOf returns the TypeReflectCacheEntry of the provided reflect.Type.
func TypeReflectEntryOf(t reflect.Type) TypeReflectCacheEntry {
	if record, ok := defaultReflectCache.get(t); ok {
		return record
	}
	record := TypeReflectCacheEntry{
		isJsonMarshaler:        t.Implements(marshalerType),
		ptrIsJsonMarshaler:     reflect.PtrTo(t).Implements(marshalerType),
		isJsonUnmarshaler:      reflect.PtrTo(t).Implements(unmarshalerType),
		isStringConvertable:    t.Implements(unstructuredConvertableType),
		ptrIsStringConvertable: reflect.PtrTo(t).Implements(unstructuredConvertableType),
	}
	if t.Kind() == reflect.Struct {
		hints := map[string]*FieldCacheEntry{}
		buildStructCacheEntry(t, hints, nil)
		record.structFields = hints
	}

	defaultReflectCache.update(t, record)
	return record
}

func buildStructCacheEntry(t reflect.Type, infos map[string]*FieldCacheEntry, fieldPath [][]int) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonName, omit, isInline, isOmitempty := lookupJsonTags(field)
		if omit {
			continue
		}
		if isInline {
			buildStructCacheEntry(field.Type, infos, append(fieldPath, field.Index))
			continue
		}
		info := &FieldCacheEntry{isOmitEmpty: isOmitempty, fieldPath: append(fieldPath, field.Index)}
		infos[jsonName] = info
	}
}

// Fields returns a map of JSON field name to FieldCacheEntry for structs, or nil for non-structs.
func (e TypeReflectCacheEntry) Fields() map[string]*FieldCacheEntry {
	return e.structFields
}

// CanConvertToUnstructured returns true if this TypeReflectCacheEntry can convert values of its type to unstructured.
func (e TypeReflectCacheEntry) CanConvertToUnstructured() bool {
	return e.isJsonMarshaler || e.ptrIsJsonMarshaler || e.isStringConvertable || e.ptrIsStringConvertable
}

// ToUnstructured converts the provided value to unstructured and returns it.
func (e TypeReflectCacheEntry) ToUnstructured(sv reflect.Value) (interface{}, error) {
	// This is based on https://github.com/kubernetes/kubernetes/blob/82c9e5c814eb7acc6cc0a090c057294d0667ad66/staging/src/k8s.io/apimachinery/pkg/runtime/converter.go#L505
	// and is intended to replace it.

	// Check if the object has a custom string converter and use it if available, since it is much more efficient
	// than round tripping through json.
	if converter, ok := e.getUnstructuredConverter(sv); ok {
		return converter.ToUnstructured(), nil
	}
	// Check if the object has a custom JSON marshaller/unmarshaller.
	if marshaler, ok := e.getJsonMarshaler(sv); ok {
		if sv.Kind() == reflect.Ptr && sv.IsNil() {
			// We're done - we don't need to store anything.
			return nil, nil
		}

		data, err := marshaler.MarshalJSON()
		if err != nil {
			return nil, err
		}
		switch {
		case len(data) == 0:
			return nil, fmt.Errorf("error decoding from json: empty value")

		case bytes.Equal(data, nullBytes):
			// We're done - we don't need to store anything.
			return nil, nil

		case bytes.Equal(data, trueBytes):
			return true, nil

		case bytes.Equal(data, falseBytes):
			return false, nil

		case data[0] == '"':
			var result string
			err := json.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("error decoding string from json: %v", err)
			}
			return result, nil

		case data[0] == '{':
			result := make(map[string]interface{})
			err := json.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("error decoding object from json: %v", err)
			}
			return result, nil

		case data[0] == '[':
			result := make([]interface{}, 0)
			err := json.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("error decoding array from json: %v", err)
			}
			return result, nil

		default:
			var (
				resultInt   int64
				resultFloat float64
				err         error
			)
			if err = json.Unmarshal(data, &resultInt); err == nil {
				return resultInt, nil
			} else if err = json.Unmarshal(data, &resultFloat); err == nil {
				return resultFloat, nil
			} else {
				return nil, fmt.Errorf("error decoding number from json: %v", err)
			}
		}
	}

	return nil, fmt.Errorf("provided type cannot be converted: %v", sv.Type())
}

// CanConvertFromUnstructured returns true if this TypeReflectCacheEntry can convert objects of the type from unstructured.
func (e TypeReflectCacheEntry) CanConvertFromUnstructured() bool {
	return e.isJsonUnmarshaler
}

// FromUnstructured converts the provided source value from unstructured into the provided destination value.
func (e TypeReflectCacheEntry) FromUnstructured(sv, dv reflect.Value) error {
	// TODO: this could be made much more efficient using direct conversions like
	// UnstructuredConverter.ToUnstructured provides.
	st := dv.Type()
	data, err := json.Marshal(sv.Interface())
	if err != nil {
		return fmt.Errorf("error encoding %s to json: %v", st.String(), err)
	}
	if unmarshaler, ok := e.getJsonUnmarshaler(dv); ok {
		return unmarshaler.UnmarshalJSON(data)
	}
	return fmt.Errorf("unable to unmarshal %v into %v", sv.Type(), dv.Type())
}

var (
	nullBytes  = []byte("null")
	trueBytes  = []byte("true")
	falseBytes = []byte("false")
)

func (e TypeReflectCacheEntry) getJsonMarshaler(v reflect.Value) (json.Marshaler, bool) {
	if e.isJsonMarshaler {
		return v.Interface().(json.Marshaler), true
	}
	if e.ptrIsJsonMarshaler {
		// Check pointer receivers if v is not a pointer
		if v.Kind() != reflect.Ptr && v.CanAddr() {
			v = v.Addr()
			return v.Interface().(json.Marshaler), true
		}
	}
	return nil, false
}

func (e TypeReflectCacheEntry) getJsonUnmarshaler(v reflect.Value) (json.Unmarshaler, bool) {
	if !e.isJsonUnmarshaler {
		return nil, false
	}
	return v.Addr().Interface().(json.Unmarshaler), true
}

func (e TypeReflectCacheEntry) getUnstructuredConverter(v reflect.Value) (UnstructuredConverter, bool) {
	if e.isStringConvertable {
		return v.Interface().(UnstructuredConverter), true
	}
	if e.ptrIsStringConvertable {
		// Check pointer receivers if v is not a pointer
		if v.CanAddr() {
			v = v.Addr()
			return v.Interface().(UnstructuredConverter), true
		}
	}
	return nil, false
}

type typeReflectCache struct {
	// use an atomic and copy-on-write since there are a fixed (typically very small) number of structs compiled into any
	// go program using this cache
	value atomic.Value
	// mu is held by writers when performing load/modify/store operations on the cache, readers do not need to hold a
	// read-lock since the atomic value is always read-only
	mu sync.Mutex
}

func newReflectCache() *typeReflectCache {
	cache := &typeReflectCache{}
	cache.value.Store(make(reflectCacheMap))
	return cache
}

type reflectCacheMap map[reflect.Type]TypeReflectCacheEntry

// get returns true and TypeReflectCacheEntry for the given type if the type is in the cache. Otherwise get returns false.
func (c *typeReflectCache) get(t reflect.Type) (TypeReflectCacheEntry, bool) {
	entry, ok := c.value.Load().(reflectCacheMap)[t]
	return entry, ok
}

// update sets the TypeReflectCacheEntry for the given type via a copy-on-write update to the struct cache.
func (c *typeReflectCache) update(t reflect.Type, m TypeReflectCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	currentCacheMap := c.value.Load().(reflectCacheMap)
	if _, ok := currentCacheMap[t]; ok {
		// Bail if the entry has been set while waiting for lock acquisition.
		// This is safe since setting entries is idempotent.
		return
	}

	newCacheMap := make(reflectCacheMap, len(currentCacheMap)+1)
	for k, v := range currentCacheMap {
		newCacheMap[k] = v
	}
	newCacheMap[t] = m
	c.value.Store(newCacheMap)
}
