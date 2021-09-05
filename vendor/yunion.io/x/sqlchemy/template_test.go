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

package sqlchemy

import "testing"

func TestTemplateEvel(t *testing.T) {
	cases := []struct {
		In   string
		Var  interface{}
		Want string
	}{
		{
			In: "DROP INDEX `{{ .Index }}` ON `{{ .Table }}`",
			Var: struct {
				Table string
				Index string
			}{
				Table: "testtable",
				Index: "ix_testtable_name",
			},
			Want: "DROP INDEX `ix_testtable_name` ON `testtable`",
		},
		{
			In: "DROP INDEX `{{ .Table }}`.`{{ .Index }}`",
			Var: struct {
				Table string
				Index string
			}{
				Table: "testtable",
				Index: "ix_testtable_name",
			},
			Want: "DROP INDEX `testtable`.`ix_testtable_name`",
		},
	}
	for _, c := range cases {
		got := templateEval(c.In, c.Var)
		if got != c.Want {
			t.Errorf("want: %s != got: %s", c.Want, got)
		}
	}
}
