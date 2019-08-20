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

func TestSerialize(t *testing.T) {
	encodeFuncs := map[string]func(*Set) ([]byte, error){
		"v1":             (*Set).ToJSON,
		"v2experimental": (*Set).ToJSON_V2Experimental,
	}
	for name, sf := range encodeFuncs {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < 500; i++ {
				t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
					x := NewSet()
					for j := 0; j < 50; j++ {
						x.Insert(randomPathMaker.makePath(2, 5))
					}
					b, err := sf(x)
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
		})
	}
}

func TestSerializeV1GoldenData(t *testing.T) {
	examples := []string{
		`{"f:aaa":{},"f:aab":{},"f:aac":{},"f:aad":{},"f:aae":{},"f:aaf":{},"k:{\"name\":\"first\"}":{},"k:{\"name\":\"second\"}":{},"k:{\"port\":443,\"protocol\":\"tcp\"}":{},"k:{\"port\":443,\"protocol\":\"udp\"}":{},"v:1":{},"v:2":{},"v:3":{},"v:\"aa\"":{},"v:\"ab\"":{},"v:true":{},"i:1":{},"i:2":{},"i:3":{},"i:4":{}}`,
		`{"f:aaa":{"k:{\"name\":\"second\"}":{"v:3":{"f:aab":{}}},"v:3":{},"v:true":{}},"f:aab":{"f:aaa":{},"f:aaf":{"k:{\"port\":443,\"protocol\":\"udp\"}":{"k:{\"port\":443,\"protocol\":\"tcp\"}":{}}},"k:{\"name\":\"first\"}":{}},"f:aac":{"f:aaa":{"v:1":{}},"f:aac":{},"v:3":{"k:{\"name\":\"second\"}":{}}},"f:aad":{"f:aac":{"v:1":{}},"f:aaf":{"k:{\"name\":\"first\"}":{"k:{\"name\":\"first\"}":{}}},"i:1":{"i:1":{},"i:3":{"v:true":{}}}},"f:aae":{"f:aae":{},"k:{\"port\":443,\"protocol\":\"tcp\"}":{"k:{\"port\":443,\"protocol\":\"udp\"}":{}},"i:4":{"f:aaf":{}}},"f:aaf":{"i:1":{"f:aac":{}},"i:2":{},"i:3":{}},"k:{\"name\":\"first\"}":{"f:aad":{"f:aaf":{}}},"k:{\"port\":443,\"protocol\":\"tcp\"}":{"f:aaa":{"f:aad":{}}},"k:{\"port\":443,\"protocol\":\"udp\"}":{"f:aac":{},"k:{\"name\":\"first\"}":{"i:3":{}},"k:{\"port\":443,\"protocol\":\"udp\"}":{"i:4":{}}},"v:1":{"f:aac":{"i:4":{}},"f:aaf":{},"k:{\"port\":443,\"protocol\":\"tcp\"}":{}},"v:2":{"f:aad":{"f:aaf":{}},"i:1":{}},"v:3":{"f:aaa":{},"k:{\"name\":\"first\"}":{},"i:2":{}},"v:\"aa\"":{"f:aab":{"f:aaf":{}},"f:aae":{},"k:{\"name\":\"first\"}":{"f:aad":{}},"i:2":{}},"v:\"ab\"":{"f:aaf":{"i:4":{}},"k:{\"port\":443,\"protocol\":\"tcp\"}":{},"k:{\"port\":443,\"protocol\":\"udp\"}":{},"v:1":{"k:{\"port\":443,\"protocol\":\"udp\"}":{}},"i:1":{"f:aae":{"i:4":{}}}},"v:true":{"k:{\"name\":\"second\"}":{"f:aaa":{}},"i:2":{"k:{\"port\":443,\"protocol\":\"tcp\"}":{}}},"i:1":{"i:3":{"f:aaf":{}}},"i:2":{"f:aae":{},"k:{\"port\":443,\"protocol\":\"tcp\"}":{"v:1":{}}},"i:3":{"f:aab":{"v:true":{"v:\"aa\"":{}}},"f:aaf":{},"i:1":{}},"i:4":{"v:\"aa\"":{"f:aab":{"k:{\"name\":\"second\"}":{}}}}}`,
	}
	for i, str := range examples {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			x := NewSet()
			err := x.FromJSON(strings.NewReader(str))
			if err != nil {
				t.Fatalf("Failed to deserialize %s: %v\n%#v", str, err, x)
			}
			/*b, err := x.ToJSON_V2Experimental()
			fmt.Printf("\n\n%s\n\n", b)
			t.Fail()*/
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

func TestSerializeV2GoldenData(t *testing.T) {
	examples := []string{
		`[0,"aaa",0,"aab",0,"aac",0,"aad",0,"aae",0,"aaf",3,{"name":"first"},3,{"name":"second"},3,{"port":443,"protocol":"tcp"},3,{"port":443,"protocol":"udp"},1,1,1,2,1,3,1,"aa",1,"ab",1,true,2,1,2,2,2,3,2,4]`,
		`[4,"aaa",[7,{"name":"second"},[5,3,[0,"aab"]],1,3,1,true],4,"aab",[0,"aaa",4,"aaf",[7,{"port":443,"protocol":"udp"},[3,{"port":443,"protocol":"tcp"}]],3,{"name":"first"}],4,"aac",[4,"aaa",[1,1],0,"aac",5,3,[3,{"name":"second"}]],4,"aad",[4,"aac",[1,1],4,"aaf",[7,{"name":"first"},[3,{"name":"first"}]],6,1,[2,1,6,3,[1,true]]],4,"aae",[0,"aae",7,{"port":443,"protocol":"tcp"},[3,{"port":443,"protocol":"udp"}],6,4,[0,"aaf"]],4,"aaf",[6,1,[0,"aac"],2,2,2,3],7,{"name":"first"},[4,"aad",[0,"aaf"]],7,{"port":443,"protocol":"tcp"},[4,"aaa",[0,"aad"]],7,{"port":443,"protocol":"udp"},[0,"aac",7,{"name":"first"},[2,3],7,{"port":443,"protocol":"udp"},[2,4]],5,1,[4,"aac",[2,4],0,"aaf",3,{"port":443,"protocol":"tcp"}],5,2,[4,"aad",[0,"aaf"],2,1],5,3,[0,"aaa",3,{"name":"first"},2,2],5,"aa",[4,"aab",[0,"aaf"],0,"aae",7,{"name":"first"},[0,"aad"],2,2],5,"ab",[4,"aaf",[2,4],3,{"port":443,"protocol":"tcp"},3,{"port":443,"protocol":"udp"},5,1,[3,{"port":443,"protocol":"udp"}],6,1,[4,"aae",[2,4]]],5,true,[7,{"name":"second"},[0,"aaa"],6,2,[3,{"port":443,"protocol":"tcp"}]],6,1,[6,3,[0,"aaf"]],6,2,[0,"aae",7,{"port":443,"protocol":"tcp"},[1,1]],6,3,[4,"aab",[5,true,[1,"aa"]],0,"aaf",2,1],6,4,[5,"aa",[4,"aab",[3,{"name":"second"}]]]]`,
	}
	for i, str := range examples {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			x := NewSet()
			err := x.FromJSON(strings.NewReader(str))
			if err != nil {
				t.Errorf("Failed to deserialize %s: %v\n%#v", str, err, x)
			}
			b, err := x.ToJSON_V2Experimental()
			if err != nil {
				t.Errorf("Failed to serialize %#v: %v", x, err)
				return
			}
			if string(b) != str {
				t.Errorf("Failed;\ngot:  %s\nwant: %s\n", b, str)
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
