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

package netutils2

import (
	"io/ioutil"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SVlanConfig struct {
	Ifname string
	VlanId int
	Parent string
}

const (
	vlanConfigPath = "/proc/net/vlan/config"
)

func parseVlanConfig() (map[string]*SVlanConfig, error) {
	var content string
	if fileutils2.IsFile(vlanConfigPath) {
		cont, err := ioutil.ReadFile(vlanConfigPath)
		if err != nil {
			return nil, errors.Wrap(err, "ReadFile")
		}
		content = string(cont)
	}
	return parseVlanConfigContent(content)
}

func parseVlanConfigContent(content string) (map[string]*SVlanConfig, error) {
	vlanConfig := make(map[string]*SVlanConfig)
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		parts := strings.Split(l, "|")
		if len(parts) >= 3 {
			ifname := strings.TrimSpace(parts[0])
			vlan, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			parent := strings.TrimSpace(parts[2])
			vlanConfig[ifname] = &SVlanConfig{
				Ifname: ifname,
				VlanId: int(vlan),
				Parent: parent,
			}
		}
	}
	return vlanConfig, nil
}

func getVlanConfig(ifname string) *SVlanConfig {
	vlanConfig, err := parseVlanConfig()
	if err != nil {
		log.Errorf("fail to parseVlanConfig %s", err)
		return nil
	}
	if conf, ok := vlanConfig[ifname]; ok {
		return conf
	} else {
		return nil
	}
}
