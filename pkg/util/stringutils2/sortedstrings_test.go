package stringutils2

import (
	"testing"
)

func TestSortedString(t *testing.T) {
	input := []string{"Go", "Bravo", "Gopher", "Alpha", "Grin", "Delta"}
	// Alpha Bravo Delta Go Gopher Grin
	// 0     1     2     3  4      5
	ss := NewSortedStrings(input)
	cases := []struct {
		Needle string
		Index  int
		Want   bool
	}{
		{"Go", 3, true},
		{"Bravo", 1, true},
		{"Gopher", 4, true},
		{"Alpha", 0, true},
		{"Grin", 5, true},
		{"Delta", 2, true},
		{"Go1", 4, false},
		{"G", 3, false},
		{"A", 0, false},
		{"T", 6, false},
	}
	for _, c := range cases {
		idx, find := ss.Index(c.Needle)
		if idx != c.Index || find != c.Want {
			t.Errorf("%s: want: %d %v got: %d %v", c.Needle, c.Index, c.Want, idx, find)
		}
	}
}

func TestSplitStrings(t *testing.T) {
	input := []string{"Go", "Bravo", "Gopher", "Alpha", "Grin", "Delta"}
	input2 := []string{"Go2", "Bravo", "Gopher", "Alpha1", "Grin", "Delt"}

	ss1 := NewSortedStrings(input)
	ss2 := NewSortedStrings(input2)

	a, b, c := Split(ss1, ss2)
	t.Logf("A: %s", ss1)
	t.Logf("B: %s", ss2)
	t.Logf("A-B: %s", a)
	t.Logf("AnB: %s", b)
	t.Logf("B-A: %s", c)
}

func TestMergeStrings(t *testing.T) {
	input := []string{"Go", "Bravo", "Gopher", "Alpha", "Grin", "Delta"}
	input2 := []string{"Go2", "Bravo", "Gopher", "Alpha1", "Grin", "Delt"}

	ss1 := NewSortedStrings(input)
	ss2 := NewSortedStrings(input2)

	m := Merge(ss1, ss2)
	t.Logf("A: %s", ss1)
	t.Logf("B: %s", ss2)
	t.Logf("%s", m)
}
