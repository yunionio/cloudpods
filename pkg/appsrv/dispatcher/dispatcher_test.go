package dispatcher

import "testing"

func TestFetchContextIds(t *testing.T) {
	cases := []struct {
		segs   []string
		params map[string]string
	}{
		{
			[]string{"servers", "<resid_0>", "disks", "<resid_1>", "test"},
			map[string]string{
				"<resid_0>": "12345",
				"<resid_1>": "23",
			},
		},
		{
			[]string{"servers", "<resid_0>", "disks", "<resid_1>", "test"},
			map[string]string{
				"<resid_0>": "12345",
			},
		},
		{
			[]string{"servers", "<resid_0>"},
			map[string]string{
				"<resid_0>": "12345",
				"<resid_1>": "23",
			},
		},
	}
	for _, c := range cases {
		ctxIdx, keys := fetchContextIds(c.segs, c.params)
		t.Logf("%#v %#v", ctxIdx, keys)
	}
}
