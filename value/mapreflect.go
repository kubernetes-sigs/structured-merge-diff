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
	"reflect"
)

type mapReflect struct {
	valueReflect
}

func (r mapReflect) Length() int {
	val := r.Value
	return val.Len()
}

func (r mapReflect) Get(key string) (Value, bool) {
	k, v, ok := r.get(key)
	if !ok {
		return nil, false
	}
	return mustWrapValueReflect(v, &r.Value, &k), true
}

func (r mapReflect) get(k string) (key, value reflect.Value, ok bool) {
	mapKey := r.toMapKey(k)
	val := r.Value.MapIndex(mapKey)
	return mapKey, val, val.IsValid() && val != reflect.Value{}
}

func (r mapReflect) Has(key string) bool {
	var val reflect.Value
	val = r.Value.MapIndex(r.toMapKey(key))
	if !val.IsValid() {
		return false
	}
	return val != reflect.Value{}
}

func (r mapReflect) Set(key string, val Value) {
	r.Value.SetMapIndex(r.toMapKey(key), reflect.ValueOf(val.Unstructured()))
}

func (r mapReflect) Delete(key string) {
	val := r.Value
	val.SetMapIndex(r.toMapKey(key), reflect.Value{})
}

// TODO: Do we need to support types that implement json.Marshaler and are used as string keys?
func (r mapReflect) toMapKey(key string) reflect.Value {
	val := r.Value
	return reflect.ValueOf(key).Convert(val.Type().Key())
}

func (r mapReflect) Iterate(fn func(string, Value) bool) bool {
	if r.Value.Len() == 0 {
		return true
	}
	vr := reflectPool.Get().(*valueReflect)
	defer vr.Recycle()
	return eachMapEntry(r.Value, func(e *TypeReflectCacheEntry, key reflect.Value, value reflect.Value) bool {
		// TODO: Track cache entry
		return fn(key.String(), vr.mustReuse(value, nil, &r.Value, &key))
	})
}

func eachMapEntry(val reflect.Value, fn func(*TypeReflectCacheEntry, reflect.Value, reflect.Value) bool) bool {
	iter := val.MapRange()
	var entry *TypeReflectCacheEntry
	for iter.Next() {
		next := iter.Value()
		if !next.IsValid() {
			continue
		}
		if entry == nil {
			entry = TypeReflectEntryOf(next.Type())
		}
		if !fn(entry, iter.Key(), next) {
			return false
		}
	}
	return true
}

func (r mapReflect) Unstructured() interface{} {
	result := make(map[string]interface{}, r.Length())
	r.Iterate(func(s string, value Value) bool {
		result[s] = value.Unstructured()
		return true
	})
	return result
}

func (r mapReflect) Equals(m Map) bool {
	lhsLength := r.Length()
	rhsLength := m.Length()
	if lhsLength != rhsLength {
		return false
	}
	if lhsLength == 0 {
		return true
	}
	vr := reflectPool.Get().(*valueReflect)
	defer vr.Recycle()
	return m.Iterate(func(key string, value Value) bool {
		_, lhsVal, ok := r.get(key)
		if !ok {
			return false
		}
		// TODO: Track cache entry
		return Equals(vr.mustReuse(lhsVal, nil, nil, nil), value)
	})
}
