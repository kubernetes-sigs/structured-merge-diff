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

package merge

import (
	"sigs.k8s.io/structured-merge-diff/fieldpath"
	"sigs.k8s.io/structured-merge-diff/typed"
)

// APIVersion describes the version of an object or of a fieldset.
type APIVersion string

// VersionedSet associates a version to a set.
type VersionedSet struct {
	*fieldpath.Set
	Version APIVersion
}

// Owners is a map from workflow-id to VersionedSet (what they own in
// what version).
type Owners map[string]*VersionedSet

// Object is the root of an object stored as a TypedValue.
type Object struct {
	typed.TypedValue
}

// GroupVersion returns the version of the object.
//
// TODO: We need to be able to inspect the TypedValue in order to get
// the Version. Right now it's always using the same version.
func (o *Object) GroupVersion() APIVersion { return APIVersion("v1") }
