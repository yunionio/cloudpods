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

package disktool

import (
	"fmt"
	"math"
	"strings"

	"github.com/pkg/errors"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	raiddrivers "yunion.io/x/onecloud/pkg/baremetal/utils/raid/drivers"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	fileutils "yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

const (
	// MB_SECTORS = 2048 // 1MiB = 2014 sectors
	GPT_SECTORS = 34

	RAID_DRVIER    = "raid"
	NONRAID_DRIVER = "nonraid"
	PCIE_DRIVER    = "pcie"

	LABEL_MSDOS = "msdos"
	LABEL_GPT   = "gpt"
)

type Partition struct {
	disk     *DiskPartitions
	index    int
	bootable bool
	start    int64
	end      int64
	count    int64
	diskType string
	fs       string
	dev      string
}

func NewPartition(
	disk *DiskPartitions,
	index int, bootable bool,
	start int64, end int64, count int64,
	diskType string, fs string, dev string,
) *Partition {
	return &Partition{
		disk:     disk,
		index:    index,
		bootable: bootable,
		start:    start,
		end:      end,
		count:    count,
		diskType: diskType,
		fs:       fs,
		dev:      dev,
	}
}

func (p *Partition) GetStart() int64 {
	return p.start
}

func (p *Partition) GetEnd() int64 {
	return p.end
}

func (p *Partition) GetDev() string {
	return p.dev
}

func (p *Partition) String() string {
	bootStr := ""
	if p.bootable {
		bootStr = " boot"
	}
	return fmt.Sprintf("%s %d %d %s %s%s", p.dev, p.start, p.end, p.diskType, p.fs, bootStr)
}

func (p *Partition) Format(fs string, uuid string) error {
	cmd := []string{}
	cmdUUID := []string{}
	switch fs {
	case "swap":
		cmd = []string{"/sbin/mkswap", "-U", uuid}
	case "ext2":
		cmd = []string{"/usr/sbin/mkfs.ext2"}
		cmdUUID = []string{"/usr/sbin/tune2fs", "-U", uuid}
	case "ext3":
		cmd = []string{"/usr/sbin/mkfs.ext3"}
		cmdUUID = []string{"/usr/sbin/tune2fs", "-U", uuid}
	case "ext4":
		// for baremetal, force 64bit support large disks
		//cmd = []string{"/usr/sbin/mkfs.ext4", "-O", "64bit", "-E", "lazy_itable_init=1", "-T", "largefile"}
		cmd = []string{"/usr/sbin/mkfs.ext4", "-O", "64bit", "-E", "lazy_itable_init=1"}
		cmdUUID = []string{"/usr/sbin/tune2fs", "-U", uuid}
	case "ext4dev":
		cmd = []string{"/usr/sbin/mkfs.ext4dev", "-E", "lazy_itable_init=1"}
		cmdUUID = []string{"/usr/sbin/tune2fs", "-U", uuid}
	case "xfs":
		cmd = []string{"/sbin/mkfs.xfs", "-f", "-m", "crc=0", "-i", "projid32bit=0", "-n", "ftype=0"}
		cmdUUID = []string{"PATH=/bin:/sbin:/usr/bin:/usr/sbin /usr/sbin/xfs_admin", "-U", uuid}
	default:
		return fmt.Errorf("Unsupported filesystem %s", fs)
	}
	cmd = append(cmd, p.dev)
	cmds := []string{strings.Join(cmd, " ")}
	if len(cmdUUID) != 0 {
		cmdUUID = append(cmdUUID, p.dev)
		cmds = append(cmds, strings.Join(cmdUUID, " "))
	}
	_, err := p.Run(cmds...)
	return err
}

func (p *Partition) Fsck() error {
	if p.fs == "" {
		return fmt.Errorf("filesystem is empty")
	}
	cmd := []string{}
	if strings.HasPrefix(p.fs, "ext") {
		cmd = []string{fmt.Sprintf("/usr/sbin/fsck.%s", p.fs), "-f", "-p"}
	} else if p.fs == "xfs" {
		cmd = []string{"/sbin/fsck.xfs"}
	} else {
		return fmt.Errorf("Unsupported fsck filesystem: %s", p.fs)
	}
	cmd = append(cmd, p.dev)
	_, err := p.Run(strings.Join(cmd, " "))
	return err
}

func (p *Partition) Run(cmds ...string) ([]string, error) {
	return p.disk.Run(cmds...)
}

func (p *Partition) ResizeFs() error {
	if p.fs == "" {
		return nil
	}
	cmd := []string{}
	if strings.HasPrefix(p.fs, "linux-swap") {
		cmd = []string{"/sbin/mkswap", p.dev}
	} else if strings.HasPrefix(p.fs, "ext") {
		if err := p.Fsck(); err != nil {
			log.Warningf("FSCK error: %v", err)
		}
		cmd = []string{"/usr/sbin/resize2fs", p.dev}
	} else if p.fs == "xfs" {
		return p.ResizeXfs()
	}
	if len(cmd) == 0 {
		return nil
	}
	_, err := p.Run(strings.Join(cmd, " "))
	return err
}

func (p *Partition) ResizeXfs() error {
	mountPath := fmt.Sprintf("/tmp/%s", strings.Replace(p.dev, "/", "", -1))
	cmds := []string{
		fmt.Sprintf("mkdir -p %s", mountPath),
		fmt.Sprintf("mount -t xfs %s %s", p.dev, mountPath),
		fmt.Sprintf("/usr/sbin/xfs_growfs -d %s", mountPath),
		fmt.Sprintf("umount %s", mountPath),
		fmt.Sprintf("rm -fr %s", mountPath),
	}
	_, err := p.Run(cmds...)
	if err != nil {
		log.Errorf("Resize xfs error: %v", err)
		cmds = []string{
			fmt.Sprintf("umount %s", mountPath),
			fmt.Sprintf("rm -rf %s", mountPath),
		}
		_, err = p.Run(cmds...)
		if err != nil {
			log.Errorf("Umount error: %v", err)
			return err
		}
	}
	return nil
}

func (p *Partition) GetSizeMB() (int64, error) {
	log.Infof("GetSizeMB: start %d, end: %d, count: %d", p.start, p.end, p.count)
	if p.count != (p.end - p.start + 1) {
		return 0, fmt.Errorf("Count(%d) != End(%d)-Start(%d)+1", p.count, p.end, p.start)
	}
	return p.count * 512 / 1024 / 1024, nil
}

type DiskPartitions struct {
	driver     string
	adapter    int
	raidConfig string
	sizeMB     int64 // MB
	tool       *PartitionTool
	dev        string
	devName    string
	sectors    int64
	blockSize  int64
	rotate     bool
	desc       string
	label      string
	partitions []*Partition
}

func newDiskPartitions(driver string, adapter int, raidConfig string, sizeMB int64, blockSize int64, tool *PartitionTool) *DiskPartitions {
	ps := new(DiskPartitions)
	ps.driver = driver
	ps.adapter = adapter
	ps.raidConfig = raidConfig
	ps.sizeMB = sizeMB
	ps.tool = tool
	ps.blockSize = blockSize
	ps.partitions = make([]*Partition, 0)
	return ps
}

func (p *DiskPartitions) IsRaidDriver() bool {
	return utils.IsInStringArray(p.driver, []string{
		baremetal.DISK_DRIVER_MEGARAID,
		baremetal.DISK_DRIVER_HPSARAID,
		baremetal.DISK_DRIVER_MARVELRAID,
		baremetal.DISK_DRIVER_MPT2SAS,
	})
}

func (p *DiskPartitions) SetInfo(info *types.SDiskInfo) *DiskPartitions {
	p.dev = fmt.Sprintf("/dev/%s", info.Dev)
	p.devName = info.Dev
	p.sectors = info.Sector
	p.desc = info.ModuleInfo
	p.blockSize = info.Block
	if p.blockSize == 4096 {
		p.sectors = (p.sectors >> 3)
	}
	return p
}

func (p *DiskPartitions) ReInitInfo() error {
	cmd := "/lib/mos/lsdisk"
	lines, err := p.tool.Run(cmd)
	if err != nil {
		return errors.Wrapf(err, "Disk %#v reset info list disk", p)
	}
	for _, disk := range sysutils.ParseDiskInfo(lines, p.driver) {
		if disk.Dev == p.GetDevName() {
			p.SetInfo(disk)
		}
	}
	return p.RetrievePartitionInfo()
}

func (ps *DiskPartitions) MBSectors() int64 {
	return int64(1024 * 1024 / ps.blockSize)
}

func (ps *DiskPartitions) String() string {
	return fmt.Sprintf("%s %d %s", ps.devName, ps.sizeMB, ps.driver)
}

func (ps *DiskPartitions) DebugString() string {
	partitionsStr := []string{}
	for _, p := range ps.partitions {
		partitionsStr = append(partitionsStr, fmt.Sprintf("%#v", *p))
	}
	return fmt.Sprintf("driver: %s, dev: %s, sectors: %d, partitions: %#v", ps.driver, ps.dev, ps.sectors, partitionsStr)
}

func (ps *DiskPartitions) IsReady() bool {
	if ps.dev == "" {
		return false
	}
	return true
}

func (ps *DiskPartitions) GetDevName() string {
	devName := ps.devName
	if !ps.IsRaidDriver() || ps.raidConfig == baremetal.DISK_CONF_NONE {
		return devName
	}
	raidDrv, err := raiddrivers.GetDriverWithInit(ps.driver, ps.tool.runner.Term())
	if err != nil {
		log.Errorf("Failed to find %s raid driver: %v", ps.driver, err)
		return devName
	}
	// find first raid adapter logical volume
	lv, err := raiddrivers.GetFirstLogicalVolume(raidDrv, ps.adapter)
	if err != nil {
		log.Errorf("Failed to find raid %s adapter %d first logical volume: %v", raidDrv.GetName(), ps.adapter, err)
		return devName
	}
	if len(lv.BlockDev) == 0 {
		log.Warningf("Raid %s adapter %d first logical volume block device is empty", raidDrv.GetName(), ps.adapter)
		return devName
	}
	devName = strings.TrimLeft(lv.BlockDev, "/dev/")
	return devName
}

func (ps *DiskPartitions) RetrievePartitionInfo() error {
	ps.partitions = make([]*Partition, 0)
	cmd := []string{"/usr/sbin/parted", "-s", ps.dev, "--", "unit", "s", "print"}
	ret, err := ps.Run(strings.Join(cmd, " "))
	if err != nil {
		return err
	}
	parts, label := fileutils.ParseDiskPartitions(ps.dev, ret)
	ps.label = label
	for _, part := range parts {
		ps.addPartition(part)
	}
	return nil
}

func (ps *DiskPartitions) addPartition(p fileutils.Partition) {
	part := NewPartition(ps, p.Index, p.Bootable, p.Start, p.End, p.Count, p.DiskType, p.Fs, p.DevName)
	ps.partitions = append(ps.partitions, part)
}

func (ps *DiskPartitions) GPTEndSector() int64 {
	return ps.sectors - int64(GPT_SECTORS)
}

func (ps *DiskPartitions) FsToTypeCode(fs string) string {
	if strings.Contains(fs, "swap") {
		return "8200"
	} else if strings.HasPrefix(fs, "ntfs") || strings.HasPrefix(fs, "fat") {
		return "0700"
	}
	return "8300"
}

func (ps *DiskPartitions) doResize(dev string, cmd string) error {
	cmds := []string{}
	cmds = append(cmds, cmd)
	cmds = append(cmds, fmt.Sprintf("/sbin/hdparm -f %s", dev))
	cmds = append(cmds, fmt.Sprintf("/sbin/hdparm -z %s", dev))
	_, err := ps.tool.Run(cmds...)
	if err != nil {
		return err
	}
	return ps.RetrievePartitionInfo()
}

func (ps *DiskPartitions) GetPartitions() []*Partition {
	return ps.partitions
}

func (ps *DiskPartitions) ResizePartition(offsetMB int64) error {
	if len(ps.partitions) == 0 {
		return fmt.Errorf("ResizePartitions error: total %d partitions", len(ps.partitions))
	}
	var cmd string
	if ps.label == LABEL_MSDOS {
		part := ps.partitions[len(ps.partitions)-1]
		if part.diskType == "extended" {
			log.Infof("Find last partition an empty extended partition, removed it")
			cmd := fmt.Sprintf("/usr/sbin/parted -a none -s %s -- rm %d", part.disk.dev, part.index)
			if err := ps.doResize(part.disk.dev, cmd); err != nil {
				return fmt.Errorf("Fail to remove empty extended partition: %v", err)
			}
		}
	}
	part := ps.partitions[len(ps.partitions)-1]
	var end int64
	if offsetMB <= 0 {
		end = ps.GPTEndSector()
	} else {
		end = offsetMB*ps.MBSectors() - 1
		if end > ps.GPTEndSector() {
			end = ps.GPTEndSector()
		}
	}
	if end < part.end {
		log.Warningf("Cannot reduce size %d %d, no need to resize", end, part.end)
		end = part.end
	}
	if ps.label == LABEL_MSDOS {
		if part.diskType == "logical" {
			extendIdx := -1
			for i := range ps.partitions {
				if ps.partitions[i].diskType == "extended" {
					log.Infof("Find extended at %d", i)
					extendIdx = i
					break
				}
			}
			if extendIdx < 0 {
				return fmt.Errorf("To resize logical parition, but fail to find extend partiton")
			}
			cmd = fmt.Sprintf("/usr/sbin/parted -a none -s %s -- unit s", part.disk.dev)
			partsLen := len(ps.partitions)
			for i := partsLen - 1; i > extendIdx-1; i-- {
				cmd = fmt.Sprintf("%s rm %d", cmd, ps.partitions[i].index)
			}
			for i := extendIdx; i < partsLen; i++ {
				cmdLet := fmt.Sprintf("mkpart %s %d", ps.partitions[i].diskType, ps.partitions[i].start)
				if i == extendIdx {
					cmdLet = fmt.Sprintf("%s %d", cmdLet, ps.GPTEndSector())
				} else if i == (len(ps.partitions) - 1) {
					cmdLet = fmt.Sprintf("%s %d", cmdLet, end)
				} else {
					cmdLet = fmt.Sprintf("%s %d", cmdLet, ps.partitions[i].end)
				}
				cmd = fmt.Sprintf("%s %s", cmd, cmdLet)
			}
		} else {
			cmd = fmt.Sprintf("/usr/sbin/parted -a none -s %s -- unit s rm %d mkpart %s", part.disk.dev, part.index, part.diskType)
			cmd = fmt.Sprintf("%s %d %d", cmd, part.start, end)
			if part.bootable {
				cmd = fmt.Sprintf("%s set %d boot on", cmd, part.index)
			}
		}
	} else {
		// gpt
		cmd = fmt.Sprintf("/usr/sbin/sgdisk --set-alignment=1 --delete=%d", part.index)
		cmd = fmt.Sprintf("%s --new=%d:%d:%d", cmd, part.index, part.start, end)
		if len(part.diskType) != 0 {
			cmd = fmt.Sprintf("%s --change-name=%d:\"%s\"", cmd, part.index, part.diskType)
		}
		if len(part.fs) != 0 {
			cmd = fmt.Sprintf("%s --typecode=%d:%s", cmd, part.index, ps.FsToTypeCode(part.fs))
		}
		cmd = fmt.Sprintf("%s %s", cmd, ps.dev)
	}
	log.Infof("Resize cmd: %s", cmd)
	if err := ps.doResize(part.disk.dev, cmd); err != nil {
		return err
	}
	return ps.partitions[len(ps.partitions)-1].ResizeFs()
}

func (ps *DiskPartitions) IsSpaceAvailable(sizeMB int64) bool {
	start := ps.getNextPartStart()
	freeSect := ps.GPTEndSector() - start
	if sizeMB <= 0 {
		sizeMB = 1
	}
	reqSect := sizeMB * ps.MBSectors()
	if reqSect > freeSect {
		log.Warningf("No space require %d(%d) left %d", reqSect, sizeMB, freeSect)
		return false
	}
	return true
}

func (ps *DiskPartitions) MakeLabel() error {
	label := LABEL_GPT
	if ps.sizeMB <= 1024*1024*2 {
		label = LABEL_MSDOS
	}
	return ps.makeLabel(label)
}

func (ps *DiskPartitions) makeLabel(label string) error {
	ps.label = label
	cmd := fmt.Sprintf("/usr/sbin/parted -s %s -- mklabel %s", ps.dev, ps.label)
	// cmd = ['/usr/sbin/sgdisk', '-og', self.dev]
	_, err := ps.Run(cmd)
	return err
}

func (ps *DiskPartitions) getNextPartIndex() int {
	max := 0
	for _, part := range ps.partitions {
		if max < part.index {
			max = part.index
		}
	}
	return max + 1
}

func (ps *DiskPartitions) getNextPartStart() int64 {
	var start int64
	if len(ps.partitions) == 0 {
		start = ps.MBSectors() // 1MB
	} else {
		var gap int64 = 2
		lastPart := ps.partitions[len(ps.partitions)-1]
		start = ((lastPart.end + gap) / ps.MBSectors()) * ps.MBSectors()
		if start < lastPart.end+gap {
			start += ps.MBSectors()
		}
	}
	return start
}

func (ps *DiskPartitions) Run(cmd ...string) ([]string, error) {
	return ps.tool.Run(cmd...)
}

func (ps *DiskPartitions) CreatePartition(sizeMB int64, fs string, doformat bool, uuid string) error {
	if len(ps.partitions) == 0 {
		if err := ps.MakeLabel(); err != nil {
			return err
		}
	}
	start := ps.getNextPartStart()
	var end int64
	if sizeMB <= 0 {
		end = start + (ps.GPTEndSector()-start)/ps.MBSectors()*ps.MBSectors() - 1
	} else {
		end = start + sizeMB*ps.MBSectors() - 1
	}
	partIdx := ps.getNextPartIndex()
	var cmd string
	var diskType string
	if ps.label == LABEL_MSDOS {
		if partIdx < 5 {
			diskType = "primary"
		} else if partIdx < 9 {
			diskType = "logical"
		} else {
			return fmt.Errorf("Too many partitions on a MSDOS disk")
		}
		cmd = fmt.Sprintf("/usr/sbin/parted -a none -s %s -- unit s mkpart %s", ps.dev, diskType)
		if len(fs) != 0 {
			cmd = fmt.Sprintf("%s %s", cmd, fileutils.FsFormatToDiskType(fs))
		}
		cmd = fmt.Sprintf("%s %d %d", cmd, start, end)
	} else {
		cmd = fmt.Sprintf("/usr/sbin/sgdisk --set-alignment=1 --new=%d:%d:%d", partIdx, start, end)
		if len(fs) != 0 {
			cmd = fmt.Sprintf("%s --typecode=%d:%s", cmd, partIdx, ps.FsToTypeCode(fs))
		}
		cmd = fmt.Sprintf("%s %s", cmd, ps.dev)
	}
	_, err := ps.Run(cmd)
	if err != nil {
		return err
	}
	if err := ps.RetrievePartitionInfo(); err != nil {
		return fmt.Errorf("Fail to RetrievePartitionInfo: %v", err)
	}
	if fs != "" && doformat {
		err = ps.partitions[len(ps.partitions)-1].Format(fs, uuid)
		if err != nil {
			return fmt.Errorf("Fail to format partition: %v", err)
		}
		if err := ps.RetrievePartitionInfo(); err != nil {
			return fmt.Errorf("Fail to RetrievePartitionInfo: %v", err)
		}
	}
	return nil
}

type IPartitionRunner interface {
	Run(cmds ...string) ([]string, error)
	Term() *ssh.Client
}

type PartitionTool struct {
	disks     []*DiskPartitions
	diskTable map[string][]*DiskPartitions
	runner    IPartitionRunner
}

func NewPartitionTool(runner IPartitionRunner) *PartitionTool {
	return &PartitionTool{
		disks:     make([]*DiskPartitions, 0),
		diskTable: make(map[string][]*DiskPartitions),
		runner:    runner,
	}
}

func (tool *PartitionTool) DebugString() string {
	ret := []string{}
	disksString := func(disks []*DiskPartitions) []string {
		for _, disk := range disks {
			ret = append(ret, disk.DebugString())
		}
		return ret
	}
	for driver, disks := range tool.diskTable {
		s := fmt.Sprintf("%s: %v", driver, disksString(disks))
		ret = append(ret, s)
	}
	return strings.Join(ret, "\n")
}

func (tool *PartitionTool) Disks() []*DiskPartitions {
	return tool.disks
}

func (tool *PartitionTool) parseLsDisk(lines []string, driver string) {
	disks := sysutils.ParseDiskInfo(lines, driver)
	if len(disks) == 0 {
		return
	}
	minCnt := int(math.Min(float64(len(disks)), float64(len(tool.diskTable[driver]))))
	for i := 0; i < minCnt; i++ {
		tool.diskTable[driver][i].SetInfo(disks[i])
	}
}

func (tool *PartitionTool) FetchDiskConfs(diskConfs []baremetal.DiskConfiguration) *PartitionTool {
	for _, d := range diskConfs {
		disk := newDiskPartitions(d.Driver, d.Adapter, d.RaidConfig, d.Size, d.Block, tool)
		tool.disks = append(tool.disks, disk)
		var key string
		if d.Driver == baremetal.DISK_DRIVER_LINUX {
			key = NONRAID_DRIVER
		} else if d.Driver == baremetal.DISK_DRIVER_PCIE {
			key = PCIE_DRIVER
		} else {
			key = RAID_DRVIER
		}
		if _, ok := tool.diskTable[key]; !ok {
			tool.diskTable[key] = make([]*DiskPartitions, 0)
		}
		tool.diskTable[key] = append(tool.diskTable[key], disk)
	}
	return tool
}

func (tool *PartitionTool) IsAllDisksReady() bool {
	for idx, d := range tool.disks {
		if !d.IsReady() {
			log.Errorf("disk.%d %#v not ready", idx, d)
			return false
		}
	}
	return true
}

func (tool *PartitionTool) RetrieveDiskInfo() error {
	for _, driver := range []string{RAID_DRVIER, NONRAID_DRIVER, PCIE_DRIVER} {
		cmd := fmt.Sprintf("/lib/mos/lsdisk --%s", driver)
		ret, err := tool.Run(cmd)
		if err != nil {
			return err
		}
		tool.parseLsDisk(ret, driver)
	}
	return nil
}

func (tool *PartitionTool) RetrievePartitionInfo() error {
	for _, disk := range tool.disks {
		if err := disk.RetrievePartitionInfo(); err != nil {
			return err
		}
	}
	return nil
}

func (tool *PartitionTool) ResizePartition(diskIdx int, sizeMB int64) error {
	if diskIdx >= 0 && diskIdx < len(tool.disks) {
		return tool.disks[diskIdx].ResizePartition(sizeMB)
	}
	return fmt.Errorf("Invalid disk index: %d", diskIdx)
}

func (tool *PartitionTool) GetDisks() []*DiskPartitions {
	return tool.disks
}

func (tool *PartitionTool) GetRootDisk() *DiskPartitions {
	if len(tool.disks) == 0 {
		return nil
	}
	return tool.disks[0]
}

func (tool *PartitionTool) GetPCIEDisks() []*DiskPartitions {
	disks := make([]*DiskPartitions, 0)
	for _, disk := range tool.disks {
		if disk.driver == PCIE_DRIVER {
			disks = append(disks, disk)
		}
	}
	return disks
}

func (tool *PartitionTool) CreatePartition(diskIdx int, sizeMB int64, fs string, doformat bool, driver string, uuid string) error {
	disks := tool.disks
	if driver == PCIE_DRIVER {
		disks = tool.GetPCIEDisks()
	}
	if diskIdx < 0 || diskIdx >= len(disks) {
		for _, disk := range disks {
			if disk.IsSpaceAvailable(sizeMB) {
				return disk.CreatePartition(sizeMB, fs, doformat, uuid)
			}
		}
	} else {
		disk := disks[diskIdx]
		if disk.IsSpaceAvailable(sizeMB) {
			return disk.CreatePartition(sizeMB, fs, doformat, uuid)
		}
	}
	return nil
}

func (tool *PartitionTool) GetPartitions() []*Partition {
	parts := make([]*Partition, 0)
	for _, d := range tool.disks {
		parts = append(parts, d.partitions...)
	}
	return parts
}

func (tool *PartitionTool) Run(cmds ...string) ([]string, error) {
	return tool.runner.Run(cmds...)
}

type SSHPartitionTool struct {
	*PartitionTool
	term *ssh.Client
}

func NewSSHPartitionTool(term *ssh.Client) *SSHPartitionTool {
	tool := &SSHPartitionTool{
		term: term,
	}
	tool.PartitionTool = NewPartitionTool(tool)
	return tool
}

func (tool *SSHPartitionTool) Run(cmds ...string) ([]string, error) {
	return tool.term.Run(cmds...)
}

func (tool *SSHPartitionTool) Term() *ssh.Client {
	return tool.term
}
