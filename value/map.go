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
	"sort"
	"strings"
)

// Equals compares two maps lexically.
func MapEquals(lhs, rhs map[string]interface{}) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for k, vl := range lhs {
		vr, ok := rhs[k]
		if !ok {
			return false
		}
		if !Equals(vl, vr) {
			return false
		}
	}
	return true
}

// Less compares two maps lexically.
func MapLess(lhs, rhs map[string]interface{}) bool {
	return Compare(lhs, rhs) == -1
}

// Compare compares two maps lexically.
func MapCompare(lhs, rhs map[string]interface{}) int {
	lorder := make([]string, 0, len(lhs))
	for key := range lhs {
		lorder = append(lorder, key)
	}
	sort.Strings(lorder)
	rorder := make([]string, 0, len(rhs))
	for key := range rhs {
		rorder = append(rorder, key)
	}
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
		if c := Compare(lhs[lorder[i]], rhs[lorder[i]]); c != 0 {
			return c
		}
		// The items are equal; continue.
		i++
	}
}

func IsMap(v Value) bool {
	if _, ok := v.(map[string]interface{}); ok {
		return true
	} else if _, ok := v.(map[interface{}]interface{}); ok {
		return true
	}
	return false
}

func ValueMap(v Value) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	switch t := v.(type) {
	case map[string]interface{}:
		return t
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(t))
		for key, value := range t {
			if ks, ok := key.(string); ok {
				m[ks] = value
			}
		}
		return m
	}
	panic(fmt.Errorf("not a map: %#v", v))
}
