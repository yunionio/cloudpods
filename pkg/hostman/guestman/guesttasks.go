package guestman

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

type IGuestTasks interface {
	Start(func(...error))
}

type SGuestStopTask struct {
	*SKVMGuestInstance
	ctx            context.Context
	timeout        int64
	startPowerdown *time.Time
}

func NewGuestStopTask(guest *SKVMGuestInstance, ctx context.Context, timeout int64) *SGuestStopTask {
	return &SGuestStopTask{
		SKVMGuestInstance: guest,
		ctx:               ctx,
		timeout:           timeout,
		startPowerdown:    nil,
	}
}

func (s *SGuestStopTask) Start() {
	if s.IsRunning() && s.IsMonitorAlive() {
		// Do Powerdown,
		s.Monitor.SimpleCommand("system_powerdown", s.onPowerdownGuest)
	} else {
		s.CheckGuestRunningLater()
	}
}

func (s *SGuestStopTask) onPowerdownGuest(results string) {
	s.ExitCleanup(true)
	s.startPowerdown = &time.Now()
	s.checkGuestRunning()
}

func (s *SGuestStopTask) checkGuestRunning() {
	if !s.IsRunning() || time.Now().Sub(*s.startPowerdown) > (s.timeout*time.Second) {
		s.Stop() // force stop
		hostutils.TaskComplete(s.ctx, nil)
	} else {
		s.CheckGuestRunningLater()
	}
}

func (s *SGuestStopTask) CheckGuestRunningLater() {
	time.Sleep(time.Second * 1)
	s.checkGuestRunning()
}

type SGuestSyncConfigTaskExecutor struct {
	ctx   context.Context
	guest *SKVMGuestInstance
	tasks IGuestTasks

	errors   []error
	callback func([]error)
}

func NewGuestSyncConfigTaskExecutor(ctx context.Context, guest *SKVMGuestInstance, tasks []IGuestTasks, callback func(error)) *SGuestSyncConfigTaskExecutor {
	return &SGuestSyncConfigTaskExecutor{ctx, guest, tasks, make([]error, 0), callback}
}

func (t *SGuestSyncConfigTaskExecutor) Start(delay int) {
	timeutils2.AddTimeout(1*time.Second, t.runNextTask)
}

func (t *SGuestSyncConfigTaskExecutor) runNextTask() {
	if len(t.tasks) > 0 {
		task := t.tasks[len(t.tasks-1)]
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

type SGuestDiskSyncTask struct {
	guest    *SKVMGuestInstance
	delDisks []jsonutils.JSONObject
	addDisks []jsonutils.JSONObject
	cdrom    *string

	callback func(error)
}

func NewGuestDiskSyncTask(guest *SKVMGuestInstance, delDisks, addDisks []jsonutils.JSONObject, cdrom *string) *SGuestDiskSyncTask {
	return &SGuestDiskSyncTask{guest, delDisks, addDisks, cdrom}
}

func (d *SGuestDiskSyncTask) Start(callback func(error)) {
	d.callback = callback
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) syncDisksConf() {
	if len(d.delDisks) > 0 {
		disk = d.delDisks[len(d.delDisks)-1]
		d.delDisks = d.delDisks[:len(d.delDisks)-1]
		d.removeDisk(disk)
		return
	}
	if len(d.addDisks) > 0 {
		disk = d.addDisks[len(d.addDisks)-1]
		d.addDisks = d.addDisks[:len(d.addDisks)-1]
		d.addDisk(disk)
		return
	}
	if d.cdrom != nil {
		d.changeCdrom()
		return
	}
	d.callback(nil)
}

func (d *SGuestDiskSyncTask) changeCdrom() {
	d.guest.Monitor.GetBlocks(d.onGetBlockInfo)
}

func (d *SGuestDiskSyncTask) onGetBlockInfo(results *jsonutils.JSONArray) {
	var cdName string
	for _, r := range results.Value() {
		device, _ := r.GetString("device")
		if regexp.MustCompile(`^ide\d+-cd\d+$`, device) {
			cdName = device
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
	devId = fmt.Sprintf("drive_%s", index)
	d.guest.Monitor.DriveDel(devId,
		func(results string) { d.onRemoveDriveSucc(devId, results) })
}

func (d *SGuestDiskSyncTask) onRemoveDriveSucc(devId, results string) {
	d.guest.Monitor.DeviceDel(devId, d.onRemoveDiskSucc)
}

func (d *SGuestDiskSyncTask) onRemoveDiskSucc(results string) {
	d.syncDisksConf()
}

func (d *SGuestDiskSyncTask) addDisk(disk jsonutils.JSONObject) {
	diskPath, _ := disk.GetString("path")
	iDisk := storageman.GetManager().GetDiskByPath(diskPath)
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
		"id":    fmt.Sprintf("drive_%s", diskIndex),
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
		dev           = d.guest.GetDiskDeviceModel(diskDirver)
	)

	var params = map[string]interface{}{
		"drive": fmt.Sprintf("drive_%s", diskIndex),
		"id":    fmt.Sprintf("drive_%s", diskIndex),
	}

	if diskDirver == DISK_DRIVER_VIRTIO {
		params["addr"] = fmt.Sprintf("0x%x", d.guest.GetDiskAddr(diskIndex))
	} else if DISK_DRIVER_IDE == diskDirver {
		params["unit"] = diskIndex % 2
	}
	d.guest.Monitor.DeviceAdd(dev, params, d.onAddDeviceSucc)
}

func (d *SGuestDiskSyncTask) onAddDeviceSucc(results string) {
	d.syncDisksConf()
}

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
		nic = n.delNics[len(n.delNics)-1]
		n.delNics = n.delNics[:len(n.delNics)-1]
		n.removeNic(nic)
		return
	} else if len(n.addNics) > 0 {
		nic = n.addNics[len(n.addNics)-1]
		n.addNics = n.addNics[:len(n.addNics)-1]
		n.addNic(nic)
		return
	} else {
		n.callback()
	}
}

func (n *SGuestNetworkSyncTask) removeNic(nic jsonutils.JSONObject) {
	// pass
}

func (n *SGuestNetworkSyncTask) addNic(nic jsonutils.JSONObject) {
	// pass
}

func NewGuestNetworkSyncTask(guest *SKVMGuestInstance, delNics, addNics jsonutils.JSONObject) *SGuestNetworkSyncTask {
	return &SGuestNetworkSyncTask{guest, delNics, addNics, make(error, 0)}
}
