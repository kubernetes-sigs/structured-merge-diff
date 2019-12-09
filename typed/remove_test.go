package typed

import (
	"fmt"
	"testing"

	"sigs.k8s.io/structured-merge-diff/v3/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v3/value"
)

var (
	// Short names for readable test cases.
	_NS  = fieldpath.NewSet
	_P   = fieldpath.MakePathOrDie
	_KBF = fieldpath.KeyByFields
	_V   = value.NewValueInterface
)

func TestRemoveDeduced(t *testing.T) {
	var cases = []struct {
		object YAMLObject
		remove *fieldpath.Set
		expect YAMLObject
	}{
		{
			object: `{}`,
			remove: _NS(_P("a")),
			expect: `{}`,
		},
		{
			object: `{"a": "value"}`,
			remove: _NS(_P("a")),
			expect: `{}`,
		},
		{
			object: `{"a": "value", "b": "value"}`,
			remove: _NS(_P("a")),
			expect: `{"b": "value"}`,
		},
		{
			object: `{"a": "value", "b": {}}`,
			remove: _NS(_P("a")),
			expect: `{"b": {}}`,
		},
		{
			object: `{"a": "value", "b": {"c":"value"}}`,
			remove: _NS(_P("b")),
			expect: `{"a": "value"}`,
		},
		{
			object: `{"a": "value", "b": {"c":"value"}}`,
			remove: _NS(_P("b", "c")),
			expect: `{"a": "value", "b": {}}`,
		},
		{
			object: `{"a": "value", "b": []}`,
			remove: _NS(_P("b")),
			expect: `{"a": "value"}`,
		},
		{
			object: `{"a": "value", "b": ["c"]}`,
			remove: _NS(_P("b")),
			expect: `{"a": "value"}`,
		},
		{
			object: `{"a": "value", "b": ["c"]}`,
			remove: _NS(_P("b", "c")),
			// `c` won't get removed, as it is not a field, while being addressed as one in MakePathOrDie.
			// There is a test for removing `c` below.
			expect: `{"a": "value", "b": ["c"]}`,
		},
		{
			object: `{"a": "value", "b": ["c", "d"]}`,
			remove: _NS(_P("b", "c")),
			// `c` won't get removed, as it is not a field, while being addressed as one in MakePathOrDie.
			// There is a test for removing `c` below.
			expect: `{"a": "value", "b": ["c", "d"]}`,
		},
		{
			object: `{"a": "value", "b": ["c"]}`,
			remove: _NS(_P("b", 0)),
			expect: `{"a": "value", "b": []}`,
		},
		{
			object: `{"a": "value", "b": ["c", "d"]}`,
			remove: _NS(_P("b", 1)),
			expect: `{"a": "value", "b": ["c"]}`,
		},
		{
			object: `{"a": "value", "b": [{"c": "value"}, {"d": "value"}]}`,
			remove: _NS(_P("b", "c")),
			// `c` won't get removed, as it is not a field, but a field inside a list item, MakePath does not address it accordingly.
			// It would be unexpected to remove all `c`s from a lists items.
			// To remove only the list item or `c` the path must be specified differently (see below).
			expect: `{"a": "value", "b": [{"c": "value"}, {"d": "value"}]}`,
		},
		{
			object: `{"a": "value", "b": [{"c": "value"}, {"d": "value"}]}`,
			remove: _NS(_P("b", 0)),
			expect: `{"a": "value", "b": [{"d": "value"}]}`,
		},
		{
			object: `{"a": "value", "b": [{"c": "value"}, {"d": "value"}]}`,
			remove: _NS(_P("b", 0, "c")),
			expect: `{"a": "value", "b": [{}, {"d": "value"}]}`,
		},
		{
			object: `{"a": "value", "b": {"c":"value", "d":{"e":"value"}}}`,
			remove: _NS(_P("b", "d")),
			expect: `{"a": "value", "b": {"c":"value"}}`,
		},
		{
			object: `{"a": "value", "b": {"c":"value", "d": {"e":"value"}}}`,
			remove: _NS(_P("b", "d", "e")),
			expect: `{"a": "value", "b": {"c":"value", "d": {}}}`,
		},
	}

	for i, c := range cases {
		c := c
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			t.Parallel()

			obj, err := DeducedParseableType.FromYAML(c.object)
			if err != nil {
				t.Fatalf("unable to parse/validate object: %v\n%v", err, c.object)
			}

			parseable := &ParseableType{
				Schema:  obj.schema,
				TypeRef: obj.typeRef,
			}
			expectTyped, err := parseable.FromYAML(c.expect)
			if err != nil {
				t.Fatalf("unable to parse/validate expected object: %v\n%v", err, c.expect)
			}
			expect := expectTyped.AsValue()

			result := obj.Remove(c.remove).AsValue()
			if !value.Equals(result, expect) {
				t.Fatalf("unexpected result after Remove:\ngot: %v\nexp: %v",
					value.ToString(result), value.ToString(expect),
				)
			}

			result = obj.RemoveItems(c.remove).AsValue()
			if !value.Equals(result, expect) {
				t.Fatalf("unexpected result after RemoveItems:\ngot: %v\nexp: %v",
					value.ToString(result), value.ToString(expect),
				)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	var cases = []struct {
		object          YAMLObject
		schema          YAMLObject
		remove          *fieldpath.Set
		expect          YAMLObject
		expectItemsOnly YAMLObject
	}{
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        map:
          elementType:
            scalar: string
`,
			object:          `{"a": "value", "b":{"c":"value", "d": "value"}}`,
			remove:          _NS(_P("b", "d")),
			expect:          `{"a": "value", "b":{"c":"value"}}`,
			expectItemsOnly: `{"a": "value", "b":{"c":"value"}}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        map:
          elementType:
            scalar: string
`,
			object:          `{"a": "value", "b":{"c":"value", "d": "value"}}`,
			remove:          _NS(_P("b")),
			expect:          `{"a": "value"}`,
			expectItemsOnly: `{"a": "value", "b":{"c":"value", "d": "value"}}}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        map:
          fields:
          - name: c
            type:
              scalar: string
`,
			object:          `{"a": "value", "b":{"c":"value"}}`,
			remove:          _NS(_P("b")),
			expect:          `{"a": "value"}`,
			expectItemsOnly: `{"a": "value", "b":{"c":"value"}}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        map:
          fields:
          - name: c
            type:
              scalar: string
`,
			object:          `{"a": "value", "b":{"c":"value"}}`,
			remove:          _NS(_P("b")),
			expect:          `{"a": "value"}`,
			expectItemsOnly: `{"a": "value", "b":{"c":"value"}}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        map:
          fields:
          - name: c
            type:
              scalar: string
          elementRelationship: separable
`,
			object:          `{"a": "value", "b":{"c":"value"}}`,
			remove:          _NS(_P("b", "c")),
			expect:          `{"a": "value", "b": {}}`,
			expectItemsOnly: `{"a": "value", "b": {"c":"value"}}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        map:
          fields:
          - name: c
            type:
              scalar: string
          elementRelationship: separable
`,
			object:          `{"a": "value", "b":{"c":"value"}}`,
			remove:          _NS(_P("b")),
			expect:          `{"a": "value"}`,
			expectItemsOnly: `{"a": "value", "b":{"c":"value"}}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        map:
          fields:
          - name: c
            type:
              scalar: string
          elementRelationship: atomic
`,
			object:          `{"a": "value", "b": {"c":"value"}}`,
			remove:          _NS(_P("b", "c")),
			expect:          `{"a": "value", "b": {}}`,
			expectItemsOnly: `{"a": "value", "b": {"c":"value"}}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        map:
          fields:
          - name: c
            type:
              scalar: string
          elementRelationship: atomic
`,
			object:          `{"a": "value", "b":{"c":"value"}}`,
			remove:          _NS(_P("b")),
			expect:          `{"a": "value"}`,
			expectItemsOnly: `{"a": "value", "b":{"c":"value"}}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative
`,
			object: `{"a": "value", "b": ["c"]}`,
			remove: _NS(_P("b", "c")),
			// `c` won't get removed, as it is not a field, while being addressed as one in MakePathOrDie.
			// There is a test for removing `c` below.
			expect:          `{"a": "value", "b": ["c"]}`,
			expectItemsOnly: `{"a": "value", "b": ["c"]}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative
`,
			object:          `{"a": "value", "b": ["c"]}`,
			remove:          _NS(_P("b")),
			expect:          `{"a": "value"}`,
			expectItemsOnly: `{"a": "value", "b": ["c"]}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        list:
          keys:
          - name
          elementType:
            map:
              fields:
              - name: name
                type:
                  scalar: string
              - name: c
                type:
                  scalar: number
              - name: d
                type:
                  scalar: string
          elementRelationship: associative
`,
			object:          `{"a": "value", "b": [{"name": "item1", "c": 1, "d": "value"}, {"name": "item2", "c": 1, "d": "value"}]}`,
			remove:          _NS(_P("b", _KBF("name", "item1"))),
			expect:          `{"a": "value", "b": [{"name": "item2", "c": 1, "d": "value"}]}`,
			expectItemsOnly: `{"a": "value", "b": [{"name": "item2", "c": 1, "d": "value"}]}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: atomic
`,
			object: `{"a": "value", "b": ["c"]}`,
			remove: _NS(_P("b", "c")),
			// `c` won't get removed, as it is not a field, while being addressed as one in the path.
			expect:          `{"a": "value", "b": ["c"]}`,
			expectItemsOnly: `{"a": "value", "b": ["c"]}`,
		},
		{
			schema: `types:
- name: type
  map:
    fields:
    - name: a
      type:
        scalar: string
    - name: b
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative
`,
			object:          `{"a": "value", "b": ["c"]}`,
			remove:          _NS(_P("b", _V("c"))),
			expect:          `{"a": "value", "b": []}`,
			expectItemsOnly: `{"a": "value", "b": []}`,
		},
	}

	for i, c := range cases {
		c := c
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			t.Parallel()

			parser, err := NewParser(c.schema)
			if err != nil {
				t.Fatalf("unable to parse schema: %v\n%v", err, c.schema)
			}

			parseable := parser.Type("type")
			obj, err := parseable.FromYAML(c.object)
			if err != nil {
				t.Fatalf("unable to parse object: %v\n%v", err, c.object)
			}

			expectTyped, err := parseable.FromYAML(c.expect)
			if err != nil {
				t.Fatalf("unable to parse expected object: %v\n%v", err, c.expect)
			}
			expect := expectTyped.AsValue()

			expectItemsOnlyTyped, err := parseable.FromYAML(c.expectItemsOnly)
			if err != nil {
				t.Fatalf("unable to parse/validate expected object: %v\n%v", err, c.expectItemsOnly)
			}
			expectItemsOnly := expectItemsOnlyTyped.AsValue()

			result := obj.Remove(c.remove).AsValue()
			if !value.Equals(result, expect) {
				t.Fatalf("unexpected result after Remove:\ngot: %v\nexp: %v",
					value.ToString(result), value.ToString(expect),
				)
			}

			result = obj.RemoveItems(c.remove).AsValue()
			if !value.Equals(result, expectItemsOnly) {
				t.Fatalf("unexpected result after RemoveItems:\ngot: %v\nexp: %v",
					value.ToString(result), value.ToString(expectItemsOnly),
				)
			}
		})
	}
}
