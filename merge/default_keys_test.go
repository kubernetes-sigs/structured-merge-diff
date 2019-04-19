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

package merge_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/merge"
	"sigs.k8s.io/structured-merge-diff/typed"
)

var portListParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: type
  struct:
    fields:
      - name: containerPorts
        type:
          list:
            elementType:
              struct:
                fields:
                - name: port
                  type:
                    scalar: numeric
                - name: protocol
                  type:
                    scalar: string
            elementRelationship: associative
            keys:
            - port
            - protocol
`)
	if err != nil {
		panic(err)
	}
	return parser.Type("type")
}()

func TestDefaultKeysBroken(t *testing.T) {
	tests := map[string]TestCase{
		"apply_default_key": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						containerPorts:
						- port: 80
					`,
				},
			},
			Object: `
				containerPorts:
				- port: 80
				  protocol: TCP
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set: _NS(
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP"))),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "port"),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "protocol"),
					),
					APIVersion: "v1",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			state := NewState(portListParser)
			state.Updater.Defaulter = protocolDefaulter{ParseableType: portListParser}
			if err := test.TestWithState(state); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// protocolDefaulter sets the default protocol to TCP
type protocolDefaulter struct {
	typed.ParseableType
}

var _ merge.Defaulter = protocolDefaulter{}

// Default implements merge.Defaulter
func (d protocolDefaulter) Default(v *typed.TypedValue) (*typed.TypedValue, error) {
	// make a deep copy of v by serializing and deserializing
	y, err := v.AsValue().ToYAML()
	if err != nil {
		return nil, err
	}
	v2, err := d.ParseableType.FromYAML(typed.YAMLObject(y))
	if err != nil {
		return nil, err
	}

	// Loop over the elements of containerPorts and default the protocols
	if mapValue := v2.AsValue().MapValue; mapValue != nil {
		if containerPorts, ok := mapValue.Get("containerPorts"); ok {
			if listValue := containerPorts.Value.ListValue; listValue != nil {
				for i := range listValue.Items {
					if item := listValue.Items[i].MapValue; item != nil {
						if _, ok := item.Get("protocol"); !ok {
							item.Set("protocol", _SV("TCP"))
						}
					}
				}
			}
		}
	}

	return v2, nil
}
