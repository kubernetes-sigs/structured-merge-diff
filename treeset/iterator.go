package treeset

type Iterator struct {
	treeSet *IntWithComparator
	stack   []int
}

func newIterator(treeSet *IntWithComparator) Iterator {
	it := Iterator{
		treeSet: treeSet,
		stack:   make([]int, 0, log2(treeSet.Size())+1),
	}
	cur := treeSet.root
	for cur != empty {
		it.stack = append(it.stack, cur)
		cur = treeSet.nodeAt(cur).lChild
	}
	return it
}

func (it *Iterator) stackTop() int {
	return it.stack[len(it.stack)-1]
}

func (it *Iterator) stackPop() int {
	ret := it.stack[len(it.stack)-1]
	it.stack = it.stack[:len(it.stack)-1]
	return ret
}

func (it *Iterator) HasNext() bool {
	return len(it.stack) > 0
}

func (it *Iterator) Get() (int, bool) {
	if !it.HasNext() {
		return 0, false
	}
	return it.treeSet.nodeAt(it.stackTop()).Value, true
}

func (it *Iterator) Next() (int, bool) {
	if !it.HasNext() {
		return 0, false
	}
	ret, ok := it.Get()
	cur := it.treeSet.nodeAt(it.stackPop()).rChild
	for cur != empty {
		it.stack = append(it.stack, cur)
		cur = it.treeSet.nodeAt(cur).lChild
	}
	return ret, ok
}

func log2(x int) int {
	ret := 0
	for x > 0 {
		ret++
		x = x >> 1
	}
	return ret
}
