package lr

import (
	"math/big"
)

type CombinationIterator struct {
	r, n, i, c, max int
	s               []int
}

func NewCombinationIterator(n, r int) *CombinationIterator {
	s := make([]int, r)
	for j := range s {
		s[j] = j
	}
	c := int(new(big.Int).Binomial(int64(n), int64(r)).Int64())

	return &CombinationIterator{r: r, n: n, c: c, s: s}
}

func (ci *CombinationIterator) HasNext() bool {
	return ci.i < ci.c
}

func (ci *CombinationIterator) Next() {
	if ci.HasNext() {
		ci.updateS()
	}
}

func (ci *CombinationIterator) S() []int {
	return ci.s
}

func (ci *CombinationIterator) updateS() {
	ci.i++

	i := ci.r - 1
	ci.s[i]++
	for (i > 0) && (ci.s[i] >= ci.n-ci.r+1+i) {
		i--
		ci.s[i]++
	}

	for i = i + 1; i < ci.r; i++ {
		ci.s[i] = ci.s[i-1] + 1
	}
}
