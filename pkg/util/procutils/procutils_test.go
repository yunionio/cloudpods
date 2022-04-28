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

package procutils

import (
	"os"
	"testing"
)

func TestStat(t *testing.T) {
	cases := []struct {
		filename string
		exists   bool
		isdir    bool
	}{
		{
			filename: "/",
			exists:   true,
			isdir:    true,
		},
		{
			filename: "/tmp/__0_1_2_3_a_b_c_d____",
			exists:   false,
			isdir:    false,
		},
	}
	for _, c := range cases {
		fi, err := RemoteStat(c.filename)
		if err != nil {
			if !c.exists && os.IsNotExist(err) {
				// ok
			} else {
				t.Errorf("expect exists: %v err: %s", c.exists, err)
			}
		} else {
			if fi.IsDir() != c.isdir {
				t.Errorf("isdir want: %v got: %v", c.isdir, fi.IsDir())
			}
		}
	}
}
