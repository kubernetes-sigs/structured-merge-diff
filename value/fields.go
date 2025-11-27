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
	"io"
	"sort"
	"strings"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Field is an individual key-value pair.
type Field struct {
	Name  string
	Value Value
}

// Not meant to be used by an external library.
type FastMarshalValue struct {
	Value *Value
}

var _ json.MarshalerTo = FastMarshalValue{}

func (mv FastMarshalValue) MarshalJSONTo(enc *jsontext.Encoder) error {
	return valueMarshalJSONTo(enc, *mv.Value)
}

func valueMarshalJSONTo(enc *jsontext.Encoder, v Value) error {
	switch {
	case v.IsNull():
		return enc.WriteToken(jsontext.Null)
	case v.IsFloat():
		return enc.WriteToken(jsontext.Float(v.AsFloat()))
	case v.IsInt():
		return enc.WriteToken(jsontext.Int(v.AsInt()))
	case v.IsString():
		return enc.WriteToken(jsontext.String(v.AsString()))
	case v.IsBool():
		return enc.WriteToken(jsontext.Bool(v.AsBool()))
	case v.IsList():
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		list := v.AsList()
		for i := 0; i < list.Length(); i++ {
			if err := valueMarshalJSONTo(enc, list.At(i)); err != nil {
				return err
			}
		}
		return enc.WriteToken(jsontext.EndArray)
	case v.IsMap():
		// use the json marshaller to make sure the key ordering is deterministic
		fallthrough
	default:
		return json.MarshalEncode(enc, v.Unstructured(), json.Deterministic(true))
	}
}

// FieldList is a list of key-value pairs. Each field is expected to
// have a different name.
type FieldList []Field

var _ json.MarshalerTo = (*FieldList)(nil)
var _ json.UnmarshalerFrom = (*FieldList)(nil)

func (fl *FieldList) MarshalJSONTo(enc *jsontext.Encoder) error {
	enc.WriteToken(jsontext.BeginObject)
	for _, f := range *fl {
		if err := enc.WriteToken(jsontext.String(f.Name)); err != nil {
			return err
		}
		if err := valueMarshalJSONTo(enc, f.Value); err != nil {
			return err
		}
	}
	enc.WriteToken(jsontext.EndObject)

	return nil
}

// FieldListFromJSON is a helper function for reading a JSON document.
func (fl *FieldList) UnmarshalJSONFrom(parser *jsontext.Decoder) error {
	if objStart, err := parser.ReadToken(); err != nil {
		return fmt.Errorf("parsing JSON: %v", err)
	} else if objStart.Kind() != jsontext.BeginObject.Kind() {
		return fmt.Errorf("expected object")
	}

	var fields FieldList
	for {
		rawKey, err := parser.ReadToken()
		if err == io.EOF {
			return fmt.Errorf("unexpected EOF")
		} else if err != nil {
			return fmt.Errorf("parsing JSON: %v", err)
		}

		if rawKey.Kind() == jsontext.EndObject.Kind() {
			break
		}

		k := rawKey.String()

		var v any
		if err := json.UnmarshalDecode(parser, &v); err == io.EOF {
			return fmt.Errorf("unexpected EOF")
		} else if err != nil {
			return fmt.Errorf("parsing JSON: %v", err)
		}

		fields = append(fields, Field{Name: k, Value: NewValueInterface(v)})
	}

	fields.Sort()
	*fl = fields

	return nil
}

// Copy returns a copy of the FieldList.
// Values are not copied.
func (f FieldList) Copy() FieldList {
	c := make(FieldList, len(f))
	copy(c, f)
	return c
}

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

// Compare compares two lists lexically. The result will be 0 if f==rhs, -1
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

// Equals returns true if the two fieldslist are equals, false otherwise.
func (f FieldList) Equals(rhs FieldList) bool {
	if len(f) != len(rhs) {
		return false
	}
	for i := range f {
		if f[i].Name != rhs[i].Name {
			return false
		}
		if !Equals(f[i].Value, rhs[i].Value) {
			return false
		}
	}
	return true
}
