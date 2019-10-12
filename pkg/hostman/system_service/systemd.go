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
	SystemdServiceManager IServiceManager = &SSystemdServiceManager{}
)

type SSystemdServiceManager struct {
}

func (manager *SSystemdServiceManager) Detect() bool {
	_, err := procutils.NewCommand("systemctl", "--version").Run()
	return err == nil
}

func (manager *SSystemdServiceManager) Start(srvname string) error {
	_, err := procutils.NewCommand("systemctl", "restart", srvname).Run()
	return err
}

func (manager *SSystemdServiceManager) Enable(srvname string) error {
	_, err := procutils.NewCommand("systemctl", "enable", srvname).Run()
	return err
}

func (manager *SSystemdServiceManager) Stop(srvname string) error {
	_, err := procutils.NewCommand("systemctl", "stop", srvname).Run()
	return err
}

func (manager *SSystemdServiceManager) Disable(srvname string) error {
	_, err := procutils.NewCommand("systemctl", "disable", srvname).Run()
	return err
}

func (manager *SSystemdServiceManager) GetStatus(srvname string) SServiceStatus {
	res, _ := procutils.NewCommand("systemctl", "status", srvname).Run()
	res2, _ := procutils.NewCommand("systemctl", "is-enabled", srvname).Run()
	return parseSystemdStatus(string(res), string(res2), srvname)
}

func parseSystemdStatus(res string, res2 string, srvname string) SServiceStatus {
	var ret SServiceStatus

	lines := strings.Split(res, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Loaded:") {
			parts := strings.Split(line, " ")
			if len(parts) > 1 && parts[1] == "loaded" {
				ret.Loaded = true
			}
		} else if strings.HasPrefix(line, "Active:") {
			parts := strings.Split(line, " ")
			if len(parts) > 1 && parts[1] == "active" {
				ret.Active = true
			}
		}
	}

	lines2 := strings.Split(res2, "\n")
	for _, line := range lines2 {
		switch line {
		case "enabled", "enabled-runtime":
			ret.Enabled = true
			break
		}
	}
	return ret
}
