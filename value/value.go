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
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

var (
	readPool  = jsoniter.NewIterator(jsoniter.ConfigCompatibleWithStandardLibrary).Pool()
	writePool = jsoniter.NewStream(jsoniter.ConfigCompatibleWithStandardLibrary, nil, 1024).Pool()
)

// Equals returns true iff the two values are equal.
func Equals(lhs, rhs Value) bool {
	if IsFloat(lhs) || IsFloat(rhs) {
		var lf float64
		if IsFloat(lhs) {
			lf = ValueFloat(lhs)
		} else if IsInt(lhs) {
			lf = float64(ValueInt(lhs))
		} else {
			return false
		}
		var rf float64
		if IsFloat(rhs) {
			rf = ValueFloat(rhs)
		} else if IsInt(rhs) {
			rf = float64(ValueInt(rhs))
		} else {
			return false
		}
		return lf == rf
	}
	if IsInt(lhs) {
		if IsInt(rhs) {
			return ValueInt(lhs) == ValueInt(rhs)
		}
		return false
	}
	if IsString(lhs) {
		if IsString(rhs) {
			return ValueString(lhs) == ValueString(rhs)
		}
		return false
	}
	if IsBool(lhs) {
		if IsBool(rhs) {
			return ValueBool(lhs) == ValueBool(rhs)
		}
		return false
	}
	if IsList(lhs) {
		if IsList(rhs) {
			return ListEquals(ValueList(lhs), ValueList(rhs))
		}
		return false
	}
	if IsMap(lhs) {
		if IsMap(rhs) {
			return MapEquals(ValueMap(lhs), ValueMap(rhs))
		}
		return false
	}
	if IsNull(lhs) {
		if IsNull(rhs) {
			return true
		}
		return false
	}
	// No field is set, on either objects.
	return true
}

// Less provides a total ordering for Value (so that they can be sorted, even
// if they are of different types).
func Less(lhs, rhs Value) bool {
	return Compare(lhs, rhs) == -1
}

// Compare provides a total ordering for Value (so that they can be
// sorted, even if they are of different types). The result will be 0 if
// v==rhs, -1 if v < rhs, and +1 if v > rhs.
func Compare(lhs, rhs Value) int {
	if IsFloat(lhs) {
		if !IsFloat(rhs) {
			// Extra: compare floats and ints numerically.
			if IsInt(rhs) {
				return FloatCompare(ValueFloat(lhs), float64(ValueInt(rhs)))
			}
			return -1
		}
		return FloatCompare(ValueFloat(lhs), ValueFloat(rhs))
	} else if IsFloat(rhs) {
		// Extra: compare floats and ints numerically.
		if IsInt(lhs) {
			return FloatCompare(float64(ValueInt(lhs)), ValueFloat(rhs))
		}
		return 1
	}

	if IsInt(lhs) {
		if !IsInt(rhs) {
			return -1
		}
		return IntCompare(ValueInt(lhs), ValueInt(rhs))
	} else if IsInt(rhs) {
		return 1
	}

	if IsString(lhs) {
		if !IsString(rhs) {
			return -1
		}
		return strings.Compare(ValueString(lhs), ValueString(rhs))
	} else if IsString(rhs) {
		return 1
	}

	if IsBool(lhs) {
		if !IsBool(rhs) {
			return -1
		}
		return BoolCompare(ValueBool(lhs), ValueBool(rhs))
	} else if IsBool(rhs) {
		return 1
	}

	if IsList(lhs) {
		if !IsList(rhs) {
			return -1
		}
		return ListCompare(ValueList(lhs), ValueList(rhs))
	} else if IsList(rhs) {
		return 1
	}
	if IsMap(lhs) {
		if !IsMap(rhs) {
			return -1
		}
		return MapCompare(ValueMap(lhs), ValueMap(rhs))
	} else if IsMap(rhs) {
		return 1
	}
	if IsNull(lhs) {
		if !IsNull(rhs) {
			return -1
		}
		return 0
	} else if IsNull(rhs) {
		return 1
	}

	// Invalid Value-- nothing is set.
	return 0
}

func FromJSON(input []byte) (Value, error) {
	return FromJSONFast(input)
}

func ToJSON(val Value) ([]byte, error) {
	buf := bytes.Buffer{}
	stream := writePool.BorrowStream(&buf)
	defer writePool.ReturnStream(stream)
	WriteJSONStream(val, stream)
	b := stream.Buffer()
	err := stream.Flush()
	// Help jsoniter manage its buffers--without this, the next
	// use of the stream is likely to require an allocation. Look
	// at the jsoniter stream code to understand why. They were probably
	// optimizing for folks using the buffer directly.
	stream.SetBuffer(b[:0])
	return buf.Bytes(), err
}

func FromJSONFast(input []byte) (Value, error) {
	iter := readPool.BorrowIterator(input)
	defer readPool.ReturnIterator(iter)
	return ReadJSONIter(iter)
}

func ReadJSONIter(iter *jsoniter.Iterator) (Value, error) {
	v := iter.Read()
	if iter.Error != nil && iter.Error != io.EOF {
		return nil, iter.Error
	}
	return v, nil
}

func WriteJSONStream(v Value, stream *jsoniter.Stream) {
	stream.WriteVal(v)
}

// IntCompare compares integers. The result will be 0 if i==rhs, -1 if i <
// rhs, and +1 if i > rhs.
func IntCompare(lhs, rhs int64) int {
	if lhs > rhs {
		return 1
	} else if lhs < rhs {
		return -1
	}
	return 0
}

// Compare compares floats. The result will be 0 if lhs==rhs, -1 if f <
// rhs, and +1 if f > rhs.
func FloatCompare(lhs, rhs float64) int {
	if lhs > rhs {
		return 1
	} else if lhs < rhs {
		return -1
	}
	return 0
}

// Compare compares booleans. The result will be 0 if b==rhs, -1 if b <
// rhs, and +1 if b > rhs.
func BoolCompare(lhs, rhs bool) int {
	if lhs == rhs {
		return 0
	} else if lhs == false {
		return -1
	}
	return 1
}

// Field is an individual key-value pair.
type Field struct {
	Name  string
	Value Value
}

// FieldList is a list of key-value pairs. Each field is expected to
// have a different name.
type FieldList []Field

// Sort sorts the field list by Name.
func (f FieldList) Sort() {
	if len(f) < 2 {
		return
	}
	if len(f) == 2 {
		if f[1].Name < f[0].Name {
			f[0], f[1] = f[1], f[0]
		}
		return
	}
	sort.SliceStable(f, func(i, j int) bool {
		return f[i].Name < f[j].Name
	})
}

// Less compares two lists lexically.
func (f FieldList) Less(rhs FieldList) bool {
	return f.Compare(rhs) == -1
}

// Less compares two lists lexically. The result will be 0 if f==rhs, -1
// if f < rhs, and +1 if f > rhs.
func (f FieldList) Compare(rhs FieldList) int {
	i := 0
	for {
		if i >= len(f) && i >= len(rhs) {
			// Maps are the same length and all items are equal.
			return 0
		}
		if i >= len(f) {
			// F is shorter.
			return -1
		}
		if i >= len(rhs) {
			// RHS is shorter.
			return 1
		}
		if c := strings.Compare(f[i].Name, rhs[i].Name); c != 0 {
			return c
		}
		if c := Compare(f[i].Value, rhs[i].Value); c != 0 {
			return c
		}
		// The items are equal; continue.
		i++
	}
}

type Value interface{}

func Copy(v Value) Value {
	if IsList(v) {
		l := make([]interface{}, 0, len(ValueList(v)))
		for _, item := range ValueList(v) {
			l = append(l, Copy(item))
		}
		return l
	}
	if IsMap(v) {
		m := make(map[string]interface{}, len(ValueMap(v)))
		for key, item := range ValueMap(v) {
			m[key] = Copy(item)
		}
		return m
	}
	// Scalars don't have to be copied
	return v
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

func ToString(v Value) string {
	if v == nil {
		return "null"
	}
	switch {
	case IsFloat(v):
		return fmt.Sprintf("%v", ValueFloat(v))
	case IsInt(v):
		return fmt.Sprintf("%v", ValueInt(v))
	case IsString(v):
		return fmt.Sprintf("%q", ValueString(v))
	case IsBool(v):
		return fmt.Sprintf("%v", ValueBool(v))
	case IsList(v):
		strs := []string{}
		for _, item := range ValueList(v) {
			strs = append(strs, ToString(item))
		}
		return "[" + strings.Join(strs, ",") + "]"
	case IsMap(v):
		strs := []string{}
		for k, v := range ValueMap(v) {
			strs = append(strs, fmt.Sprintf("%v=%v", k, ToString(v)))
		}
		return "{" + strings.Join(strs, ";") + "}"
	}
	return fmt.Sprintf("{{undefined(%#v)}}", v)
}

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

func IsFloat(v Value) bool {
	if v == nil {
		return false
	} else if _, ok := v.(float64); ok {
		return true
	} else if _, ok := v.(float32); ok {
		return true
	}
	return false
}

func ValueFloat(v Value) float64 {
	if f, ok := v.(float32); ok {
		return float64(f)
	}
	return v.(float64)
}

func IsInt(v Value) bool {
	if v == nil {
		return false
	} else if _, ok := v.(int); ok {
		return true
	} else if _, ok := v.(int8); ok {
		return true
	} else if _, ok := v.(int16); ok {
		return true
	} else if _, ok := v.(int32); ok {
		return true
	} else if _, ok := v.(int64); ok {
		return true
	}
	return false
}

func ValueInt(v Value) int64 {
	if i, ok := v.(int); ok {
		return int64(i)
	} else if i, ok := v.(int8); ok {
		return int64(i)
	} else if i, ok := v.(int16); ok {
		return int64(i)
	} else if i, ok := v.(int32); ok {
		return int64(i)
	}
	return v.(int64)
}

func IsString(v Value) bool {
	if v == nil {
		return false
	}
	_, ok := v.(string)
	return ok
}

func ValueString(v Value) string {
	return v.(string)
}

func IsBool(v Value) bool {
	if v == nil {
		return false
	}
	_, ok := v.(bool)
	return ok
}

func ValueBool(v Value) bool {
	return v.(bool)
}

func IsNull(v Value) bool {
	return v == nil
}
