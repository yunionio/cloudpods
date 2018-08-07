package esxi

import "testing"

func TestReverseArray(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}

	reverseArray(a)
	t.Logf("%#v", a)

	b := []string{"1", "2", "3", "4", "5"}
	reverseArray(b)
	t.Logf("%#v", b)
}
