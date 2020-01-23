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
	"sync"
)

type listReflect struct {
	Value reflect.Value
}

func (r listReflect) Length() int {
	val := r.Value
	return val.Len()
}

func (r listReflect) At(i int) Value {
	val := r.Value
	return mustWrapValueReflect(val.Index(i))
}

func (r listReflect) Unstructured() interface{} {
	l := r.Length()
	result := make([]interface{}, l)
	for i := 0; i < l; i++ {
		result[i] = r.At(i).Unstructured()
	}
	return result
}

var lrrPool = sync.Pool{
	New: func() interface{} {
		return &listReflectRange{vr: &valueReflect{}}
	},
}

func (r listReflect) Range() ListRange {
	length := r.Value.Len()
	if length == 0 {
		return EmptyRange
	}
	rr := lrrPool.Get().(*listReflectRange)
	rr.list = r.Value
	rr.i = -1
	return rr
}

type listReflectRange struct {
	list reflect.Value
	vr   *valueReflect
	i    int
}

func (r *listReflectRange) Next() bool {
	r.i += 1
	return r.i < r.list.Len()
}

func (r *listReflectRange) Item() (index int, value Value) {
	if r.i < 0 {
		panic("Item() called before first calling Next()")
	}
	if r.i >= r.list.Len() {
		panic("Item() called on ListRange with no more items")
	}
	return r.i, r.vr.reuse(r.list.Index(r.i))
}

func (r *listReflectRange) Recycle() {
	lrrPool.Put(r)
}
