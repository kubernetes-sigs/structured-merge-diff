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
	"fmt"
	"strings"
	"testing"
)

func TestSerializeV1(t *testing.T) {
	for i := 0; i < 500; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			x := NewSet()
			for j := 0; j < 50; j++ {
				x.Insert(randomPathMaker.makePath(2, 5))
			}
			b, err := x.ToJSON()
			if err != nil {
				t.Fatalf("Failed to serialize %#v: %v", x, err)
			}
			x2 := NewSet()
			err = x2.FromJSON(bytes.NewReader(b))
			if err != nil {
				t.Fatalf("Failed to deserialize %s: %v\n%#v", b, err, x)
			}
			if !x2.Equals(x) {
				b2, _ := x2.ToJSON()
				t.Fatalf("failed to reproduce original:\n\n%s\n\n%s\n\n%s\n\n%s\n", x, b, b2, x2)
			}
		})
	}
}

func TestSerializeV1GoldenData(t *testing.T) {
	examples := []string{
		`{"f:aaa":{},"f:aab":{},"f:aac":{},"f:aad":{},"f:aae":{},"f:aaf":{},"k:{\"name\":\"first\"}":{},"k:{\"name\":\"second\"}":{},"k:{\"port\":443,\"protocol\":\"tcp\"}":{},"k:{\"port\":443,\"protocol\":\"udp\"}":{},"v:1":{},"v:2":{},"v:3":{},"v:\"aa\"":{},"v:\"ab\"":{},"v:true":{}}`,
		`{"f:aaa":{"k:{\"name\":\"second\"}":{"v:3":{"f:aab":{}}},"v:3":{},"v:true":{}},"f:aab":{"f:aaa":{},"f:aaf":{"k:{\"port\":443,\"protocol\":\"udp\"}":{"k:{\"port\":443,\"protocol\":\"tcp\"}":{}}},"k:{\"name\":\"first\"}":{}},"f:aac":{"f:aaa":{"v:1":{}},"f:aac":{},"v:3":{"k:{\"name\":\"second\"}":{}}},"f:aad":{"f:aac":{"v:1":{}},"f:aaf":{"k:{\"name\":\"first\"}":{"k:{\"name\":\"first\"}":{}}}},"f:aae":{"f:aae":{},"k:{\"port\":443,\"protocol\":\"tcp\"}":{"k:{\"port\":443,\"protocol\":\"udp\"}":{}}},"f:aaf":{},"k:{\"name\":\"first\"}":{"f:aad":{"f:aaf":{}}},"k:{\"port\":443,\"protocol\":\"tcp\"}":{"f:aaa":{"f:aad":{}}},"k:{\"port\":443,\"protocol\":\"udp\"}":{"f:aac":{},"k:{\"name\":\"first\"}":{},"k:{\"port\":443,\"protocol\":\"udp\"}":{}},"v:1":{"f:aac":{},"f:aaf":{},"k:{\"port\":443,\"protocol\":\"tcp\"}":{}},"v:2":{"f:aad":{"f:aaf":{}}},"v:3":{"f:aaa":{},"k:{\"name\":\"first\"}":{}},"v:\"aa\"":{"f:aab":{"f:aaf":{}},"f:aae":{},"k:{\"name\":\"first\"}":{"f:aad":{}}},"v:\"ab\"":{"f:aaf":{},"k:{\"port\":443,\"protocol\":\"tcp\"}":{},"k:{\"port\":443,\"protocol\":\"udp\"}":{},"v:1":{"k:{\"port\":443,\"protocol\":\"udp\"}":{}}},"v:true":{"k:{\"name\":\"second\"}":{"f:aaa":{}}}}`,
	}
	for i, str := range examples {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			x := NewSet()
			err := x.FromJSON(strings.NewReader(str))
			if err != nil {
				t.Fatalf("Failed to deserialize %s: %v\n%#v", str, err, x)
			}
			b, err := x.ToJSON()
			if err != nil {
				t.Fatalf("Failed to serialize %#v: %v", x, err)
			}
			if string(b) != str {
				t.Fatalf("Failed;\ngot:  %s\nwant: %s\n", b, str)
			}
		})
	}
}

func TestDropUnknown(t *testing.T) {
	input := `{"f:aaa":{},"r:aab":{}}`
	expect := `{"f:aaa":{}}`
	x := NewSet()
	err := x.FromJSON(strings.NewReader(input))
	if err != nil {
		t.Errorf("Failed to deserialize %s: %v\n%#v", input, err, x)
	}
	b, err := x.ToJSON()
	if err != nil {
		t.Errorf("Failed to serialize %#v: %v", x, err)
		return
	}
	if string(b) != expect {
		t.Errorf("Failed;\ngot:  %s\nwant: %s\n", b, expect)
	}
}
