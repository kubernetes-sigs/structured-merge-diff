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

package value

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// A Value is an object; it corresponds to an 'atom' in the schema.
type Value struct {
	// Exactly one of the below must be set.
	*Float
	*Int
	*String
	*Boolean
	*List
	*Map
	Null bool // represents an explicit `"foo" = null`
}

type Int int64
type Float float64
type String string
type Boolean bool

// Field is an individual key-value pair.
type Field struct {
	Name  string
	Value Value
}

// String returns a human-readable representation of the field.
func (f Field) String() string {
	return fmt.Sprintf("%v=%v", f.Name, f.Value.HumanReadable())
}

// List is a list of items.
type List struct {
	Items []Value
}

// FollowKeys returns the first value with the corresponding keys or an
// error if no item can be found.
func (l *List) FollowKeys(keys []Field) (Value, error) {
	if l == nil {
		return Value{}, errors.New("can't lookup keys in nil-list")
	}
	return Value{}, fmt.Errorf("couldn't find item for keys %v in list", keys)
}

// FollowValue returns the first value that matches in the list, or an
// error if no item can be found.
func (l *List) FollowValue(value Value) (Value, error) {
	if l == nil {
		return Value{}, errors.New("can't lookup value in nil-list")
	}
	for _, item := range l.Items {
		if reflect.DeepEqual(value, item) {
			return item, nil
		}
	}
	return Value{}, fmt.Errorf("couldn't find value %q in list", value.HumanReadable())
}

// FollowIndex returns the value at the given index, if the index is
// valid for the list.
func (l *List) FollowIndex(index int) (Value, error) {
	if l == nil {
		return Value{}, errors.New("can't lookup index in nil-list")
	}
	if index < 0 {
		return Value{}, errors.New("can't lookup negative index in list")
	}
	if index >= len(l.Items) {
		return Value{}, fmt.Errorf("index out of range: %d/%d", index, len(l.Items))
	}
	return l.Items[index], nil
}

// Map is a map of key-value pairs. It represents both structs and maps. We use
// a list and a go-language map to preserve order.
//
// Set and Get helpers are provided.
type Map struct {
	Items []Field

	// may be nil; lazily constructed.
	// TODO: Direct modifications to Items above will cause serious problems.
	index map[string]*Field
}

// Get returns the (Field, true) or (nil, false) if it is not present
func (m *Map) Get(key string) (*Field, bool) {
	if m.index == nil {
		m.index = map[string]*Field{}
		for i := range m.Items {
			f := &m.Items[i]
			m.index[f.Name] = f
		}
	}
	f, ok := m.index[key]
	return f, ok
}

// Set inserts or updates the given item.
func (m *Map) Set(key string, value Value) {
	if f, ok := m.Get(key); ok {
		f.Value = value
		return
	}
	m.Items = append(m.Items, Field{Name: key, Value: value})
	m.index = nil // Since the append might have reallocated
}

// Follow returns the Value corresponding to the key, or an error if it
// can't be found.
func (m *Map) Follow(key string) (Value, error) {
	if m == nil {
		return Value{}, errors.New("can't look-up field in nil-map")
	}
	f, ok := m.Get(key)
	if !ok {
		return Value{}, fmt.Errorf("could not look-up field %q in map", key)
	}
	return f.Value, nil
}

// StringValue returns s as a scalar string Value.
func StringValue(s string) Value {
	s2 := String(s)
	return Value{String: &s2}
}

// IntValue returns i as a scalar numeric (integer) Value.
func IntValue(i int) Value {
	i2 := Int(i)
	return Value{Int: &i2}
}

// FloatValue returns f as a scalar numeric (float) Value.
func FloatValue(f float64) Value {
	f2 := Float(f)
	return Value{Float: &f2}
}

// BooleanValue returns b as a scalar boolean Value.
func BooleanValue(b bool) Value {
	b2 := Boolean(b)
	return Value{Boolean: &b2}
}

// MatchKeys returns true if the Value would match against the given keys.
func (v Value) MatchKeys(keys []Field) bool {
	if v.Map == nil {
		// Non-map never match the given keys.
		return false
	}

	for _, key := range keys {
		value, found := v.Map.Get(key.Name)
		if !found || !reflect.DeepEqual(value, key.Value) {
			return false
		}
	}
	return true
}

// HumanReadable returns a human-readable representation of the value.
// TODO: Rename this to "String".
func (v Value) HumanReadable() string {
	switch {
	case v.Float != nil:
		return fmt.Sprintf("%v", *v.Float)
	case v.Int != nil:
		return fmt.Sprintf("%v", *v.Int)
	case v.String != nil:
		return fmt.Sprintf("%q", *v.String)
	case v.Boolean != nil:
		return fmt.Sprintf("%v", *v.Boolean)
	case v.List != nil:
		strs := []string{}
		for _, item := range v.List.Items {
			strs = append(strs, item.HumanReadable())
		}
		return "[" + strings.Join(strs, ",") + "]"
	case v.Map != nil:
		strs := []string{}
		for _, i := range v.Map.Items {
			strs = append(strs, i.String())
		}
		return "{" + strings.Join(strs, ";") + "}"
	default:
		fallthrough
	case v.Null == true:
		return "null"
	}
}
