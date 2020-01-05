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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

var reflectPool = sync.Pool{
	New: func() interface{} {
		return &valueReflect{}
	},
}

var (
	MarshalerCache = newMarshalerCache()
)

type marshalerTypeCache struct {
	// use an atomic and copy-on-write since there are a fixed (typically very small) number of structs compiled into any
	// go program using this cache
	value atomic.Value
	// mu is held by writers when performing load/modify/store operations on the cache, readers do not need to hold a
	// read-lock since the atomic value is always read-only
	mu sync.Mutex
}

func newMarshalerCache() *marshalerTypeCache {
	cache := &marshalerTypeCache{}
	cache.value.Store(make(marshalerCacheMap))
	return cache
}

type marshalerCacheMap map[reflect.Type]marshalerCacheEntry

type marshalerCacheEntry struct {
	isJsonMarshaler    bool
	isPtrJsonMarshaler bool

	typeConverter UnstructuredStringConverter
}

// Get returns true and marshalerCacheEntry for the given type if the type is in the cache. Otherwise Get returns false.
func (c *marshalerTypeCache) Get(t reflect.Type) (marshalerCacheEntry, bool) {
	entry, ok := c.value.Load().(marshalerCacheMap)[t]
	return entry, ok
}

// Update sets the marshalerCacheEntry for the given type via a copy-on-write update to the struct cache.
func (c *marshalerTypeCache) Update(t reflect.Type, m marshalerCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldCacheMap := c.value.Load().(marshalerCacheMap)
	newCacheMap := make(marshalerCacheMap, len(oldCacheMap)+1)
	for k, v := range oldCacheMap {
		newCacheMap[k] = v
	}
	newCacheMap[t] = m
	c.value.Store(newCacheMap)
}

func (c *marshalerTypeCache) RegisterConverter(t reflect.Type, converter UnstructuredStringConverter) {
	c.Update(t, marshalerCacheEntry{typeConverter: converter})
}

// The below getMarshalerCacheEntry function is an improvement to getMarshaler from
// https://github.com/kubernetes/kubernetes/blob/40df9f82d0572a123f5ad13f48312978a2ff5877/staging/src/k8s.io/apimachinery/pkg/runtime/converter.go#L509
// and should somehow be consolidated with it

var marshalerType = reflect.TypeOf(new(json.Marshaler)).Elem()

func getMarshalerCacheEntry(t reflect.Type) marshalerCacheEntry {
	if record, ok := MarshalerCache.Get(t); ok {
		return record
	}
	record := marshalerCacheEntry{
		isJsonMarshaler:    t.Implements(marshalerType),
		isPtrJsonMarshaler: reflect.PtrTo(t).Implements(marshalerType),
	}
	MarshalerCache.Update(t, record)
	return record
}

// NewValueReflect creates a Value backed by an "interface{}" type,
// typically an structured object in Kubernetes world that is uses reflection to expose.
// The provided "interface{}" may contain structs and types that are converted to Values
// by the jsonMarshaler interface.
func NewValueReflect(value interface{}) (Value, error) {
	if value == nil {
		return NewValueInterface(nil), nil
	}
	return wrapValueReflect(reflect.ValueOf(value))
}

func wrapValueReflect(value reflect.Value) (Value, error) {
	marshelerEntry := getMarshalerCacheEntry(value.Type())
	if marshaler, ok := getMarshaler(marshelerEntry, value); ok {
		return toUnstructured(marshaler, value)
	}
	if marshelerEntry.typeConverter != nil {
		return reflectConverted{Value: value, Converter: marshelerEntry.typeConverter}, nil
	}
	value = dereference(value)
	val := reflectPool.Get().(*valueReflect)
	val.Value = value
	return Value(val), nil
}

func mustWrapValueReflect(value reflect.Value) Value {
	v, err := wrapValueReflect(value)
	if err != nil {
		panic(err)
	}
	return v
}

func dereference(val reflect.Value) reflect.Value {
	kind := val.Kind()
	if (kind == reflect.Interface || kind == reflect.Ptr) && !safeIsNil(val) {
		return val.Elem()
	}
	return val
}

type valueReflect struct {
	Value reflect.Value
}

func (r valueReflect) IsMap() bool {
	return r.isKind(reflect.Map, reflect.Struct)
}

func (r valueReflect) IsList() bool {
	typ := r.Value.Type()
	return typ.Kind() == reflect.Slice && !(typ.Elem().Kind() == reflect.Uint8)
}

func (r valueReflect) IsBool() bool {
	return r.isKind(reflect.Bool)
}

func (r valueReflect) IsInt() bool {
	// Uint64 deliberately excluded, see valueUnstructured.Int.
	return r.isKind(reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Uint, reflect.Uint32, reflect.Uint16, reflect.Uint8)
}

func (r valueReflect) IsFloat() bool {
	return r.isKind(reflect.Float64, reflect.Float32)
}

func (r valueReflect) IsString() bool {
	kind := r.Value.Kind()
	if kind == reflect.String {
		return true
	}
	if kind == reflect.Slice && r.Value.Type().Elem().Kind() == reflect.Uint8 {
		return true
	}
	return false
}

func (r valueReflect) IsNull() bool {
	return safeIsNil(r.Value)
}

func (r valueReflect) isKind(kinds ...reflect.Kind) bool {
	kind := r.Value.Kind()
	for _, k := range kinds {
		if kind == k {
			return true
		}
	}
	return false
}

// TODO find a cleaner way to avoid panics from reflect.IsNil()
func safeIsNil(v reflect.Value) bool {
	k := v.Kind()
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func (r valueReflect) Map() Map {
	val := r.Value
	switch val.Kind() {
	case reflect.Struct:
		return structReflect{Value: r.Value}
	case reflect.Map:
		return mapReflect{Value: r.Value}
	default:
		panic("value is not a map or struct")
	}
}

func (r *valueReflect) Recycle() {
	reflectPool.Put(r)
}

func (r valueReflect) List() List {
	if r.IsList() {
		return listReflect{r.Value}
	}
	panic("value is not a list")
}

func (r valueReflect) Bool() bool {
	if r.IsBool() {
		return r.Value.Bool()
	}
	panic("value is not a bool")
}

func (r valueReflect) Int() int64 {
	if r.isKind(reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8) {
		return r.Value.Int()
	}
	if r.isKind(reflect.Uint, reflect.Uint32, reflect.Uint16, reflect.Uint8) {
		return int64(r.Value.Uint())
	}

	panic("value is not an int")
}

func (r valueReflect) Float() float64 {
	if r.IsFloat() {
		return r.Value.Float()
	}
	panic("value is not a float")
}

func (r valueReflect) String() string {
	kind := r.Value.Kind()
	if kind == reflect.String {
		return r.Value.String()
	}
	if kind == reflect.Slice && r.Value.Type().Elem().Kind() == reflect.Uint8 {
		return base64.StdEncoding.EncodeToString(r.Value.Bytes())
	}
	panic("value is not a string")
}

func (r valueReflect) Unstructured() interface{} {
	val := r.Value
	switch {
	case r.IsNull():
		return nil
	case val.Kind() == reflect.Struct:
		return structReflect{Value: r.Value}.Unstructured()
	case val.Kind() == reflect.Map:
		return mapReflect{Value: r.Value}.Unstructured()
	case r.IsList():
		return listReflect{Value: r.Value}.Unstructured()
	case r.IsString():
		return r.String()
	case r.IsInt():
		return r.Int()
	case r.IsBool():
		return r.Bool()
	case r.IsFloat():
		return r.Float()
	default:
		panic(fmt.Sprintf("value of type %s is not a supported by value reflector", val.Type()))
	}
}

// The below toUnstructured functions are based on
// https://github.com/kubernetes/kubernetes/blob/40df9f82d0572a123f5ad13f48312978a2ff5877/staging/src/k8s.io/apimachinery/pkg/runtime/converter.go#L509
// and should somehow be consolidated with it

func getMarshaler(entry marshalerCacheEntry, v reflect.Value) (json.Marshaler, bool) {
	if entry.isJsonMarshaler {
		return v.Interface().(json.Marshaler), true
	}
	if entry.isPtrJsonMarshaler {
		// Check pointer receivers if v is not a pointer
		if v.Kind() != reflect.Ptr && v.CanAddr() {
			return v.Addr().Interface().(json.Marshaler), true
		}
	}
	return nil, false
}

var (
	nullBytes  = []byte("null")
	trueBytes  = []byte("true")
	falseBytes = []byte("false")
)

func toUnstructured(marshaler json.Marshaler, sv reflect.Value) (Value, error) {
	data, err := marshaler.MarshalJSON()
	if err != nil {
		return nil, err
	}
	switch {
	case len(data) == 0:
		return nil, fmt.Errorf("error decoding from json: empty value")

	case bytes.Equal(data, nullBytes):
		// We're done - we don't need to store anything.
		return NewValueInterface(nil), nil

	case bytes.Equal(data, trueBytes):
		return NewValueInterface(true), nil

	case bytes.Equal(data, falseBytes):
		return NewValueInterface(false), nil

	case data[0] == '"':
		var result string
		err := json.Unmarshal(data, &result)
		if err != nil {
			return nil, fmt.Errorf("error decoding string from json: %v", err)
		}
		return NewValueInterface(result), nil

	case data[0] == '{':
		result := make(map[string]interface{})
		err := json.Unmarshal(data, &result)
		if err != nil {
			return nil, fmt.Errorf("error decoding object from json: %v", err)
		}
		return NewValueInterface(result), nil

	case data[0] == '[':
		result := make([]interface{}, 0)
		err := json.Unmarshal(data, &result)
		if err != nil {
			return nil, fmt.Errorf("error decoding array from json: %v", err)
		}
		return NewValueInterface(result), nil

	default:
		var (
			resultInt   int64
			resultFloat float64
			err         error
		)
		if err = json.Unmarshal(data, &resultInt); err == nil {
			return NewValueInterface(resultInt), nil
		}
		if err = json.Unmarshal(data, &resultFloat); err == nil {
			return NewValueInterface(resultFloat), nil
		}
		return nil, fmt.Errorf("error decoding number from json: %v", err)
	}
}

type UnstructuredStringConverter interface {
	ToString(v reflect.Value) string
	IsNull(v reflect.Value) bool
}

type reflectConverted struct {
	Value     reflect.Value
	Converter UnstructuredStringConverter
}

func (r reflectConverted) IsMap() bool {
	return false
}

func (r reflectConverted) IsList() bool {
	return false
}

func (r reflectConverted) IsBool() bool {
	return false
}

func (r reflectConverted) IsInt() bool {
	return false
}

func (r reflectConverted) IsFloat() bool {
	return false
}

func (r reflectConverted) IsString() bool {
	return !r.IsNull()
}

func (r reflectConverted) IsNull() bool {
	if safeIsNil(r.Value) {
		return true
	}
	if r.Value.Kind() == reflect.Ptr {
		return r.Converter.IsNull(r.Value.Elem())
	}
	return r.Converter.IsNull(r.Value)
}

func (r reflectConverted) Map() Map {
	panic("value is not a map")
}

func (r reflectConverted) List() List {
	panic("value is not a fieldList")
}

func (r reflectConverted) Bool() bool {
	panic("value is not a boolean")
}

func (r reflectConverted) Int() int64 {
	panic("value is not a int")
}

func (r reflectConverted) Float() float64 {
	panic("value is not a float")
}

func (r reflectConverted) String() string {
	if r.Value.Kind() == reflect.Ptr {
		return r.Converter.ToString(r.Value.Elem())
	}
	return r.Converter.ToString(r.Value)
}

func (r reflectConverted) Recycle() {

}

func (r reflectConverted) Unstructured() interface{} {
	if r.IsNull() {
		return nil
	}
	return r.String()
}
