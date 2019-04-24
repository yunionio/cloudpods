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

package printutils

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestPrintJSONObjectRecursive(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "basic",
			in: `{
				"k0": {
					"k00": "v00",
					"k01": [
						{
							"k010": "v010",
							"k011": "v011"
						}
					]
				}
			}`,
			out: `+---------------+-------+
|     Field     | Value |
+---------------+-------+
| k0.k00        | v00   |
| k0.k01.0.k010 | v010  |
| k0.k01.0.k011 | v011  |
+---------------+-------+
`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			j, err := jsonutils.ParseString(c.in)
			if err != nil {
				t.Fatalf("unexpected parse json error: %s", err)
			}
			PrintJSONObjectRecursiveEx(j, func(obj jsonutils.JSONObject) {
				dict := obj.(*jsonutils.JSONDict)
				printJSONObject(dict, func(s string) {
					if s != c.out {
						t.Errorf("want:\n%s\ngot:\n%s", c.out, s)
					}
				})
			})
		})
	}
}
