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

package typed_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	yaml "gopkg.in/yaml.v2"
	"sigs.k8s.io/structured-merge-diff/typed"
)

func testdata(file string) string {
	return filepath.Join("..", "internal", "testdata", file)
}

func BenchmarkFromUnstructured(b *testing.B) {
	pod, err := ioutil.ReadFile(testdata("pod.yaml"))
	if err != nil {
		b.Fatal(err)
	}

	s, err := ioutil.ReadFile(testdata("k8s-schema.yaml"))
	if err != nil {
		b.Fatal(err)
	}
	parser, err := typed.NewParser(typed.YAMLObject(s))
	if err != nil {
		b.Fatal(err)
	}
	pt := parser.Type("io.k8s.api.core.v1.Pod")

	obj := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(pod), &obj); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := pt.FromUnstructured(obj); err != nil {
			b.Fatal(err)
		}
	}

}
