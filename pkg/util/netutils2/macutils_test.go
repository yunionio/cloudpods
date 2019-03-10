package netutils2

import (
	"testing"
	"yunion.io/x/pkg/util/netutils"
)

func TestParseMac(t *testing.T) {
	cases := []struct {
		In   string
		Inc  int
		Want string
	}{
		{
			In:   "00:50:56:C0:00:01",
			Inc:  1,
			Want: "00:50:56:c0:00:02",
		},
		{
			In:   "00:50:56:c0:00:01",
			Inc:  -1,
			Want: "00:50:56:c0:00:00",
		},
		{
			In:   "00:50:56:c0:00:01",
			Inc:  0xff,
			Want: "00:50:56:c0:01:00",
		},
		{
			In:   "00:50:56:C0:00:01",
			Inc:  -2,
			Want: "00:50:56:bf:ff:ff",
		},
		{
			In:   "00:50:56:C0:00:01",
			Inc:  -0xff,
			Want: "00:50:56:bf:ff:02",
		},
	}
	for i, c := range cases {
		mac, err := ParseMac(c.In)
		if err != nil {
			t.Errorf("parse mac errror %s", err)
		} else {
			mac2 := mac.Add(c.Inc)
			if mac2.String() != c.Want {
				t.Errorf("%d) %s inc %d want: %s got: %s", i, c.In, c.Inc, c.Want, mac2.String())
			}
			mac3 := mac2.Add(-c.Inc)
			if mac3.String() != netutils.FormatMacAddr(c.In) {
				t.Errorf("%d) %s inc %d want: %s got: %s", i, c.In, c.Inc, c.Want, mac2.String())
			}
		}
	}
}
