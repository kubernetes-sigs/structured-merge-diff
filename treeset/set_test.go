package treeset

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	mathrand "math/rand"
	"testing"
)

type intSet struct {
	a   []int
	set *IntWithComparator
}

func (s *intSet) Compare(i, j int) int {
	return s.a[i] - s.a[j]
}

func (s *intSet) CompareNew(k interface{}, i int) int {
	return k.(int) - s.a[i]
}

func (s *intSet) Insert(k int) bool {
	s.a = append(s.a, k)
	created := s.set.Insert(len(s.a) - 1)
	if !created {
		s.a = s.a[:len(s.a)-1]
	}
	return created
}

func (s *intSet) Get(k int) bool {
	idx, ok := s.set.Find(k)
	if ok {
		if s.a[idx] != k {
			panic("index did not match")
		}
	}
	return ok
}

func (s *intSet) Union(rhs *intSet) *intSet {
	return &intSet{
		a:   append(s.a, rhs.a...),
		set: s.set.Union(rhs.set),
	}
}

func newIntSet() *intSet {
	s := new(intSet)
	s.set = NewIntWithComparator(0, s)
	return s
}

func TestIntSet(t *testing.T) {
	for i := 0; i < 4; i++ {
		t.Run(fmt.Sprintf("iteration-%d", i), func(t *testing.T) {
			t.Parallel()

			b := make([]byte, 8)
			_, err := cryptorand.Reader.Read(b)
			if err != nil {
				t.Fatalf("cannot create seed: %v", err)
			}
			seed := int64(binary.LittleEndian.Uint64(b))

			rand := mathrand.New(mathrand.NewSource(seed))
			s := newIntSet()
			hashSet := make(map[int]bool)
			ceil := rand.Intn(1000000)
			ops := 1000000 + rand.Intn(1000000)
			for j := 0; j < ops; j++ {
				k := rand.Intn(ceil)
				if rand.Int63()%2 == 0 {
					s.Insert(k)
					hashSet[k] = true
				}
				if ok := s.Get(k); ok != hashSet[k] {
					t.Fatalf("expected %v found = %v, but got %v", k, hashSet[k], ok)
				}
			}
		})
	}
}
