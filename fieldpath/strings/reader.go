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
	"fmt"
	"io"
)

func NewReaderWithStringTable(r io.Reader) *io.PipeReader {
	out, in := io.Pipe()
	go func() {
		defer in.Close()
		p := newReplacer(in)
		inputBuffer := make([]byte, 100)
		for {
			if _, err := r.Read(inputBuffer); err != nil {
				return
			}
			if p.skipTheRest {
				p.write(inputBuffer...)
			} else {
				for _, b := range inputBuffer {
					if err := p.read(b); err != nil {
						in.CloseWithError(err)
						return
					}
				}
			}
			if err := p.flush(); err != nil {
				return
			}
		}
	}()
	return out
}

type replacer struct {
	writer                 io.Writer
	first                  bool
	skipTheRest            bool
	readStringTableVersion bool
	inEscapeSequence       bool
	inQuotes               bool
	readIndex              bool
	inputBuffer            []byte
	outputBuffer           []byte
	indexBuffer            []byte
	stringTable            []string
}

func newReplacer(writer io.Writer) *replacer {
	p := replacer{}
	p.writer = writer
	p.first = true
	return &p
}

func (p *replacer) write(b ...byte) {
	p.outputBuffer = append(p.outputBuffer, b...)
}

func (p *replacer) flush() error {
	_, err := p.writer.Write(p.outputBuffer)
	p.outputBuffer = p.outputBuffer[:0]
	return err
}

func (p *replacer) read(b byte) (err error) {
	if p.skipTheRest {
		p.write(b)
		return nil
	}

	// Parse the string table version
	// This will be at the beginning of the entire object and start with a '['
	// if the object doesn't start that way, just skip the rest of it.
	if p.first {
		if b == byte('[') {
			p.readStringTableVersion = true
		} else {
			p.skipTheRest = true
		}
		p.first = false
		p.write(b)
		return nil
	}
	if p.readStringTableVersion {
		if b == byte(',') || b == byte(':') || b == byte('}') || b == byte(']') {
			k := parseInt(p.indexBuffer)

			if p.stringTable, err = GetTable(k); err != nil {
				return err
			}

			p.readStringTableVersion = false
			p.indexBuffer = p.indexBuffer[:0]
		} else {
			p.indexBuffer = append(p.indexBuffer, b)
		}
		return nil
	}

	// Identify and parse an index of an item in the string table
	// This will start with a '!'.
	if !p.inQuotes && b == byte('!') {
		p.readIndex = true
		return nil
	}
	if p.readIndex {
		if b == byte(',') || b == byte(':') || b == byte('}') || b == byte(']') {
			k := parseBase64(p.indexBuffer)

			if k < len(p.stringTable) {
				p.write([]byte(fmt.Sprintf("%q", p.stringTable[k]))...)
				p.write(b)
			} else {
				return fmt.Errorf("unable to look up %v in the string table", k)
			}

			p.readIndex = false
			p.indexBuffer = p.indexBuffer[:0]
		} else {
			p.indexBuffer = append(p.indexBuffer, b)
		}
		return nil
	}

	// Update the state of the parser so it knows what part of json it's reading
	p.inQuotes = !p.inQuotes && (b == byte('"')) || p.inQuotes && (p.inEscapeSequence || !(b == byte('"')))
	p.inEscapeSequence = !p.inEscapeSequence && b == byte('\\')

	p.write(b)
	return nil
}

func parseInt(b []byte) int {
	n := 0
	for _, d := range b {
		if d >= byte('0') && d <= byte('9') {
			n = n*10 + int(d) - int('0')
		}
	}
	return n
}

func parseBase64(b []byte) int {
	n := 0
	for _, d := range b {
		if d >= byte('A') && d <= byte('Z') {
			n = n*64 + int(d) - int('A')
		} else if d >= byte('a') && d <= byte('z') {
			n = n*64 + int(d) - int('a') + 26
		} else if d >= byte('0') && d <= byte('9') {
			n = n*64 + int(d) - int('0') + 52
		} else if d == byte('+') {
			n = n*64 + 62
		} else if d == byte('/') {
			n = n*64 + 63
		}
	}
	return n
}
