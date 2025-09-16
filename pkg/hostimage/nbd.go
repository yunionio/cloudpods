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

package hostimage

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

var EXPORT_NBD_BASE_PORT = 7777
var LAST_USED_NBD_SERVER_PORT = 0

type SNbdExportManager struct {
	portsLock *sync.Mutex
}

func NewNbdExportManager() *SNbdExportManager {
	return &SNbdExportManager{
		portsLock: new(sync.Mutex),
	}
}

func (m *SNbdExportManager) GetFreePortByBase(basePort int) int {
	var port = 1
	for {
		if netutils2.IsTcpPortUsed("0.0.0.0", basePort+port) {
			port += 1
		} else {
			break
		}
	}
	return port + basePort
}

func (m *SNbdExportManager) GetNBDServerFreePort() int {
	basePort := EXPORT_NBD_BASE_PORT + LAST_USED_NBD_SERVER_PORT
	var port = 1
	for {
		if netutils2.IsTcpPortUsed("0.0.0.0", basePort+port) {
			port += 1
		} else {
			break
		}
	}
	LAST_USED_NBD_SERVER_PORT = port
	if LAST_USED_NBD_SERVER_PORT > 1000 {
		LAST_USED_NBD_SERVER_PORT = 0
	}
	return port + basePort
}

func (m *SNbdExportManager) getQemuNbdVersion() (string, error) {
	output, err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuNbd(), "--version").Output()
	if err != nil {
		log.Errorf("qemu-nbd version failed %s %s", output, err.Error())
		return "", errors.Wrapf(err, "qemu-nbd version failed %s", output)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		parts := strings.Split(lines[0], " ")
		return parts[1], nil
	}
	return "", errors.Error("empty version output")
}

func (m *SNbdExportManager) QemuNbdStartExport(imageInfo qemuimg.SImageInfo, diskId string) (int, error) {
	m.portsLock.Lock()
	defer m.portsLock.Unlock()

	nbdPort := m.GetNBDServerFreePort()
	pidFilePath := path.Join(HostImageOptions.HostImageNbdPidDir, fmt.Sprintf("nbd_%s.pid", diskId))

	nbdVer, err := m.getQemuNbdVersion()
	if err != nil {
		return -1, errors.Wrap(err, "getQemuNbdVersion")
	}
	var cmd []string
	if imageInfo.Encrypted() {
		cmd = []string{
			qemutils.GetQemuNbd(),
			"--read-only", "--persistent", "-x", diskId, "-p", strconv.Itoa(nbdPort),
			"--object", imageInfo.SecretOptions(),
			"--image-opts", imageInfo.ImageOptions(),
		}
	} else {
		cmd = []string{
			qemutils.GetQemuNbd(),
			"--read-only", "--persistent", "-x", diskId, "-p", strconv.Itoa(nbdPort),
			imageInfo.Path,
		}
	}
	cmd = append(cmd, "--pid-file", pidFilePath)
	if version.GE(nbdVer, "4.0.0") {
		cmd = append(cmd, "--fork")
	}
	cmdStr := strings.Join(cmd, " ")
	err = procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmdStr).Run()
	if err != nil {
		log.Errorf("qemu-nbd connect failed %s %s", err.Error())
		return -1, errors.Wrapf(err, "qemu-nbd connect failed")
	}
	return nbdPort, nil
}

func (m *SNbdExportManager) QemuNbdCloseExport(diskId string) error {
	pidFilePath := path.Join(HostImageOptions.HostImageNbdPidDir, fmt.Sprintf("nbd_%s.pid", diskId))
	if !m.nbdProcessExist(diskId) {
		if fileutils2.Exists(pidFilePath) {
			if err := os.Remove(pidFilePath); err != nil {
				log.Errorf("failed remove nbd pid file %s", pidFilePath)
			}
		}
		return nil
	}
	if fileutils2.Exists(pidFilePath) {
		pid, err := fileutils2.FileGetIntContent(pidFilePath)
		if err != nil {
			return errors.Wrapf(err, "failed get pid of qemu-nbd process %s", pidFilePath)
		}
		out, err := procutils.NewRemoteCommandAsFarAsPossible("kill", "-9", strconv.Itoa(pid)).Output()
		if err != nil {
			log.Errorf("failed kill nbd export process %s %s", err, out)
			return errors.Wrapf(err, "kill nbd export failed: %s", out)
		}

		if err := os.Remove(pidFilePath); err != nil {
			log.Errorf("failed remove nbd pid file %s", pidFilePath)
		}
	}
	return nil
}

func (m *SNbdExportManager) nbdProcessExist(diskId string) bool {
	return procutils.NewRemoteCommandAsFarAsPossible("sh", "-c",
		fmt.Sprintf("ps -ef | grep [q]emu-nbd | grep %s", diskId)).Run() == nil
}
