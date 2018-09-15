package seclib2

import (
	"testing"
	"math/rand"
	"time"
)

func TestRandomPassword2(t *testing.T) {
	rand.Seed(time.Now().Unix())
	t.Logf("%s", RandomPassword2(12))
}

func TestMeetComplxity(t *testing.T) {
	cases := [] struct {
		in string
		want bool
	} {
		{"123456", false},
		{"123abcABC!@#", true},
	}
	for _, c := range cases {
		if c.want != MeetComplxity(c.in) {
			t.Errorf("%s != %v", c.in, c.want)
		}
	}
}
