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

package mdadm

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

const (
	MDADM_BIN = "/sbin/mdadm"
)

func init() {
	raid.RegisterDriver(baremetal.DISK_DRIVER_LINUX, NewMdadmRaidLinux)
	raid.RegisterDriver(baremetal.DISK_DRIVER_PCIE, NewMdadmRaidPcie)
}

type MdadmRaid struct {
	term       raid.IExecTerm
	adapter    *MdadmRaidAdapter
	driverName string
}

func NewMdadmRaidLinux(term raid.IExecTerm) raid.IRaidDriver {
	return &MdadmRaid{
		term:       term,
		driverName: baremetal.DISK_DRIVER_LINUX,
	}
}

func NewMdadmRaidPcie(term raid.IExecTerm) raid.IRaidDriver {
	return &MdadmRaid{
		term:       term,
		driverName: baremetal.DISK_DRIVER_PCIE,
	}
}

func (r *MdadmRaid) GetName() string {
	return r.driverName
}

func (r *MdadmRaid) ParsePhyDevs() error {
	if r.adapter == nil {
		r.adapter = &MdadmRaidAdapter{
			raid:  r,
			term:  r.term,
			index: 0,
		}
	}
	return nil
}

func (r *MdadmRaid) SetDevicesForAdapter(adapterIdx int, devs []*baremetal.BaremetalStorage) {
	r.adapter.setDevices(devs)
	for i := range devs {
		devPath := path.Join("/dev", devs[i].Dev)
		cmd := fmt.Sprintf("%s --examine %s | grep UUID", MDADM_BIN, devPath)
		output, err := r.term.Run(cmd)
		if err == nil && len(output) > 0 {
			for _, line := range output {
				segs := strings.SplitN(strings.TrimSpace(line), ":", 2)
				if len(segs) == 2 {
					uuid := strings.TrimSpace(segs[1])
					cmd = fmt.Sprintf("%s --assemble --scan --uuid=%s", MDADM_BIN, uuid)
					output, err := r.term.Run(cmd)
					if err != nil {
						log.Errorf("faield assemble mdadm %s: %s", uuid, output)
					}
				}
			}
		}
	}
}

func (r *MdadmRaid) GetAdapters() []raid.IRaidAdapter {
	return []raid.IRaidAdapter{r.adapter}
}

func (r *MdadmRaid) PreBuildRaid(confs []*api.BaremetalDiskConfig, adapterIdx int) error {
	return nil
}

func deviceHasRaid(devPath string, term *ssh.Client) bool {
	cmd := fmt.Sprintf("%s --examine %s 2>/dev/null || true", MDADM_BIN, devPath)
	output, err := term.Run(cmd)
	if err != nil {
		log.Errorf("examine device %s: %s", devPath, err)
		return false
	}

	for _, line := range output {
		if strings.Contains(line, "mdadm") || strings.Contains(line, "ARRAY") {
			return true
		}
	}
	return false
}

func (r *MdadmRaid) CleanRaid() error {
	return nil
}

func CleanMdadmPartitions(term *ssh.Client) {
	out, err := term.Run("ls -1 /dev/md/")
	if err != nil {
		log.Errorf("failed get md devices %s, %s", out, err)
		return
	}
	// destory mdadm soft raid
	for _, line := range out {
		dev := strings.TrimSpace(line)
		if !strings.HasPrefix(dev, "md") {
			continue
		}
		out, err = term.Run(fmt.Sprintf("dd if=/dev/zero of=/dev/md/%s bs=512 count=34", dev))
		if err != nil {
			log.Errorf("failed clean mdadm partitions %s %s", out, err)
		}
		out, err = term.Run(fmt.Sprintf("dd if=/dev/zero of=/dev/md/%s bs=512 count=34 seek=$(( $(blockdev --getsz /dev/md/%s) - 34 ))", dev, dev))
		if err != nil {
			log.Errorf("failed clean mdadm partitions %s %s", out, err)
		}
		out, err = term.Run(fmt.Sprintf("hdparm -z /dev/md/%s", dev))
		if err != nil {
			log.Errorf("failed clean mdadm partitions %s %s", out, err)
		}
	}
}

func CleanRaid(term *ssh.Client) error {
	CleanMdadmPartitions(term)

	// stop md devices
	cmd := fmt.Sprintf("%s --stop --scan", MDADM_BIN)
	_, err := term.Run(cmd)
	if err != nil {
		log.Warningf("Stop md devices: %s", err)
	}

	pcieRet, err := term.Run("/lib/mos/lsdisk --pcie")
	if err != nil {
		log.Warningf("Fail to retrieve PCIE DISK info %s", err)
	} else {
		pcieDiskInfo := sysutils.ParsePCIEDiskInfo(pcieRet)
		for i := range pcieDiskInfo {
			devPath := path.Join("/dev", pcieDiskInfo[i].Dev)
			if deviceHasRaid(devPath, term) {
				cmd := fmt.Sprintf("%s --zero-superblock --force %s", MDADM_BIN, devPath)
				out, err := term.Run(cmd)
				if err != nil {
					return errors.Wrapf(err, "zero superblock on %s: %s", devPath, out)
				}
			}
		}
	}

	nonraidRet, err := term.Run("/lib/mos/lsdisk --nonraid")
	if err != nil {
		log.Warningf("Fail to retrieve SCSI DISK info %s", err)
	} else {
		nonraidDiskInfo := sysutils.ParseSCSIDiskInfo(nonraidRet)
		for i := range nonraidDiskInfo {
			devPath := path.Join("/dev", nonraidDiskInfo[i].Dev)
			if deviceHasRaid(devPath, term) {
				cmd := fmt.Sprintf("%s --zero-superblock --force %s", MDADM_BIN, devPath)
				out, err := term.Run(cmd)
				if err != nil {
					return errors.Wrapf(err, "zero superblock on %s: %s", devPath, out)
				}
			}
		}
	}

	out, err := term.Run("rm /dev/md/*")
	if err != nil {
		log.Warningf("failed soft link at /dev/md %s", out)
	}

	return nil
}

type MdadmRaidAdapter struct {
	raid  *MdadmRaid
	term  raid.IExecTerm
	index int
	devs  []*baremetal.BaremetalStorage
}

func (a *MdadmRaidAdapter) GetIndex() int {
	return a.index
}

func (a *MdadmRaidAdapter) PreBuildRaid(confs []*api.BaremetalDiskConfig) error {
	return nil
}

func (a *MdadmRaidAdapter) GetLogicVolumes() ([]*raid.RaidLogicalVolume, error) {
	lvs := make([]*raid.RaidLogicalVolume, 0)
	cmd := "ls -1 /dev/md/* 2>/dev/null || true"
	output, err := a.term.Run(cmd)
	if err != nil {
		return lvs, nil
	}

	for _, line := range output {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/dev/md/md") {
			mdPath := line
			numStr := strings.TrimPrefix(line, "/dev/md/md")
			if strings.HasSuffix(numStr, "_0") {
				numStr = strings.TrimSuffix(numStr, "_0")
			}
			if num, err := strconv.Atoi(numStr); err == nil {
				res, err := a.term.Run(fmt.Sprintf("readlink -f %s", line))
				if err == nil && len(res) > 0 {
					mdPath = strings.TrimSpace(res[0])
					lv := &raid.RaidLogicalVolume{
						Index:    num,
						Adapter:  a.index,
						BlockDev: mdPath,
					}
					lvs = append(lvs, lv)
				}
			}
		}
	}

	return lvs, nil
}

func (a *MdadmRaidAdapter) RemoveLogicVolumes() error {
	//cmd := fmt.Sprintf("%s --stop --scan", MDADM_BIN)
	//_, err := a.term.Run(cmd)
	//if err != nil {
	//	log.Warningf("Stop md devices: %v", err)
	//}
	return nil
}

func (a *MdadmRaidAdapter) GetDevices() []*baremetal.BaremetalStorage {
	return a.devs
}

func (a *MdadmRaidAdapter) setDevices(devs []*baremetal.BaremetalStorage) {
	a.devs = devs
}

func (a *MdadmRaidAdapter) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return a.buildRaid("0", devs, conf)
}

func (a *MdadmRaidAdapter) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return a.buildRaid("1", devs, conf)
}

func (a *MdadmRaidAdapter) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return a.buildRaid("5", devs, conf)
}

func (a *MdadmRaidAdapter) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return a.buildRaid("10", devs, conf)
}

func (a *MdadmRaidAdapter) BuildNoneRaid(devs []*baremetal.BaremetalStorage) error {
	return nil
}

func (a *MdadmRaidAdapter) PostBuildRaid() error {
	return nil
}

func (a *MdadmRaidAdapter) buildRaid(level string, devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	if len(devs) == 0 {
		return fmt.Errorf("no devices provided for RAID %s", level)
	}

	var mdNum int
	var err error
	if conf.SoftRaidIdx != nil {
		mdNum = *conf.SoftRaidIdx
	} else {
		mdNum, err = a.getNextMdNum()
		if err != nil {
			return errors.Wrap(err, "get next md number")
		}
	}

	devPaths := make([]string, 0, len(devs))
	for _, dev := range devs {
		if dev.Dev == "" {
			return fmt.Errorf("device path is empty for storage")
		}
		devPaths = append(devPaths, path.Join("/dev", dev.Dev))
	}

	for _, dev := range devPaths {
		if err := a.ensureDeviceClean(dev); err != nil {
			return errors.Wrapf(err, "clean device %s", dev)
		}
	}

	mdDev := fmt.Sprintf("/dev/md/md%d", mdNum)

	imsmDev := fmt.Sprintf("/dev/md/imsm%d", mdNum)
	cmdImsm := fmt.Sprintf("%s --create %s --metadata=imsm --raid-devices=%d --run --force %s", MDADM_BIN, imsmDev, len(devs), strings.Join(devPaths, " "))
	output, err := a.term.Run(cmdImsm)
	if err != nil {
		log.Errorf("mdadm create imsm raid %s failed, output: %v %s", level, output, err)
	} else {
		a.term.Run(fmt.Sprintf("%s --wait %s", MDADM_BIN, imsmDev))
		time.Sleep(time.Second * 3)
	}

	args := []string{
		"--create",
		mdDev,
		fmt.Sprintf("--level=%s", level),
		fmt.Sprintf("--raid-devices=%d", len(devs)),
		"--force",
		"--run",
	}

	for _, dev := range devPaths {
		args = append(args, dev)
	}

	args = append(args, "--assume-clean")

	cmd := fmt.Sprintf("%s %s", MDADM_BIN, strings.Join(args, " "))
	log.Infof("Building software RAID %s: %s", level, cmd)

	output, err = a.term.Run(cmd)
	if err != nil {
		return errors.Wrapf(err, "mdadm create raid %s failed, output: %v", level, output)
	}

	cmd = fmt.Sprintf("%s --wait %s", MDADM_BIN, mdDev)
	output, err = a.term.Run(cmd)
	if err != nil {
		log.Errorf("mdadm wait raid %s failed: %s", mdDev, output)
		//return errors.Wrapf(err, "mdadm wait raid %s failed, output: %v", mdDev, output)
	}

	log.Infof("Successfully created software RAID %s: /dev/md/md%d, start sync block devs", level, mdNum)

	for i := range devPaths {
		flushCmd := fmt.Sprintf("blockdev --flushbufs %s", devPaths[i])
		output, err = a.term.Run(flushCmd)
		if err != nil {
			return errors.Wrapf(err, "mdadm blockdev flushbufs %s failed, output: %v", devPaths[i], output)
		}
	}

	output, err = a.term.Run("sync")
	if err != nil {
		return errors.Wrapf(err, "mdadm %s sync failed, output: %v", mdDev, output)
	}

	return nil
}

func (a *MdadmRaidAdapter) getNextMdNum() (int, error) {
	cmd := "ls -1 /dev/md/ 2>/dev/null | grep -E '/dev/md/md[0-9]+$' || true"
	output, err := a.term.Run(cmd)
	if err != nil {
		return 0, errors.Wrap(err, "list md devices")
	}

	usedNums := make(map[int]bool)
	mdNumRe := regexp.MustCompile(`/dev/md/md(\d+)`)
	for _, line := range output {
		matches := mdNumRe.FindStringSubmatch(line)
		if len(matches) > 1 {
			if num, err := strconv.Atoi(matches[1]); err == nil {
				usedNums[num] = true
			}
		}
	}

	for i := 0; i < 256; i++ {
		if !usedNums[i] {
			return i, nil
		}
	}

	return 0, fmt.Errorf("no available md device number")
}

func (a *MdadmRaidAdapter) ensureDeviceClean(dev string) error {
	cmd := fmt.Sprintf("%s --examine %s 2>/dev/null || true", MDADM_BIN, dev)
	output, err := a.term.Run(cmd)
	if err != nil {
		return errors.Wrapf(err, "examine device %s", dev)
	}

	for _, line := range output {
		if strings.Contains(line, "mdadm") || strings.Contains(line, "ARRAY") {
			cmd := fmt.Sprintf("%s --zero-superblock --force %s", MDADM_BIN, dev)
			_, err := a.term.Run(cmd)
			if err != nil {
				return errors.Wrapf(err, "zero superblock on %s", dev)
			}
			break
		}
	}

	return nil
}
