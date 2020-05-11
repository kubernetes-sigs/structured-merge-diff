package typed_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/v3/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v3/typed"
)

func TestComparisonRemove(t *testing.T) {
	cases := []struct {
		name       string
		Comparison *typed.Comparison
		Remove     *fieldpath.Set
		Expect     *typed.Comparison
		Fails      bool
	}{
		{
			name: "works on nil set",
			Comparison: &typed.Comparison{
				Added:    fieldpath.NewSet(fieldpath.MakePathOrDie("a")),
				Modified: fieldpath.NewSet(fieldpath.MakePathOrDie("b")),
				Removed:  fieldpath.NewSet(fieldpath.MakePathOrDie("c")),
			},
			Remove: nil,
			Expect: &typed.Comparison{
				Added:    fieldpath.NewSet(fieldpath.MakePathOrDie("a")),
				Modified: fieldpath.NewSet(fieldpath.MakePathOrDie("b")),
				Removed:  fieldpath.NewSet(fieldpath.MakePathOrDie("c")),
			},
		},
		{
			name: "works on empty set",
			Comparison: &typed.Comparison{
				Added:    fieldpath.NewSet(fieldpath.MakePathOrDie("a")),
				Modified: fieldpath.NewSet(fieldpath.MakePathOrDie("b")),
				Removed:  fieldpath.NewSet(fieldpath.MakePathOrDie("c")),
			},
			Remove: fieldpath.NewSet(),
			Expect: &typed.Comparison{
				Added:    fieldpath.NewSet(fieldpath.MakePathOrDie("a")),
				Modified: fieldpath.NewSet(fieldpath.MakePathOrDie("b")),
				Removed:  fieldpath.NewSet(fieldpath.MakePathOrDie("c")),
			},
		},
		{
			name: "does not result in empty comparison on empty set",
			Comparison: &typed.Comparison{
				Added:    fieldpath.NewSet(fieldpath.MakePathOrDie("a")),
				Modified: fieldpath.NewSet(fieldpath.MakePathOrDie("b")),
				Removed:  fieldpath.NewSet(fieldpath.MakePathOrDie("c")),
			},
			Remove: fieldpath.NewSet(),
			Expect: &typed.Comparison{
				Added:    fieldpath.NewSet(),
				Modified: fieldpath.NewSet(),
				Removed:  fieldpath.NewSet(),
			},
			Fails: true,
		},
		{
			name: "removes simple nested paths",
			Comparison: &typed.Comparison{
				Added: fieldpath.NewSet(
					fieldpath.MakePathOrDie("a"),
					fieldpath.MakePathOrDie("a", "ab", "aba"),
				),
				Modified: fieldpath.NewSet(),
				Removed:  fieldpath.NewSet(),
			},
			Remove: fieldpath.NewSet(
				fieldpath.MakePathOrDie("a"),
			),
			Expect: &typed.Comparison{
				Added:    fieldpath.NewSet(),
				Modified: fieldpath.NewSet(),
				Removed:  fieldpath.NewSet(),
			},
		},
		{
			name: "removes nested paths",
			Comparison: &typed.Comparison{
				Added: fieldpath.NewSet(
					fieldpath.MakePathOrDie("a"),
					fieldpath.MakePathOrDie("a", "aa", "aaa"),
					fieldpath.MakePathOrDie("a", "aa", "aab"),
					fieldpath.MakePathOrDie("a", "ab", "aba"),
					fieldpath.MakePathOrDie("a", "ab", "abb"),
				),
				Modified: fieldpath.NewSet(
					fieldpath.MakePathOrDie("b"),
					fieldpath.MakePathOrDie("b", "ba", "baa"),
					fieldpath.MakePathOrDie("b", "ba", "bab"),
					fieldpath.MakePathOrDie("b", "bb", "bba"),
					fieldpath.MakePathOrDie("b", "bb", "bbb"),
				),
				Removed: fieldpath.NewSet(
					fieldpath.MakePathOrDie("c"),
					fieldpath.MakePathOrDie("c", "ca", "caa"),
					fieldpath.MakePathOrDie("c", "ca", "cab"),
					fieldpath.MakePathOrDie("c", "cb", "cba"),
					fieldpath.MakePathOrDie("c", "cb", "cbb"),
				),
			},
			Remove: fieldpath.NewSet(
				fieldpath.MakePathOrDie("a", "aa"),
				fieldpath.MakePathOrDie("b", "bb", "bba"),
				fieldpath.MakePathOrDie("c"),
			),
			Expect: &typed.Comparison{
				Added: fieldpath.NewSet(
					fieldpath.MakePathOrDie("a"),
					fieldpath.MakePathOrDie("a", "ab", "aba"),
					fieldpath.MakePathOrDie("a", "ab", "abb"),
				),
				Modified: fieldpath.NewSet(
					fieldpath.MakePathOrDie("b"),
					fieldpath.MakePathOrDie("b", "ba", "baa"),
					fieldpath.MakePathOrDie("b", "ba", "bab"),
					fieldpath.MakePathOrDie("b", "bb", "bbb"),
				),
				Removed: fieldpath.NewSet(),
			},
		},
		{
			name: "does not remove every child",
			Comparison: &typed.Comparison{
				Added: fieldpath.NewSet(
					fieldpath.MakePathOrDie("a", "aa", "aaa"),
					fieldpath.MakePathOrDie("a", "aa", "aab"),
					fieldpath.MakePathOrDie("a", "ab", "aba"),
					fieldpath.MakePathOrDie("a", "ab", "abb"),
				),
				Modified: fieldpath.NewSet(),
				Removed:  fieldpath.NewSet(),
			},
			Remove: fieldpath.NewSet(
				fieldpath.MakePathOrDie("a", "ab"),
			),
			Expect: &typed.Comparison{
				Added: fieldpath.NewSet(
					fieldpath.MakePathOrDie("a", "aa", "aaa"),
					fieldpath.MakePathOrDie("a", "aa", "aab"),
				),
				Modified: fieldpath.NewSet(),
				Removed:  fieldpath.NewSet(),
			},
		},
		{
			name: "removes simple path",
			Comparison: &typed.Comparison{
				Added: fieldpath.NewSet(
					fieldpath.MakePathOrDie("a"),
				),
				Modified: fieldpath.NewSet(
					fieldpath.MakePathOrDie("b"),
				),
				Removed: fieldpath.NewSet(
					fieldpath.MakePathOrDie("c"),
				),
			},
			Remove: fieldpath.NewSet(
				fieldpath.MakePathOrDie("a"),
				fieldpath.MakePathOrDie("b"),
				fieldpath.MakePathOrDie("c"),
			),
			Expect: &typed.Comparison{
				Added:    fieldpath.NewSet(),
				Modified: fieldpath.NewSet(),
				Removed:  fieldpath.NewSet(),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.Comparison.Remove(c.Remove)
			if (!c.Comparison.Added.Equals(c.Expect.Added) ||
				!c.Comparison.Modified.Equals(c.Expect.Modified) ||
				!c.Comparison.Removed.Equals(c.Expect.Removed)) != c.Fails {
				t.Fatalf("remove expected: \n%v\nremoved:\n%v\ngot:\n%v\n", c.Expect, c.Remove, c.Comparison)
			}
		})
	}
}
