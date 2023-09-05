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

package fsdriver

import (
	"fmt"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/cloudinit"
)

func (d *sGuestRootFsDriver) DeployUserData(userData string) error {
	log.Errorf("DeployUserData not implemented for %s", d.GetIRootFsDriver().String())
	return nil
}

func (d *sLinuxRootFs) DeployUserData(userData string) error {
	return d.deployUserDataBySystemd(userData)
}

func (d *sLinuxRootFs) deployUserDataByCron(userData string) error {
	const userDataPath = "/etc/userdata.sh"
	const userDataCronPath = "/etc/cron.d/userdata"

	config, err := cloudinit.ParseUserData(userData)
	var scripts string
	if err != nil {
		scripts = userData
	} else {
		scripts = config.UserDataScript()
	}
	// scripts += "\n"
	// scripts += fmt.Sprintf("rm -f %s\n", userDataCronPath)

	err = d.rootFs.FilePutContents(userDataPath, scripts, false, false)
	if err != nil {
		return errors.Wrap(err, "save user_data fail")
	}
	err = d.rootFs.Chmod(userDataPath, 0755, false)
	if err != nil {
		return errors.Wrap(err, "chmod user_data fail")
	}
	cron := fmt.Sprintf("@reboot %s\n", userDataPath)
	err = d.rootFs.FilePutContents(userDataCronPath, cron, false, false)
	if err != nil {
		return errors.Wrap(err, "save user_data cron fail")
	}
	return nil
}

func (d *sLinuxRootFs) deployUserDataBySystemd(userData string) error {
	const scriptPath = "/etc/userdata.sh"
	const runScriptPath = "/etc/run-userdata.sh"
	const serviceName = "cloud-userdata.service"

	{
		config, err := cloudinit.ParseUserData(userData)
		var scripts string
		if err != nil {
			scripts = userData
		} else {
			scripts = config.UserDataScript()
		}
		scripts += "\n"
		scripts += fmt.Sprintf("systemctl disable --now %s\n", serviceName)

		err = d.rootFs.FilePutContents(scriptPath, scripts, false, false)
		if err != nil {
			return errors.Wrap(err, "save user_data fail")
		}
	}

	{
		runScripts := fmt.Sprintf(`#!/bin/bash
/bin/bash %s >> /var/log/cloud-userdata.log 2>&1
`, scriptPath)
		err := d.rootFs.FilePutContents(runScriptPath, runScripts, false, false)
		if err != nil {
			return errors.Wrap(err, "save user_data fail")
		}
		err = d.rootFs.Chmod(runScriptPath, 0755, false)
		if err != nil {
			return errors.Wrap(err, "chmod user_data fail")
		}
	}

	{
		var unitPath = fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName)
		var enablePath = fmt.Sprintf("/etc/systemd/system/multi-user.target.wants/%s", serviceName)
		var unitContent = fmt.Sprintf(`[Unit]
Description=Run once
After=local-fs.target
After=network.target

[Service]
ExecStart=%s
RemainAfterExit=true
Type=oneshot

[Install]
WantedBy=multi-user.target
`, runScriptPath)
		err := d.rootFs.FilePutContents(unitPath, unitContent, false, false)
		if err != nil {
			return errors.Wrap(err, "save user_data unit fail")
		}

		err = d.rootFs.Symlink(unitPath, enablePath, false)
		if err != nil {
			return errors.Wrap(err, "create user_data symlink fail")
		}
	}

	return nil
}

func (w *SWindowsRootFs) DeployUserData(userData string) error {
	config, err := cloudinit.ParseUserData(userData)
	var scripts string
	if err != nil {
		scripts = userData
	} else {
		scripts = config.UserDataScript()
	}
	const scriptPath = "/windows/userdata.bat"
	err = w.rootFs.FilePutContents(scriptPath, scripts, false, true)
	if err != nil {
		return errors.Wrap(err, "save user_data fail")
	}
	userDataScript := strings.Join([]string{
		`set USER_DATA_SCRIPT=%SystemRoot%\userdata.bat`,
		`if exist %%USER_DATA_SCRIPT% (`,
		`    call %%USER_DATA_SCRIPT%`,
		`    del %%USER_DATA_SCRIPT%`,
		`)`,
	}, "\r\n")
	w.appendGuestBootScript("userdata", userDataScript)
	return nil
}
