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

package guestman

import (
	"context"
	"fmt"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/guestman/qemu"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

type IGuestTasks interface {
	Start(func(...error))
}

/**
 *  GuestStopTask
**/

type SGuestStopTask struct {
	*SKVMGuestInstance
	ctx            context.Context
	timeout        int64
	startPowerdown time.Time
}

func NewGuestStopTask(guest *SKVMGuestInstance, ctx context.Context, timeout int64) *SGuestStopTask {
	return &SGuestStopTask{
		SKVMGuestInstance: guest,
		ctx:               ctx,
		timeout:           timeout,
		startPowerdown:    time.Time{},
	}
}

func (s *SGuestStopTask) Start() {
	s.stopping = true
	if s.IsRunning() && s.IsMonitorAlive() {
		s.Monitor.SimpleCommand("system_powerdown", s.onPowerdownGuest)
	} else {
		s.checkGuestRunning()
	}
}

func (s *SGuestStopTask) onPowerdownGuest(results string) {
	//s.ExitCleanup(true)
	s.startPowerdown = time.Now()
	s.checkGuestRunning()
}

func (s *SGuestStopTask) checkGuestRunning() {
	if !s.IsRunning() || time.Now().Sub(s.startPowerdown) > time.Duration(s.timeout)*time.Second {
		s.Stop() // force stop
		s.stopping = false
		hostutils.TaskComplete(s.ctx, nil)
	} else {
		s.CheckGuestRunningLater()
	}
}

func (s *SGuestStopTask) CheckGuestRunningLater() {
	time.Sleep(time.Second * 1)
	s.checkGuestRunning()
}

// SGuestStartRescueTask Start a rescue vm
type SGuestStartRescueTask struct {
	*SKVMGuestInstance
	ctx                 context.Context
	BaremetalManagerUri string
}

func NewGuestStartRescueTask(guest *SKVMGuestInstance, ctx context.Context, baremetalManagerUri string) *SGuestStartRescueTask {
	return &SGuestStartRescueTask{
		SKVMGuestInstance:   guest,
		ctx:                 ctx,
		BaremetalManagerUri: baremetalManagerUri,
	}
}

func (s *SGuestStartRescueTask) Start() {
	if err := s.prepareRescue(s.ctx, s.BaremetalManagerUri); err != nil {
		log.Errorf("prepareRescue fail %s", err)
		hostutils.TaskFailed(s.ctx, err.Error())
		return
	}

	hostutils.TaskComplete(s.ctx, nil)
}

// SGuestStopRescueTask Stop a rescue vm, clean rescue files
type SGuestStopRescueTask struct {
	*SKVMGuestInstance
	ctx context.Context
}

func NewGuestStopRescueTask(guest *SKVMGuestInstance, ctx context.Context) *SGuestStopRescueTask {
	return &SGuestStopRescueTask{
		SKVMGuestInstance: guest,
		ctx:               ctx,
	}
}

func (s *SGuestStopRescueTask) Start() {
	if err := s.clearRescue(s.ctx); err != nil {
		log.Errorf("clearRescue fail %s", err)
		hostutils.TaskFailed(s.ctx, err.Error())
		return
	}

	hostutils.TaskComplete(s.ctx, nil)
}

type SGuestSuspendTask struct {
	*SKVMGuestInstance
	ctx              context.Context
	onFinishCallback func(*SGuestSuspendTask, string)
}

func NewGuestSuspendTask(
	guest *SKVMGuestInstance,
	ctx context.Context,
	onFinishCallback func(*SGuestSuspendTask, string),
) *SGuestSuspendTask {
	t := &SGuestSuspendTask{
		SKVMGuestInstance: guest,
		ctx:               ctx,
	}
	if onFinishCallback == nil {
		onFinishCallback = t.onSaveMemStateComplete
	}
	t.onFinishCallback = onFinishCallback
	return t
}

func (s *SGuestSuspendTask) Start() {
	s.Monitor.SimpleCommand("stop", s.onSuspendGuest)
}

func (s *SGuestSuspendTask) GetStateFilePath() string {
	return s.SKVMGuestInstance.GetStateFilePath("")
}

func (s *SGuestSuspendTask) onSuspendGuest(results string) {
	if strings.Contains(strings.ToLower(results), "error") {
		hostutils.TaskFailed(s.ctx, fmt.Sprintf("Suspend error: %s", results))
		return
	}
	statFile := s.GetStateFilePath()
	s.Monitor.SaveState(statFile, s.onSaveMemStateWait)
}

func (s *SGuestSuspendTask) onSaveMemStateWait(results string) {
	if strings.Contains(strings.ToLower(results), "error") {
		hostutils.TaskFailed(s.ctx, fmt.Sprintf("Save memory state error: %s", results))
		// TODO: send cont command
		return
	}
	s.Monitor.GetMigrateStatus(s.onSaveMemStateCheck)
}

func (s *SGuestSuspendTask) onSaveMemStateCheck(status string) {
	if status == "failed" {
		hostutils.TaskFailed(s.ctx, "Save memory state failed")
		// TODO: send cont command
		return
	} else if status != "completed" {
		time.Sleep(time.Second * 3)
		log.Infof("Server %s saving memory state status %q", s.GetName(), status)
		s.onSaveMemStateWait("")
	} else {
		log.Infof("Server %s save memory completed", s.GetName())
		s.onFinishCallback(s, s.GetStateFilePath())
	}
}

func (s *SGuestSuspendTask) onSaveMemStateComplete(_ *SGuestSuspendTask, _ string) {
	log.Infof("Server %s memory state saved, stopping server", s.GetName())
	s.ExecStopTask(s.ctx, int64(3))
}

/**
 *  GuestSyncConfigTaskExecutor
**/

type SGuestSyncConfigTaskExecutor struct {
	ctx   context.Context
	guest *SKVMGuestInstance
	tasks []IGuestTasks

	errors   []error
	callback func([]error)
}

func NewGuestSyncConfigTaskExecutor(ctx context.Context, guest *SKVMGuestInstance, tasks []IGuestTasks, callback func([]error)) *SGuestSyncConfigTaskExecutor {
	return &SGuestSyncConfigTaskExecutor{ctx, guest, tasks, make([]error, 0), callback}
}

func (t *SGuestSyncConfigTaskExecutor) Start(delay int) {
	timeutils2.AddTimeout(1*time.Second, t.runNextTask)
}

func (t *SGuestSyncConfigTaskExecutor) runNextTask() {
	if len(t.tasks) > 0 {
		task := t.tasks[len(t.tasks)-1]
		t.tasks = t.tasks[:len(t.tasks)-1]
		task.Start(t.runNextTaskCallback)
	} else {
		t.doCallback()
	}
}

func (t *SGuestSyncConfigTaskExecutor) doCallback() {
	if t.callback != nil {
		t.callback(t.errors)
		t.callback = nil
	}
}

func (t *SGuestSyncConfigTaskExecutor) runNextTaskCallback(err ...error) {
	if err != nil {
		t.errors = append(t.errors, err...)
	}
	t.runNextTask()
}

/**
 *  GuestDiskSyncTask
**/

type SGuestDiskSyncTask struct {
	guest    *SKVMGuestInstance
	delDisks []*desc.SGuestDisk
	addDisks []*desc.SGuestDisk
	cdroms   []*desc.SGuestCdrom
	floppys  []*desc.SGuestFloppy
	errors   []error

	callback     func(...error)
	checkDrivers []string
}

func NewGuestDiskSyncTask(guest *SKVMGuestInstance, delDisks, addDisks []*desc.SGuestDisk, cdroms []*desc.SGuestCdrom, floppys []*desc.SGuestFloppy) *SGuestDiskSyncTask {
	return &SGuestDiskSyncTask{guest, delDisks, addDisks, cdroms, floppys, make([]error, 0), nil, nil}
}

func (d *SGuestDiskSyncTask) Start(callback func(...error)) {
	d.callback = callback
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) syncDisksConf() {
	if len(d.delDisks) > 0 {
		disk := d.delDisks[len(d.delDisks)-1]
		d.delDisks = d.delDisks[:len(d.delDisks)-1]
		d.removeDisk(disk)
		return
	}
	if len(d.addDisks) > 0 {
		disk := d.addDisks[len(d.addDisks)-1]
		d.addDisks = d.addDisks[:len(d.addDisks)-1]
		d.addDisk(disk)
		return
	}
	if len(d.cdroms) > 0 {
		cdrom := d.cdroms[len(d.cdroms)-1]
		d.cdroms = d.cdroms[:len(d.cdroms)-1]
		d.changeCdrom(cdrom)
		return
	}
	if len(d.floppys) > 0 {
		floppy := d.floppys[len(d.floppys)-1]
		d.floppys = d.floppys[:len(d.floppys)-1]
		d.changeFloppy(floppy)
		return
	}
	if idxs := d.guest.GetNeedMergeBackingFileDiskIndexs(); len(idxs) > 0 {
		d.guest.StreamDisks(context.Background(),
			func() { d.guest.streamDisksComplete(context.Background()) }, idxs,
		)
	}
	d.callback(d.errors...)
}

func (d *SGuestDiskSyncTask) changeCdrom(cdrom *desc.SGuestCdrom) {
	d.guest.Monitor.GetBlocks(func(blocks []monitor.QemuBlock) {
		var cdName string
		for _, r := range blocks {
			if regexp.MustCompile(fmt.Sprintf(`^(ide|scsi)?(%d-)?cd\d+$`, cdrom.Ordinal)).MatchString(r.Device) {
				cdName = r.Device
				break
			}
		}
		if len(cdName) > 0 {
			d.changeCdromContent(cdName, cdrom)
		} else {
			if cdrom.Path != "" && cdrom.Scsi != nil {
				cb2 := func(res string) {
					d.changeCdromContent(cdName, cdrom)
				}
				cb := func(res string) {
					cdrom.Scsi.Options["drive"] = cdrom.Id
					d.guest.Monitor.DeviceAdd(cdrom.Scsi.DevType, cdrom.Scsi.Options, cb2)
				}
				params := map[string]string{}
				for k, v := range cdrom.DriveOptions {
					params[k] = v
				}
				params["file"] = cdrom.Path
				params["id"] = cdrom.Id
				d.guest.Monitor.DriveAdd("0", "", params, cb)
			} else {
				cdrom.Path = ""
				d.syncDisksConf()
			}
		}
	})
}

func (d *SGuestDiskSyncTask) changeCdromContent(cdName string, cdrom *desc.SGuestCdrom) {
	if cdrom.Path == "" {
		d.guest.Monitor.EjectCdrom(cdName, func(s string) {
			d.OnEjectCdromContentSucc(cdName)
		})
	} else {
		d.guest.Monitor.ChangeCdrom(cdName, cdrom.Path, func(s string) {
			d.OnChangeCdromContentSucc(cdrom)
		})
	}
}

func (d *SGuestDiskSyncTask) OnEjectCdromContentSucc(cdName string) {
	for i, cdrom := range d.guest.Desc.Cdroms {
		if cdrom.Id == cdName {
			d.guest.Desc.Cdroms[i].Path = ""
			d.guest.Desc.Cdroms[i].BootIndex = nil
		}
	}
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) OnChangeCdromContentSucc(cdrom *desc.SGuestCdrom) {
	if d.guest.Desc.Cdroms == nil {
		d.guest.Desc.Cdroms = make([]*desc.SGuestCdrom, options.HostOptions.CdromCount)
		for i := range d.guest.Desc.Cdroms {
			d.guest.Desc.Cdroms[i] = new(desc.SGuestCdrom)
			d.guest.Desc.Cdroms[i].Ordinal = int64(i)
		}
	}
	for i := range d.guest.Desc.Cdroms {
		if cdrom.Ordinal == d.guest.Desc.Cdroms[i].Ordinal {
			d.guest.Desc.Cdroms[i] = cdrom
		}
	}
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) changeFloppy(floppy *desc.SGuestFloppy) {
	d.guest.Monitor.GetBlocks(func(blocks []monitor.QemuBlock) {
		var flName string
		for _, r := range blocks {
			if regexp.MustCompile(fmt.Sprintf(`^floppy%d$`, floppy.Ordinal)).MatchString(r.Device) {
				flName = r.Device
				break
			}
		}
		if len(flName) > 0 {
			d.changeFloppyContent(flName, floppy)
		} else {
			floppy.Path = ""
			d.syncDisksConf()
		}
	})
}

func (d *SGuestDiskSyncTask) changeFloppyContent(flName string, floppy *desc.SGuestFloppy) {
	if floppy.Path == "" {
		d.guest.Monitor.EjectCdrom(flName, func(s string) {
			d.OnEjectFloppyContentSucc(flName)
		})
	} else {
		d.guest.Monitor.ChangeCdrom(flName, floppy.Path, func(s string) {
			d.OnChangeFloppyContentSucc(floppy)
		})
	}
}

func (d *SGuestDiskSyncTask) OnEjectFloppyContentSucc(flName string) {
	for i, floppy := range d.guest.Desc.Floppys {
		if floppy.Id == flName {
			d.guest.Desc.Floppys[i].Path = ""
		}
	}
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) OnChangeFloppyContentSucc(floppy *desc.SGuestFloppy) {
	if d.guest.Desc.Floppys == nil {
		d.guest.Desc.Floppys = make([]*desc.SGuestFloppy, options.HostOptions.FloppyCount)
		for i := range d.guest.Desc.Floppys {
			d.guest.Desc.Floppys[i] = new(desc.SGuestFloppy)
			d.guest.Desc.Floppys[i].Ordinal = int64(i)
		}
	}
	for i := range d.guest.Desc.Floppys {
		if floppy.Ordinal == d.guest.Desc.Floppys[i].Ordinal {
			d.guest.Desc.Floppys[i] = floppy
		}
	}
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) removeDisk(disk *desc.SGuestDisk) {
	devId := fmt.Sprintf("drive_%d", disk.Index)
	d.guest.Monitor.DriveDel(devId, func(results string) {
		d.onRemoveDriveSucc(devId, results, disk.Index)
	})
}

func (d *SGuestDiskSyncTask) onRemoveDriveSucc(devId, results string, diskIdx int8) {
	log.Infof("remove drive %s results: %s", devId, results)
	d.guest.Monitor.DeviceDel(devId, func(results string) {
		d.onRemoveDiskSucc(results, diskIdx)
	})
}

func (d *SGuestDiskSyncTask) onRemoveDiskSucc(results string, diskIdx int8) {
	var i = 0
	for ; i < len(d.guest.Desc.Disks); i++ {
		if d.guest.Desc.Disks[i].Index == diskIdx {
			if d.guest.Desc.Disks[i].Pci != nil {
				err := d.guest.pciAddrs.ReleasePCIAddress(d.guest.Desc.Disks[i].Pci.PCIAddr)
				if err != nil {
					log.Errorf("failed release disk pci addr %s", d.guest.Desc.Disks[i].Pci.PCIAddr)
				}
			}
			break
		}
	}

	if i < len(d.guest.Desc.Disks) {
		d.guest.Desc.Disks = append(d.guest.Desc.Disks[:i], d.guest.Desc.Disks[i+1:]...)
	}

	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) checkDiskDriver(disk *desc.SGuestDisk) {
	if d.checkDrivers == nil {
		d.checkDrivers = make([]string, 0)
	}
	log.Debugf("sync disk driver: %s", disk.Driver)

	if disk.Driver == DISK_DRIVER_SCSI && d.guest.Desc.VirtioScsi == nil {
		// insert virtio scsi
		var cType = d.guest.getHotPlugPciControllerType()
		if cType == nil {
			err := errors.Errorf("failed get hotplugable pci controller")
			d.errors = append(d.errors, err)
			d.syncDisksConf()
			return
		}

		d.guest.Desc.VirtioScsi = &desc.SGuestVirtioScsi{
			PCIDevice: desc.NewPCIDevice(*cType, "virtio-scsi-pci", "scsi"),
		}
		cb := func(ret string) {
			log.Infof("Add scsi controller %s", ret)
			d.checkDrivers = append(d.checkDrivers, DISK_DRIVER_SCSI)
			d.startAddDisk(disk)
		}
		params := map[string]string{
			"id":   d.guest.Desc.VirtioScsi.Id,
			"bus":  d.guest.Desc.VirtioScsi.BusStr(),
			"addr": d.guest.Desc.VirtioScsi.SlotFunc(),
		}
		d.guest.Monitor.DeviceAdd(d.guest.Desc.VirtioScsi.DevType, params, cb)
	} else {
		d.startAddDisk(disk)
	}
}

func (d *SGuestDiskSyncTask) addDisk(disk *desc.SGuestDisk) {
	d.checkDiskDriver(disk)
}

func (d *SGuestDiskSyncTask) startAddDisk(disk *desc.SGuestDisk) {
	iDisk, _ := storageman.GetManager().GetDiskByPath(disk.Path)
	if iDisk == nil {
		d.syncDisksConf()
		return
	}

	var (
		diskIndex  = disk.Index
		aio        = disk.AioMode
		diskDriver = disk.Driver
		cacheMode  = disk.CacheMode
	)

	var params = map[string]string{
		"file":  iDisk.GetPath(),
		"if":    "none",
		"id":    fmt.Sprintf("drive_%d", diskIndex),
		"cache": cacheMode,
		"aio":   aio,
	}

	if iDisk.IsFile() {
		params["file.locking"] = "off"
	}
	if d.guest.isEncrypted() {
		params["encrypt.format"] = "luks"
		params["encrypt.key-secret"] = "sec0"
	}

	var bus string
	var cType *desc.PCI_CONTROLLER_TYPE
	switch diskDriver {
	case DISK_DRIVER_SCSI:
		bus = "scsi.0"
	case DISK_DRIVER_VIRTIO:
		cType = d.guest.getHotPlugPciControllerType()
		if cType == nil {
			log.Errorf("no hotplugable pci controller found")
			d.errors = append(d.errors, errors.Errorf("no hotplugable pci controller found"))
			d.syncDisksConf()
			return
		}
		bus = d.guest.GetPciBus()
	case DISK_DRIVER_IDE:
		bus = fmt.Sprintf("ide.%d", diskIndex/2)
	case DISK_DRIVER_SATA:
		bus = fmt.Sprintf("ide.%d", diskIndex)
	}
	// drive_add bus is a placeholder
	d.guest.Monitor.DriveAdd(bus, "", params, func(result string) { d.onAddDiskSucc(disk, result, cType) })
}

func (d *SGuestDiskSyncTask) onAddDiskSucc(disk *desc.SGuestDisk, results string, cType *desc.PCI_CONTROLLER_TYPE) {
	var (
		diskIndex  = disk.Index
		diskDriver = disk.Driver
		devType    = qemu.GetDiskDeviceModel(diskDriver)
		id         = fmt.Sprintf("drive_%d", diskIndex)
	)
	switch diskDriver {
	case DISK_DRIVER_VIRTIO:
		disk.Pci = desc.NewPCIDevice(*cType, devType, id)
		err := d.guest.ensureDevicePciAddress(disk.Pci, -1, nil)
		if err != nil {
			log.Errorln(err)
			d.guest.Monitor.DriveDel(id, func(res string) {
				log.Infof("drive %s del %s", id, res)
			})

			d.errors = append(d.errors, err)
			d.syncDisksConf()
			return
		}
	case DISK_DRIVER_SCSI:
		disk.Scsi = desc.NewScsiDevice(d.guest.Desc.VirtioScsi.Id, devType, id)
	case DISK_DRIVER_PVSCSI:
		disk.Scsi = desc.NewScsiDevice(d.guest.Desc.PvScsi.Id, devType, id)
	case DISK_DRIVER_IDE:
		disk.Ide = desc.NewIdeDevice(devType, id)
	case DISK_DRIVER_SATA: // -device ahci,id=ahci pci device
		disk.Ide = desc.NewIdeDevice(devType, id)
	}

	d.guest.Desc.Disks = append(d.guest.Desc.Disks, disk)
	var params = map[string]string{
		"drive": fmt.Sprintf("drive_%d", diskIndex),
		"id":    fmt.Sprintf("drive_%d", diskIndex),
	}

	if diskDriver == DISK_DRIVER_VIRTIO && disk.Pci != nil {
		params["bus"] = disk.Pci.BusStr()
		params["addr"] = disk.Pci.SlotFunc()
	} else if DISK_DRIVER_IDE == diskDriver {
		params["unit"] = strconv.Itoa(int(diskIndex % 2))
	}
	d.guest.Monitor.DeviceAdd(devType, params, d.onAddDeviceSucc)
}

func (d *SGuestDiskSyncTask) onAddDeviceSucc(results string) {
	log.Infof("%s device add res: %s", d.guest.GetName(), results)
	d.syncDisksConf()
}

/**
 *  GuestNetworkSyncTask
**/

type SGuestNetworkSyncTask struct {
	guest   *SKVMGuestInstance
	delNics []*desc.SGuestNetwork
	addNics []*desc.SGuestNetwork
	errors  []error

	callback func(...error)
}

func (n *SGuestNetworkSyncTask) Start(callback func(...error)) {
	n.callback = callback
	n.syncNetworkConf()
}

func (n *SGuestNetworkSyncTask) syncNetworkConf() {
	if len(n.delNics) > 0 {
		nic := n.delNics[len(n.delNics)-1]
		n.delNics = n.delNics[:len(n.delNics)-1]
		n.removeNic(nic)
	} else if len(n.addNics) > 0 {
		nic := n.addNics[len(n.addNics)-1]
		n.addNics = n.addNics[:len(n.addNics)-1]
		n.addNic(nic)
	} else {
		n.callback(n.errors...)
	}
}

func (n *SGuestNetworkSyncTask) removeNic(nic *desc.SGuestNetwork) {
	callback := func(res string) {
		if len(res) > 0 && !strings.Contains(res, "not found") {
			log.Errorf("netdev del failed %s", res)
			n.errors = append(n.errors, fmt.Errorf("netdev del failed %s", res))
			n.syncNetworkConf()
		} else {
			n.onNetdevDel(nic)
		}
	}
	n.guest.Monitor.NetdevDel(nic.Ifname, callback)
}

func (n *SGuestNetworkSyncTask) onNetdevDel(nic *desc.SGuestNetwork) {
	downScript := n.guest.getNicDownScriptPath(nic)
	output, err := procutils.NewCommand("sh", downScript).Output()
	if err != nil {
		log.Errorf("script down nic failed %s", output)
		n.errors = append(n.errors, err)
	}
	n.delNicDevice(nic)
}

func (n *SGuestNetworkSyncTask) delNicDevice(nic *desc.SGuestNetwork) {
	callback := func(res string) {
		if len(res) > 0 {
			log.Errorf("network device del failed %s", res)
			n.errors = append(n.errors, fmt.Errorf("network device del failed %s", res))
		} else {
			var i = 0
			for ; i < len(n.guest.Desc.Nics); i++ {
				if n.guest.Desc.Nics[i].Index == nic.Index {
					if nic.Pci != nil {
						err := n.guest.pciAddrs.ReleasePCIAddress(nic.Pci.PCIAddr)
						if err != nil {
							log.Errorf("failed release nic pci addr %s", nic.Pci.PCIAddr)
						}
					}
					break
				}
			}

			if i < len(n.guest.Desc.Nics) {
				n.guest.Desc.Nics = append(n.guest.Desc.Nics[:i], n.guest.Desc.Nics[i+1:]...)
			}

			n.syncNetworkConf()
		}
	}
	n.guest.Monitor.DeviceDel(fmt.Sprintf("netdev-%s", nic.Ifname), callback)
}

func (n *SGuestNetworkSyncTask) addNic(nic *desc.SGuestNetwork) {
	if err := n.guest.generateNicScripts(nic); err != nil {
		log.Errorln(err)
		n.errors = append(n.errors, err)
		n.syncNetworkConf()
		return
	}

	if nic.Driver == "vfio-pci" {
		// vfio device will add on isolated devices sync task
		n.onDeviceAdd(nic)
		return
	}
	var (
		netType    = "tap"
		upscript   = n.guest.getNicUpScriptPath(nic)
		downscript = n.guest.getNicDownScriptPath(nic)
		params     = map[string]string{
			"ifname": nic.Ifname, "script": upscript, "downscript": downscript,
			"vhost": "on", "vhostforce": "off",
		}
		cType = n.guest.getHotPlugPciControllerType()
	)

	if cType == nil {
		err := errors.Errorf("no hotplugable pci controller found")
		log.Errorln(err)
		n.errors = append(n.errors, err)
		n.syncNetworkConf()
		return
	}

	callback := func(res string) {
		if len(res) > 0 {
			log.Errorf("netdev add failed %s", res)
			n.errors = append(n.errors, fmt.Errorf("netdev add failed %s", res))
			n.syncNetworkConf()
		} else {
			nic.UpscriptPath = upscript
			nic.DownscriptPath = downscript
			n.onNetdevAdd(nic, cType)
		}
	}

	n.guest.Monitor.NetdevAdd(nic.Ifname, netType, params, callback)
}

func (n *SGuestNetworkSyncTask) onNetdevAdd(nic *desc.SGuestNetwork, cType *desc.PCI_CONTROLLER_TYPE) {
	id := fmt.Sprintf("netdev-%s", nic.Ifname)
	devType := n.guest.getNicDeviceModel(nic.Driver)
	onFail := func(e error) {
		log.Errorln(e)
		n.errors = append(n.errors, e)
		n.guest.Monitor.NetdevDel(nic.Ifname, func(res string) {
			log.Infof("netdev %s del %s", id, res)
		})
		n.syncNetworkConf()
	}

	switch nic.Driver {
	case "virtio":
		nic.Pci = desc.NewPCIDevice(*cType, devType, id)
	case "e1000":
		nic.Pci = desc.NewPCIDevice(*cType, devType, id)
	case "vmxnet3":
		nic.Pci = desc.NewPCIDevice(*cType, devType, id)
	default:
		err := errors.Errorf("unknown nic driver %s", nic.Driver)
		onFail(err)
		return
	}
	err := n.guest.ensureDevicePciAddress(nic.Pci, -1, nil)
	if err != nil {
		onFail(err)
		return
	}

	params := map[string]string{
		"id":     fmt.Sprintf("netdev-%s", nic.Ifname),
		"netdev": nic.Ifname,
		"mac":    nic.Mac,
		"bus":    nic.Pci.BusStr(),
		"addr":   nic.Pci.SlotFunc(),
	}

	callback := func(res string) {
		if len(res) > 0 {
			err := fmt.Errorf("device add failed %s", res)
			onFail(err)
			return
		} else {
			n.onDeviceAdd(nic)
		}
	}
	n.guest.Monitor.DeviceAdd(devType, params, callback)
}

func (n *SGuestNetworkSyncTask) onDeviceAdd(nic *desc.SGuestNetwork) {
	n.guest.Desc.Nics = append(n.guest.Desc.Nics, nic)
	n.syncNetworkConf()
}

func NewGuestNetworkSyncTask(
	guest *SKVMGuestInstance, delNics, addNics []*desc.SGuestNetwork,
) *SGuestNetworkSyncTask {
	return &SGuestNetworkSyncTask{guest, delNics, addNics, make([]error, 0), nil}
}

/**
 *  GuestIsolatedDeviceSyncTask
**/

type SGuestIsolatedDeviceSyncTask struct {
	guest   *SKVMGuestInstance
	delDevs []*desc.SGuestIsolatedDevice
	addDevs []*desc.SGuestIsolatedDevice
	errors  []error

	callback func(...error)
}

func NewGuestIsolatedDeviceSyncTask(guest *SKVMGuestInstance, delDevs, addDevs []*desc.SGuestIsolatedDevice) *SGuestIsolatedDeviceSyncTask {
	return &SGuestIsolatedDeviceSyncTask{guest, delDevs, addDevs, make([]error, 0), nil}
}

func (t *SGuestIsolatedDeviceSyncTask) Start(cb func(...error)) {
	t.callback = cb
	t.syncDevice()
}

func (t *SGuestIsolatedDeviceSyncTask) syncDevice() {
	if len(t.delDevs) > 0 {
		dev := t.delDevs[len(t.delDevs)-1]
		t.delDevs = t.delDevs[:len(t.delDevs)-1]
		t.removeDevice(dev)
	} else if len(t.addDevs) > 0 {
		dev := t.addDevs[len(t.addDevs)-1]
		t.addDevs = t.addDevs[:len(t.addDevs)-1]
		t.addDevice(dev)
	} else {
		t.callback(t.errors...)
	}
}

func (t *SGuestIsolatedDeviceSyncTask) removeDevice(dev *desc.SGuestIsolatedDevice) {
	cb := func(res string) {
		if len(res) > 0 {
			t.errors = append(t.errors, fmt.Errorf("device del failed: %s", res))
		} else {
			var i = 0
			for ; i < len(t.guest.Desc.IsolatedDevices); i++ {
				if t.guest.Desc.IsolatedDevices[i].Id == dev.Id {
					for j := 0; j < len(t.guest.Desc.IsolatedDevices[i].VfioDevs); j++ {
						pciaddr := t.guest.Desc.IsolatedDevices[i].VfioDevs[j].PCIAddr
						if pciaddr != nil {
							if e := t.guest.pciAddrs.ReleasePCIAddress(pciaddr); e != nil {
								log.Errorf("failed release vfio pci address %s", pciaddr)
							}
						}
					}
					break
				}
			}
			if i < len(t.guest.Desc.IsolatedDevices) {
				// remove device
				t.guest.Desc.IsolatedDevices = append(t.guest.Desc.IsolatedDevices[:i], t.guest.Desc.IsolatedDevices[i+1:]...)
			}
		}
		t.syncDevice()
	}

	devObj := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByIdent(dev.VendorDeviceId, dev.Addr, dev.MdevId)
	if devObj == nil {
		cb(fmt.Sprintf("Not found host isolated_device by %s %s", dev.VendorDeviceId, dev.Addr))
		return
	}

	opts, err := devObj.GetHotUnplugOptions(dev)
	if err != nil {
		cb(errors.Wrap(err, "GetHotPlugOptions").Error())
		return
	}

	t.delDeviceCallBack(opts, 0, cb)
}

func (t *SGuestIsolatedDeviceSyncTask) addDevice(dev *desc.SGuestIsolatedDevice) {
	var err error
	devObj := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByIdent(dev.VendorDeviceId, dev.Addr, dev.MdevId)
	if devObj == nil {
		err = errors.Errorf("Not found host isolated_device by %s %s", dev.VendorDeviceId, dev.Addr)
		log.Errorln(err)
		t.errors = append(t.errors, err)
		t.syncDevice()
		return
	}

	var setupScript string
	if dev.DevType == api.NIC_TYPE {
		setupScript, err = t.guest.sriovNicAttachInitScript(dev.NetworkIndex, devObj)
		if err != nil {
			err = errors.Errorf("sriovNicAttachInitScript %s", err)
			log.Errorln(err)
			t.errors = append(t.errors, err)
			t.syncDevice()
			return
		}
	}

	onFail := func(err error) {
		for i := 0; i < len(dev.VfioDevs); i++ {
			if dev.VfioDevs[i].PCIAddr != nil {
				if eRelease := t.guest.pciAddrs.ReleasePCIAddress(dev.VfioDevs[i].PCIAddr); eRelease != nil {
					log.Errorf("failed release pci pci address %s: %s", dev.VfioDevs[i].PCIAddr, eRelease)
				}
			}
		}
		log.Errorln(err)
		t.errors = append(t.errors, err)
		t.syncDevice()
		return
	}

	var cType *desc.PCI_CONTROLLER_TYPE
	if dev.DevType == api.USB_TYPE {
		dev.Usb = desc.NewUsbDevice("usb-host", devObj.GetQemuId())
		dev.Usb.Options = devObj.GetPassthroughOptions()
	} else {
		cType = t.guest.getVfioDeviceHotPlugPciControllerType()
		if cType == nil {
			log.Errorf("no hotplugable pci controller found")
			t.errors = append(t.errors, errors.Errorf("no hotplugable pci controller found"))
			t.syncDevice()
			return
		}
		id := devObj.GetQemuId()
		dev.VfioDevs = make([]*desc.VFIODevice, 0)
		vfioDev := desc.NewVfioDevice(
			*cType, "vfio-pci", id, devObj.GetAddr(), devObj.GetDeviceType() == api.GPU_VGA_TYPE,
		)
		dev.VfioDevs = append(dev.VfioDevs, vfioDev)

		groupDevAddrs := devObj.GetIOMMUGroupRestAddrs()
		for j := 0; j < len(groupDevAddrs); j++ {
			gid := fmt.Sprintf("%s-%d", id, j+1)
			vfioDev = desc.NewVfioDevice(*cType, "vfio-pci", gid, groupDevAddrs[j], false)
			dev.VfioDevs = append(dev.VfioDevs, vfioDev)
		}
		multiFunc := true
		err := t.guest.ensureDevicePciAddress(dev.VfioDevs[0].PCIDevice, 0, &multiFunc)
		if err != nil {
			err = errors.Wrapf(err, "ensure isolated device %s pci address", dev.VfioDevs[0].PCIAddr)
		} else {
			for j := 1; j < len(dev.VfioDevs); j++ {
				dev.VfioDevs[j].PCIAddr = dev.VfioDevs[0].PCIAddr.Copy()
				err = t.guest.ensureDevicePciAddress(dev.VfioDevs[j].PCIDevice, j, nil)
				if err != nil {
					err = errors.Wrapf(err, "ensure isolated device %s pci address", dev.VfioDevs[j].PCIAddr)
					break
				}
			}
		}
		if err != nil {
			onFail(err)
			return
		}
	}

	cb := func(res string) {
		if len(res) > 0 {
			onFail(fmt.Errorf("device add failed: %s", res))
		} else {
			t.guest.Desc.IsolatedDevices = append(t.guest.Desc.IsolatedDevices, dev)
			t.syncDevice()
		}
	}

	if len(setupScript) > 0 {
		output, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", setupScript).Output()
		if err != nil {
			log.Errorf("isolated device setup error %s, %s", output, err)
		}
	}

	opts, err := devObj.GetHotPlugOptions(dev, t.guest.Desc)
	if err != nil {
		cb(errors.Wrap(err, "GetHotPlugOptions").Error())
		return
	}

	t.addDeviceCallBack(opts, 0, cb)
}

func (t *SGuestIsolatedDeviceSyncTask) addDeviceCallBack(opts []*isolated_device.HotPlugOption, idx int, onAddFinish func(string)) {
	if idx >= len(opts) {
		onAddFinish("")
		return
	}

	opt := opts[idx]
	t.guest.Monitor.DeviceAdd(opt.Device, opt.Options, func(err string) {
		if err != "" {
			onAddFinish(fmt.Sprintf("monitor add %d device: %s", idx, err))
			return
		}
		t.addDeviceCallBack(opts, idx+1, onAddFinish)
	})
}

func (t *SGuestIsolatedDeviceSyncTask) delDeviceCallBack(opts []*isolated_device.HotUnplugOption, idx int, onDelFinish func(string)) {
	if idx >= len(opts) {
		onDelFinish("")
		return
	}

	opt := opts[idx]

	t.guest.Monitor.DeviceDel(opt.Id, func(err string) {
		if err != "" {
			onDelFinish(fmt.Sprintf("monitor del %d device: %s", idx, err))
			return
		}
		t.delDeviceCallBack(opts, idx+1, onDelFinish)
	})
}

/**
 *  GuestLiveMigrateTask
**/

type SGuestLiveMigrateTask struct {
	*SKVMGuestInstance

	ctx    context.Context
	params *SLiveMigrate

	c chan struct{}

	timeoutAt        time.Time
	doTimeoutMigrate bool
	cancelled        bool

	expectDowntime        int64
	dirtySyncCount        int64
	diskDriverMirrorIndex int

	onBlockJobsCancelled func()
	totalTransferMb      int64
}

func NewGuestLiveMigrateTask(
	ctx context.Context, guest *SKVMGuestInstance, params *SLiveMigrate,
) *SGuestLiveMigrateTask {
	task := &SGuestLiveMigrateTask{SKVMGuestInstance: guest, ctx: ctx, params: params}
	task.expectDowntime = 300 // qemu default downtime 300ms
	task.MigrateTask = task

	task.totalTransferMb = task.Desc.Mem
	for i := 0; i < len(task.Desc.Disks); i++ {
		if utils.IsInStringArray(task.Desc.Disks[i].StorageType, api.STORAGE_LOCAL_TYPES) {
			task.totalTransferMb += int64(task.Desc.Disks[i].Size)
		}
	}
	return task
}

func (s *SGuestLiveMigrateTask) Start() {
	s.Monitor.MigrateSetCapability("zero-blocks", "on", s.onSetZeroBlocks)
}

func (s *SGuestLiveMigrateTask) onSetZeroBlocks(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateFailed(fmt.Sprintf("Migrate set capability zero-blocks error: %s", res))
		return
	}

	// https://wiki.qemu.org/Features/AutoconvergeLiveMigration
	s.Monitor.MigrateSetCapability("auto-converge", "on", s.onSetAutoConverge)
}

func (s *SGuestLiveMigrateTask) onSetAutoConverge(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateFailed(fmt.Sprintf("Migrate set capability zero-blocks error: %s", res))
		return
	}

	// https://wiki.qemu.org/Features/AutoconvergeLiveMigration
	s.Monitor.MigrateSetCapability("events", "on", s.onMigrateEnableEvents)
}

func (s *SGuestLiveMigrateTask) onMigrateEnableEvents(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateFailed(fmt.Sprintf("Migrate set capability events error: %s", res))
		return
	}
	s.Monitor.MigrateSetCapability("pause-before-switchover", "on", s.onMigrateSetPauseBeforeSwitchover)
}

func (s *SGuestLiveMigrateTask) onMigrateSetPauseBeforeSwitchover(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateFailed(fmt.Sprintf("Migrate set capability pause-before-switchover error: %s", res))
		return
	}

	if version.LT(s.QemuVersion, "4.0.0") {
		s.startMigrate()
		return
	}

	if s.params.EnableTLS {
		s.Monitor.MigrateSetCapability("multifd", "off", s.onSetMulitfd)
		return
	}

	log.Infof("migrate src guest enable multifd")
	s.Monitor.MigrateSetCapability("multifd", "on", s.onSetMulitfd)
}

func (s *SGuestLiveMigrateTask) onSetMulitfd(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateFailed(fmt.Sprintf("Migrate set capability multifd error: %s", res))
		return
	}
	s.startMigrate()
}

func (s *SGuestLiveMigrateTask) startRamMigrateTimeout() {
	if !s.timeoutAt.IsZero() {
		// timeout has been set
		return
	}
	memMb := s.Desc.Mem
	migSeconds := int(memMb) / options.HostOptions.MigrateExpectRate
	if migSeconds < options.HostOptions.MinMigrateTimeoutSeconds {
		migSeconds = options.HostOptions.MinMigrateTimeoutSeconds
	}
	s.timeoutAt = time.Now().Add(time.Second * time.Duration(migSeconds))
	log.Infof("migrate timeout seconds: %d now: %v expectfinial: %v", migSeconds, time.Now(), s.timeoutAt)
}

func (s *SGuestLiveMigrateTask) startMigrate() {
	if s.params.EnableTLS {
		// https://wiki.qemu.org/Features/MigrationTLS
		// first remove possible existing tls0
		s.Monitor.ObjectDel("tls0", func(res string) {
			log.Infof("cleanup possible existing tls0: %s", res)
			s.Monitor.ObjectAdd("tls-creds-x509", map[string]string{
				"dir":         s.getPKIDirPath(),
				"endpoint":    "client",
				"id":          "tls0",
				"verify-peer": "no",
			}, func(res string) {
				if strings.Contains(strings.ToLower(res), "error") {
					s.migrateFailed(fmt.Sprintf("Migrate add tls-creds-x509 object client tls0 error: %s", res))
					return
				}
				s.Monitor.MigrateSetParameter("tls-creds", "tls0", func(res string) {
					if strings.Contains(strings.ToLower(res), "error") {
						s.migrateFailed(fmt.Sprintf("Migrate set tls-creds tls0 error: %s", res))
						return
					}
					s.doMigrate()
				})
			})
		})
	} else {
		s.Monitor.MigrateSetParameter("tls-creds", "", func(res string) {
			if strings.Contains(strings.ToLower(res), "error") {
				s.migrateFailed(fmt.Sprintf("Migrate set tls-creds to empty error: %s", res))
				return
			}
			s.doMigrate()
		})
	}
}

func (s *SGuestLiveMigrateTask) waitMirrorJobsReady() {
	cb := func(jobs []monitor.BlockJob) {
		var allReady = true
		var remaining = s.Desc.Mem * 1024 * 1024
		var mbps float64
		for i := 0; i < len(jobs); i++ {
			if jobs[i].Status != "ready" {
				allReady = false
				remaining += (jobs[i].Len - jobs[i].Offset)
				mbps += float64(jobs[i].Speed) / 1024 / 1024
			}
		}

		if !allReady {
			progress := (1 - float64(remaining)/float64(s.totalTransferMb*1024*1024)) * 100.0
			hostutils.UpdateServerProgress(context.Background(), s.Id, progress, mbps)
			time.Sleep(time.Second * 3)
			s.waitMirrorJobsReady()
			return
		}
		s.Monitor.Migrate(fmt.Sprintf("tcp:%s:%d", s.params.DestIp, s.params.DestPort),
			false, false, s.setMaxBandwidth)
	}
	s.Monitor.GetBlockJobs(cb)
}

func (s *SGuestLiveMigrateTask) mirrorDisks(res string) {
	if len(res) > 0 {
		log.Errorf("disk %d driver mirror failed %s", s.diskDriverMirrorIndex, res)
		s.onDriveMirrorDisksFailed(res)
		return
	}
	if s.diskDriverMirrorIndex == len(s.Desc.Disks) {
		s.waitMirrorJobsReady()
		return
	}

	i := s.diskDriverMirrorIndex
	s.diskDriverMirrorIndex += 1

	storageType := s.Desc.Disks[i].StorageType
	if storageType == "" {
		storageType = storageman.GetManager().GetStorage(s.Desc.Disks[i].StorageId).StorageType()
	}

	if utils.IsInStringArray(storageType, api.STORAGE_LOCAL_TYPES) {
		var drive = fmt.Sprintf("drive_%d", s.Desc.Disks[i].Index)
		var target = fmt.Sprintf("nbd:%s:%d:exportname=drive_%d", s.params.DestIp, s.params.NbdServerPort, s.Desc.Disks[i].Index)
		var speed int64 = 0
		if s.params.MaxBandwidthMB != nil {
			speed = *s.params.MaxBandwidthMB * 1024 * 1024
		}
		s.Monitor.DriveMirror(s.mirrorDisks, drive, target, "top", s.Desc.Disks[i].Format, true, false, speed)
	} else {
		s.mirrorDisks("")
	}
}

func (s *SGuestLiveMigrateTask) onDriveMirrorDisksFailed(res string) {
	s.migrateFailed(fmt.Sprintf("Migrate error: %s", res))
}

func (s *SGuestLiveMigrateTask) doMigrate() {
	if s.params.NbdServerPort > 0 {
		s.mirrorDisks("")
	} else {
		var copyIncremental = false
		if s.params.IsLocal {
			// copy disk data
			copyIncremental = true
		}
		s.Monitor.Migrate(fmt.Sprintf("tcp:%s:%d", s.params.DestIp, s.params.DestPort),
			copyIncremental, false, s.setMaxBandwidth)
	}
}

func (s *SGuestLiveMigrateTask) setMaxBandwidth(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateFailed(fmt.Sprintf("Migrate set capability auto-converge error: %s", res))
		return
	}

	// default set bandwidth no limit
	var maxBandwidth int64 = math.MaxInt64
	if s.params.MaxBandwidthMB != nil && *s.params.MaxBandwidthMB > 0 {
		maxBandwidth = *s.params.MaxBandwidthMB * 1024 * 1024
	}

	s.Monitor.MigrateSetParameter("max-bandwidth", maxBandwidth, s.startMigrateStatusCheck)
}

func (s *SGuestLiveMigrateTask) startMigrateStatusCheck(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateFailed(fmt.Sprintf("Migrate error: %s", res))
		return
	}

	s.c = make(chan struct{})
	for s.c != nil {
		select {
		case <-s.c: // on c close
			s.c = nil
			break
		case <-time.After(time.Second * 5):
			if s.Monitor != nil {
				s.Monitor.GetMigrateStats(s.onGetMigrateStats)
			} else {
				log.Errorf("server %s(%s) migrate stopped unexpectedly", s.GetId(), s.GetName())
				s.migrateFailed(fmt.Sprintf("Migrate error: %s", res))
				return
			}
		}
	}
}

func (s *SGuestLiveMigrateTask) onGetMigrateStats(stats *monitor.MigrationInfo, err error) {
	if err != nil {
		log.Errorf("%s get migrate stats failed %s", s.GetName(), err)
		s.migrateFailed(fmt.Sprintf("%s get migrate stats failed %s", s.GetName(), err))
		return
	}
	s.onGetMigrateStatus(stats)
}

/*
{ 'enum': 'MigrationStatus',
  'data': [ 'none', 'setup', 'cancelling', 'cancelled',
            'active', 'postcopy-active', 'postcopy-paused',
            'postcopy-recover', 'completed', 'failed', 'colo',
            'pre-switchover', 'device', 'wait-unplug' ] }
*/

func (s *SGuestLiveMigrateTask) onGetMigrateStatus(stats *monitor.MigrationInfo) {
	status := string(*stats.Status)
	if status == "completed" {
		jsonStats := jsonutils.Marshal(stats)
		log.Infof("migration info %s", jsonStats)
	} else if status == "failed" {
		s.migrateFailed(fmt.Sprintf("Query migrate got status: %s", status))
	} else if status == "cancelled" {
		s.migrateFailed(status)
	} else if status == "active" {
		var (
			ramRemain int64
			mbps      float64
		)
		if stats.RAM != nil {
			mbps = stats.RAM.Mbps
			ramRemain = stats.RAM.Remaining
		}
		if ramRemain > 0 {
			progress := (1 - float64(ramRemain)/float64(s.totalTransferMb*1024*1024)) * 100.0
			hostutils.UpdateServerProgress(context.Background(), s.Id, progress, mbps)
		}

		if s.params.QuicklyFinish && stats.RAM != nil && stats.RAM.Remaining > 0 {
			if stats.CPUThrottlePercentage == nil {
				// qemu do not enable cpu throttle, don't need set downtime
				return
			}

			if *stats.CPUThrottlePercentage < options.HostOptions.LiveMigrateCpuThrottleMax {
				return
			}

			if stats.ExpectedDowntime != nil && *stats.ExpectedDowntime > s.expectDowntime {
				if s.dirtySyncCount == 0 {
					// record dirty sync count
					s.dirtySyncCount = stats.RAM.DirtySyncCount
					return
				}
				// run more than one round dirty ram sync after cpu throttle 99%
				if stats.RAM.DirtySyncCount <= s.dirtySyncCount+1 {
					return
				}

				cb := func(res string) {
					if len(res) == 0 {
						s.expectDowntime = *stats.ExpectedDowntime
						log.Infof("migrate update downtime to %d", *stats.ExpectedDowntime)
					} else {
						log.Errorf("failed set migrate downtime %s", res)
					}
				}
				s.Monitor.MigrateSetDowntime(float64(*stats.ExpectedDowntime)/1000.0, cb)
			}
		}
	}
}

func (s *SGuestLiveMigrateTask) onMigrateStartPostcopy(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateFailed(fmt.Sprintf("onMigrateStartPostcopy error: %s", res))
		return
	} else {
		log.Infof("onMigrateStartPostcopy success")
	}
}

func (s *SGuestLiveMigrateTask) migrateContinueFromPreSwitchover() {
	s.Monitor.MigrateContinue("pre-switchover", s.onMigrateContinue)
}

func (s *SGuestLiveMigrateTask) onMigrateContinue(res string) {
	if len(res) > 0 {
		s.migrateFailed(res)
	}
}

func (s *SGuestLiveMigrateTask) onMigrateReceivedPreSwitchoverEvent() {
	s.onBlockJobsCancelled = s.migrateContinueFromPreSwitchover
	s.cancelBlockJobs("")
}

func (s *SGuestLiveMigrateTask) onMigrateReceivedBlockJobError(res string) {
	if !s.cancelled {
		s.migrateFailed(res)
	}
}

func (s *SGuestLiveMigrateTask) migrateComplete(stats jsonutils.JSONObject) {
	s.MigrateTask = nil
	if s.c != nil {
		close(s.c)
		s.c = nil
	}
	s.Monitor.Disconnect()
	s.Monitor = nil
	res := jsonutils.NewDict()
	if stats != nil {
		res.Set("migration_info", stats)
	}
	hostutils.TaskComplete(s.ctx, res)
	hostutils.UpdateServerProgress(context.Background(), s.Id, 0.0, 0)
}

func (s *SGuestLiveMigrateTask) cancelBlockJobs(res string) {
	log.Infof("%s cancel block jobs %s", s.GetName(), res)
	if s.diskDriverMirrorIndex == 0 {
		s.onBlockJobsCancelled()
		return
	}

	s.diskDriverMirrorIndex -= 1
	i := s.diskDriverMirrorIndex
	if utils.IsInStringArray(s.Desc.Disks[i].StorageType, api.STORAGE_LOCAL_TYPES) {
		s.Monitor.CancelBlockJob(fmt.Sprintf("drive_%d", s.Desc.Disks[i].Index), false, s.cancelBlockJobs)
	} else {
		s.cancelBlockJobs("")
	}
}

func (s *SGuestLiveMigrateTask) migrateFailed(msg string) {
	s.onBlockJobsCancelled = func() {
		s.onMigrateFailBlockJobsCancelled(msg)
	}
	s.cancelBlockJobs("")
}

func (s *SGuestLiveMigrateTask) onMigrateFailBlockJobsCancelled(msg string) {
	cleanup := func() {
		s.MigrateTask = nil
		if s.c != nil {
			close(s.c)
			s.c = nil
		}
		hostutils.TaskFailed(s.ctx, msg)
	}
	if s.params.EnableTLS {
		s.Monitor.ObjectDel("tls0", func(res string) {
			log.Infof("cleanup possible existing tls0: %s", res)
			cleanup()
		})
	} else {
		cleanup()
	}
}

func (s *SGuestLiveMigrateTask) SetLiveMigrateCancelled() {
	s.cancelled = true
}

/**
 *  GuestResumeTask
**/

type SGuestResumeTask struct {
	*SKVMGuestInstance

	ctx       context.Context
	startTime time.Time

	isTimeout           bool
	cleanTLS            bool
	resumed             bool
	isResumeFromMigrate bool

	getTaskData func() (jsonutils.JSONObject, error)
}

func NewGuestResumeTask(ctx context.Context, s *SKVMGuestInstance, isTimeout bool, cleanTLS bool) *SGuestResumeTask {
	return &SGuestResumeTask{
		SKVMGuestInstance: s,
		ctx:               ctx,
		isTimeout:         isTimeout,
		cleanTLS:          cleanTLS,
		getTaskData:       nil,
	}
}

func (s *SGuestResumeTask) Start() {
	log.Debugf("[%s] GuestResumeTask start", s.GetId())
	s.startTime = time.Now()
	if s.cleanTLS {
		s.Monitor.ObjectDel("tls0", func(res string) {
			log.Infof("Clean %s tls0 object: %s", s.GetName(), res)
			pkiPath := s.getPKIDirPath()
			if err := os.RemoveAll(pkiPath); err != nil {
				log.Warningf("Remove tls pki dir %s error: %v", pkiPath, err)
			}
			s.confirmRunning()
		})
		return
	}
	s.confirmRunning()
}

func (s *SGuestResumeTask) GetStateFilePath() string {
	return s.SKVMGuestInstance.GetStateFilePath("")
}

func (s *SGuestResumeTask) Stop() {
	// TODO
	// stop stream disk
	s.taskFailed(fmt.Sprintf("[%s] qemu quit unexpectedly on resume", s.GetId()))
}

func (s *SGuestResumeTask) confirmRunning() {
	if s.Monitor != nil {
		s.Monitor.QueryStatus(s.onConfirmRunning)
	} else {
		s.taskFailed(fmt.Sprintf("[%s] qemu quit unexpectedly on resume confirmRunning", s.GetId()))
	}
}

func (s *SGuestResumeTask) onConfirmRunning(status string) {
	log.Infof("[%s] onConfirmRunning status %s", s.GetId(), status)
	switch status {
	case "paused (prelaunch)":
		/* ref: qemu/src/qapi/run-state.json
		 * prelaunch: QEMU was started with -S and guest has not started.
		 * we need resume guest at state prelaunch */
		if err := s.onGuestPrelaunch(); err != nil {
			s.ForceStop()
			s.taskFailed(err.Error())
			return
		}
		s.resumeGuest()
	case "running", "paused (suspended)":
		s.onStartRunning()
	case "paused (postmigrate)":
		s.isResumeFromMigrate = true
		s.resumeGuest()
	case "paused (inmigrate)":
		// guest is paused waiting for an incoming migration
		time.Sleep(time.Second * 1)
		s.confirmRunning()
	default:
		switch {
		case strings.Contains(status, "error"):
			// handle error first, results may be 'paused (internal-error)'
			s.taskFailed(status)
		case strings.Contains(status, "paused"):
			if s.resumed {
				s.taskFailed("resume guest twice")
				return
			}
			if err := s.onGuestPrelaunch(); err != nil {
				s.ForceStop()
				s.taskFailed(err.Error())
				return
			}
			s.Monitor.GetBlocks(s.onGetBlockInfo)
		default:
			memMb := s.Desc.Mem
			migSeconds := int(memMb) / options.HostOptions.MigrateExpectRate
			if migSeconds < options.HostOptions.MinMigrateTimeoutSeconds {
				migSeconds = options.HostOptions.MinMigrateTimeoutSeconds
			}
			log.Infof("start guest timeout seconds: %d", migSeconds)
			if s.isTimeout && time.Now().Sub(s.startTime) >= time.Second*time.Duration(migSeconds) {
				s.taskFailed("Timeout")
				return
			} else {
				time.Sleep(time.Second * 3)
				s.confirmRunning()
			}
		}
	}
}

func (s *SGuestResumeTask) taskFailed(reason string) {
	log.Infof("Start guest %s failed: %s", s.Id, reason)
	s.ForceStop()
	if s.ctx != nil && len(appctx.AppContextTaskId(s.ctx)) > 0 {
		hostutils.TaskFailed(s.ctx, reason)
	} else {
		s.SyncStatus(reason)
	}
}

func (s *SGuestResumeTask) onGetBlockInfo(blocks []monitor.QemuBlock) {
	log.Debugf("onGetBlockInfo %v", blocks)
	// for _, drv := range results.GetArray() {
	// 	// encryption not work
	// }
	//time.Sleep(time.Second * 1)
	s.resumeGuest()
}

func (s *SGuestResumeTask) resumeGuest() {
	if s.resumed {
		s.taskFailed("resume guest twice")
		return
	}

	if s.Desc.IsVolatileHost {
		if err := s.prepareNicsForVolatileGuestResume(); err != nil {
			s.taskFailed(err.Error())
			return
		}
		s.Desc.IsVolatileHost = false
		SaveLiveDesc(s, s.Desc)
	}

	s.startTime = time.Now()
	s.Monitor.SimpleCommand("cont", s.onResumeSucc)
}

func (s *SGuestResumeTask) onResumeSucc(res string) {
	s.resumed = true
	s.confirmRunning()
}

func (s *SGuestResumeTask) SetGetTaskData(f func() (jsonutils.JSONObject, error)) {
	s.getTaskData = f
}

func (s *SGuestResumeTask) onStartRunning() {
	if s.Desc.IsVolatileHost {
		s.Desc.IsVolatileHost = false
		SaveLiveDesc(s, s.Desc)
	}

	s.setCgroupPid()
	s.removeStatefile()
	if s.ctx != nil && len(appctx.AppContextTaskId(s.ctx)) > 0 {
		var (
			data jsonutils.JSONObject
			err  error
		)
		if s.getTaskData != nil {
			data, err = s.getTaskData()
			if err != nil {
				s.taskFailed(err.Error())
				return
			}
		}
		hostutils.TaskComplete(s.ctx, data)
		if s.isResumeFromMigrate {
			return
		}
	}

	disksIdx := s.GetNeedMergeBackingFileDiskIndexs()
	if len(disksIdx) > 0 {
		s.SyncStatus("")
		timeutils2.AddTimeout(
			time.Second*time.Duration(options.HostOptions.AutoMergeDelaySeconds),
			func() { s.startStreamDisks(disksIdx) })
	} else if options.HostOptions.AutoMergeBackingTemplate {
		s.SyncStatus("")
		timeutils2.AddTimeout(
			time.Second*time.Duration(options.HostOptions.AutoMergeDelaySeconds),
			func() { s.startStreamDisks(nil) })
	} else {
		s.SyncStatus("")
		s.detachStartupTask()
	}
}

func (s *SGuestResumeTask) startStreamDisks(disksIdx []int) {
	s.startTime = time.Time{}
	s.detachStartupTask()
	if s.IsMonitorAlive() {
		s.StreamDisks(s.ctx, func() { s.onStreamComplete(disksIdx) }, disksIdx)
	}
}

func (s *SGuestResumeTask) onStreamComplete(disksIdx []int) {
	if len(disksIdx) == 0 {
		// if disks idx length == 0 indicate merge backing template
		s.SyncStatus("")
	} else {
		s.streamDisksComplete(context.Background())
	}
}

func (s *SGuestResumeTask) removeStatefile() {
	go s.CleanStatefiles()
}

type IGuestBlockProgressTask interface {
	OnGetBlockJobs(jobs []monitor.BlockJob)
	StreamingDiskCompletedCount() int
	StreamingDiskCount() int
}

type SGuestBlockProgressBaseTask struct {
	*SKVMGuestInstance

	ctx context.Context

	totalSizeMb     int
	completedSizeMb int

	c    chan struct{}
	jobs map[string]monitor.BlockJob
	task IGuestBlockProgressTask
}

func NewGuestBlockProgressBaseTask(
	ctx context.Context, guest *SKVMGuestInstance, blkTask IGuestBlockProgressTask,
) *SGuestBlockProgressBaseTask {
	return &SGuestBlockProgressBaseTask{
		SKVMGuestInstance: guest,
		ctx:               ctx,
		task:              blkTask,
		jobs:              map[string]monitor.BlockJob{},
	}
}

func (s *SGuestBlockProgressBaseTask) startWaitBlockJob(res string) {
	s.c = make(chan struct{})
	for {
		select {
		case <-s.c:
			s.c = nil
			return
		case <-time.After(time.Second * 10):
			s.Monitor.GetBlockJobs(s.onGetBlockJobs)
		}
	}
}

func (s *SGuestBlockProgressBaseTask) onGetBlockJobs(jobs []monitor.BlockJob) {
	if len(jobs) == 0 && s.c != nil {
		close(s.c)
		s.c = nil
	}
	for i := range jobs {
		job := jobs[i]
		_job, ok := s.jobs[job.Device]
		if !ok {
			job.CalcOffset(0)
			s.jobs[job.Device] = job
			continue
		}
		if _job.Status == "ready" {
			delete(s.jobs, _job.Device)
			continue
		}
		job.Start, job.Now = _job.Start, _job.Now
		job.CalcOffset(_job.Offset)
		s.jobs[job.Device] = job
	}
	mbps, progress := 0.0, 0.0
	totalSize, totalOffset := int64(1), int64(0)
	for _, job := range s.jobs {
		mbps += job.SpeedMbps
		totalSize += job.Len
		totalOffset += job.Offset
	}
	if len(s.jobs) == 0 && len(jobs) == 0 {
		progress = 100.0
	} else {
		progress = float64(totalOffset) / float64(totalSize) * 100
	}

	diskCount := s.task.StreamingDiskCount()
	if diskCount > 0 {
		progress = float64(s.task.StreamingDiskCompletedCount())/float64(diskCount)*100.0 + 1.0/float64(diskCount)*progress
	}
	hostutils.UpdateServerProgress(context.Background(), s.GetId(), progress, mbps)
	s.task.OnGetBlockJobs(jobs)
}

func (s *SGuestBlockProgressBaseTask) cancelWaitBlockJobs() {
	if s.c != nil {
		close(s.c)
		s.c = nil
	}
}

/**
 *  GuestStreamDisksTask
**/

type SGuestStreamDisksTask struct {
	*SGuestBlockProgressBaseTask

	callback func()
	disksIdx []int

	c          chan struct{}
	streamDevs []string
}

func NewGuestStreamDisksTask(ctx context.Context, guest *SKVMGuestInstance, callback func(), disksIdx []int) *SGuestStreamDisksTask {
	task := &SGuestStreamDisksTask{
		callback: callback,
		disksIdx: disksIdx,
	}
	task.SGuestBlockProgressBaseTask = NewGuestBlockProgressBaseTask(ctx, guest, task)
	return task
}

func (s *SGuestStreamDisksTask) Start() {
	s.Monitor.GetBlockJobCounts(s.onInitCheckStreamJobs)
}

func (s *SGuestStreamDisksTask) onInitCheckStreamJobs(jobs int) {
	if jobs > 0 {
		log.Warningf("GuestStreamDisksTask: duplicate block streaming???")
		s.startWaitBlockJob("")
	} else if jobs == 0 {
		s.startBlockStreaming()
	}
}

func (s *SGuestStreamDisksTask) startBlockStreaming() {
	s.checkBlockDrives()
}

func (s *SGuestStreamDisksTask) checkBlockDrives() {
	s.Monitor.GetBlocks(s.onBlockDrivesSucc)
}

func (s *SGuestStreamDisksTask) onBlockDrivesSucc(blocks []monitor.QemuBlock) {
	s.streamDevs = []string{}
	for _, block := range blocks {
		if len(block.Inserted.File) > 0 && len(block.Inserted.BackingFile) > 0 {
			var stream = false
			idx := block.Device[len(block.Device)-1] - '0'
			for i := 0; i < len(s.disksIdx); i++ {
				if int(idx) == s.disksIdx[i] {
					stream = true
				}
			}
			if !stream {
				continue
			}
			s.streamDevs = append(s.streamDevs, block.Device)
		}
	}
	log.Infof("Stream devices %s: %v", s.GetName(), s.streamDevs)
	if len(s.streamDevs) == 0 {
		s.taskComplete()
	} else {
		s.startDoBlockStream()
		s.SyncStatus("")
	}
}

func (s *SGuestStreamDisksTask) startDoBlockStream() {
	if len(s.streamDevs) > 0 {
		dev := s.streamDevs[0]
		s.streamDevs = s.streamDevs[1:]
		s.Monitor.BlockStream(dev, s.startWaitBlockJob)
	} else {
		s.taskComplete()
	}
}

func (s *SGuestStreamDisksTask) StreamingDiskCompletedCount() int {
	return len(s.disksIdx) - len(s.streamDevs) - 1
}

func (s *SGuestStreamDisksTask) StreamingDiskCount() int {
	return len(s.disksIdx)
}

func (s *SGuestStreamDisksTask) OnGetBlockJobs(jobs []monitor.BlockJob) {
	if len(jobs) == 0 {
		s.cancelWaitBlockJobs()
		if s.streamDevs == nil {
			s.checkBlockDrives()
		} else {
			s.startDoBlockStream()
		}
	}
}

func (s *SGuestStreamDisksTask) taskComplete() {
	hostutils.UpdateServerProgress(context.Background(), s.Id, 100.0, 0.0)
	s.SyncStatus("")

	if s.callback != nil {
		s.callback()
	}
}

/**
 *  GuestReloadDiskTask
**/

type SGuestReloadDiskTask struct {
	*SKVMGuestInstance

	ctx  context.Context
	disk storageman.IDisk
}

func NewGuestReloadDiskTask(
	ctx context.Context, s *SKVMGuestInstance, disk storageman.IDisk,
) *SGuestReloadDiskTask {
	return &SGuestReloadDiskTask{
		SKVMGuestInstance: s,
		ctx:               ctx,
		disk:              disk,
	}
}

func (s *SGuestReloadDiskTask) WaitSnapshotReplaced(callback func()) error {
	var retry = 0
	for {
		retry += 1
		if retry == 300 {
			return fmt.Errorf(
				"SnapshotDeleteJob.deleting_disk_snapshot always has %s", s.disk.GetId())
		}

		if _, ok := storageman.DELETEING_SNAPSHOTS.Load(s.disk.GetId()); ok {
			time.Sleep(time.Second * 1)
		} else {
			break
		}
	}

	callback()
	return nil
}

func (s *SGuestReloadDiskTask) Start() {
	s.fetchDisksInfo(s.startReloadDisk)
}

func (s *SGuestReloadDiskTask) fetchDisksInfo(callback func(string)) {
	s.Monitor.GetBlocks(func(blocks []monitor.QemuBlock) { s.onGetBlocksSucc(blocks, callback) })
}

func (s *SGuestReloadDiskTask) onGetBlocksSucc(blocks []monitor.QemuBlock, callback func(string)) {
	var device string
	for i := range blocks {
		device = s.getDiskOfDrive(blocks[i])
		if len(device) > 0 {
			callback(device)
			break
		}
	}

	if len(device) == 0 {
		s.taskFailed("Device not found")
	}
}

func (s *SGuestReloadDiskTask) getDiskOfDrive(block monitor.QemuBlock) string {
	if len(block.Inserted.File) == 0 {
		return ""
	}
	filePath, err := qemuimg.ParseQemuFilepath(block.Inserted.File)
	if err != nil {
		log.Errorf("qemuimg.ParseQemuFilepath %s fail %s", block.Inserted.File, err)
		return ""
	}
	if filePath == s.disk.GetPath() {
		return block.Device
	}
	return ""
}

func (s *SGuestReloadDiskTask) startReloadDisk(device string) {
	s.doReloadDisk(device, s.onReloadSucc)
}

func (s *SGuestReloadDiskTask) doReloadDisk(device string, callback func(string)) {
	s.Monitor.SimpleCommand("stop", func(string) {
		path := s.disk.GetPath()
		if s.isEncrypted() {
			path = qemuimg.GetQemuFilepath(path, "sec0", qemuimg.EncryptFormatLuks)
		}
		s.Monitor.ReloadDiskBlkdev(device, path, callback)
	})
}

func (s *SGuestReloadDiskTask) onReloadSucc(err string) {
	if len(err) > 0 {
		log.Errorf("monitor new snapshot blkdev error: %s", err)
	}
	s.Monitor.SimpleCommand("cont", s.onResumeSucc)
}

func (s *SGuestReloadDiskTask) onResumeSucc(results string) {
	log.Infof("guest reload disk task resume succ %s", results)
	params := jsonutils.NewDict()
	params.Set("reopen", jsonutils.JSONTrue)
	hostutils.TaskComplete(s.ctx, params)
}

func (s *SGuestReloadDiskTask) taskFailed(reason string) {
	hostutils.TaskFailed(s.ctx, reason)
}

/**
 *  GuestDiskSnapshotTask
**/

type SGuestDiskSnapshotTask struct {
	*SGuestReloadDiskTask

	snapshotId string
}

func NewGuestDiskSnapshotTask(
	ctx context.Context, s *SKVMGuestInstance, disk storageman.IDisk, snapshotId string,
) *SGuestDiskSnapshotTask {
	return &SGuestDiskSnapshotTask{
		SGuestReloadDiskTask: NewGuestReloadDiskTask(ctx, s, disk),
		snapshotId:           snapshotId,
	}
}

func (s *SGuestDiskSnapshotTask) Start() {
	s.fetchDisksInfo(s.startSnapshot)
}

func (s *SGuestDiskSnapshotTask) startSnapshot(device string) {
	s.doReloadDisk(device, s.onReloadBlkdevSucc)
}

func (s *SGuestDiskSnapshotTask) onReloadBlkdevSucc(res string) {
	var cb = s.onResumeSucc
	if len(res) > 0 {
		cb = func(string) {
			s.onSnapshotBlkdevFail(fmt.Sprintf("onReloadBlkdevFail: %s", res))
		}
	}
	s.Monitor.SimpleCommand("cont", cb)
}

func (s *SGuestDiskSnapshotTask) onSnapshotBlkdevFail(reason string) {
	snapshotDir := s.disk.GetSnapshotDir()
	snapshotPath := path.Join(snapshotDir, s.snapshotId)
	output, err := procutils.NewCommand("mv", "-f", snapshotPath, s.disk.GetPath()).Output()
	if err != nil {
		log.Errorf("mv %s to %s failed: %s, %s", snapshotPath, s.disk.GetPath(), err, output)
	}
	hostutils.TaskFailed(s.ctx, fmt.Sprintf("Reload blkdev error: %s", reason))
}

func (s *SGuestDiskSnapshotTask) onResumeSucc(res string) {
	log.Infof("guest disk snapshot task resume succ %s", res)
	snapshotLocation := path.Join(s.disk.GetSnapshotLocation(), s.snapshotId)
	body := jsonutils.NewDict()
	body.Set("location", jsonutils.NewString(snapshotLocation))
	hostutils.TaskComplete(s.ctx, body)
}

/**
 *  GuestSnapshotDeleteTask
**/

type SGuestSnapshotDeleteTask struct {
	*SGuestReloadDiskTask
	deleteSnapshot  string
	convertSnapshot string
	pendingDelete   bool

	tmpPath string
}

func NewGuestSnapshotDeleteTask(
	ctx context.Context, s *SKVMGuestInstance, disk storageman.IDisk,
	deleteSnapshot, convertSnapshot string, pendingDelete bool,
) *SGuestSnapshotDeleteTask {
	return &SGuestSnapshotDeleteTask{
		SGuestReloadDiskTask: NewGuestReloadDiskTask(ctx, s, disk),
		deleteSnapshot:       deleteSnapshot,
		convertSnapshot:      convertSnapshot,
		pendingDelete:        pendingDelete,
	}
}

func (s *SGuestSnapshotDeleteTask) Start() {
	if err := s.doDiskConvert(); err != nil {
		s.taskFailed(err.Error())
	}
	s.fetchDisksInfo(s.doReloadDisk)
}

func (s *SGuestSnapshotDeleteTask) doDiskConvert() error {
	snapshotDir := s.disk.GetSnapshotDir()
	snapshotPath := path.Join(snapshotDir, s.convertSnapshot)
	img, err := qemuimg.NewQemuImage(snapshotPath)
	if err != nil {
		log.Errorln(err)
		return err
	}
	convertedDisk := snapshotPath + ".tmp"
	if err = img.Convert2Qcow2To(convertedDisk, true, "", "", ""); err != nil {
		log.Errorln(err)
		if fileutils2.Exists(convertedDisk) {
			os.Remove(convertedDisk)
		}
		return err
	}

	s.tmpPath = snapshotPath + ".swap"
	if output, err := procutils.NewCommand("mv", "-f", snapshotPath, s.tmpPath).Output(); err != nil {
		log.Errorf("mv %s to %s failed: %s, %s", snapshotPath, s.tmpPath, err, output)
		if fileutils2.Exists(s.tmpPath) {
			procutils.NewCommand("mv", "-f", s.tmpPath, snapshotPath).Output()
		}
		return err
	}
	if output, err := procutils.NewCommand("mv", "-f", convertedDisk, snapshotPath).Output(); err != nil {
		log.Errorf("mv %s to %s failed: %s, %s", convertedDisk, snapshotPath, err, output)
		if fileutils2.Exists(s.tmpPath) {
			procutils.NewCommand("mv", "-f", s.tmpPath, snapshotPath).Output()
		}
		return err
	}
	return nil
}

func (s *SGuestSnapshotDeleteTask) doReloadDisk(device string) {
	s.SGuestReloadDiskTask.doReloadDisk(device, s.onReloadBlkdevSucc)
}

func (s *SGuestSnapshotDeleteTask) onReloadBlkdevSucc(err string) {
	var callback = s.onResumeSucc
	if len(err) > 0 {
		callback = func(string) {
			s.onSnapshotBlkdevFail(fmt.Sprintf("onReloadBlkdevFail %s", err))
		}
	}
	s.Monitor.SimpleCommand("cont", callback)
}

func (s *SGuestSnapshotDeleteTask) onSnapshotBlkdevFail(res string) {
	snapshotPath := path.Join(s.disk.GetSnapshotDir(), s.convertSnapshot)
	if output, err := procutils.NewCommand("mv", "-f", s.tmpPath, snapshotPath).Output(); err != nil {
		log.Errorf("mv %s to %s failed: %s, %s", s.tmpPath, snapshotPath, err, output)
	}
	s.taskFailed(fmt.Sprintf("Reload blkdev failed %s", res))
}

func (s *SGuestSnapshotDeleteTask) onResumeSucc(res string) {
	log.Infof("guest do new snapshot task resume succ %s", res)
	if len(s.tmpPath) > 0 {
		output, err := procutils.NewCommand("rm", "-f", s.tmpPath).Output()
		if err != nil {
			log.Errorf("rm %s failed: %s, %s", s.tmpPath, err, output)
		}
	}
	if !s.pendingDelete {
		s.disk.DoDeleteSnapshot(s.deleteSnapshot)
	}
	body := jsonutils.NewDict()
	body.Set("deleted", jsonutils.JSONTrue)
	hostutils.TaskComplete(s.ctx, body)
}

/**
 *  GuestDriveMirrorTask
**/

type SDriveMirrorTask struct {
	*SKVMGuestInstance

	ctx              context.Context
	nbdUri           string
	onSucc           func()
	syncMode         string
	index            int
	blockReplication bool
}

func NewDriveMirrorTask(
	ctx context.Context, s *SKVMGuestInstance, nbdUri, syncMode string,
	blockReplication bool, onSucc func(),
) *SDriveMirrorTask {
	return &SDriveMirrorTask{
		SKVMGuestInstance: s,
		ctx:               ctx,
		nbdUri:            nbdUri,
		syncMode:          syncMode,
		onSucc:            onSucc,
		blockReplication:  blockReplication,
	}
}

func (s *SDriveMirrorTask) Start() {
	s.startMirror("")
}

func (s *SDriveMirrorTask) supportBlockReplication() bool {
	c := make(chan bool)
	s.Monitor.HumanMonitorCommand("help drive_mirror", func(res string) {
		if strings.Index(res, "[-c]") > 0 {
			c <- true
		} else {
			c <- false
		}
	})
	return <-c
}

func (s *SDriveMirrorTask) startMirror(res string) {
	log.Infof("drive mirror results:%s", res)
	if len(res) > 0 {
		hostutils.TaskFailed(s.ctx, res)
		return
	}
	var blockReplication = false
	if s.blockReplication && s.supportBlockReplication() {
		blockReplication = true
		log.Infof("mirror block replication supported")
	}
	if s.index < len(s.Desc.Disks) {
		target := fmt.Sprintf("%s:exportname=drive_%d_backend", s.nbdUri, s.index)
		s.Monitor.DriveMirror(s.startMirror, fmt.Sprintf("drive_%d", s.index), target, s.syncMode, "", true, blockReplication, 0)
		s.index += 1
	} else {
		if s.onSucc != nil {
			s.onSucc()
		} else {
			hostutils.TaskComplete(s.ctx, nil)
		}
	}
}

type SDriveBackupTask struct {
	*SKVMGuestInstance

	ctx      context.Context
	nbdUri   string
	onSucc   func()
	syncMode string
	index    int
}

func NewDriveBackupTask(
	ctx context.Context, s *SKVMGuestInstance, nbdUri, syncMode string, onSucc func(),
) *SDriveBackupTask {
	return &SDriveBackupTask{
		SKVMGuestInstance: s,
		ctx:               ctx,
		nbdUri:            nbdUri,
		syncMode:          syncMode,
		onSucc:            onSucc,
	}
}

func (s *SDriveBackupTask) Start() {
	s.startBackup("")
}

func (s *SDriveBackupTask) startBackup(res string) {
	log.Infof("drive backup results:%s", res)
	if len(res) > 0 {
		hostutils.TaskFailed(s.ctx, res)
		return
	}
	disks := s.Desc.Disks
	if s.index < len(disks) {
		target := fmt.Sprintf("%s:exportname=drive_%d_backend", s.nbdUri, s.index)
		s.Monitor.DriveBackup(s.startBackup, fmt.Sprintf("drive_%d", s.index), target, s.syncMode, "raw")
		s.index += 1
	} else {
		if s.onSucc != nil {
			s.onSucc()
		} else {
			hostutils.TaskComplete(s.ctx, nil)
		}
	}
}

type SGuestBlockReplicationTask struct {
	*SKVMGuestInstance

	ctx      context.Context
	nbdHost  string
	nbdPort  string
	onSucc   func()
	onFail   func(string)
	syncMode string
	index    int
}

func NewGuestBlockReplicationTask(
	ctx context.Context, s *SKVMGuestInstance,
	nbdHost, nbdPort, syncMode string, onSucc func(), onFail func(string),
) *SGuestBlockReplicationTask {
	return &SGuestBlockReplicationTask{
		SKVMGuestInstance: s,
		ctx:               ctx,
		nbdHost:           nbdHost,
		nbdPort:           nbdPort,
		syncMode:          syncMode,
		onSucc:            onSucc,
		onFail:            onFail,
	}
}

func (s *SGuestBlockReplicationTask) Start() {
	s.onXBlockdevChange("")
}

func (s *SGuestBlockReplicationTask) onXBlockdevChange(res string) {
	if len(res) > 0 {
		log.Errorf("SGuestBlockReplicationTask onXBlockdevChange %s", res)
		if s.onFail != nil {
			s.onFail(res)
		} else {
			hostutils.TaskFailed(s.ctx, res)
		}
		return
	}

	disks := s.Desc.Disks
	if s.index < len(disks) {
		diskIndex := disks[s.index].Index
		drive := fmt.Sprintf("drive_%d", diskIndex)
		node := fmt.Sprintf("node_%d", diskIndex)

		s.Monitor.DriveAdd("", "buddy", map[string]string{
			"file.driver": "nbd", "file.host": s.nbdHost, "file.port": s.nbdPort,
			"file.export": drive, "node-name": node,
		}, s.onNbdDriveAddSucc(drive, node))
		s.index += 1
	} else {
		if s.onSucc != nil {
			s.onSucc()
		} else {
			hostutils.TaskComplete(s.ctx, nil)
		}
	}
}

func (s *SGuestBlockReplicationTask) onNbdDriveAddSucc(parent, node string) monitor.StringCallback {
	return func(res string) {
		if len(res) > 0 {
			log.Errorf("SGuestBlockReplicationTask onNbdDriveAddSucc %s", res)
			if s.onFail != nil {
				s.onFail(res)
			} else {
				hostutils.TaskFailed(s.ctx, res)
			}
			return
		}

		s.Monitor.XBlockdevChange(parent, node, "", s.onXBlockdevChange)
	}
}

/**
 *  GuestOnlineResizeDiskTask
**/

type SGuestOnlineResizeDiskTask struct {
	*SKVMGuestInstance

	ctx    context.Context
	diskId string
	sizeMB int64
}

func NewGuestOnlineResizeDiskTask(
	ctx context.Context, s *SKVMGuestInstance, diskId string, sizeMB int64,
) *SGuestOnlineResizeDiskTask {
	return &SGuestOnlineResizeDiskTask{
		SKVMGuestInstance: s,
		ctx:               ctx,
		diskId:            diskId,
		sizeMB:            sizeMB,
	}
}

func (task *SGuestOnlineResizeDiskTask) Start() {
	task.Monitor.GetBlocks(task.OnGetBlocksSucc)
}

func (task *SGuestOnlineResizeDiskTask) OnGetBlocksSucc(blocks []monitor.QemuBlock) {
	for i := 0; i < len(blocks); i += 1 {
		image := ""
		if strings.HasPrefix(blocks[i].Inserted.File, "json:") {
			//RBD磁盘格式如下
			//json:{"driver": "raw", "file": {"pool": "testpool01", "image": "952636e3-73ed-4a19-8648-05e69e6bb57a", "driver": "rbd", "=keyvalue-pairs": "[\"mon_host\", \"10.127.10.230;10.127.10.237;10.127.10.238\", \"key\", \"AQBZ/Ddd0j5BCxAAfuvl5oHWsmuTGer6T9LzeQ==\", \"rados_mon_op_timeout\", \"5\", \"rados_osd_op_timeout\", \"1200\", \"client_mount_timeout\", \"120\"]"}
			fileJson, err := jsonutils.ParseString(blocks[i].Inserted.File[5:])
			if err != nil {
				hostutils.TaskFailed(task.ctx, fmt.Sprintf("parse file json %s error: %v", blocks[i].Inserted.File, err))
				return
			}
			image, _ = fileJson.GetString("file", "image")
		}
		if len(blocks[i].Inserted.File) > 0 && strings.HasSuffix(blocks[i].Inserted.File, task.diskId) || image == task.diskId {
			task.Monitor.ResizeDisk(blocks[i].Device, task.sizeMB, task.OnResizeSucc)
			return
		}
	}
	hostutils.TaskFailed(task.ctx, fmt.Sprintf("disk %s not found on this guest", task.diskId))
}

func (task *SGuestOnlineResizeDiskTask) OnResizeSucc(err string) {
	if len(err) == 0 {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewInt(task.sizeMB), "disk_size")
		hostutils.TaskComplete(task.ctx, params)
		return
	}
	hostutils.TaskFailed(task.ctx, fmt.Sprintf("resize disk %s %dMb error: %v", task.diskId, task.sizeMB, err))
}

/**
 *  GuestHotplugCpuMem
**/

type SGuestHotplugCpuMemTask struct {
	*SKVMGuestInstance

	ctx         context.Context
	addCpuCount int
	addMemSize  int

	originalCpuCount int
	addedCpuCount    int

	addedMemSize    int
	memSlotNewIndex *int
	memSlot         *desc.SMemSlot
}

func NewGuestHotplugCpuMemTask(
	ctx context.Context, s *SKVMGuestInstance, addCpuCount, addMemSize int,
) *SGuestHotplugCpuMemTask {
	return &SGuestHotplugCpuMemTask{
		SKVMGuestInstance: s,
		ctx:               ctx,
		addCpuCount:       addCpuCount,
		addMemSize:        addMemSize,
	}
}

// First at all add cpu count, second add mem size
func (task *SGuestHotplugCpuMemTask) Start() {
	if task.addCpuCount > 0 {
		task.startAddCpu()
	} else if task.addMemSize > 0 {
		task.startAddMem()
	} else {
		task.onSucc()
	}
}

func (task *SGuestHotplugCpuMemTask) startAddCpu() {
	task.Monitor.GetCpuCount(task.onGetCpuCount)
}

func (task *SGuestHotplugCpuMemTask) onGetCpuCount(count int) {
	task.originalCpuCount = count
	task.doAddCpu()
}

func (task *SGuestHotplugCpuMemTask) doAddCpu() {
	if task.addedCpuCount < task.addCpuCount {
		task.Monitor.AddCpu(task.originalCpuCount+task.addedCpuCount, task.onAddCpu)
	} else {
		task.startAddMem()
	}
}

func (task *SGuestHotplugCpuMemTask) onAddCpu(reason string) {
	if len(reason) > 0 {
		log.Errorln(reason)
		task.onFail(reason)
		return
	}
	task.addedCpuCount += 1
	task.doAddCpu()
}

func (task *SGuestHotplugCpuMemTask) startAddMem() {
	if task.addMemSize > 0 {
		task.Monitor.GeMemtSlotIndex(task.onGetSlotIndex)
	} else {
		task.onSucc()
	}
}

func (task *SGuestHotplugCpuMemTask) onGetSlotIndex(index int) {
	var newIndex = index
	task.memSlotNewIndex = &newIndex

	var objType string
	var id = fmt.Sprintf("mem%d", *task.memSlotNewIndex)
	var options map[string]string

	if task.manager.host.IsHugepagesEnabled() {
		memPath := fmt.Sprintf("/dev/hugepages/%s-%d", task.GetId(), index)

		err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", memPath).Run()
		if err != nil {
			reason := fmt.Sprintf("mkdir %s fail: %s", memPath, err)
			log.Errorf("%s", reason)
			task.onFail(reason)
			return
		}
		err = procutils.NewRemoteCommandAsFarAsPossible("mount", "-t", "hugetlbfs", "-o",
			fmt.Sprintf("pagesize=%dK,size=%dM", task.manager.host.HugepageSizeKb(), task.addMemSize),
			fmt.Sprintf("hugetlbfs-%s-%d", task.GetId(), index),
			memPath,
		).Run()
		if err != nil {
			reason := fmt.Sprintf("mount %s fail: %s", memPath, err)
			log.Errorf("%s", reason)
			task.onFail(reason)
			return
		}

		objType = "memory-backend-file"
		options = map[string]string{
			"size":     fmt.Sprintf("%dM", task.addMemSize),
			"mem-path": memPath,
			"share":    "on",
			"prealloc": "on",
		}
	} else {
		objType = "memory-backend-ram"
		options = map[string]string{
			"size": fmt.Sprintf("%dM", task.addMemSize),
		}
	}
	options["id"] = id
	cb := func(reason string) {
		if reason == "" {
			memObj := desc.NewObject(objType, id)
			memObj.Options = options
			task.memSlot = new(desc.SMemSlot)
			task.memSlot.MemObj = memObj
			task.memSlot.SizeMB = int64(task.addMemSize)
		}
		task.onAddMemObject(reason)
	}
	task.Monitor.ObjectAdd(objType, options, cb)
}

func (task *SGuestHotplugCpuMemTask) onAddMemFailed(reason string) {
	log.Errorln(reason)
	cb := func(res string) { log.Infof("%s", res) }
	task.Monitor.ObjectDel(fmt.Sprintf("mem%d", *task.memSlotNewIndex), cb)
	task.onFail(reason)
}

func (task *SGuestHotplugCpuMemTask) onAddMemObject(reason string) {
	if len(reason) > 0 {
		task.onAddMemFailed(reason)
		return
	}
	params := map[string]string{
		"id":     fmt.Sprintf("dimm%d", *task.memSlotNewIndex),
		"memdev": fmt.Sprintf("mem%d", *task.memSlotNewIndex),
	}
	cb := func(reason string) {
		if reason == "" {
			task.memSlot.MemDev = &desc.SMemDevice{
				Type: "pc-dimm",
				Id:   fmt.Sprintf("dimm%d", *task.memSlotNewIndex),
			}
		}
		task.onAddMemDevice(reason)
	}

	task.Monitor.DeviceAdd("pc-dimm", params, cb)
}

func (task *SGuestHotplugCpuMemTask) onAddMemDevice(reason string) {
	if len(reason) > 0 {
		task.onAddMemFailed(reason)
		return
	}
	task.addedMemSize = task.addMemSize
	task.onSucc()
}

func (task *SGuestHotplugCpuMemTask) updateGuestDesc() {
	task.Desc.Cpu += int64(task.addedCpuCount)
	task.Desc.CpuDesc.Cpus += uint(task.addedCpuCount)
	task.Desc.Mem += int64(task.addedMemSize)
	if task.addedMemSize > 0 {
		if task.Desc.MemDesc.MemSlots == nil {
			task.Desc.MemDesc.MemSlots = make([]*desc.SMemSlot, 0)
		}
		task.Desc.MemDesc.MemSlots = append(task.Desc.MemDesc.MemSlots, task.memSlot)
	}
	if task.addedCpuCount > 0 && len(task.Desc.VcpuPin) == 1 {
		task.Desc.VcpuPin[0].Vcpus = fmt.Sprintf("0-%d", task.Desc.Cpu-1)
	}

	if task.addedCpuCount > 0 || task.addedMemSize > 0 {
		SaveLiveDesc(task, task.Desc)
	}
	if task.addedMemSize > 0 {
		vncPort := task.GetVncPort()
		data := jsonutils.NewDict()
		data.Set("vnc_port", jsonutils.NewInt(int64(vncPort)))
		data.Set("sync_qemu_cmdline", jsonutils.JSONTrue)
		if err := task.saveScripts(data); err != nil {
			log.Errorf("failed save script: %s", err)
		}
	}
}

func (task *SGuestHotplugCpuMemTask) onFail(reason string) {
	body := jsonutils.NewDict()
	if task.addedCpuCount < task.addCpuCount {
		body.Set("add_cpu_failed", jsonutils.JSONTrue)
		body.Set("added_cpu", jsonutils.NewInt(int64(task.addedCpuCount)))
	} else if task.memSlotNewIndex != nil {
		body.Set("add_mem_failed", jsonutils.JSONTrue)
	}
	task.updateGuestDesc()
	hostutils.TaskFailed2(task.ctx, reason, body)
}

func (task *SGuestHotplugCpuMemTask) onSucc() {
	task.updateGuestDesc()
	hostutils.TaskComplete(task.ctx, nil)
}

type SGuestBlockIoThrottleTask struct {
	*SKVMGuestInstance

	ctx context.Context
}

func (task *SGuestBlockIoThrottleTask) Start() {
	go task.startDoIoThrottle(0)
}

func (task *SGuestBlockIoThrottleTask) startDoIoThrottle(idx int) {
	if idx < len(task.Desc.Disks) {
		_cb := func(res string) {
			if len(res) > 0 {
				task.taskFail(res)
			} else {
				task.startDoIoThrottle(idx + 1)
			}
		}
		task.Monitor.BlockIoThrottle(
			fmt.Sprintf("drive_%d", task.Desc.Disks[idx].Index),
			int64(task.Desc.Disks[idx].Bps), int64(task.Desc.Disks[idx].Iops), _cb)
	} else {
		task.taskComplete(nil)
	}
}

func (task *SGuestBlockIoThrottleTask) taskFail(reason string) {
	if taskId := task.ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		hostutils.TaskFailed(task.ctx, reason)
	} else {
		log.Errorln(reason)
	}
}

func (task *SGuestBlockIoThrottleTask) taskComplete(data jsonutils.JSONObject) {
	if taskId := task.ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		hostutils.TaskComplete(task.ctx, data)
	}
}

type CancelBlockReplication struct {
	SCancelBlockJobs
}

func NewCancelBlockReplicationTask(ctx context.Context, guest *SKVMGuestInstance) *CancelBlockReplication {
	return &CancelBlockReplication{SCancelBlockJobs{guest, ctx}}
}

func (task *CancelBlockReplication) Start() {
	// start remove child node of block device
	disks := task.Desc.Disks
	for i := 0; i < len(disks); i++ {
		diskIndex := disks[i].Index
		drive := fmt.Sprintf("drive_%d", diskIndex)
		node := fmt.Sprintf("node_%d", diskIndex)
		child := fmt.Sprintf("children.%d", task.getQuorumChildIndex())
		task.Monitor.XBlockdevChange(drive, "", child, func(res string) {
			if len(res) > 0 {
				log.Errorf("failed remove child %s for parent %s: %s", drive, node, res)
				return
			}
			task.Monitor.DriveDel(node, func(res string) {
				if len(res) > 0 {
					log.Errorf("failed remove drive %s: %s", node, res)
					return
				}
			})
		})
	}
	if task.ctx != nil {
		hostutils.TaskComplete(task.ctx, nil)
	}
}

type SCancelBlockJobs struct {
	*SKVMGuestInstance

	ctx context.Context
}

func NewCancelBlockJobsTask(ctx context.Context, guest *SKVMGuestInstance) *SCancelBlockJobs {
	return &SCancelBlockJobs{guest, ctx}
}

func (task *SCancelBlockJobs) Start() {
	task.findBlockDevices()
}

func (task *SCancelBlockJobs) findBlockDevices() {
	task.Monitor.GetBlocks(task.onBlockDriversSucc)
}

func (task *SCancelBlockJobs) onBlockDriversSucc(blocks []monitor.QemuBlock) {
	drivers := make([]string, 0)
	for i := 0; i < len(blocks); i++ {
		if strings.HasPrefix(blocks[i].Device, "drive_") {
			drivers = append(drivers, blocks[i].Device)
		}
	}
	task.StartCancelBlockJobs(drivers)
}

func (task *SCancelBlockJobs) StartCancelBlockJobs(drivers []string) {
	if len(drivers) > 0 {
		driver := drivers[0]
		drivers = drivers[1:]
		onCancelBlockJob := func(res string) {
			if len(res) > 0 {
				log.Errorln(res)
			}
			task.StartCancelBlockJobs(drivers)
		}
		task.Monitor.CancelBlockJob(driver, false, onCancelBlockJob)
	} else {
		task.taskComplete()
	}
}

func (task *SCancelBlockJobs) taskComplete() {
	if task.ctx != nil {
		hostutils.TaskComplete(task.ctx, nil)
	}
}

type SGuestStorageCloneDiskTask struct {
	*SGuestBlockProgressBaseTask

	started   bool
	diskIndex int
	resp      *hostapi.ServerCloneDiskFromStorageResponse
	params    *SStorageCloneDisk
}

func NewGuestStorageCloneDiskTask(ctx context.Context, guest *SKVMGuestInstance, params *SStorageCloneDisk) *SGuestStorageCloneDiskTask {
	task := &SGuestStorageCloneDiskTask{
		params: params,
	}
	task.SGuestBlockProgressBaseTask = NewGuestBlockProgressBaseTask(ctx, guest, task)
	return task
}

func (t *SGuestStorageCloneDiskTask) Start(guestRunning bool) {
	var diskIndex = -1
	disks := t.Desc.Disks
	for diskIndex = 0; diskIndex < len(disks); diskIndex++ {
		if disks[diskIndex].DiskId == t.params.SourceDisk.GetId() {
			diskIndex = int(disks[diskIndex].Index)
			break
		}
	}
	if diskIndex < 0 {
		hostutils.TaskFailed(
			t.ctx, fmt.Sprintf("failed find disk index %s", t.params.SourceDisk.GetId()),
		)
		return
	}

	resp, err := t.params.TargetStorage.CloneDiskFromStorage(
		t.ctx, t.params.SourceStorage, t.params.SourceDisk, t.params.TargetDiskId, !guestRunning)
	if err != nil {
		hostutils.TaskFailed(
			t.ctx, fmt.Sprintf("Clone disk %s to storage %s failed %s",
				t.params.SourceDisk.GetPath(), t.params.TargetStorage.GetId(), err),
		)
		return
	}

	targetDisk, err := t.params.TargetStorage.GetDiskById(t.params.TargetDiskId)
	if err != nil {
		hostutils.TaskFailed(
			t.ctx, fmt.Sprintf("Failed get target disk %s %s", t.params.TargetDiskId, err),
		)
		return
	}
	targetDiskFormat, err := targetDisk.GetFormat()
	if err != nil {
		hostutils.TaskFailed(
			t.ctx, fmt.Sprintf("Failed get target disk format %s %s", t.params.TargetDiskId, err),
		)
		return
	}

	if !guestRunning {
		hostutils.TaskComplete(t.ctx, jsonutils.Marshal(resp))
		return
	} else {
		t.resp = resp
		t.diskIndex = diskIndex
	}

	t.Monitor.DriveMirror(t.onDriveMirror, fmt.Sprintf("drive_%d", diskIndex), targetDisk.GetPath(), "full", targetDiskFormat, true, false, 0)
}

func (t *SGuestStorageCloneDiskTask) onDriveMirror(res string) {
	if len(res) > 0 {
		hostutils.TaskFailed(
			t.ctx, fmt.Sprintf("Clone disk %s to storage %s drive mirror failed %s",
				t.params.SourceDisk.GetPath(), t.params.TargetStorage.GetId(), res),
		)
	} else {
		t.started = true
		hostutils.TaskComplete(t.ctx, jsonutils.Marshal(t.resp))
		go t.startWaitBlockJob(res)
		timeutils2.AddTimeout(time.Second*3, func() {
			t.SyncStatus("drive mirror started")
		})
	}
}

func (t *SGuestStorageCloneDiskTask) OnGetBlockJobs(jobs []monitor.BlockJob) {
	if len(jobs) == 0 {
		if !t.started {
			targetDisk, err := t.params.TargetStorage.GetDiskById(t.params.TargetDiskId)
			if err != nil {
				hostutils.TaskFailed(
					t.ctx, fmt.Sprintf("Failed get target disk %s %s", t.params.TargetDiskId, err),
				)
				return
			}
			targetDiskFormat, err := targetDisk.GetFormat()
			if err != nil {
				hostutils.TaskFailed(
					t.ctx, fmt.Sprintf("Failed get target disk format %s %s", t.params.TargetDiskId, err),
				)
				return
			}
			t.Monitor.DriveMirror(t.onDriveMirror, fmt.Sprintf("drive_%d", t.diskIndex), targetDisk.GetPath(), "full", targetDiskFormat, true, false, 0)
		} else {
			hostutils.TaskFailed(t.ctx, fmt.Sprintf("Disk %s Block job not found", t.params.SourceDisk.GetId()))
		}
		return
	}
	for i := range jobs {
		if jobs[i].Status == "ready" && jobs[i].Device == fmt.Sprintf("drive_%d", t.diskIndex) {
			t.cancelWaitBlockJobs()
			params := jsonutils.NewDict()
			params.Set("block_jobs_ready", jsonutils.JSONTrue)
			hostutils.TaskComplete(t.ctx, params)
			break
		}
	}

}

func (t *SGuestStorageCloneDiskTask) StreamingDiskCompletedCount() int {
	return t.params.CompletedDiskCount
}

func (t *SGuestStorageCloneDiskTask) StreamingDiskCount() int {
	return t.params.CloneDiskCount
}

type SGuestLiveChangeDisk struct {
	*SKVMGuestInstance

	ctx             context.Context
	params          *SStorageCloneDisk
	guestNeedResume bool
	diskIndex       int
	targetDisk      storageman.IDisk
}

func NewGuestLiveChangeDiskTask(ctx context.Context, guest *SKVMGuestInstance, params *SStorageCloneDisk) (*SGuestLiveChangeDisk, error) {
	disk, err := params.TargetStorage.GetDiskById(params.TargetDiskId)
	if err != nil {
		return nil, err
	}

	var diskIndex = -1
	disks := guest.Desc.Disks
	for diskIndex = 0; diskIndex < len(disks); diskIndex++ {
		if disks[diskIndex].DiskId == params.SourceDisk.GetId() {
			diskIndex = int(disks[diskIndex].Index)
			break
		}
	}
	if diskIndex < 0 {
		return nil, fmt.Errorf("failed found disk %s index", params.SourceDisk.GetId())
	}

	diskFormat, err := params.SourceDisk.GetFormat()
	if err != nil {
		return nil, errors.Wrap(err, "failed get disk fromat")
	}
	params.DiskFormat = diskFormat

	return &SGuestLiveChangeDisk{
		SKVMGuestInstance: guest,
		ctx:               ctx,
		params:            params,
		guestNeedResume:   false,
		diskIndex:         diskIndex,
		targetDisk:        disk,
	}, nil
}

func (t *SGuestLiveChangeDisk) Start() {
	// pause guest first
	if !t.IsSuspend() {
		t.Monitor.SimpleCommand("stop", t.onGuestPaused)
		t.guestNeedResume = true
	} else {
		t.onGuestPaused("")
	}
}

func (t *SGuestLiveChangeDisk) onGuestPaused(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		hostutils.TaskFailed(t.ctx, fmt.Sprintf("pause error: %s", res))
		return
	}

	t.Monitor.BlockJobComplete(fmt.Sprintf("drive_%d", t.diskIndex), t.onReopenImageSuccess)
}

func (t *SGuestLiveChangeDisk) onReopenImageSuccess(res string) {
	// resume guest first
	if t.guestNeedResume {
		t.Monitor.SimpleCommand("cont", nil)
	}
	if len(res) > 0 {
		hostutils.TaskFailed(t.ctx, fmt.Sprintf("reopen image failed: %s", res))
		return
	}
	if t.params.TargetDiskDesc != nil {
		for i := 0; i < len(t.Desc.Disks); i++ {
			if t.Desc.Disks[i].Index == int8(t.diskIndex) {
				log.Debugf("update guest disk %s desc", t.Desc.Disks[i].DiskId)
				t.Desc.Disks[i].GuestdiskJsonDesc = *t.params.TargetDiskDesc
				SaveLiveDesc(t, t.Desc)
				break
			}
		}
	}

	resp := &hostapi.ServerCloneDiskFromStorageResponse{
		TargetAccessPath: t.targetDisk.GetPath(),
	}
	hostutils.TaskComplete(t.ctx, jsonutils.Marshal(resp))
}
