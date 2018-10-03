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

package schema

// Schema is a list of types.
type Schema struct {
	Types []TypeDef `yaml:"types,omitempty"`
}

// A TypeSpecifier references a particular type in a schema.
type TypeSpecifier struct {
	Type   TypeRef `yaml:"type,omitempty"`
	Schema Schema  `yaml:"schema,omitempty"`
}

// TypeDef represents a node in a schema.
type TypeDef struct {
	// Top level types should be named. Every type must have a unique name.
	Name string `yaml:"name,omitempty"`

	Atom `yaml:"atom,omitempty,inline"`
}

// TypeRef either refers to a named type or declares an inlined type.
type TypeRef struct {
	// Either the name or one member of Atom should be set.
	NamedType *string `yaml:"namedType,omitempty"`
	Inlined   Atom    `yaml:"inlined,inline,omitempty"`
}

// Atom represents the smallest possible pieces of the type system.
type Atom struct {
	// Exactly one of the below must be set.
	*Scalar  `yaml:"scalar,omitempty"`
	*Struct  `yaml:"struct,omitempty"`
	*List    `yaml:"list,omitempty"`
	*Map     `yaml:"map,omitempty"`
	*Untyped `yaml:"untyped,omitempty"`
}

// Scalar (AKA "primitive") has a single value which is either numeric, string,
// or boolean.
// TODO: split numeric into float/int? Something even more fine-grained?
type Scalar string

const (
	Numeric = Scalar("numeric")
	String  = Scalar("string")
	Boolean = Scalar("boolean")
)

// ElementRelationship is an enum of the different possible relationships
// between the elements of container types.
type ElementRelationship string

const (
	// Associative only applies to lists (see the documentation there).
	Associative = ElementRelationship("associative")
	// Atomic makes container types (lists, maps, structs, untyped) behave
	// as scalars / leaf fields (default for untyped data).
	Atomic = ElementRelationship("atomic")
	// Separable means the items of the container type have no particular
	// relationship (default behavior for maps and structs).
	Separable = ElementRelationship("separable")
)

// Struct is a list of fields. Each field has a name and a type. Some fields
// may be grouped into unions.
type Struct struct {
	// Each struct field appears exactly once in this list. The order in
	// this list defines the canonical field ordering.
	Fields []StructField `yaml:"fields,omitempty"`

	// TODO: Implement unions, either this way or by inlining.
	// Unions are groupings of fields with special rules. They may refer to
	// one or more fields in the above list. A given field from the above
	// list may be referenced in exactly 0 or 1 places in the below list.
	// Unions []Union `yaml:"unions,omitempty"`

	// ElementRelationship states the relationship between the struct's items.
	// * `separable` (or unset) implies that each element is 100% independent.
	// * `atomic` implies that all elements depend on each other, and this
	//   is effectively a scalar / leaf field; it doesn't make sense for
	//   separate actors to set the elements. Example: an RGB color struct;
	//   it would never make sense to "own" only one component of the
	//   color.
	// The default behavior for structs is `separable`; it's permitted to
	// leave this unset to get the default behavior.
	ElementRelationship ElementRelationship `yaml:"elementRelationship,omitempty"`
}

// StructField pairs a field name with a field type.
type StructField struct {
	// Name is the field name.
	Name string `yaml:"name,omitempty"`
	// Type is the field type.
	Type TypeRef `yaml:"type,omitempty"`
}

/*

TODO: incorporate unions. Likely not like this.

type Union struct {
	// Discriminator is optional, if it is set, it identifies the
	// discriminator field for this union. The field identified must be a
	// string typed scalar.
	DiscriminatorName *string `yaml:"discriminatorName,omitempty"`

	// Fields lists the members of the union, and their discriminator
	// value (if Discriminator is non-nil).
	Fields []UnionField `yaml:"fields,omitempty"`

	// Attribute says whether the union is optional, required, or defaulted.
	Attribute FieldAttribute `yaml:"attribute,omitempty"`
}


// UnionField pairs a field name with a field type.
type UnionField struct {
	// Name is the field name.
	FieldName string `yaml:"fieldName,omitempty"`

	// DiscriminatorValue is the value that the Discriminator field must
	// have if this member of the union is set. It must be non-nil iff the
	// Union's DiscriminatorName member is non-nil.
	DiscriminatorValue *string `yaml:"discriminatorValue,omitempty"`
}
*/

// List has zero or more elements of some type.
type List struct {
	ElementType TypeRef `yaml:"elementType,omitempty"`

	// ElementRelationship states the relationship between the list's elements
	// and must have one of these values:
	// * `atomic`: the list is treated as a single entity, like a scalar.
	// * `associative`:
	//   - If the list element is a scalar, the list is treated as a set.
	//   - If the list element is a struct, the list is treated as a map.
	//   - The list element must not be a map or a list itself.
	// There is no default for this value for lists; all schemas must
	// explicitly state the element relationship for all lists.
	ElementRelationship ElementRelationship `yaml:"elementRelationship,omitempty"`

	// Iff ElementRelationship is `associative`, and the element type is
	// struct, then Keys must have non-zero length, and it lists the fields
	// of the element's struct type which are to be used as the keys of the
	// list.
	//
	// TODO: change this to "non-atomic struct" above and make the code reflect this.
	//
	// Each key must refer to a single field name (no nesting, not JSONPath).
	Keys []string `yaml:"keys,omitempty"`
}

// Map is a key-value pair. Its semantics are the same as an associative list, but:
// * It is serialized differently:
//     map:  {"k": {"value": "v"}}
//     list: [{"key": "k", "value": "v"}]
// * Keys must be string typed.
// * Keys can't have multiple components.
type Map struct {
	ElementType TypeRef `yaml:"elementType,omitempty"`

	// ElementRelationship states the relationship between the map's items.
	// * `separable` implies that each element is 100% independent.
	// * `atomic` implies that all elements depend on each other, and this
	//   is effectively a scalar / leaf field; it doesn't make sense for
	//   separate actors to set the elements.
	//   TODO: find a simple example.
	// The default behavior for maps is `separable`; it's permitted to
	// leave this unset to get the default behavior.
	ElementRelationship ElementRelationship `yaml:"elementRelationship,omitempty"`
}

// Untyped is used for fields that allow arbitrary content. (Think: plugin
// objects.)
type Untyped struct {
	// ElementRelationship states the relationship between the items, if
	// container-typed data happens to be present here.
	// * `atomic` implies that all elements depend on each other, and this
	//   is effectively a scalar / leaf field; it doesn't make sense for
	//   separate actors to set the elements.
	// TODO: support "guess" (guesses at associative list keys)
	// TODO: support "lookup" (calls a lookup function to figure out the
	//       schema based on the data)
	// The default behavior for untyped data is `atomic`; it's permitted to
	// leave this unset to get the default behavior.
	ElementRelationship ElementRelationship `yaml:"elementRelationship,omitempty"`
}

// FindNamedType returns the referenced TypeDef, if it exists, or (nil, false)
// if it doesn't.
func (s Schema) FindNamedType(name string) (TypeDef, bool) {
	for _, t := range s.Types {
		if t.Name == name {
			return t, true
		}
	}
	return TypeDef{}, false
}

// Resolve returns the atom referenced, whether it is inline or
// named. Returns Atom{}, false if the type can't be resolved. Allows callers
// to not care about the difference between a (possibly inlined) reference and
// a definition.
func (s Schema) Resolve(tr TypeRef) (Atom, bool) {
	if tr.NamedType != nil {
		t, ok := s.FindNamedType(*tr.NamedType)
		if !ok {
			return Atom{}, false
		}
		return t.Atom, true
	}
	return tr.Inlined, true
}
