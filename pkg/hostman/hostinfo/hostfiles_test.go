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

package hostinfo

import (
	"testing"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestFixTelegrafConfPath(t *testing.T) {
	cases := []struct {
		hostFile api.SHostFile
		path     string
	}{
		{
			hostFile: api.SHostFile{
				SInfrasResourceBase: apis.SInfrasResourceBase{
					SDomainLevelResourceBase: apis.SDomainLevelResourceBase{
						SStandaloneResourceBase: apis.SStandaloneResourceBase{
							Name: "mysql1",
						},
					},
				},
				Type: string(api.TelegrafConf),
				Path: "/etc/telegraf/mysqlmon.conf",
			},
			path: "/etc/telegraf/telegraf.d/mysqlmon.conf",
		},
		{
			hostFile: api.SHostFile{
				SInfrasResourceBase: apis.SInfrasResourceBase{
					SDomainLevelResourceBase: apis.SDomainLevelResourceBase{
						SStandaloneResourceBase: apis.SStandaloneResourceBase{
							Name: "mysql2",
						},
					},
				},
				Type: string(api.TelegrafConf),
			},
			path: "/etc/telegraf/telegraf.d/mysql2.conf",
		},
	}
	for _, c := range cases {
		fixed := fixTelegrafConfPath(&c.hostFile)
		if fixed.Path != c.path {
			t.Errorf("fixTelegrafConfPath(%s) = %s, expected %s", c.hostFile.Name, fixed.Path, c.path)
		}
	}
}
