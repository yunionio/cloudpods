package diskutils

import "testing"

func TestParseNbdkitOutputNbdURI(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   string
		ok     bool
	}{
		{
			name:   "bound to any ip address",
			output: "nbdkit: debug: bound to IP address <any>:10809 (2 socket(s))",
			want:   "nbd://127.0.0.1:10809/",
			ok:     true,
		},
		{
			name:   "direct nbd uri",
			output: "ready at nbd://localhost:10810",
			want:   "nbd://127.0.0.1:10810/",
			ok:     true,
		},
		{
			name:   "port key value",
			output: "nbdkit: listening port=10811",
			want:   "nbd://127.0.0.1:10811/",
			ok:     true,
		},
		{
			name:   "cannot parse",
			output: "nbdkit: debug: waiting for connection",
			want:   "",
			ok:     false,
		},
	}

	for _, tc := range cases {
		got, ok := parseNbdkitOutputNbdURI(tc.output)
		if ok != tc.ok {
			t.Fatalf("%s: ok mismatch, got %v want %v", tc.name, ok, tc.ok)
		}
		if got != tc.want {
			t.Fatalf("%s: uri mismatch, got %q want %q", tc.name, got, tc.want)
		}
	}
}
