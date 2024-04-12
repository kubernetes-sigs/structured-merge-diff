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
	"sort"

	json "sigs.k8s.io/json"
	"sigs.k8s.io/structured-merge-diff/v4/internal/builder"
)

func (s *Set) ToJSON() ([]byte, error) {
	buf := builder.JSONBuilder{}
	if err := s.emitContentsV1(false, &buf, &builder.ReusableBuilder{}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Set) ToJSONStream(w io.Writer) error {
	buf := builder.JSONBuilder{}
	if err := s.emitContentsV1(false, &buf, &builder.ReusableBuilder{}); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

type orderedMapItemWriter struct {
	w        *builder.JSONBuilder
	hasItems bool

	builder *builder.ReusableBuilder
}

// writeKey writes a key to the writer, including a leading comma if necessary.
// The key is expected to be an already-serialized JSON string (including quotes).
// e.g. writeKey([]byte("\"foo\""))
// After writing the key, the caller should write the encoded value, e.g. using
// writeEmptyValue or by directly writing the value to the writer.
func (om *orderedMapItemWriter) writeKey(key []byte) error {
	if om.hasItems {
		if _, err := om.w.Write([]byte{','}); err != nil {
			return err
		}
	}

	if _, err := om.w.Write(key); err != nil {
		return err
	}
	if _, err := om.w.Write([]byte{':'}); err != nil {
		return err
	}
	om.hasItems = true
	return nil
}

// writePathKey writes a path element as a key to the writer, including a leading comma if necessary.
// The path will be serialized as a JSON string (including quotes) and passed to writeKey.
// After writing the key, the caller should write the encoded value, e.g. using
// writeEmptyValue or by directly writing the value to the writer.
func (om *orderedMapItemWriter) writePathKey(pe PathElement) error {
	if om.hasItems {
		if _, err := om.w.Write([]byte{','}); err != nil {
			return err
		}
	}

	if err := serializePathElementToWriter(om.builder.Reset(), pe); err != nil {
		return err
	}
	if err := om.w.WriteJSON(om.builder.UnsafeString()); err != nil {
		return err
	}

	if _, err := om.w.Write([]byte{':'}); err != nil {
		return err
	}
	om.hasItems = true
	return nil
}

// writeEmptyValue writes an empty JSON object to the writer.
// This should be used after writeKey.
func (om orderedMapItemWriter) writeEmptyValue() error {
	if _, err := om.w.Write([]byte("{}")); err != nil {
		return err
	}
	return nil
}

func (s *Set) emitContentsV1(includeSelf bool, w *builder.JSONBuilder, r *builder.ReusableBuilder) error {
	om := orderedMapItemWriter{w: w, builder: r}
	mi, ci := 0, 0

	if _, err := w.Write([]byte{'{'}); err != nil {
		return err
	}

	if includeSelf && !(len(s.Members.members) == 0 && len(s.Children.members) == 0) {
		if err := om.writeKey([]byte("\".\"")); err != nil {
			return err
		}
		if err := om.writeEmptyValue(); err != nil {
			return err
		}
	}

	for mi < len(s.Members.members) && ci < len(s.Children.members) {
		mpe := s.Members.members[mi]
		cpe := s.Children.members[ci].pathElement

		if c := mpe.Compare(cpe); c < 0 {
			if err := om.writePathKey(mpe); err != nil {
				return err
			}
			if err := om.writeEmptyValue(); err != nil {
				return err
			}

			mi++
		} else {
			if err := om.writePathKey(cpe); err != nil {
				return err
			}
			if err := s.Children.members[ci].set.emitContentsV1(c == 0, w, r); err != nil {
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

		if err := om.writePathKey(mpe); err != nil {
			return err
		}
		if err := om.writeEmptyValue(); err != nil {
			return err
		}

		mi++
	}

	for ci < len(s.Children.members) {
		cpe := s.Children.members[ci].pathElement

		if err := om.writePathKey(cpe); err != nil {
			return err
		}
		if err := s.Children.members[ci].set.emitContentsV1(false, w, r); err != nil {
			return err
		}

		ci++
	}

	if _, err := w.Write([]byte{'}'}); err != nil {
		return err
	}

	return nil
}

// FromJSON clears s and reads a JSON formatted set structure.
func (s *Set) FromJSON(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	found, _, err := readIterV1(b)
	if err != nil {
		return err
	} else if found == nil {
		*s = Set{}
	} else {
		*s = *found
	}
	return nil
}

type setReader struct {
	target   *Set
	isMember bool
}

func (sr *setReader) UnmarshalJSON(data []byte) error {
	children, isMember, err := readIterV1(data)
	if err != nil {
		return err
	}
	sr.target = children
	sr.isMember = isMember
	return nil
}

// returns true if this subtree is also (or only) a member of parent; s is nil
// if there are no further children.
func readIterV1(data []byte) (children *Set, isMember bool, err error) {
	m := map[string]setReader{}

	if err := json.UnmarshalCaseSensitivePreserveInts(data, &m); err != nil {
		return nil, false, err
	}

	for k, v := range m {
		if k == "." {
			isMember = true
			continue
		}

		pe, err := DeserializePathElement(k)
		if err == ErrUnknownPathElementType {
			// Ignore these-- a future version maybe knows what
			// they are. We drop these completely rather than try
			// to preserve things we don't understand.
			continue
		} else if err != nil {
			return nil, false, fmt.Errorf("parsing key as path element: %v", err)
		}

		if v.isMember {
			if children == nil {
				children = &Set{}
			}

			// Append the member to the members list, we will sort it later
			m := &children.Members.members
			*m = append(*m, pe)
		}

		if v.target != nil {
			if children == nil {
				children = &Set{}
			}

			// Append the child to the children list, we will sort it later
			m := &children.Children.members
			*m = append(*m, setNode{pe, v.target})
		}
	}

	// Sort the members and children
	if children != nil {
		sort.Slice(children.Members.members, func(i, j int) bool {
			return children.Members.members[i].Less(children.Members.members[j])
		})

		sort.Slice(children.Children.members, func(i, j int) bool {
			return children.Children.members[i].pathElement.Less(children.Children.members[j].pathElement)
		})
	}

	if children == nil {
		isMember = true
	}

	return children, isMember, nil
}
