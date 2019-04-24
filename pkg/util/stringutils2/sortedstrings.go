package stringutils2

import (
	"sort"
)

type SSortedStrings []string

func NewSortedStrings(strs []string) SSortedStrings {
	if strs == nil {
		return nil
	}
	sort.Strings(strs)
	return SSortedStrings(strs)
}

func (ss SSortedStrings) Index(needle string) (int, bool) {
	i := 0
	j := len(ss) - 1
	for i <= j {
		m := (i + j) / 2
		if ss[m] < needle {
			i = m + 1
		} else if ss[m] > needle {
			j = m - 1
		} else {
			return m, true
		}
	}
	return j + 1, false
}

func (ss SSortedStrings) Contains(needle string) bool {
	_, find := ss.Index(needle)
	return find
}

func (ss SSortedStrings) ContainsAny(needles ...string) bool {
	for i := range needles {
		_, find := ss.Index(needles[i])
		if find {
			return true
		}
	}
	return false
}

func (ss SSortedStrings) ContainsAll(needles ...string) bool {
	for i := range needles {
		_, find := ss.Index(needles[i])
		if !find {
			return false
		}
	}
	return true
}

func Split(a, b SSortedStrings) (aNoB SSortedStrings, aAndB SSortedStrings, bNoA SSortedStrings) {
	a_b := make([]string, 0)
	b_a := make([]string, 0)
	anb := make([]string, 0)
	i := 0
	j := 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			anb = append(anb, a[i])
			i += 1
			j += 1
		} else if a[i] < b[j] {
			a_b = append(a_b, a[i])
			i += 1
		} else if a[i] > b[j] {
			b_a = append(b_a, b[j])
			j += 1
		}
	}
	if i < len(a) {
		a_b = append(a_b, a[i:]...)
	}
	if j < len(b) {
		b_a = append(b_a, b[j:]...)
	}
	aNoB = SSortedStrings(a_b)
	aAndB = SSortedStrings(anb)
	bNoA = SSortedStrings(b_a)
	return
}

func Merge(a, b SSortedStrings) SSortedStrings {
	ret := make([]string, 0)
	i := 0
	j := 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			ret = append(ret, a[i])
			i += 1
			j += 1
		} else if a[i] < b[j] {
			ret = append(ret, a[i])
			i += 1
		} else if a[i] > b[j] {
			ret = append(ret, b[j])
			j += 1
		}
	}
	if i < len(a) {
		ret = append(ret, a[i:]...)
	}
	if j < len(b) {
		ret = append(ret, b[j:]...)
	}
	return SSortedStrings(ret)
}
