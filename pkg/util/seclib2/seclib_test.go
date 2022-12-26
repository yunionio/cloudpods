// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package seclib2

import (
	"math/rand"
	"testing"
	"time"

	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/httperrors"
)

func TestRandomPassword2(t *testing.T) {
	rand.Seed(time.Now().Unix())
	t.Logf("%s", RandomPassword2(12))
}

func TestMeetComplxity(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"123456", false},
		{"123abcABC!@#", true},
		{"123abcABC-@=", true},
	}
	for _, c := range cases {
		if c.want != MeetComplxity(c.in) {
			t.Errorf("%s != %v", c.in, c.want)
		}
	}
}

func TestPassword(t *testing.T) {
	cases := []struct {
		in                string
		valid             bool
		errClass          string
		invalidCharacters []byte
	}{
		{in: "123456", valid: false, errClass: httperrors.ErrWeakPassword.Error()},
		{in: "123abcABC!@#", valid: true},
		{in: "123abcABC-@=", valid: true},
	}
	for _, c := range cases {
		ap := AnalyzePasswordStrenth(c.in)
		if string(ap.Invalid) != string(c.invalidCharacters) {
			t.Fatalf("%s invalid character %s != %s", c.in, string(ap.Invalid), string(c.invalidCharacters))
		}
		err := ValidatePassword(c.in)
		if err != nil {
			t.Logf("%s -> %v", c.in, err)
			e := err.(*httputils.JSONClientError)
			if e.Class != c.errClass {
				t.Fatalf("%s invalid error class %s != %s", c.in, e.Class, c.errClass)
			}
		} else if !c.valid {
			t.Fatalf("%s should be invalid", c.in)
		}
	}
}
