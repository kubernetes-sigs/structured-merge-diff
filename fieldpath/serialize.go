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

package fieldpath

import (
	"fmt"
	"io"
	"slices"
	"sync"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func (s *Set) ToJSON() ([]byte, error) {
	return json.Marshal((*setContentsV1)(s))
}

func (s *Set) ToJSONStream(w io.Writer) error {
	return json.MarshalWrite(w, (*setContentsV1)(s))
}

var pool = sync.Pool{
	New: func() any {
		return &pathElementSerializer{}
	},
}

func writePathKey(enc *jsontext.Encoder, pe PathElement) error {
	serializer := pool.Get().(*pathElementSerializer)
	defer func() {
		serializer.reset()
		pool.Put(serializer)
	}()

	if err := serializer.serialize(pe); err != nil {
		return err
	}

	if err := enc.WriteToken(jsontext.String(serializer.builder.String())); err != nil {
		return err
	}
	return nil
}

type setContentsV1 Set

var _ json.MarshalerTo = (*setContentsV1)(nil)
var _ json.UnmarshalerFrom = (*setContentsV1)(nil)

func (s *setContentsV1) MarshalJSONTo(enc *jsontext.Encoder) error {
	return s.emitContentsV1(false, enc)
}

func (s *setContentsV1) emitContentsV1(includeSelf bool, om *jsontext.Encoder) error {
	if err := om.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}

	if includeSelf && !(len(s.Members.members) == 0 && len(s.Children.members) == 0) {
		if err := om.WriteToken(jsontext.String(".")); err != nil {
			return err
		}
		if err := om.WriteValue(jsontext.Value("{}")); err != nil {
			return err
		}
	}

	mi, ci := 0, 0
	for mi < len(s.Members.members) && ci < len(s.Children.members) {
		mpe := s.Members.members[mi]
		cpe := s.Children.members[ci].pathElement

		if c := mpe.Compare(cpe); c < 0 {
			if err := writePathKey(om, mpe); err != nil {
				return err
			}
			if err := om.WriteValue(jsontext.Value("{}")); err != nil {
				return err
			}

			mi++
		} else {
			if err := writePathKey(om, cpe); err != nil {
				return err
			}
			if err := (*setContentsV1)(s.Children.members[ci].set).emitContentsV1(c == 0, om); err != nil {
				return err
			}

			// If we also found a member with the same path, we skip this member.
			if c == 0 {
				mi++
			}
			ci++
		}
	}

	for mi < len(s.Members.members) {
		mpe := s.Members.members[mi]

		if err := writePathKey(om, mpe); err != nil {
			return err
		}
		if err := om.WriteValue(jsontext.Value("{}")); err != nil {
			return err
		}

		mi++
	}

	for ci < len(s.Children.members) {
		cpe := s.Children.members[ci].pathElement

		if err := writePathKey(om, cpe); err != nil {
			return err
		}
		if err := (*setContentsV1)(s.Children.members[ci].set).emitContentsV1(false, om); err != nil {
			return err
		}

		ci++
	}

	if err := om.WriteToken(jsontext.EndObject); err != nil {
		return err
	}

	return nil
}

func (s *setContentsV1) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	found, _, err := s.readIterV1(dec)
	if err != nil {
		return err
	} else if found == nil {
		*(*Set)(s) = Set{}
	} else {
		*(*Set)(s) = *found
	}
	return nil
}

// returns true if this subtree is also (or only) a member of parent; s is nil
// if there are no further children.
func (s *setContentsV1) readIterV1(parser *jsontext.Decoder) (children *Set, isMember bool, err error) {
	if objStart, err := parser.ReadToken(); err != nil {
		return nil, false, fmt.Errorf("parsing JSON: %v", err)
	} else if objStart.Kind() != jsontext.BeginObject.Kind() {
		return nil, false, fmt.Errorf("expected object")
	}

	for {
		rawKey, err := parser.ReadToken()
		if err == io.EOF {
			return nil, false, fmt.Errorf("unexpected EOF")
		} else if err != nil {
			return nil, false, fmt.Errorf("parsing JSON: %v", err)
		}

		if rawKey.Kind() == jsontext.EndObject.Kind() {
			break
		}

		k := rawKey.String()
		if k == "." {
			isMember = true
			if err := parser.SkipValue(); err != nil {
				return nil, false, fmt.Errorf("parsing JSON: %v", err)
			}
			continue
		}
		pe, err := DeserializePathElement(k)
		if err == ErrUnknownPathElementType {
			// Ignore these-- a future version maybe knows what
			// they are. We drop these completely rather than try
			// to preserve things we don't understand.
			if err := parser.SkipValue(); err != nil {
				return nil, false, fmt.Errorf("parsing JSON: %v", err)
			}
			continue
		} else if err != nil {
			return nil, false, fmt.Errorf("parsing key as path element: %v", err)
		}

		grandChildren, isChildMember, err := s.readIterV1(parser)
		if err != nil {
			return nil, false, fmt.Errorf("parsing value as set: %v", err)
		}

		if isChildMember {
			if children == nil {
				children = &Set{}
			}

			// Append the member to the members list, we will sort it later
			m := &children.Members.members
			*m = append(*m, pe)
		}

		if grandChildren != nil {
			if children == nil {
				children = &Set{}
			}

			// Append the child to the children list, we will sort it later
			m := &children.Children.members
			*m = append(*m, setNode{pe, grandChildren})
		}
	}

	// Sort the members and children
	if children != nil {
		slices.SortFunc(children.Members.members, func(a, b PathElement) int {
			return a.Compare(b)
		})
		slices.SortFunc(children.Children.members, func(a, b setNode) int {
			return a.pathElement.Compare(b.pathElement)
		})
	}

	if children == nil {
		isMember = true
	}

	return children, isMember, nil
}

// FromJSON clears s and reads a JSON formatted set structure.
func (s *Set) FromJSON(r io.Reader) error {
	return json.UnmarshalRead(r, (*setContentsV1)(s))
}
