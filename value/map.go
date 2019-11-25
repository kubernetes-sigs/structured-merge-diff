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
	"sort"
	"strings"
)

type Map interface {
	Set(key string, val Value)
	Get(key string) (Value, bool)
	Delete(key string)
	Equals(other Map) bool
	Iterate(func(key string, value Value) bool) bool
	Length() int
}

// Less compares two maps lexically.
func MapLess(lhs, rhs Map) bool {
	return MapCompare(lhs, rhs) == -1
}

// Compare compares two maps lexically.
func MapCompare(lhs, rhs Map) int {
	lorder := make([]string, 0, lhs.Length())
	lhs.Iterate(func(key string, _ Value) bool {
		lorder = append(lorder, key)
		return true
	})
	sort.Strings(lorder)
	rorder := make([]string, 0, rhs.Length())
	rhs.Iterate(func(key string, _ Value) bool {
		rorder = append(rorder, key)
		return true
	})
	sort.Strings(rorder)

	i := 0
	for {
		if i >= len(lorder) && i >= len(rorder) {
			// Maps are the same length and all items are equal.
			return 0
		}
		if i >= len(lorder) {
			// LHS is shorter.
			return -1
		}
		if i >= len(rorder) {
			// RHS is shorter.
			return 1
		}
		if c := strings.Compare(lorder[i], rorder[i]); c != 0 {
			return c
		}
		litem, _ := lhs.Get(lorder[i])
		ritem, _ := rhs.Get(rorder[i])
		if c := Compare(litem, ritem); c != 0 {
			return c
		}
		litem.Recycle()
		ritem.Recycle()
		// The items are equal; continue.
		i++
	}
}

type MapInterface map[interface{}]interface{}

func (m MapInterface) Set(key string, val Value) {
	m[key] = val.Interface()
}

func (m MapInterface) Get(key string) (Value, bool) {
	if v, ok := m[key]; !ok {
		return nil, false
	} else {
		return NewValueInterface(v), true
	}
}

func (m MapInterface) Delete(key string) {
	delete(m, key)
}

func (m MapInterface) Iterate(fn func(key string, value Value) bool) bool {
	for k, v := range m {
		if ks, ok := k.(string); !ok {
			continue
		} else {
			vv := NewValueInterface(v)
			if !fn(ks, vv) {
				vv.Recycle()
				return false
			}
			vv.Recycle()
		}
	}
	return true
}

func (m MapInterface) Length() int {
	return len(m)
}

func (m MapInterface) Equals(other Map) bool {
	if m.Length() != other.Length() {
		return false
	}
	for k, v := range m {
		ks, ok := k.(string)
		if !ok {
			return false
		}
		vo, ok := other.Get(ks)
		if !ok {
			return false
		}
		vv := NewValueInterface(v)
		if !Equals(vv, vo) {
			vv.Recycle()
			vo.Recycle()
			return false
		}
		vo.Recycle()
		vv.Recycle()
	}
	return true
}

type MapString map[string]interface{}

func (m MapString) Set(key string, val Value) {
	m[key] = val.Interface()
}

func (m MapString) Get(key string) (Value, bool) {
	if v, ok := m[key]; !ok {
		return nil, false
	} else {
		return NewValueInterface(v), true
	}
}

func (m MapString) Delete(key string) {
	delete(m, key)
}

func (m MapString) Iterate(fn func(key string, value Value) bool) bool {
	for k, v := range m {
		vv := NewValueInterface(v)
		if !fn(k, vv) {
			vv.Recycle()
			return false
		}
		vv.Recycle()
	}
	return true
}

func (m MapString) Length() int {
	return len(m)
}

func (m MapString) Equals(other Map) bool {
	if m.Length() != other.Length() {
		return false
	}
	for k, v := range m {
		vo, ok := other.Get(k)
		if !ok {
			return false
		}
		vv := NewValueInterface(v)
		if !Equals(vv, vo) {
			vo.Recycle()
			vv.Recycle()
			return false
		}
		vo.Recycle()
		vv.Recycle()
	}
	return true
}
