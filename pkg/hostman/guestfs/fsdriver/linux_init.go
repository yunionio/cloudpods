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

	"yunion.io/x/pkg/errors"
)

const (
	unitDirPath   = "/usr/lib/systemd/system"
	enableDirPath = "/etc/systemd/system/multi-user.target.wants"
)

func (d *sLinuxRootFs) isSupportSystemd() bool {
	return d.rootFs.Exists(unitDirPath, false) && d.rootFs.Exists(enableDirPath, false)
}

func (d *sLinuxRootFs) installInitScript(name, cmd string, oneshot bool) error {
	if d.isSupportSystemd() {
		return d.installSystemd(name, cmd, oneshot)
	} else {
		return d.installCrond(cmd)
	}
}

func (d *sLinuxRootFs) installCrond(cmd string) error {
	cronJob := fmt.Sprintf("@reboot %s", cmd)
	outs, err := d.rootFs.ExecCommand("crontab", "-l")
	if err != nil {
		return errors.Wrapf(err, "exec crontab -l: %v", outs)
	}
	for _, line := range outs {
		if strings.Contains(line, cronJob) {
			// if cronjob exist, return success
			return nil
		}
	}
	outs, err = d.rootFs.ExecCommand("sh", "-c", fmt.Sprintf("(crontab -l 2>/dev/null; echo '%s') |crontab -", cronJob))
	if err != nil {
		return errors.Wrapf(err, "add crontab %v", outs)
	}
	return nil
}

func (d *sLinuxRootFs) installSystemd(name, cmd string, oneshot bool) error {
	var serviceName = fmt.Sprintf("%s.service", name)
	var unitPath = fmt.Sprintf("%s/%s", unitDirPath, serviceName)
	var enablePath = fmt.Sprintf("%s/%s", enableDirPath, serviceName)
	var serviceType = "simple"
	if oneshot {
		serviceType = "oneshot"
	}

	var unitContent = fmt.Sprintf(`[Unit]
Description=Run once
After=local-fs.target
After=network.target

[Service]
LimitNOFILE=65535
CapabilityBoundingSet=CAP_NET_RAW
AmbientCapabilities=CAP_NET_RAW
ExecStart=%s
RemainAfterExit=true
Type=%s

[Install]
WantedBy=multi-user.target
`, cmd, serviceType)
	err := d.rootFs.FilePutContents(unitPath, unitContent, false, false)
	if err != nil {
		return errors.Wrap(err, "save user_data unit fail")
	}

	err = d.rootFs.Symlink(unitPath, enablePath, false)
	if err != nil {
		return errors.Wrap(err, "create user_data symlink fail")
	}

	return nil
}

func (d *sLinuxRootFs) InstallQemuGuestAgentSystemd() error {
	var serviceName = "qemu-guest-agent.service"
	var unitPath = fmt.Sprintf("%s/%s", unitDirPath, serviceName)
	var unitContent = fmt.Sprintf(`[Unit]
Description=QEMU Guest Agent
BindsTo=dev-virtio\x2dports-org.qemu.guest_agent.0.device
After=dev-virtio\x2dports-org.qemu.guest_agent.0.device

[Service]
ExecStart=/usr/bin/qemu-ga
Restart=always
RestartSec=0

[Install]
`)
	err := d.rootFs.FilePutContents(unitPath, unitContent, false, false)
	if err != nil {
		return errors.Wrap(err, "save qemu-guest-agent.service unit fail")
	}

	return nil
}
