package identity

import (
	"reflect"
	"testing"
)

func TestJoinLabel(t *testing.T) {
	cases := []struct {
		segs []string
		want string
	}{
		{
			segs: []string{"L1", "L2", "L3"},
			want: "L1/L2/L3",
		},
		{
			segs: []string{"L1/", "L2", "L3"},
			want: "L1/L2/L3",
		},
		{
			segs: []string{"L1/ ", "/L2", "/L3/"},
			want: "L1/L2/L3",
		},
	}
	for _, c := range cases {
		got := JoinLabels(c.segs...)
		if got != c.want {
			t.Errorf("got %s want %s", got, c.want)
		}
	}
}

func TestSplitLabel(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{
			in:   "L1/L2/L3",
			want: []string{"L1", "L2", "L3"},
		},
		{
			in:   "L1/L2//L3",
			want: []string{"L1", "L2", "L3"},
		},
		{
			in:   "/L1/L2/L3/",
			want: []string{"L1", "L2", "L3"},
		},
	}
	for _, c := range cases {
		got := SplitLabel(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("want %s got %s", c.want, got)
		}
	}
}
