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

package container_device

import (
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	hygonDrmVendorID     = "0x1d94"
	hygonDrmPrefixCard   = "card"
	hygonDrmPrefixRender = "renderD"
	hygonMaxVdevIdx      = 200
	hygonMaxPipePerDev   = 20
)

type hygonDrmSlice []string

func (s hygonDrmSlice) Len() int           { return len(s) }
func (s hygonDrmSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s hygonDrmSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// listHygonDrmDevices lists Hygon DCU card and render DRM device names, sorted.
func listHygonDrmDevices() (cards []string, renders []string, err error) {
	const driDir = "/dev/dri"
	if !hygonPathExists(driDir) {
		return nil, nil, nil
	}
	entries, err := hygonReadDir(driDir)
	if err != nil {
		return nil, nil, errors.Wrap(err, "read /dev/dri")
	}
	names := make([]string, 0)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, hygonDrmPrefixCard) || strings.HasPrefix(name, hygonDrmPrefixRender) {
			names = append(names, name)
		}
	}
	sort.Sort(hygonDrmSlice(names))
	for _, name := range names {
		vendorPath := fmt.Sprintf("/sys/class/drm/%s/device/vendor", name)
		vendorBytes, err := hygonReadFile(vendorPath)
		if err != nil {
			log.Warningf("read drm vendor %s: %v", vendorPath, err)
			continue
		}
		vendorID := strings.TrimSpace(string(vendorBytes))
		if vendorID != hygonDrmVendorID {
			continue
		}
		if strings.HasPrefix(name, hygonDrmPrefixCard) {
			cards = append(cards, name)
		}
		if strings.HasPrefix(name, hygonDrmPrefixRender) {
			renders = append(renders, name)
		}
	}
	if len(cards) != len(renders) {
		return cards, renders, errors.Errorf("hygon drm card count %d != render count %d", len(cards), len(renders))
	}
	return cards, renders, nil
}

func hygonCardPathForRender(renderPath string) string {
	renderBase := path.Base(renderPath)
	cards, renders, err := listHygonDrmDevices()
	if err != nil {
		log.Warningf("list hygon drm devices: %v", err)
		return ""
	}
	for i, render := range renders {
		if render == renderBase {
			return path.Join("/dev/dri", cards[i])
		}
	}
	return ""
}

func hygonPciBusIdFromAddr(pciAddr string) string {
	pciAddr = strings.TrimSpace(pciAddr)
	if pciAddr == "" {
		return ""
	}
	if strings.HasPrefix(pciAddr, "0000:") {
		return pciAddr
	}
	return "0000:" + pciAddr
}

func createHygonVdevConfFile(pciBusId, coremsk1, coremsk2 string, reqcores, memMiB int32, deviceID, vdevIdx, pipeID int, dir, fileName string) error {
	content := fmt.Sprintf("PciBusId: %s\n", pciBusId)
	content += fmt.Sprintf("cu_mask: 0x%s\n", coremsk1)
	content += fmt.Sprintf("cu_mask: 0x%s\n", coremsk2)
	content += fmt.Sprintf("cu_count: %d\n", reqcores)
	content += fmt.Sprintf("mem: %d MiB\n", memMiB)
	content += fmt.Sprintf("device_id: %d\n", deviceID)
	content += fmt.Sprintf("vdev_id: %d\n", vdevIdx)
	content += fmt.Sprintf("pipe_id: %d\n", pipeID)
	content += "enable: 1\n"

	if err := hygonMkdirAll(dir, 0o777); err != nil {
		return errors.Wrapf(err, "mkdir %s", dir)
	}
	filePath := path.Join(dir, fileName)
	if err := hygonWriteFile(filePath, []byte(content), 0o666); err != nil {
		return errors.Wrapf(err, "write vdev conf %s", filePath)
	}
	log.Infof("created hygon vdev conf: %s", filePath)
	return nil
}

func hygonVgpuCacheDirName(guestId, containerName string, devIdx, pipeID, vdevIdx int, coremsk1, coremsk2 string) string {
	return fmt.Sprintf("%s_%s_%d_%d_%d_%s_%s", guestId, containerName, devIdx, pipeID, vdevIdx, coremsk1, coremsk2)
}

func hygonReadFile(filePath string) ([]byte, error) {
	if hygonUseRemoteFS() {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("cat", filePath).Output()
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	return os.ReadFile(filePath)
}

func hygonMkdirAll(dir string, perm os.FileMode) error {
	if hygonUseRemoteFS() {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", dir).Output()
		if err != nil {
			return errors.Wrapf(err, "remote mkdir %s: %s", dir, out)
		}
		if perm != 0 {
			out, err = procutils.NewRemoteCommandAsFarAsPossible("chmod", fmt.Sprintf("%o", perm), dir).Output()
			if err != nil {
				return errors.Wrapf(err, "remote chmod %s: %s", dir, out)
			}
		}
		return nil
	}
	if err := os.MkdirAll(dir, perm); err != nil {
		return err
	}
	return os.Chmod(dir, perm)
}

func hygonWriteFile(filePath string, content []byte, perm os.FileMode) error {
	if hygonUseRemoteFS() {
		b64 := base64.StdEncoding.EncodeToString(content)
		script := fmt.Sprintf("mkdir -p $(dirname %q) && echo %q | base64 -d > %q && chmod %o %q",
			filePath, b64, filePath, perm, filePath)
		out, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", script).Output()
		if err != nil {
			return errors.Wrapf(err, "remote write %s: %s", filePath, out)
		}
		return nil
	}
	return os.WriteFile(filePath, content, perm)
}

func hygonRemoveAll(path string) error {
	if hygonUseRemoteFS() {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("rm", "-rf", path).Output()
		if err != nil {
			return errors.Wrapf(err, "remote rm -rf %s: %s", path, out)
		}
		return nil
	}
	return os.RemoveAll(path)
}

func hygonRemove(filePath string) error {
	if hygonUseRemoteFS() {
		out, err := procutils.NewRemoteCommandAsFarAsPossible("rm", "-f", filePath).Output()
		if err != nil {
			return errors.Wrapf(err, "remote rm %s: %s", filePath, out)
		}
		return nil
	}
	return os.Remove(filePath)
}
