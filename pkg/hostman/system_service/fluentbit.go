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

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/log"
)

type SFluentbit struct {
	*SBaseSystemService
}

func NewFluentbitService() *SFluentbit {
	return &SFluentbit{NewBaseSystemService("fluentbit", nil)}
}

func (s *SFluentbit) GetConfig(kwargs map[string]interface{}) string {
	// 写到这
	conf := ""
	conf += "[SERVICE]\n"
	conf += "    Flush 5\n"
	conf += "    Daemon Off\n"
	conf += "    Log_Level info\n"
	conf += "    Parsers_File parsers.conf\n"
	conf += "    Plugins_File plugins.conf\n"
	conf += "    HTTP_Server Off\n"
	conf += "\n"

	conf += "[INPUT]\n"
	conf += "    Name systemd\n"
	unitsInf := kwargs["units"]
	units, _ := unitsInf.([]string)
	for _, u := range units {
		conf += fmt.Sprintf("    Systemd_Filter  _SYSTEMD_UNIT=%s.service\n", u)
	}
	conf += "    Tag host.*\n"
	conf += "\n"

	ielUrl := kwargs["elasticsearch"]
	mesUrl, _ := ielUrl.(map[string]string)
	sesUrl := mesUrl["url"]
	esurl, err := url.Parse(sesUrl)
	if err != nil {
		log.Errorln(err)
		return ""
	}
	esHostname := strings.Split(esurl.Host, ":")[0]
	conf += "[OUTPUT]\n"
	conf += "    Name es\n"
	conf += "    Match *\n"
	conf += fmt.Sprintf("    Host %s\n", esHostname)
	conf += "    Port 9200\n"
	conf += "    Logstash_Format on\n"
	conf += "    Retry_Limit False\n"
	conf += "    Type flb_type\n"
	conf += "    Time_Key @timestamp\n"
	conf += "    Logstash_Prefix onecloud\n"
	return conf
}

func (s *SFluentbit) GetConfigFile() string {
	return "/etc/fluent-bit/fluent-bit.conf"
}

func (s *SFluentbit) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SFluentbit) BgReload(kwargs map[string]interface{}) {
	go s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SFluentbit) BgReloadConf(kwargs map[string]interface{}) {
	go s.reloadConf(s.GetConfig(kwargs), s.GetConfigFile())
}
