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

package notify

import "testing"

func TestMobileExt(t *testing.T) {
	cases := []struct {
		data SInternationalMobile
		want string
	}{
		{
			SInternationalMobile{
				Mobile:   "+8612345678901",
				AreaCode: "+86",
			},
			"12345678901",
		},
		{
			SInternationalMobile{
				Mobile:   "+8612345678901;ext=2",
				AreaCode: "+86",
			},
			"12345678901",
		},
		{
			SInternationalMobile{
				"+8812345678901",
				"+86",
			},
			"",
		},
		{
			SInternationalMobile{
				"+8612345678901",
				"",
			},
			"12345678901",
		},
		{
			SInternationalMobile{
				"+8612345678901;ext=2",
				"",
			},
			"12345678901",
		},
		{
			SInternationalMobile{
				"13811111111",
				"",
			},
			"13811111111",
		},
		{
			SInternationalMobile{
				"+85213811111111",
				"",
			},
			"13811111111",
		},
	}

	for _, c := range cases {
		temp := c.data
		temp.AcceptExtMobile()
		if temp.Mobile != c.want {
			t.Errorf("reset mobile err,old:%s,new:%s", c.data, temp)
		}
	}
}
