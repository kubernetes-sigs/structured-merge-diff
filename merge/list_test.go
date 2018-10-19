package merge_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/typed"

	"sigs.k8s.io/structured-merge-diff/merge"
)

func TestInlineListMerge(t *testing.T) {
	parser, err := typed.NewParser(`types:
- name: lists
  struct:
    fields:
    - name: items
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative`)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("disjointItemsOrder", func(t *testing.T) {
		state := &State{
			Updater:  &merge.Updater{},
			Parser:   parser,
			Typename: "lists",
		}

		lhs := typed.YAMLObject(`
items:
- a
- b
- c
`)

		rhs := typed.YAMLObject(`
items:
- d
- e
`)

		output := typed.YAMLObject(`
items:
- a
- b
- c
- d
- e
`)

		twoWayMerge(t, state, lhs, rhs, output)
	})

	t.Run("overlappingItemsOrder", func(t *testing.T) {
		state := &State{
			Updater:  &merge.Updater{},
			Parser:   parser,
			Typename: "lists",
		}

		lhs := typed.YAMLObject(`
items:
- a
- b
- c
`)

		rhs := typed.YAMLObject(`
items:
- a
- b
- d
`)

		output := typed.YAMLObject(`
items:
- a
- b
- c
- d
`)

		twoWayMerge(t, state, lhs, rhs, output)
	})

	t.Run("disjointOverlappingItemsOrder", func(t *testing.T) {
		state := &State{
			Updater:  &merge.Updater{},
			Parser:   parser,
			Typename: "lists",
		}

		lhs := typed.YAMLObject(`
items:
- a
- b
- c
`)

		rhs := typed.YAMLObject(`
items:
- b
- a
- d
`)

		output := typed.YAMLObject(`
items:
- b
- a
- c
- d
`)

		twoWayMerge(t, state, lhs, rhs, output)
	})

	t.Run("orderViolationHandling1", func(t *testing.T) {
		state := &State{
			Updater:  &merge.Updater{},
			Parser:   parser,
			Typename: "lists",
		}

		lhs := typed.YAMLObject(`
items:
- a
- b
- c
- d
`)

		rhs := typed.YAMLObject(`
items:
- d
- b
- a
`)

		output := typed.YAMLObject(`
items:
- d
- b
- a
- c
`)

		twoWayMerge(t, state, lhs, rhs, output)
	})

	t.Run("orderViolationHandling2", func(t *testing.T) {
		state := &State{
			Updater:  &merge.Updater{},
			Parser:   parser,
			Typename: "lists",
		}

		lhs := typed.YAMLObject(`
items:
- a
- b
- c
- d
- e
`)

		rhs := typed.YAMLObject(`
items:
- e
- d
- b
- a
`)

		output := typed.YAMLObject(`
items:
- c
- e
- d
- b
- a
`)

		twoWayMerge(t, state, lhs, rhs, output)
	})

	t.Run("orderViolationHandling3", func(t *testing.T) {
		state := &State{
			Updater:  &merge.Updater{},
			Parser:   parser,
			Typename: "lists",
		}

		lhs := typed.YAMLObject(`
items:
- a
- b
- c
- d
- e
`)

		rhs := typed.YAMLObject(`
items:
- a
- d
- b
- e
`)

		output := typed.YAMLObject(`
items:
- a
- c
- d
- b
- e
`)

		twoWayMerge(t, state, lhs, rhs, output)
	})

	t.Run("orderConstraintCheck", func(t *testing.T) {
		state := &State{
			Updater:  &merge.Updater{},
			Parser:   parser,
			Typename: "lists",
		}

		lhs := typed.YAMLObject(`
items:
- a
- b
- c
`)

		rhs := typed.YAMLObject(`
items:
- a
- x
- y
- z
- b
`)

		output := typed.YAMLObject(`
items:
- a
- x
- y
- z
- b
- c
`)

		twoWayMerge(t, state, lhs, rhs, output)
	})
}

func twoWayMerge(t *testing.T, state *State, lhs, rhs, output typed.YAMLObject) {
	err := state.Apply(lhs, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	err = state.Apply(rhs, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	res, err := state.CompareLive(output)
	if err != nil {
		t.Fatalf("Failed to compare live with config: %v", err)
	}
	if !res.IsSame() {
		t.Fatalf("Merge result was not as expected, got:\n%v\nwanted:%v\n", res, output)
	}
}
