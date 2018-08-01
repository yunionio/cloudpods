package appsrv

import (
	"testing"
)

func TestSplitPath(t *testing.T) {
	cases := []struct {
		in  string
		out int
	}{
		{in: "/v2.0/tokens/123", out: 3},
		{in: "/v2.0//tokens//123", out: 3},
		{in: "/", out: 0},
		{in: "/v2.0//123//", out: 2},
	}
	for _, p := range cases {
		ret := SplitPath(p.in)
		if len(ret) != p.out {
			t.Error("Split error for ", p.in, " out ", ret)
		}
	}
}
