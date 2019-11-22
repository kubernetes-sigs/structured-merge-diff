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

type List interface {
	Interface() []interface{}
	Length() int
	At(int) Value
}

type ListInterface []interface{}

func (l ListInterface) Interface() []interface{} {
	return l
}

func (l ListInterface) Length() int {
	return len(l)
}

func (l ListInterface) At(i int) Value {
	return ValueInterface{Value: l[i]}
}

// Equals compares two lists lexically.
func ListEquals(lhs, rhs List) bool {
	if lhs.Length() != rhs.Length() {
		return false
	}

	for i := 0; i < lhs.Length(); i++ {
		lv := lhs.At(i)
		if !Equals(lv, rhs.At(i)) {
			return false
		}
	}
	return true
}

// Less compares two lists lexically.
func ListLess(lhs, rhs List) bool {
	return ListCompare(lhs, rhs) == -1
}

// Compare compares two lists lexically. The result will be 0 if l==rhs, -1
// if l < rhs, and +1 if l > rhs.
func ListCompare(lhs, rhs List) int {
	i := 0
	for {
		if i >= lhs.Length() && i >= rhs.Length() {
			// Lists are the same length and all items are equal.
			return 0
		}
		if i >= lhs.Length() {
			// LHS is shorter.
			return -1
		}
		if i >= rhs.Length() {
			// RHS is shorter.
			return 1
		}
		if c := Compare(lhs.At(i), rhs.At(i)); c != 0 {
			return c
		}
		// The items are equal; continue.
		i++
	}
}