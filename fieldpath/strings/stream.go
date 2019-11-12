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

package strings

import (
	"sigs.k8s.io/structured-merge-diff/value"
)

type streamWithStringTable struct {
	value.Stream

	stringTable map[string]int
}

var _ value.Stream = &streamWithStringTable{}

func NewStreamWithStringTable(s value.Stream) (value.Stream, error) {
	reverseStringTable, err := GetReverseTable(DefaultVersion)
	if err != nil {
		return nil, err
	}
	stream := &streamWithStringTable{
		Stream:      s,
		stringTable: reverseStringTable,
	}
	return stream, nil
}

func (s *streamWithStringTable) WriteString(str string) {
	if x, ok := s.stringTable[str]; ok {
		s.Stream.WriteRaw("!")
		s.Stream.WriteInt(x)
	} else {
		s.Stream.WriteString(str)
	}
}

func (s *streamWithStringTable) WriteObjectField(str string) {
	if x, ok := s.stringTable[str]; ok {
		s.Stream.WriteRaw("!")
		s.Stream.WriteInt(x)
		s.Stream.WriteRaw(":")
	} else {
		s.Stream.WriteObjectField(str)
	}
}
