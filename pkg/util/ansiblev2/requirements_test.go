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

package ansiblev2

import "testing"

func TestRequirementsString(t *testing.T) {
	rss := []RoleSource{
		{
			Src: "zaiste.nginx",
		},
		{
			Src:     "geerlingguy.gitlab",
			Version: "1.1.0",
		},
		{
			Name: "essentials",
			Src:  "https://github.com/zaiste/ansible-essentials",
		},
		{
			Name:    "java",
			Src:     "https://github.com/geerlingguy/ansible-role-java",
			Version: "origin/main",
		},
		{
			Name:    "jenkins",
			Src:     "https://github.com/geerlingguy/ansible-role-jenkins",
			Version: "4.2.1",
		},
	}
	rm := NewRequirements(rss...)
	t.Logf("\n%s", rm.String())
}
