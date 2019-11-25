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
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

var (
	readPool  = jsoniter.NewIterator(jsoniter.ConfigCompatibleWithStandardLibrary).Pool()
	writePool = jsoniter.NewStream(jsoniter.ConfigCompatibleWithStandardLibrary, nil, 1024).Pool()
)

func FromJSON(input []byte) (Value, error) {
	return FromJSONFast(input)
}

func FromJSONFast(input []byte) (Value, error) {
	iter := readPool.BorrowIterator(input)
	defer readPool.ReturnIterator(iter)
	return ReadJSONIter(iter)
}

func ToJSON(v Value) ([]byte, error) {
	buf := bytes.Buffer{}
	stream := writePool.BorrowStream(&buf)
	defer writePool.ReturnStream(stream)
	WriteJSONStream(v, stream)
	b := stream.Buffer()
	err := stream.Flush()
	// Help jsoniter manage its buffers--without this, the next
	// use of the stream is likely to require an allocation. Look
	// at the jsoniter stream code to understand why. They were probably
	// optimizing for folks using the buffer directly.
	stream.SetBuffer(b[:0])
	return buf.Bytes(), err
}

func ReadJSONIter(iter *jsoniter.Iterator) (Value, error) {
	v := iter.Read()
	if iter.Error != nil && iter.Error != io.EOF {
		return nil, iter.Error
	}
	return NewValueInterface(v), nil
}

func WriteJSONStream(v Value, stream *jsoniter.Stream) {
	stream.WriteVal(v.Interface())
}

type Value interface {
	IsMap() bool
	IsList() bool
	IsBool() bool
	IsInt() bool
	IsFloat() bool
	IsString() bool
	IsNull() bool

	Map() Map
	List() List
	Bool() bool
	Int() int64
	Float() float64
	String() string

	// Returns a value of this type that is no longer needed. The
	// value shouldn't be used after this call.
	Recycle()

	Copy() Value
	Interface() interface{}
}

var viPool = sync.Pool{
	New: func() interface{} {
		return &ValueInterface{}
	},
}

func NewValueInterface(v interface{}) Value {
	vi := viPool.Get().(*ValueInterface)
	vi.Value = v
	return Value(vi)
}

type ValueInterface struct {
	Value interface{}
}

func (v ValueInterface) IsMap() bool {
	if _, ok := v.Value.(map[string]interface{}); ok {
		return true
	}
	if _, ok := v.Value.(map[interface{}]interface{}); ok {
		return true
	}
	return false
}

func (v ValueInterface) Map() Map {
	if v.Value == nil {
		return MapString(nil)
	}
	switch t := v.Value.(type) {
	case map[string]interface{}:
		return MapString(t)
	case map[interface{}]interface{}:
		return MapInterface(t)
	}
	panic(fmt.Errorf("not a map: %#v", v))
}

func (v ValueInterface) IsList() bool {
	if v.Value == nil {
		return false
	}
	_, ok := v.Value.([]interface{})
	return ok
}

func (v ValueInterface) List() List {
	return ListInterface(v.Value.([]interface{}))
}

func (v ValueInterface) IsFloat() bool {
	if v.Value == nil {
		return false
	} else if _, ok := v.Value.(float64); ok {
		return true
	} else if _, ok := v.Value.(float32); ok {
		return true
	}
	return false
}

func (v ValueInterface) Float() float64 {
	if f, ok := v.Value.(float32); ok {
		return float64(f)
	}
	return v.Value.(float64)
}

func (v ValueInterface) IsInt() bool {
	if v.Value == nil {
		return false
	} else if _, ok := v.Value.(int); ok {
		return true
	} else if _, ok := v.Value.(int8); ok {
		return true
	} else if _, ok := v.Value.(int16); ok {
		return true
	} else if _, ok := v.Value.(int32); ok {
		return true
	} else if _, ok := v.Value.(int64); ok {
		return true
	}
	return false
}

func (v ValueInterface) Int() int64 {
	if i, ok := v.Value.(int); ok {
		return int64(i)
	} else if i, ok := v.Value.(int8); ok {
		return int64(i)
	} else if i, ok := v.Value.(int16); ok {
		return int64(i)
	} else if i, ok := v.Value.(int32); ok {
		return int64(i)
	}
	return v.Value.(int64)
}

func (v ValueInterface) IsString() bool {
	if v.Value == nil {
		return false
	}
	_, ok := v.Value.(string)
	return ok
}

func (v ValueInterface) String() string {
	return v.Value.(string)
}

func (v ValueInterface) IsBool() bool {
	if v.Value == nil {
		return false
	}
	_, ok := v.Value.(bool)
	return ok
}

func (v ValueInterface) Bool() bool {
	return v.Value.(bool)
}

func (v ValueInterface) IsNull() bool {
	return v.Value == nil
}

func (v *ValueInterface) Recycle() {
	viPool.Put(v)
}

func (v ValueInterface) Interface() interface{} {
	return v.Value
}

func (v *ValueInterface) Copy() Value {
	if v.IsList() {
		l := make([]interface{}, 0, v.List().Length())
		for i := 0; i < v.List().Length(); i++ {
			l = append(l, v.List().At(i).Copy().Interface())
		}
		return NewValueInterface(l)
	}
	if v.IsMap() {
		m := make(map[string]interface{}, v.Map().Length())
		v.Map().Iterate(func(key string, item Value) bool {
			m[key] = item.Copy().Interface()
			return true
		})
		return NewValueInterface(m)
	}
	// Scalars don't have to be copied
	return v
}

// Equals returns true iff the two values are equal.
func Equals(lhs, rhs Value) bool {
	if lhs.IsFloat() || rhs.IsFloat() {
		var lf float64
		if lhs.IsFloat() {
			lf = lhs.Float()
		} else if lhs.IsInt() {
			lf = float64(lhs.Int())
		} else {
			return false
		}
		var rf float64
		if rhs.IsFloat() {
			rf = rhs.Float()
		} else if rhs.IsInt() {
			rf = float64(rhs.Int())
		} else {
			return false
		}
		return lf == rf
	}
	if lhs.IsInt() {
		if rhs.IsInt() {
			return lhs.Int() == rhs.Int()
		}
		return false
	}
	if lhs.IsString() {
		if rhs.IsString() {
			return lhs.String() == rhs.String()
		}
		return false
	}
	if lhs.IsBool() {
		if rhs.IsBool() {
			return lhs.Bool() == rhs.Bool()
		}
		return false
	}
	if lhs.IsList() {
		if rhs.IsList() {
			return ListEquals(lhs.List(), rhs.List())
		}
		return false
	}
	if lhs.IsMap() {
		if rhs.IsMap() {
			return lhs.Map().Equals(rhs.Map())
		}
		return false
	}
	if lhs.IsNull() {
		if rhs.IsNull() {
			return true
		}
		return false
	}
	// No field is set, on either objects.
	return true
}

func ToString(v Value) string {
	if v.IsNull() {
		return "null"
	}
	switch {
	case v.IsFloat():
		return fmt.Sprintf("%v", v.Float())
	case v.IsInt():
		return fmt.Sprintf("%v", v.Int())
	case v.IsString():
		return fmt.Sprintf("%q", v.String())
	case v.IsBool():
		return fmt.Sprintf("%v", v.Bool())
	case v.IsList():
		strs := []string{}
		for i := 0; i < v.List().Length(); i++ {
			strs = append(strs, ToString(v.List().At(i)))
		}
		return "[" + strings.Join(strs, ",") + "]"
	case v.IsMap():
		strs := []string{}
		v.Map().Iterate(func(k string, v Value) bool {
			strs = append(strs, fmt.Sprintf("%v=%v", k, ToString(v)))
			return true
		})
		return strings.Join(strs, "")
	}
	// No field is set, on either objects.
	return "{{undefined}}"
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
	if lhs.IsFloat() {
		if !rhs.IsFloat() {
			// Extra: compare floats and ints numerically.
			if rhs.IsInt() {
				return FloatCompare(lhs.Float(), float64(rhs.Int()))
			}
			return -1
		}
		return FloatCompare(lhs.Float(), rhs.Float())
	} else if rhs.IsFloat() {
		// Extra: compare floats and ints numerically.
		if lhs.IsInt() {
			return FloatCompare(float64(lhs.Int()), rhs.Float())
		}
		return 1
	}

	if lhs.IsInt() {
		if !rhs.IsInt() {
			return -1
		}
		return IntCompare(lhs.Int(), rhs.Int())
	} else if rhs.IsInt() {
		return 1
	}

	if lhs.IsString() {
		if !rhs.IsString() {
			return -1
		}
		return strings.Compare(lhs.String(), rhs.String())
	} else if rhs.IsString() {
		return 1
	}

	if lhs.IsBool() {
		if !rhs.IsBool() {
			return -1
		}
		return BoolCompare(lhs.Bool(), rhs.Bool())
	} else if rhs.IsBool() {
		return 1
	}

	if lhs.IsList() {
		if !rhs.IsList() {
			return -1
		}
		return ListCompare(lhs.List(), rhs.List())
	} else if rhs.IsList() {
		return 1
	}
	if lhs.IsMap() {
		if !rhs.IsMap() {
			return -1
		}
		return MapCompare(lhs.Map(), rhs.Map())
	} else if rhs.IsMap() {
		return 1
	}
	if lhs.IsNull() {
		if !rhs.IsNull() {
			return -1
		}
		return 0
	} else if rhs.IsNull() {
		return 1
	}

	// Invalid Value-- nothing is set.
	return 0
}
