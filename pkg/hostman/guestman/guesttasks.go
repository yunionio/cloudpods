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
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appctx"
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
	s.ExitCleanup(true)
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
		hostutils.TaskFailed(s.ctx, fmt.Sprintf("Save memory state failed"))
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
	delDisks []jsonutils.JSONObject
	addDisks []jsonutils.JSONObject
	cdrom    *string

	callback      func(...error)
	checkeDrivers []string
}

func NewGuestDiskSyncTask(guest *SKVMGuestInstance, delDisks, addDisks []jsonutils.JSONObject, cdrom *string) *SGuestDiskSyncTask {
	return &SGuestDiskSyncTask{guest, delDisks, addDisks, cdrom, nil, nil}
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
	if d.cdrom != nil {
		d.changeCdrom()
		return
	}
	if idxs := d.guest.GetNeedMergeBackingFileDiskIndexs(); len(idxs) > 0 {
		d.guest.StreamDisks(context.Background(),
			func() { d.guest.streamDisksComplete(context.Background()) }, idxs,
		)
	}
	d.callback()
}

func (d *SGuestDiskSyncTask) changeCdrom() {
	d.guest.Monitor.GetBlocks(d.onGetBlockInfo)
}

func (d *SGuestDiskSyncTask) onGetBlockInfo(blocks []monitor.QemuBlock) {
	var cdName string
	for _, r := range blocks {
		if regexp.MustCompile(`^ide\d+-cd\d+$`).MatchString(r.Device) {
			cdName = r.Device
			break
		}
	}
	if len(cdName) > 0 {
		d.changeCdromContent(cdName)
	}
}

func (d *SGuestDiskSyncTask) changeCdromContent(cdName string) {
	if *d.cdrom == "" {
		d.guest.Monitor.EjectCdrom(cdName, d.OnChangeCdromContentSucc)
	} else {
		d.guest.Monitor.ChangeCdrom(cdName, *d.cdrom, d.OnChangeCdromContentSucc)
	}
}

func (d *SGuestDiskSyncTask) OnChangeCdromContentSucc(results string) {
	d.cdrom = nil
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) removeDisk(disk jsonutils.JSONObject) {
	index, _ := disk.Int("index")
	devId := fmt.Sprintf("drive_%d", index)
	d.guest.Monitor.DriveDel(devId,
		func(results string) { d.onRemoveDriveSucc(devId, results) })
}

func (d *SGuestDiskSyncTask) onRemoveDriveSucc(devId, results string) {
	d.guest.Monitor.DeviceDel(devId, d.onRemoveDiskSucc)
}

func (d *SGuestDiskSyncTask) onRemoveDiskSucc(results string) {
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) checkDiskDriver(disk jsonutils.JSONObject) {
	if d.checkeDrivers == nil {
		d.checkeDrivers = make([]string, 0)
	}
	driver, _ := disk.GetString("driver")
	log.Debugf("sync disk driver: %s", driver)
	if driver == DISK_DRIVER_SCSI {
		if utils.IsInStringArray(DISK_DRIVER_SCSI, d.checkeDrivers) {
			d.startAddDisk(disk)
		} else {
			cb := func(ret string) { d.checkScsiDriver(ret, disk) }
			d.guest.Monitor.HumanMonitorCommand("info pci", cb)
		}
	} else {
		d.startAddDisk(disk)
	}
}

func (d *SGuestDiskSyncTask) checkScsiDriver(ret string, disk jsonutils.JSONObject) {
	if strings.Contains(ret, "SCSI controller") {
		d.checkeDrivers = append(d.checkeDrivers, DISK_DRIVER_SCSI)
		d.startAddDisk(disk)
	} else {
		cb := func(ret string) {
			log.Infof("Add scsi controller %s", ret)
			d.checkeDrivers = append(d.checkeDrivers, DISK_DRIVER_SCSI)
			d.startAddDisk(disk)
		}
		d.guest.Monitor.DeviceAdd("virtio-scsi-pci", map[string]interface{}{"id": "scsi"}, cb)
	}
}

func (d *SGuestDiskSyncTask) addDisk(disk jsonutils.JSONObject) {
	d.checkDiskDriver(disk)
}

func (d *SGuestDiskSyncTask) startAddDisk(disk jsonutils.JSONObject) {
	diskPath, _ := disk.GetString("path")
	iDisk, _ := storageman.GetManager().GetDiskByPath(diskPath)
	if iDisk == nil {
		d.syncDisksConf()
		return
	}

	var (
		diskIndex, _  = disk.Int("index")
		aio, _        = disk.GetString("aio_mode")
		diskDirver, _ = disk.GetString("driver")
		cacheMode, _  = disk.GetString("cache_mode")
	)

	var params = map[string]string{
		"file":  iDisk.GetPath(),
		"if":    "none",
		"id":    fmt.Sprintf("drive_%d", diskIndex),
		"cache": cacheMode,
		"aio":   aio,
	}

	var bus string
	switch diskDirver {
	case DISK_DRIVER_SCSI:
		bus = "scsi.0"
	case DISK_DRIVER_VIRTIO:
		bus = d.guest.GetPciBus()
	case DISK_DRIVER_IDE:
		bus = fmt.Sprintf("ide.%d", diskIndex/2)
	case DISK_DRIVER_SATA:
		bus = fmt.Sprintf("ide.%d", diskIndex)
	}
	d.guest.Monitor.DriveAdd(bus, params, func(result string) { d.onAddDiskSucc(disk, result) })
}

func (d *SGuestDiskSyncTask) onAddDiskSucc(disk jsonutils.JSONObject, results string) {
	var (
		diskIndex, _  = disk.Int("index")
		diskDirver, _ = disk.GetString("driver")
		dev           = qemu.GetDiskDeviceModel(diskDirver)
	)

	var params = map[string]interface{}{
		"drive": fmt.Sprintf("drive_%d", diskIndex),
		"id":    fmt.Sprintf("drive_%d", diskIndex),
	}

	if diskDirver == DISK_DRIVER_VIRTIO {
		params["addr"] = fmt.Sprintf("0x%x", d.guest.GetDiskAddr(int(diskIndex)))
	} else if DISK_DRIVER_IDE == diskDirver {
		params["unit"] = diskIndex % 2
	}
	d.guest.Monitor.DeviceAdd(dev, params, d.onAddDeviceSucc)
}

func (d *SGuestDiskSyncTask) onAddDeviceSucc(results string) {
	d.syncDisksConf()
}

/**
 *  GuestNetworkSyncTask
**/

type SGuestNetworkSyncTask struct {
	guest   *SKVMGuestInstance
	delNics []jsonutils.JSONObject
	addNics []jsonutils.JSONObject
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

func (n *SGuestNetworkSyncTask) removeNic(nic jsonutils.JSONObject) {
	ifname, _ := nic.GetString("ifname")
	callback := func(res string) {
		if len(res) > 0 {
			log.Errorf("netdev del failed %s", res)
			n.errors = append(n.errors, fmt.Errorf("netdev del failed %s", res))
			n.syncNetworkConf()
		} else {
			n.onNetdevDel(nic)
		}
	}
	n.guest.Monitor.NetdevDel(ifname, callback)
}

func (n *SGuestNetworkSyncTask) onNetdevDel(nic jsonutils.JSONObject) {
	downScript := n.guest.getNicDownScriptPath(nic)
	output, err := procutils.NewCommand("sh", downScript).Output()
	if err != nil {
		log.Errorf("script down nic failed %s", output)
		n.errors = append(n.errors, err)
	}
	n.delNicDevice(nic)
}

func (n *SGuestNetworkSyncTask) delNicDevice(nic jsonutils.JSONObject) {
	callback := func(res string) {
		if len(res) > 0 {
			log.Errorf("network device del failed %s", res)
			n.errors = append(n.errors, fmt.Errorf("network device del failed %s", res))
		} else {
			n.syncNetworkConf()
		}
	}
	ifname, _ := nic.GetString("ifname")
	n.guest.Monitor.DeviceDel(fmt.Sprintf("netdev-%s", ifname), callback)
}

func (n *SGuestNetworkSyncTask) addNic(nic jsonutils.JSONObject) {
	if err := n.guest.generateNicScripts(nic); err != nil {
		log.Errorln(err)
		n.errors = append(n.errors, err)
		n.syncNetworkConf()
		return
	}
	upscript := n.guest.getNicUpScriptPath(nic)
	downscript := n.guest.getNicDownScriptPath(nic)
	ifname, _ := nic.GetString("ifname")
	params := map[string]string{
		"ifname": ifname, "script": upscript, "downscript": downscript,
		"vhost": "on", "vhostforce": "off",
	}
	netType := "tap"

	callback := func(res string) {
		if len(res) > 0 {
			log.Errorf("netdev add failed %s", res)
			n.errors = append(n.errors, fmt.Errorf("netdev add failed %s", res))
			n.syncNetworkConf()
		} else {
			n.onNetdevAdd(nic)
		}
	}

	n.guest.Monitor.NetdevAdd(ifname, netType, params, callback)
}

func (n *SGuestNetworkSyncTask) onNetdevAdd(nic jsonutils.JSONObject) {
	index, _ := nic.Int("index")
	ifname, _ := nic.GetString("ifname")
	mac, _ := nic.GetString("mac")
	driver, _ := nic.GetString("driver")
	dev := n.guest.getNicDeviceModel(driver)
	addr := n.guest.getNicAddr(int(index))
	params := map[string]interface{}{
		"id":     fmt.Sprintf("netdev-%s", ifname),
		"netdev": ifname,
		"addr":   fmt.Sprintf("0x%x", addr),
		"mac":    mac,
		"bus":    "pci.0",
	}
	callback := func(res string) {
		if len(res) > 0 {
			log.Errorf("device add failed %s", res)
			n.errors = append(n.errors, fmt.Errorf("device add failed %s", res))
			n.syncNetworkConf()
		} else {
			n.onDeviceAdd(nic)
		}
	}
	n.guest.Monitor.DeviceAdd(dev, params, callback)
}

func (n *SGuestNetworkSyncTask) onDeviceAdd(nic jsonutils.JSONObject) {
	n.syncNetworkConf()
}

func NewGuestNetworkSyncTask(guest *SKVMGuestInstance, delNics, addNics []jsonutils.JSONObject) *SGuestNetworkSyncTask {
	return &SGuestNetworkSyncTask{guest, delNics, addNics, make([]error, 0), nil}
}

/**
 *  GuestIsolatedDeviceSyncTask
**/

type SGuestIsolatedDeviceSyncTask struct {
	guest   *SKVMGuestInstance
	delDevs []jsonutils.JSONObject
	addDevs []jsonutils.JSONObject
	errors  []error

	callback func(...error)
}

func NewGuestIsolatedDeviceSyncTask(guest *SKVMGuestInstance, delDevs, addDevs []jsonutils.JSONObject) *SGuestIsolatedDeviceSyncTask {
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

func (t *SGuestIsolatedDeviceSyncTask) removeDevice(dev jsonutils.JSONObject) {
	cb := func(res string) {
		if len(res) > 0 {
			t.errors = append(t.errors, fmt.Errorf("device del failed: %s", res))
		}
		t.syncDevice()
	}

	vendorDevId, err := dev.GetString("vendor_device_id")
	if err != nil {
		cb(err.Error())
		return
	}
	addr, err := dev.GetString("addr")
	if err != nil {
		cb(err.Error())
		return
	}

	devObj := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByIdent(vendorDevId, addr)
	if devObj == nil {
		cb(fmt.Sprintf("Not found host isolated_device by %s %s", vendorDevId, addr))
		return
	}

	opts, err := devObj.GetHotUnplugOptions()
	if err != nil {
		cb(errors.Wrap(err, "GetHotPlugOptions").Error())
		return
	}

	t.delDeviceCallBack(opts, 0, cb)
}

func (t *SGuestIsolatedDeviceSyncTask) addDevice(dev jsonutils.JSONObject) {
	cb := func(res string) {
		if len(res) > 0 {
			t.errors = append(t.errors, fmt.Errorf("device add failed: %s", res))
		}
		t.syncDevice()
	}

	vendorDevId, err := dev.GetString("vendor_device_id")
	if err != nil {
		cb(err.Error())
		return
	}
	addr, err := dev.GetString("addr")
	if err != nil {
		cb(err.Error())
		return
	}

	devObj := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByIdent(vendorDevId, addr)
	if devObj == nil {
		cb(fmt.Sprintf("Not found host isolated_device by %s %s", vendorDevId, addr))
		return
	}

	opts, err := devObj.GetHotPlugOptions()
	if err != nil {
		cb(errors.Wrap(err, "GetHotPlugOptions").Error())
		return
	}

	// TODO: support GPU
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
}

func NewGuestLiveMigrateTask(
	ctx context.Context, guest *SKVMGuestInstance, params *SLiveMigrate,
) *SGuestLiveMigrateTask {
	task := &SGuestLiveMigrateTask{SKVMGuestInstance: guest, ctx: ctx, params: params}
	task.migrateTask = task
	return task
}

func (s *SGuestLiveMigrateTask) Start() {
	s.Monitor.MigrateSetCapability("zero-blocks", "on", s.onSetZeroBlocks)
}

func (s *SGuestLiveMigrateTask) onSetZeroBlocks(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateTask = nil
		hostutils.TaskFailed(s.ctx, fmt.Sprintf("Migrate set capability zero-blocks error: %s", res))
		return
	}
	// https://wiki.qemu.org/Features/AutoconvergeLiveMigration
	s.Monitor.MigrateSetCapability("auto-converge", "on", s.startMigrate)
}

func (s *SGuestLiveMigrateTask) startRamMigrateTimeout() {
	if !s.timeoutAt.IsZero() {
		// timeout has been set
		return
	}
	memMb, _ := s.Desc.Int("mem")
	migSeconds := int(memMb) / options.HostOptions.MigrateExpectRate
	if migSeconds < options.HostOptions.MinMigrateTimeoutSeconds {
		migSeconds = options.HostOptions.MinMigrateTimeoutSeconds
	}
	s.timeoutAt = time.Now().Add(time.Second * time.Duration(migSeconds))
	log.Infof("migrate timeout seconds: %d now: %v expectfinial: %v", migSeconds, time.Now(), s.timeoutAt)
}

func (s *SGuestLiveMigrateTask) startMigrate(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateTask = nil
		hostutils.TaskFailed(s.ctx, fmt.Sprintf("Migrate set capability auto-converge error: %s", res))
		return
	}
	if s.params.EnableTLS {
		// https://wiki.qemu.org/Features/MigrationTLS
		s.Monitor.ObjectAdd("tls-creds-x509", map[string]string{
			"dir":         s.getPKIDirPath(),
			"endpoint":    "client",
			"id":          "tls0",
			"verify-peer": "no",
		}, func(res string) {
			if strings.Contains(strings.ToLower(res), "error") {
				s.migrateTask = nil
				hostutils.TaskFailed(s.ctx, fmt.Sprintf("Migrate add tls-creds-x509 object client tls0 error: %s", res))
				return
			}
			s.Monitor.MigrateSetParameter("tls-creds", "tls0", func(res string) {
				if strings.Contains(strings.ToLower(res), "error") {
					s.migrateTask = nil
					hostutils.TaskFailed(s.ctx, fmt.Sprintf("Migrate set tls-creds tls0 error: %s", res))
					return
				}
				s.doMigrate()
			})
		})
	} else {
		s.Monitor.MigrateSetParameter("tls-creds", "", func(res string) {
			if strings.Contains(strings.ToLower(res), "error") {
				s.migrateTask = nil
				hostutils.TaskFailed(s.ctx, fmt.Sprintf("Migrate set tls-creds to empty error: %s", res))
				return
			}
			s.doMigrate()
		})
	}
}

func (s *SGuestLiveMigrateTask) doMigrate() {
	var copyIncremental = false
	if s.params.IsLocal {
		// copy disk data
		copyIncremental = true
	}
	s.Monitor.Migrate(fmt.Sprintf("tcp:%s:%d", s.params.DestIp, s.params.DestPort),
		copyIncremental, false, s.startMigrateStatusCheck)
}

func (s *SGuestLiveMigrateTask) startMigrateStatusCheck(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateTask = nil
		hostutils.TaskFailed(s.ctx, fmt.Sprintf("Migrate error: %s", res))
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
				s.Monitor.GetMigrateStatus(s.onGetMigrateStatus)
			} else {
				log.Errorf("server %s(%s) migrate stopped unexpectedly", s.GetId(), s.GetName())
				s.migrateTask = nil
				hostutils.TaskFailed(s.ctx, fmt.Sprintf("Migrate error: %s", res))
				return
			}
		}
	}
}

func (s *SGuestLiveMigrateTask) onGetMigrateStatus(status string) {
	if status == "completed" {
		s.migrateComplete()
	} else if status == "failed" || status == "cancelled" {
		s.migrateTask = nil
		close(s.c)
		hostutils.TaskFailed(s.ctx, fmt.Sprintf("Query migrate got status: %s", status))
	} else if status == "migrate_disk_copy" {
		// do nothing, simply wait
	} else if status == "migrate_ram_copy" {
		if s.timeoutAt.IsZero() {
			s.startRamMigrateTimeout()
		} else if !s.doTimeoutMigrate && s.timeoutAt.Before(time.Now()) {
			log.Warningf("migrate timeout, force stop to finish migrate")
			// timeout, start memory postcopy
			// https://wiki.qemu.org/Features/PostCopyLiveMigration
			s.Monitor.SimpleCommand("stop", s.onMigrateStartPostcopy)
			s.doTimeoutMigrate = true
		}
	}
}

func (s *SGuestLiveMigrateTask) onMigrateStartPostcopy(res string) {
	if strings.Contains(strings.ToLower(res), "error") {
		s.migrateTask = nil
		close(s.c)
		hostutils.TaskFailed(s.ctx, fmt.Sprintf("onMigrateStartPostcopy error: %s", res))
		return
	} else {
		log.Infof("onMigrateStartPostcopy success")
	}
}

func (s *SGuestLiveMigrateTask) migrateComplete() {
	s.migrateTask = nil
	close(s.c)
	s.Monitor.Disconnect()
	s.Monitor = nil
	hostutils.TaskComplete(s.ctx, nil)
}

/**
 *  GuestResumeTask
**/

type SGuestResumeTask struct {
	*SKVMGuestInstance

	ctx       context.Context
	startTime time.Time

	isTimeout bool
	cleanTLS  bool

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
	log.Debugf("[%s] onConfirmRunning status %s", s.GetId(), status)
	if status == "running" || status == "paused (suspended)" || status == "paused (perlaunch)" {
		s.onStartRunning()
	} else if strings.Contains(status, "error") {
		// handle error first, results may be 'paused (internal-error)'
		s.taskFailed(status)
	} else if strings.Contains(status, "paused") {
		s.Monitor.GetBlocks(s.onGetBlockInfo)
	} else if status == "postmigrate" {
		s.resumeGuest()
	} else {
		memMb, _ := s.Desc.Int("mem")
		migSeconds := int(memMb) / options.HostOptions.MigrateExpectRate
		if migSeconds < options.HostOptions.MinMigrateTimeoutSeconds {
			migSeconds = options.HostOptions.MinMigrateTimeoutSeconds
		}
		log.Infof("start guest timeout seconds: %d", migSeconds)
		if s.isTimeout && time.Now().Sub(s.startTime) >= time.Second*time.Duration(migSeconds) {
			s.taskFailed("Timeout")
		} else {
			time.Sleep(time.Second * 3)
			s.confirmRunning()
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
	time.Sleep(time.Second * 1)
	s.resumeGuest()
}

func (s *SGuestResumeTask) resumeGuest() {
	s.startTime = time.Now()
	s.Monitor.SimpleCommand("cont", s.onResumeSucc)
}

func (s *SGuestResumeTask) onResumeSucc(res string) {
	s.confirmRunning()
}

func (s *SGuestResumeTask) SetGetTaskData(f func() (jsonutils.JSONObject, error)) {
	s.getTaskData = f
}

func (s *SGuestResumeTask) onStartRunning() {
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
	}
	if options.HostOptions.SetVncPassword {
		s.SetVncPassword()
	}
	s.OnResumeSyncMetadataInfo()
	s.optimizeOom()
	s.doBlockIoThrottle()
	if !options.HostOptions.DisableSetCgroup {
		timeutils2.AddTimeout(time.Second*5, s.SetCgroup)
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
	}
}

func (s *SGuestResumeTask) doBlockIoThrottle() {
	disks, _ := s.Desc.GetArray("disks")
	if len(disks) > 0 {
		bps, _ := disks[0].Int("bps")
		iops, _ := disks[0].Int("iops")
		if bps > 0 || iops > 0 {
			s.BlockIoThrottle(context.Background(), bps, iops)
		}
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

/**
 *  GuestStreamDisksTask
**/

type SGuestStreamDisksTask struct {
	*SKVMGuestInstance

	ctx      context.Context
	callback func()
	disksIdx []int

	c          chan struct{}
	streamDevs []string
}

func NewGuestStreamDisksTask(ctx context.Context, guest *SKVMGuestInstance, callback func(), disksIdx []int) *SGuestStreamDisksTask {
	return &SGuestStreamDisksTask{
		SKVMGuestInstance: guest,
		ctx:               ctx,
		callback:          callback,
		disksIdx:          disksIdx,
	}
}

func (s *SGuestStreamDisksTask) Start() {
	s.Monitor.GetBlockJobCounts(s.onInitCheckStreamJobs)
}

func (s *SGuestStreamDisksTask) onInitCheckStreamJobs(jobs int) {
	if jobs > 0 {
		log.Warningf("GuestStreamDisksTask: duplicate block streaming???")
		s.startWaitBlockStream("")
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
		s.Monitor.BlockStream(dev, len(s.disksIdx)-len(s.streamDevs), len(s.disksIdx), s.startWaitBlockStream)
	} else {
		s.taskComplete()
	}
}

func (s *SGuestStreamDisksTask) startWaitBlockStream(res string) {
	log.Infof("Block stream command res %s: %q", s.GetName(), res)
	s.c = make(chan struct{})
	for {
		select {
		case <-s.c:
			s.c = nil
			return
		case <-time.After(time.Second * 10):
			s.Monitor.GetBlockJobCounts(s.checkStreamJobs)
		}
	}
}

func (s *SGuestStreamDisksTask) checkStreamJobs(jobs int) {
	if jobs == 0 && s.c != nil {
		close(s.c)
		s.startDoBlockStream()
	}
}

func (s *SGuestStreamDisksTask) taskComplete() {
	hostutils.UpdateServerProgress(context.Background(), s.Id, 100.0, 0.0)
	s.SyncStatus("")

	// XXX: region disk post-migrate not implement

	// disks, _ := s.Desc.GetArray("disks")
	// var needSync = fale
	// for i, disk := range disks {
	// 	if disk.Contains("url") && disk.Contains("path") {
	// 		diskId, _ := disk.GetString("disk_id")
	// 		targetStroageId, _ := disk.GetString("target_storage_id")
	// 		params := jsonutils.NewDict()
	// 		params.Set("storage_id", jsonutils.NewString(targetStroageId))
	// 		modules.Disks.PerformAction(hostutils.GetComputeSession(context.Background()),
	// 			diskId, "post-migrate", params)
	// 		needSync = true
	// 	}
	// }
	// if needSync {
	// 	s.SaveDesc(s.Desc)
	// }

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
	if block.Inserted.File == s.disk.GetPath() {
		return block.Device
	}
	return ""
}

func (s *SGuestReloadDiskTask) startReloadDisk(device string) {
	s.doReloadDisk(device, s.onReloadSucc)
}

func (s *SGuestReloadDiskTask) doReloadDisk(device string, callback func(string)) {
	s.Monitor.SimpleCommand("stop", func(string) {
		s.Monitor.ReloadDiskBlkdev(device, s.disk.GetPath(), callback)
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
	disks, _ := s.Desc.GetArray("disks")
	if s.index < len(disks) {
		target := fmt.Sprintf("%s:exportname=drive_%d", s.nbdUri, s.index)
		s.Monitor.DriveMirror(s.startMirror, fmt.Sprintf("drive_%d", s.index),
			target, s.syncMode, true, blockReplication)
		s.index += 1
	} else {
		if s.onSucc != nil {
			s.onSucc()
		} else {
			hostutils.TaskComplete(s.ctx, nil)
		}
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

	memSlotNewIndex *int
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

		params := map[string]string{
			"id":       fmt.Sprintf("mem%d", *task.memSlotNewIndex),
			"size":     fmt.Sprintf("%dM", task.addMemSize),
			"mem-path": memPath,
			"share":    "on",
			"prealloc": "on",
		}
		task.Monitor.ObjectAdd("memory-backend-file", params, task.onAddMemObject)
	} else {
		params := map[string]string{
			"id":   fmt.Sprintf("mem%d", *task.memSlotNewIndex),
			"size": fmt.Sprintf("%dM", task.addMemSize),
		}
		task.Monitor.ObjectAdd("memory-backend-ram", params, task.onAddMemObject)
	}
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
	params := map[string]interface{}{
		"id":     fmt.Sprintf("dimm%d", *task.memSlotNewIndex),
		"memdev": fmt.Sprintf("mem%d", *task.memSlotNewIndex),
	}
	task.Monitor.DeviceAdd("pc-dimm", params, task.onAddMemDevice)
}

func (task *SGuestHotplugCpuMemTask) onAddMemDevice(reason string) {
	if len(reason) > 0 {
		task.onAddMemFailed(reason)
		return
	}
	task.onSucc()
}

func (task *SGuestHotplugCpuMemTask) onFail(reason string) {
	body := jsonutils.NewDict()
	if task.addedCpuCount < task.addCpuCount {
		body.Set("add_cpu_failed", jsonutils.JSONTrue)
		body.Set("added_cpu", jsonutils.NewInt(int64(task.addedCpuCount)))
	} else if task.memSlotNewIndex != nil {
		body.Set("add_mem_failed", jsonutils.JSONTrue)
	}
	hostutils.TaskFailed2(task.ctx, reason, body)
}

func (task *SGuestHotplugCpuMemTask) onSucc() {
	hostutils.TaskComplete(task.ctx, nil)
}

type SGuestBlockIoThrottleTask struct {
	*SKVMGuestInstance

	ctx  context.Context
	bps  int64
	iops int64
}

func (task *SGuestBlockIoThrottleTask) Start() error {
	task.findBlockDevices()
	return nil
}

func (task *SGuestBlockIoThrottleTask) findBlockDevices() {
	task.Monitor.GetBlocks(task.onBlockDriversSucc)
}

func (task *SGuestBlockIoThrottleTask) onBlockDriversSucc(blocks []monitor.QemuBlock) {
	drivers := make([]string, 0)
	for i := 0; i < len(blocks); i++ {
		if strings.HasPrefix(blocks[i].Device, "drive_") {
			drivers = append(drivers, blocks[i].Device)
		}
	}
	log.Infof("Drivers %s do io throttle bps %d iops %d", drivers, task.bps, task.iops)
	task.doIoThrottle(drivers)
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

func (task *SGuestBlockIoThrottleTask) doIoThrottle(drivers []string) {
	if len(drivers) == 0 {
		task.taskComplete(nil)
	} else {
		driver := drivers[0]
		drivers = drivers[1:]
		_cb := func(res string) {
			if len(res) > 0 {
				task.taskFail(res)
			} else {
				task.doIoThrottle(drivers)
			}
		}
		task.Monitor.BlockIoThrottle(driver, task.bps, task.iops, _cb)
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
		task.Monitor.CancelBlockJob(driver, true, onCancelBlockJob)
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
	guest  *SKVMGuestInstance
	params *SStorageCloneDisk
}

func NewGuestStorageCloneDiskTask(guest *SKVMGuestInstance, params *SStorageCloneDisk) *SGuestStorageCloneDiskTask {
	return &SGuestStorageCloneDiskTask{
		guest:  guest,
		params: params,
	}
}

func (t *SGuestStorageCloneDiskTask) Start(ctx context.Context) {
	resp, err := t.params.TargetStorage.CloneDiskFromStorage(ctx, t.params.SourceStorage, t.params.SourceDisk, t.params.TargetDiskId)
	if err != nil {
		hostutils.TaskFailed(
			ctx,
			errors.Wrapf(err, "Clone disk %s to storage %s", t.params.SourceDisk.GetPath(), t.params.TargetStorage.GetId()).Error())
		return
	}
	hostutils.TaskComplete(ctx, jsonutils.Marshal(resp))
}
