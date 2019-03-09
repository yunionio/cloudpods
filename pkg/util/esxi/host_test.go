package esxi

import (
	"testing"
)

func TestFormatName(t *testing.T) {
	cases := []struct {
		In   string
		Want string
	}{
		{
			In:   "esxi-172.16.23.1",
			Want: "esxi-172-16-23-1",
		},
		{
			In:   "esxi6.yunion.cn",
			Want: "esxi6",
		},
	}
	for _, c := range cases {
		got := formatName(c.In)
		if got != c.Want {
			t.Errorf("got: %s want %s", got, c.Want)
		}
	}
}
