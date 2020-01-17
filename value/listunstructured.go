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

type listUnstructured []interface{}

func (l listUnstructured) Length() int {
	return len(l)
}

func (l listUnstructured) At(i int) Value {
	return NewValueInterface(l[i])
}

func (l listUnstructured) Range() ListRange {
	if len(l) == 0 {
		return &listUnstructuredRange{l, nil, -1, 0}
	}
	vv := viPool.Get().(*valueUnstructured)
	return &listUnstructuredRange{l, vv, -1, len(l)}
}

type listUnstructuredRange struct {
	list   listUnstructured
	vv     *valueUnstructured
	i      int
	length int
}

func (r *listUnstructuredRange) Next() bool {
	r.i += 1
	return r.i < r.length
}

func (r *listUnstructuredRange) Item() (index int, value Value) {
	if r.i < 0 {
		panic("Item() called before first calling Next()")
	}
	if r.i >= r.length {
		panic("Item() called on ListRange with no more items")
	}

	r.vv.Value = r.list[r.i]
	return r.i, r.vv
}

func (r *listUnstructuredRange) Recycle() {
	if r.vv != nil {
		r.vv.Recycle()
	}
}
