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
