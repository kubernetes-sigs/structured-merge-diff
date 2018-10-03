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
	"github.com/kubernetes-sigs/structured-merge-diff/fieldpath"
	"github.com/kubernetes-sigs/structured-merge-diff/value"
)

type Version string

type VersionedSet struct {
	*fieldpath.Set
	Version Version
}

type Converter interface {
	Convert(object *Object, version Version) *Object
}

type Object struct {
	value.Value
}

func (o *Object) OwnerSet() map[string]*VersionedSet          { return nil }
func (o *Object) SetOwnerSet(owners map[string]*VersionedSet) {}
func (o *Object) GroupVersion() Version                       { return Version("v1") }

func ConflictsToError(conflicts map[string]*VersionedSet) error { return nil }
func MergeApplied(live, config *Object) *Object                 { return nil }
