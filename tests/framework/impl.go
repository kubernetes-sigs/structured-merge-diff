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

package framework

import "sigs.k8s.io/structured-merge-diff/typed"

// Implementation defines the interface required by the actual merge code to be used in Kubernetes and this test framework
type Implementation interface {
	// Apply returns the new post merge object and errors (including conflicts if any occur)
	Apply(live, config typed.YAMLObject, workflow string, force bool) (typed.YAMLObject, error)
	// Update returns the new object and errors (including conflicts if any occur)
	Update(live, new typed.YAMLObject, workflow string) (typed.YAMLObject, error)
}
