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
	"strings"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

var (
	SysVServiceManager IServiceManager = &SSysVServiceManager{}
)

type SSysVServiceManager struct {
}

func (manager *SSysVServiceManager) Detect() bool {
	return procutils.NewCommand("chkconfig", "--version").Run() == nil
}

func (manager *SSysVServiceManager) Start(srvname string) error {
	return procutils.NewCommand("service", srvname, "restart").Run()
}

func (manager *SSysVServiceManager) Enable(srvname string) error {
	return procutils.NewCommand("chkconfig", srvname, "on").Run()
}

func (manager *SSysVServiceManager) Stop(srvname string) error {
	return procutils.NewCommand("service", srvname, "stop").Run()
}

func (manager *SSysVServiceManager) Disable(srvname string) error {
	return procutils.NewCommand("chkconfig", srvname, "off").Run()
}

func (manager *SSysVServiceManager) GetStatus(srvname string) SServiceStatus {
	res, _ := procutils.NewCommand("chkconfig", "--list", srvname).Output()
	res2, _ := procutils.NewCommand("service", srvname, "status").Output()
	return parseSysvStatus(string(res), string(res2), srvname)
}

func parseSysvStatus(res string, res2 string, srvname string) SServiceStatus {
	var ret SServiceStatus
	lines := strings.Split(res, "\n")
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		for i := 1; i < len(parts); i++ {
			part := parts[i]
			part = strings.TrimSpace(part)
			if strings.HasSuffix(part, ":on") {
				ret.Enabled = true
				break
			}
		}
		if len(parts) > 1 && strings.TrimSpace(parts[0]) == srvname {
			ret.Loaded = true
			if strings.Index(res2, "running") > 0 {
				ret.Active = true
			}
			break
		}
	}
	return ret
}
