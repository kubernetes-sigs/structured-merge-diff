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

package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/googleapis/gnostic/compiler"
	jsoniter "github.com/json-iterator/go"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"

	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"
	"k8s.io/klog"
)

var (
	file = flag.String("file", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
)

func main() {
	// Switch over to depending on this instead of client-go:
	// https://github.com/kubernetes/kubernetes/blob/master/api/openapi-spec/swagger.json
	flag.Parse()

	f, err := os.Open(*file)
	if err != nil {
		klog.Fatalf("Failed to open --file=%s: %v", *file, err)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		klog.Fatalf("Failed reading %s: %v", *file, err)
	}
	var json map[string]interface{}
	err = jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(b, &json)
	if err != nil {
		klog.Fatalf("Failed parsing %s: %v", *file, err)
	}
	doc, err := openapi_v2.NewDocument(jsonToYAML(json), compiler.NewContext("$root", nil))
	if err != nil {
		klog.Fatalf("Failed parse %s: %v", *file, err)
	}
	set := map[string]struct{}{}
	for _, prop := range doc.Definitions.AdditionalProperties {
		collectStrings(prop.Value, set)
	}
	list := order(set)
	list = trim(list)
	for _, s := range list {
		fmt.Printf("%s\n", s)
	}
}

// collectStrings traverses all the types in the schema and collects all strings
// into the set.
func collectStrings(schema *openapi_v2.Schema, set map[string]struct{}) {
	if schema.Type == nil {
		return
	}
	// TODO: defaults? They're buried in the defaulting code
	// TODO: anyof, allof, oneof do not appear to be in use except in apiextensions.v1.JSONSchemaProps
	//
	for _, t := range schema.Type.Value {
		switch t {
		case "object":
			if schema.Properties != nil {
				for _, namedSchema := range schema.Properties.AdditionalProperties {
					set[namedSchema.Name] = struct{}{}
					collectStrings(namedSchema.Value, set)
				}
			}
		case "array":
			for _, schema := range schema.Items.Schema {
				collectStrings(schema, set)
			}
		case "string":
			for _, e := range schema.Enum {
				if e.Yaml != "" {
					// The enum strings are yaml!?
					d := yaml.NewDecoder(strings.NewReader(e.Yaml))
					var s string
					err := d.Decode(&s)
					if err != nil {
						klog.Fatalf("Error parsing enum yaml value to string: %v", err)
					}
					set[s] = struct{}{}
				} else {
					klog.Fatalf("Expected enum to contain yaml strings, but got: %v", e)
				}
			}
		case "number":
		case "integer":
		case "boolean":
		default:
			klog.Fatalf("Unsupported type: %s", t)
		}
	}
}

// order sorts the strings primarily by length, and lexically if the strings are the same
// length.
func order(set map[string]struct{}) []string {
	result := make([]string, len(set), len(set))
	i := 0
	for s := range set {
		result[i] = s
		i++
	}
	sort.SliceStable(result, func(i, j int) bool {
		cmp := len(result[i]) - len(result[j])
		if cmp == 0 {
			return result[i] < result[j] // lexically sort equal length strings
		}
		return cmp < 0
	})
	return result
}

// trim removes any strings that are not longer than the `!<base64>` they would be replaced
// by.
func trim(strings []string) []string {
	results := make([]string, 0, len(strings))
	for i, s := range strings {
		encoded := toBase64(uint64(i))

		//fmt.Printf("!%s -> \"%s\"\n", encoded, s)
		if len(s)+2 >= len(encoded)+1 {
			results = append(results, s)
		}
	}
	return results
}

// TODO: just need to calculate length. Computing the string is only helpful for debugging here.
// TODO: Review encoding to base64, using base64.RawStdEncoding here is slightly less dense than taking 6-bits at a time and producing a base64 bytestring
func toBase64(i uint64) string {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, uint64(i))
	return base64.RawStdEncoding.EncodeToString(buf[:n])
}

// TODO: remove copy-paste
// from https://github.com/kubernetes/kube-openapi/blob/0270cf2f1c1d995d34b36019a6f65d58e6e33ad4/pkg/handler/handler.go#L141
func jsonToYAML(j map[string]interface{}) yaml.MapSlice {
	if j == nil {
		return nil
	}
	ret := make(yaml.MapSlice, 0, len(j))
	for k, v := range j {
		ret = append(ret, yaml.MapItem{k, jsonToYAMLValue(v)})
	}
	return ret
}

// TODO: remove copy-paste
// from https://github.com/kubernetes/kube-openapi/blob/0270cf2f1c1d995d34b36019a6f65d58e6e33ad4/pkg/handler/handler.go#L152
func jsonToYAMLValue(j interface{}) interface{} {
	switch j := j.(type) {
	case map[string]interface{}:
		return jsonToYAML(j)
	case []interface{}:
		ret := make([]interface{}, len(j))
		for i := range j {
			ret[i] = jsonToYAMLValue(j[i])
		}
		return ret
	case float64:
		// replicate the logic in https://github.com/go-yaml/yaml/blob/51d6538a90f86fe93ac480b35f37b2be17fef232/resolve.go#L151
		if i64 := int64(j); j == float64(i64) {
			if i := int(i64); i64 == int64(i) {
				return i
			}
			return i64
		}
		if ui64 := uint64(j); j == float64(ui64) {
			return ui64
		}
		return j
	case int64:
		if i := int(j); j == int64(i) {
			return i
		}
		return j
	}
	return j
}