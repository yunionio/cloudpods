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
	cron := fmt.Sprintf("@reboot /bin/sh %s\n", userDataPath)
	err = d.rootFs.FilePutContents(userDataCronPath, cron, false, false)
	if err != nil {
		return errors.Wrap(err, "save user_data cron fail")
	}
	return nil
}

func (d *sLinuxRootFs) deployUserDataBySystemd(userData string) error {
	const scriptPath = "/etc/userdata.sh"
	const serviceName = "cloud-userdata"

	{
		config, err := cloudinit.ParseUserData(userData)
		var scripts string
		if err != nil {
			scripts = userData
		} else {
			scripts = config.UserDataScript()
		}
		scripts += "\n"
		if d.isSupportSystemd() {
			// diable systemd service
			scripts += fmt.Sprintf("systemctl disable --now %s\n", serviceName)
		} else {
			// cleanup crontab
			scripts += fmt.Sprintf("crontab -l 2>/dev/null | grep -v '%s') |crontab -\n", scriptPath)
		}

		err = d.rootFs.FilePutContents(scriptPath, scripts, false, false)
		if err != nil {
			return errors.Wrap(err, "save user_data fail")
		}
		err = d.rootFs.Chmod(scriptPath, 0755, false)
		if err != nil {
			return errors.Wrap(err, "chmod user_data fail")
		}
	}
	{
		err := d.installInitScript(serviceName, "/bin/sh "+scriptPath, true)
		if err != nil {
			return errors.Wrap(err, "installInitScript")
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
