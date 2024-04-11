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
	"bytes"
	gojson "encoding/json"
	"fmt"
	"io"

	json "sigs.k8s.io/json"
)

func (s *Set) ToJSON() ([]byte, error) {
	buf := bytes.Buffer{}
	err := s.ToJSONStream(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Set) ToJSONStream(w io.Writer) error {
	err := s.emitContentsV1(false, w)
	if err != nil {
		return err
	}
	return nil
}

type orderedMapItemWriter struct {
	w        io.Writer
	hasItems bool
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
	pev, err := SerializePathElement(pe)
	if err != nil {
		return err
	}
	key, err := gojson.Marshal(pev)
	if err != nil {
		return err
	}

	return om.writeKey(key)
}

// writeEmptyValue writes an empty JSON object to the writer.
// This should be used after writeKey.
func (om orderedMapItemWriter) writeEmptyValue() error {
	if _, err := om.w.Write([]byte("{}")); err != nil {
		return err
	}
	return nil
}

func (s *Set) emitContentsV1(includeSelf bool, w io.Writer) error {
	om := orderedMapItemWriter{w: w}
	mi, ci := 0, 0

	if _, err := om.w.Write([]byte{'{'}); err != nil {
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
			if err := s.Children.members[ci].set.emitContentsV1(c == 0, om.w); err != nil {
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
		if err := s.Children.members[ci].set.emitContentsV1(false, om.w); err != nil {
			return err
		}

		ci++
	}

	if _, err := om.w.Write([]byte{'}'}); err != nil {
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

			m := &children.Members.members
			// Since we expect that most of the time these will have been
			// serialized in the right order, we just verify that and append.
			appendOK := len(*m) == 0 || (*m)[len(*m)-1].Less(pe)
			if appendOK {
				*m = append(*m, pe)
			} else {
				children.Members.Insert(pe)
			}
		}

		if v.target != nil {
			if children == nil {
				children = &Set{}
			}

			// Since we expect that most of the time these will have been
			// serialized in the right order, we just verify that and append.
			m := &children.Children.members
			appendOK := len(*m) == 0 || (*m)[len(*m)-1].pathElement.Less(pe)
			if appendOK {
				*m = append(*m, setNode{pe, v.target})
			} else {
				*children.Children.Descend(pe) = *v.target
			}
		}
	}

	if children == nil {
		isMember = true
	}

	return children, isMember, nil
}
