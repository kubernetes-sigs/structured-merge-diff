package treeset

import (
	"fmt"
	fuzz "github.com/google/gofuzz"
	"sort"
	"testing"
)

type intArray []int

func (a intArray) Compare(i, j int) int {
	// do not a[i] - a[j], watch for int under-/over- flow
	if a[i] < a[j] {
		return -1
	}
	if a[i] > a[j] {
		return 1
	}
	return 0
}

func (a intArray) CompareNew(k interface{}, i int) int {
	return k.(int) - a[i]
}

func TestIterator(t *testing.T) {
	for i := 0; i < 16; i++ {
		t.Run(fmt.Sprintf("iteration-%d", i), func(t *testing.T) {
			var a []int
			fuzz.New().NilChance(.01).NumElements(0, 100000).Fuzz(&a)
			s := NewIntWithComparator(len(a), intArray(a))
			b := make([]int, len(a))
			copy(b, a)
			sort.Ints(b)
			for i := range a {
				s.Insert(i)
			}
			it := s.Iterator()
			for i := range b {
				if v, ok := it.Next(); !ok || a[v] != b[i] {
					t.Errorf("failed to get %v at %v, got %v, %v %v", b[i], i, ok, v, a[v])
				}
			}
		})
	}
}
