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

package conditionparser

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestEvalTemplate(t *testing.T) {
	input := jsonutils.NewDict()
	input.Add(jsonutils.NewString("myhost"), "host")
	input.Add(jsonutils.NewString("jxq"), "zone")
	input.Add(jsonutils.NewString("myprojId"), "project", "id")
	input.Add(jsonutils.NewString("my"), "project", "name")
	input.Add(jsonutils.NewString("my"), "proj_name")
	cases := []struct {
		input string
		want  string
	}{
		{
			input: "${host}-1",
			want:  "myhost-1",
		},
		{
			input: "${zone}-${host}-1",
			want:  "jxq-myhost-1",
		},
		{
			input: "${zone}-${host}-${project.name}-1",
			want:  "jxq-myhost-my-1",
		},
		{
			input: "${zone}-${host}-${proj_name}-1",
			want:  "jxq-myhost-my-1",
		},
	}
	for _, c := range cases {
		got, err := EvalTemplate(c.input, input)
		if err != nil {
			t.Errorf("EvalTemplate %s error %s", c.input, err)
		} else if got != c.want {
			t.Errorf("EvalTemplate %s got %s want %s", c.input, got, c.want)
		}
	}
}
