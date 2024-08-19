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
	"fmt"
	"os"
	"strconv"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	bondingModePath   = "/sys/class/net/%s/bonding/mode"
	bondingSlavesPath = "/sys/class/net/%s/bonding/slaves"
)

type SBondingConfig struct {
	Mode     int
	ModeName string
	Slaves   []string
}

func getBondingConfig(ifname string) *SBondingConfig {
	modePath := fmt.Sprintf(bondingModePath, ifname)
	slavesPath := fmt.Sprintf(bondingSlavesPath, ifname)
	if !fileutils2.Exists(modePath) || !fileutils2.Exists(slavesPath) {
		return nil
	}
	conf := SBondingConfig{}
	{
		content, err := os.ReadFile(modePath)
		if err != nil {
			log.Errorf("fail to read %s", modePath)
			return nil
		}
		parts := strings.Split(string(content), " ")
		mode, _ := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1]))
		modeName := strings.Join(parts[:len(parts)-1], " ")
		conf.Mode = int(mode)
		conf.ModeName = modeName
	}
	{
		content, err := os.ReadFile(slavesPath)
		if err != nil {
			log.Errorf("fail to read %s", modePath)
			return nil
		}
		parts := strings.Split(string(content), " ")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if len(part) > 0 {
				conf.Slaves = append(conf.Slaves, part)
			}
		}
	}
	return &conf
}
