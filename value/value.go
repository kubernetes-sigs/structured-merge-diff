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

	jsoniter "github.com/json-iterator/go"
)

var (
	readPool  = jsoniter.NewIterator(jsoniter.ConfigCompatibleWithStandardLibrary).Pool()
	writePool = jsoniter.NewStream(jsoniter.ConfigCompatibleWithStandardLibrary, nil, 1024).Pool()
)

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
		m := make(map[string]interface{}, ValueMap(v).Length())
		ValueMap(v).Iterate(func(key string, value Value) bool {
			m[key] = Copy(value)
			return true
		})
		return m
	}
	// Scalars don't have to be copied
	return v
}

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
		ValueMap(v).Iterate(func(k string, v Value) bool {
			strs = append(strs, fmt.Sprintf("%v=%v", k, ToString(v)))
			return true
		})
		return "{" + strings.Join(strs, ";") + "}"
	}
	return fmt.Sprintf("{{undefined(%#v)}}", v)
}
