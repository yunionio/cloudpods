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

package dbutils

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestParseMySQLConnStr(t *testing.T) {
	cases := []struct {
		connStr string
		cfg     SDBConfig
	}{
		{
			connStr: "regionadmin:abcdefg@tcp(192.168.222.22:3306)/yunioncloud?parseTime=True",
			cfg: SDBConfig{
				Hostport: "192.168.222.22:3306",
				Database: "yunioncloud",
				Username: "regionadmin",
				Password: "abcdefg",
			},
		},
	}
	for _, c := range cases {
		got := ParseMySQLConnStr(c.connStr)
		if jsonutils.Marshal(got).String() != jsonutils.Marshal(c.cfg).String() {
			t.Errorf("parse %s got %s want %s", c.connStr, jsonutils.Marshal(got), jsonutils.Marshal(c.cfg))
		}
	}
}
