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

package system_service

import "fmt"

type SNtpd struct {
	*SBaseSystemService
}

func NewNtpdService() *SNtpd {
	return &SNtpd{NewBaseSystemService("ntpd", nil)}
}

func (s *SNtpd) GetConfig(kwargs map[string]interface{}) string {
	var srvs = []string{}
	if servers, ok := kwargs["servers"]; ok {
		ss, _ := servers.([]string)
		for _, srv := range ss {
			srvs = append(srvs, srv[len("ntp://"):])
		}
	} else {
		srvs = []string{"1.cn.pool.ntp.org",
			"2.cn.pool.ntp.org",
			"3.cn.pool.ntp.org",
			"0.cn.pool.ntp.org",
			"cn.pool.ntp.org"}
	}

	conf := ""
	conf += "driftfile /var/lib/ntp/drift\n"
	conf += "restrict default nomodify notrap nopeer noquery kod limited\n"
	conf += "restrict 127.0.0.1\n"
	conf += "restrict ::1\n"
	for _, srv := range srvs {
		conf += fmt.Sprintf("server %s iburst\n", srv)
	}
	conf += "includefile /etc/ntp/crypto/pw\n"
	conf += "keys /etc/ntp/keys\n"
	conf += "disable monitor\n"
	return conf
}

func (s *SNtpd) GetConfigFile() string {
	return "/etc/ntp.conf"
}

func (s *SNtpd) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SNtpd) BgReload(kwargs map[string]interface{}) {
	go s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SNtpd) BgReloadConf(kwargs map[string]interface{}) {
	go s.reloadConf(s.GetConfig(kwargs), s.GetConfigFile())
}
