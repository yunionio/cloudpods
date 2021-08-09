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

package responses

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestTransColonToDot(t *testing.T) {
	raw := `{"A:b::C": "1:2:3", "A": true, "B": ["1:2", ":", "c"], "D:E": true}`
	obj, err := jsonutils.ParseString(raw)
	if err != nil {
		t.Fatalf("json parse: %v", err)
	}
	gotj, err := TransColonToDot(obj)
	if err != nil {
		t.Fatalf("trans: %v", err)
	}
	wantj, _ := jsonutils.ParseString(`{"A":true,"A.b..C":"1:2:3","B":["1:2",":","c"],"D.E":true}`)
	if !wantj.Equals(gotj) {
		t.Errorf("trans failed, want:\n%s\ngot:\n%s", wantj, gotj)
	}
}
