package baremetal

import (
	"testing"
)

func TestReplaceHostAddr(t *testing.T) {
	cases := []struct {
		In   string
		Addr string
		Want string
	}{
		{
			In:   "https://www.sina.com.cn",
			Addr: "118.187.65.237",
			Want: "https://118.187.65.237",
		},
		{
			In:   "https://192.168.223.22:9292/v1/images",
			Addr: "10.168.24.23",
			Want: "https://10.168.24.23:9292/v1/images",
		},
	}
	for _, c := range cases {
		got := replaceHostAddr(c.In, c.Addr)
		if got != c.Want {
			t.Errorf("In: %s Addr: %s Got: %s Want: %s", c.In, c.Addr, got, c.Want)
		}
	}
}
