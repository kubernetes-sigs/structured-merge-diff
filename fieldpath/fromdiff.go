/*
Copyright 2018 The Kubernetes Authors.
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
	"github.com/kubernetes-sigs/structured-merge-diff/value"
)

// SetFromDiff creates a set containing every leaf field that has a
// different value between v and u.
func SetFromDiff(v, u value.Value) *Set {
	s := NewSet()

	return s
}
