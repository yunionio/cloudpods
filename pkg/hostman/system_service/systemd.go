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

func systemctl(args ...string) *procutils.Command {
	return procutils.NewRemoteCommandAsFarAsPossible("systemctl", args...)
}

func (manager *SSystemdServiceManager) Detect() bool {
	return systemctl("--version").Run() == nil
}

func (manager *SSystemdServiceManager) Start(srvname string) error {
	return systemctl("restart", srvname).Run()
}

func (manager *SSystemdServiceManager) Enable(srvname string) error {
	return systemctl("enable", srvname).Run()
}

func (manager *SSystemdServiceManager) Stop(srvname string) error {
	return systemctl("stop", srvname).Run()
}

func (manager *SSystemdServiceManager) Disable(srvname string) error {
	return systemctl("disable", srvname).Run()
}

func (manager *SSystemdServiceManager) GetStatus(srvname string) SServiceStatus {
	res, _ := systemctl("status", srvname).Output()
	res2, _ := systemctl("is-enabled", srvname).Output()
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
