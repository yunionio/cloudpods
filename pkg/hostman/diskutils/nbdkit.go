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

package diskutils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type NbdkitDisk struct {
	Host     string
	Port     int
	User     string
	Passwd   string
	DiskPath string

	Proc   *Command
	NbdURI string

	kvmDisk      *SKVMGuestDisk
	readOnly     bool
	deployDriver string
}

func NewNbdkitDisk(vddkInfo *apis.VDDKConInfo, diskPath, deployDriver string, readOnly bool) (*NbdkitDisk, error) {
	return &NbdkitDisk{
		Host:         vddkInfo.Host,
		Port:         int(vddkInfo.Port),
		User:         vddkInfo.User,
		Passwd:       vddkInfo.Passwd,
		DiskPath:     diskPath,
		deployDriver: deployDriver,
		readOnly:     readOnly,
	}, nil
}

func (vd *NbdkitDisk) Cleanup() {
	if vd.kvmDisk != nil {
		vd.kvmDisk.Cleanup()
		vd.kvmDisk = nil
	}
}

func (vd *NbdkitDisk) Connect(*apis.GuestDesc) error {
	flatFile, err := vd.ConnectBlockDevice()
	if err != nil {
		return errors.Wrap(err, "ConnectBlockDevice")
	}
	vd.kvmDisk, err = NewKVMGuestDisk(qemuimg.SImageInfo{Path: flatFile}, vd.deployDriver, vd.readOnly)
	if err != nil {
		vd.DisconnectBlockDevice()
		return errors.Wrap(err, "NewKVMGuestDisk")
	}
	err = vd.kvmDisk.Connect(nil)
	if err != nil {
		vd.DisconnectBlockDevice()
		return errors.Wrap(err, "kvmDisk connect")
	}
	return nil
}

func (vd *NbdkitDisk) ConnectWithDiskId(desc *apis.GuestDesc, diskId string) error {
	return vd.Connect(desc)
}

func (vd *NbdkitDisk) Disconnect() error {
	if vd.kvmDisk != nil {
		if err := vd.kvmDisk.Disconnect(); err != nil {
			log.Errorf("kvm disk disconnect failed %s", err)
		}
		vd.kvmDisk.Cleanup()
		vd.kvmDisk = nil
	}
	return vd.DisconnectBlockDevice()
}

func (vd *NbdkitDisk) MountRootfs() (fsdriver.IRootFsDriver, error) {
	if vd.kvmDisk == nil {
		return nil, fmt.Errorf("kvmDisk is nil")
	}
	return vd.kvmDisk.MountRootfs()
}

func (vd *NbdkitDisk) UmountRootfs(fd fsdriver.IRootFsDriver) error {
	if vd.kvmDisk == nil {
		return nil
	}
	return vd.kvmDisk.UmountRootfs(fd)
}

func (vd *NbdkitDisk) Mount() (err error) {
	_, err = vd.ConnectBlockDevice()
	return err
}

func (vd *NbdkitDisk) Umount() error {
	if vd.Proc != nil {
		if !vd.Proc.Exited() {
			if err := vd.Proc.Kill(); err != nil {
				log.Errorf("kill nbdkit process failed: %s", err)
			}
		}
		if err := vd.Proc.Wait(); err != nil {
			log.Warningf("wait nbdkit process failed: %s", err)
		}
		vd.Proc = nil
	}
	vd.NbdURI = ""
	return nil
}

func (vd *NbdkitDisk) ExecProg() error {
	args := []string{
		"--foreground",
		"--exit-with-parent",
		"-v",
		"ssh",
		"verify-remote-host=false",
		fmt.Sprintf("host=%s", vd.Host),
		fmt.Sprintf("user=%s", vd.User),
		fmt.Sprintf("path=%s", vd.DiskPath),
	}
	if vd.readOnly {
		args = append([]string{"-r"}, args...)
	}
	if vd.Port > 0 {
		args = append(args, fmt.Sprintf("port=%d", vd.Port))
	}
	if len(vd.Passwd) > 0 {
		args = append(args, fmt.Sprintf("password=%s", vd.Passwd))
	}
	cmd := NewCommand("nbdkit", args...)
	log.Debugf("command to mount: %s", cmd)
	vd.Proc = cmd
	vd.NbdURI = ""
	err := vd.Proc.Start()
	if err != nil {
		return errors.Wrap(err, "vd.Proc.Start")
	}
	return nil
}

func (vd *NbdkitDisk) WaitMounted() error {
	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.Errorf("wait nbdkit ready timeout: %s", vd.Proc.stdouterr.String())
		default:
			if vd.Proc != nil {
				output := vd.Proc.stdouterr.String()
				if nbdURI, ok := parseNbdkitOutputNbdURI(output); ok {
					vd.NbdURI = nbdURI
					return nil
				}
			}
			if vd.Proc != nil && vd.Proc.Exited() {
				backup := vd.Proc.stdouterr.String()
				return errors.Errorf("nbdkit exited unexpectedly: %s", backup)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

var (
	nbdkitURIRegex      = regexp.MustCompile(`nbd://(?:127\.0\.0\.1|localhost):(\d{2,5})`)
	nbdkitPortRegexList = []*regexp.Regexp{
		regexp.MustCompile(`(?:127\.0\.0\.1|localhost):(\d{2,5})`),
		regexp.MustCompile(`\bbound\s+to\s+IP\s+address\s+[^\s:]+:(\d{2,5})\b`),
		regexp.MustCompile(`\bport(?:\s+number)?\s*[=:]?\s*(\d{2,5})\b`),
	}
)

// nbdkit: debug: bound to IP address <any>:10809 (2 socket(s))
func parseNbdkitOutputNbdURI(output string) (string, bool) {
	if matched := nbdkitURIRegex.FindStringSubmatch(output); len(matched) == 2 {
		port, err := strconv.Atoi(matched[1])
		if err == nil && port > 0 {
			return fmt.Sprintf("nbd://127.0.0.1:%d/", port), true
		}
	}
	for _, regex := range nbdkitPortRegexList {
		if matched := regex.FindStringSubmatch(output); len(matched) == 2 {
			port, err := strconv.Atoi(matched[1])
			if err == nil && port > 0 {
				return fmt.Sprintf("nbd://127.0.0.1:%d/", port), true
			}
		}
	}
	if idx := strings.Index(output, "nbd://"); idx >= 0 {
		candidate := output[idx:]
		if end := strings.IndexAny(candidate, " \n\r\t"); end >= 0 {
			candidate = candidate[:end]
		}
		if strings.HasPrefix(candidate, "nbd://") && strings.Contains(candidate, ":") {
			parts := strings.Split(candidate, ":")
			port, err := strconv.Atoi(parts[len(parts)-1])
			if err == nil && port > 0 {
				return fmt.Sprintf("nbd://127.0.0.1:%d/", port), true
			}
		}
	}
	return "", false
}

// ConnectBlockDevice starts nbdkit ssh plugin and returns a local NBD URI.
func (vd *NbdkitDisk) ConnectBlockDevice() (string, error) {
	if vd.Proc != nil && !vd.Proc.Exited() && len(vd.NbdURI) > 0 {
		return vd.NbdURI, nil
	}
	if err := vd.ExecProg(); err != nil {
		return "", errors.Wrap(err, "ExecProg")
	}
	if err := vd.WaitMounted(); err != nil {
		return "", errors.Wrap(err, "WaitMounted")
	}
	log.Infof("disk mounted by nbdkit over %s", vd.NbdURI)
	return vd.NbdURI, nil
}

func (vd *NbdkitDisk) DisconnectBlockDevice() error {
	if vd.Proc != nil {
		return vd.Umount()
	}
	return fmt.Errorf("nbdkit disk has not connected")
}

func (vd *NbdkitDisk) DeployGuestfs(req *apis.DeployParams) (res *apis.DeployGuestFsResponse, err error) {
	return vd.kvmDisk.DeployGuestfs(req)
}

func (d *NbdkitDisk) ResizeFs(req *apis.ResizeFsParams) (*apis.Empty, error) {
	return d.kvmDisk.ResizeFs(req)
}

func (d *NbdkitDisk) FormatFs(req *apis.FormatFsParams) (*apis.Empty, error) {
	return d.kvmDisk.FormatFs(req)
}

func (d *NbdkitDisk) SaveToGlance(req *apis.SaveToGlanceParams) (*apis.SaveToGlanceResponse, error) {
	return d.kvmDisk.SaveToGlance(req)
}

func (d *NbdkitDisk) ProbeImageInfo(req *apis.ProbeImageInfoPramas) (*apis.ImageInfo, error) {
	return d.kvmDisk.ProbeImageInfo(req)
}
