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

package models

import (
	"testing"
)

func TestNormalizeProjectName(t *testing.T) {
	cases := []struct {
		In   string
		Want string
	}{
		{"分公司1", "fengongsi1"},
		{"集团/分公司/项目A", "jituan-fengongsi-xiangmua"},
	}
	for _, c := range cases {
		got := NormalizeProjectName(c.In)
		if got != c.Want {
			t.Errorf("NormalizeProjectName %s got %s want %s", c.In, got, c.Want)
		}
	}
}
