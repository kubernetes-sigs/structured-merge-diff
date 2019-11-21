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

func IsList(v Value) bool {
	if v == nil {
		return false
	}
	_, ok := v.([]interface{})
	return ok
}

func ValueList(v Value) []interface{} {
	return v.([]interface{})
}

// Equals compares two lists lexically.
func ListEquals(lhs, rhs []interface{}) bool {
	if len(lhs) != len(rhs) {
		return false
	}

	for i, lv := range lhs {
		if !Equals(lv, rhs[i]) {
			return false
		}
	}
	return true
}

// Less compares two lists lexically.
func ListLess(lhs, rhs []interface{}) bool {
	return ListCompare(lhs, rhs) == -1
}

// Compare compares two lists lexically. The result will be 0 if l==rhs, -1
// if l < rhs, and +1 if l > rhs.
func ListCompare(lhs, rhs []interface{}) int {
	i := 0
	for {
		if i >= len(lhs) && i >= len(rhs) {
			// Lists are the same length and all items are equal.
			return 0
		}
		if i >= len(lhs) {
			// LHS is shorter.
			return -1
		}
		if i >= len(rhs) {
			// RHS is shorter.
			return 1
		}
		if c := Compare(lhs[i], rhs[i]); c != 0 {
			return c
		}
		// The items are equal; continue.
		i++
	}
}
