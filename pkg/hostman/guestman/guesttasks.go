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

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
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

	callback func(...error)
}

func NewGuestDiskSyncTask(guest *SKVMGuestInstance, delDisks, addDisks []jsonutils.JSONObject, cdrom *string) *SGuestDiskSyncTask {
	return &SGuestDiskSyncTask{guest, delDisks, addDisks, cdrom, nil}
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
	d.callback()
}

func (d *SGuestDiskSyncTask) changeCdrom() {
	d.guest.Monitor.GetBlocks(d.onGetBlockInfo)
}

func (d *SGuestDiskSyncTask) onGetBlockInfo(results *jsonutils.JSONArray) {
	var cdName string
	for _, r := range results.Value() {
		device, _ := r.GetString("device")
		if regexp.MustCompile(`^ide\d+-cd\d+$`).MatchString(device) {
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
		dev           = d.guest.GetDiskDeviceModel(diskDirver)
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
		return
	} else if len(n.addNics) > 0 {
		nic := n.addNics[len(n.addNics)-1]
		n.addNics = n.addNics[:len(n.addNics)-1]
		n.addNic(nic)
		return
	} else {
		n.callback()
	}
}

func (n *SGuestNetworkSyncTask) removeNic(nic jsonutils.JSONObject) {
	// pass not implement
}

func (n *SGuestNetworkSyncTask) addNic(nic jsonutils.JSONObject) {
	// pass not implement
}

func NewGuestNetworkSyncTask(guest *SKVMGuestInstance, delNics, addNics []jsonutils.JSONObject) *SGuestNetworkSyncTask {
	return &SGuestNetworkSyncTask{guest, delNics, addNics, make([]error, 0), nil}
}

/**
 *  GuestLiveMigrateTask
**/

type SGuestLiveMigrateTask struct {
	*SKVMGuestInstance

	ctx    context.Context
	params *SLiveMigrate

	c chan struct{}
}

func NewGuestLiveMigrateTask(
	ctx context.Context, guest *SKVMGuestInstance, params *SLiveMigrate,
) *SGuestLiveMigrateTask {
	return &SGuestLiveMigrateTask{SKVMGuestInstance: guest, ctx: ctx, params: params}
}

func (s *SGuestLiveMigrateTask) Start() {
	s.Monitor.MigrateSetCapability("zero-blocks", "on", s.startMigrate)
}

func (s *SGuestLiveMigrateTask) startMigrate(string) {
	var copyIncremental = false
	if s.params.IsLocal {
		copyIncremental = true
	}
	s.Monitor.Migrate(fmt.Sprintf("tcp:%s:%d", s.params.DestIp, s.params.DestPort),
		copyIncremental, false, s.startMigrateStatusCheck)
}

func (s *SGuestLiveMigrateTask) startMigrateStatusCheck(string) {
	s.c = make(chan struct{})
	for {
		select {
		case <-s.c: // on c close
			break
		case <-time.After(time.Second * 1):
			s.Monitor.GetMigrateStatus(s.onGetMigrateStatus)
		}
	}
}

func (s *SGuestLiveMigrateTask) onGetMigrateStatus(status string) {
	if status == "completed" {
		close(s.c)
		hostutils.TaskComplete(s.ctx, nil)
	} else if status == "failed" {
		close(s.c)
		hostutils.TaskFailed(s.ctx, fmt.Sprintf("Query migrate got status: %s", status))
	}
}

/**
 *  GuestResumeTask
**/

type SGuestResumeTask struct {
	*SKVMGuestInstance

	ctx       context.Context
	startTime time.Time
}

func NewGuestResumeTask(ctx context.Context, s *SKVMGuestInstance) *SGuestResumeTask {
	return &SGuestResumeTask{
		SKVMGuestInstance: s,
		ctx:               ctx,
	}
}

func (s *SGuestResumeTask) Start() {
	s.startTime = time.Now()
	s.confirmRunning()
}

func (s *SGuestResumeTask) Stop() {
	// TODO
	// stop stream disk
}

func (s *SGuestResumeTask) confirmRunning() {
	s.Monitor.QueryStatus(s.onConfirmRunning)
}

func (s *SGuestResumeTask) onConfirmRunning(status string) {
	if status == "running" || status == "paused (suspended)" || status == "paused (perlaunch)" {
		s.onStartRunning()
	} else if strings.Contains(status, "error") {
		// handle error first, results may be 'paused (internal-error)'
		s.taskFailed(status)
	} else if strings.Contains(status, "paused") {
		s.Monitor.GetBlocks(s.onGetBlockInfo)
	} else {
		if time.Now().Sub(s.startTime) >= time.Second*60 {
			s.taskFailed("Timeout")
		} else {
			time.Sleep(time.Second * 1)
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
		s.SyncStatus()
	}
}

func (s *SGuestResumeTask) onGetBlockInfo(results *jsonutils.JSONArray) {
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

func (s *SGuestResumeTask) onStartRunning() {
	// s.removeStatefile() XXX 可能不用了，先注释了
	if s.ctx != nil && len(appctx.AppContextTaskId(s.ctx)) > 0 {
		hostutils.TaskComplete(s.ctx, nil)
	}
	if options.HostOptions.SetVncPassword {
		s.SetVncPassword()
	}
	s.SyncMetadataInfo()
	s.SyncStatus()
	timeutils2.AddTimeout(time.Second*5, s.SetCgroup)
	disksIdx := s.GetNeedMergeBackingFileDiskIndexs()
	if len(disksIdx) > 0 {
		timeutils2.AddTimeout(time.Second*5, func() { s.startStreamDisks(disksIdx) })
	} else if options.HostOptions.AutoMergeBackingTemplate {
		timeutils2.AddTimeout(
			time.Second*time.Duration(options.HostOptions.AutoMergeDelaySeconds),
			func() { s.startStreamDisks(nil) })
	}
}

func (s *SGuestResumeTask) startStreamDisks(disksIdx []int) {
	s.startTime = time.Time{}
	s.CleanStartupTask()
	if s.IsMonitorAlive() {
		s.StreamDisks(s.ctx, func() { s.onStreamComplete(disksIdx) }, disksIdx)
	}
}

func (s *SGuestResumeTask) onStreamComplete(disksIdx []int) {
	if len(disksIdx) == 0 {
		s.SyncStatus()
	} else {
		s.streamDisksComplete(s.ctx)
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

func (s *SGuestStreamDisksTask) onBlockDrivesSucc(res *jsonutils.JSONArray) {
	streamDevs := []string{}
	drvs, _ := res.GetArray()
	for _, drv := range drvs {
		device, err := drv.GetString("device")
		if err != nil {
			log.Errorln(err)
			continue
		}
		inserted, err := drv.Get("inserted")
		if err != nil && inserted.Contains("file") && inserted.Contains("backing_file") {
			var stream = false
			idx := device[len(device)-1] - '0'
			for i := 0; i < len(s.disksIdx); i++ {
				if int(idx) == s.disksIdx[i] {
					stream = true
				}
			}
			if !stream {
				continue
			}
			streamDevs = append(streamDevs, device)
		}
	}
	s.streamDevs = streamDevs
	if len(streamDevs) == 0 {
		s.taskComplete()
	} else {
		s.SyncStatus()
		s.startDoBlockStream()
	}
}

func (s *SGuestStreamDisksTask) startDoBlockStream() {
	if len(s.streamDevs) > 0 {
		dev := s.streamDevs[0]
		s.streamDevs = s.streamDevs[1:]
		s.Monitor.BlockStream(dev, s.startWaitBlockStream)
	}
}

func (s *SGuestStreamDisksTask) startWaitBlockStream(res string) {
	if s.c == nil {
		s.c = make(chan struct{})
		for {
			select {
			case <-s.c:
				s.c = nil
				return
			case <-time.After(time.Second * 1):
				s.Monitor.GetBlockJobCounts(s.checkStreamJobs)
			}
		}
	}
}

func (s *SGuestStreamDisksTask) checkStreamJobs(jobs int) {
	if jobs == 0 {
		if len(s.streamDevs) == 0 {
			close(s.c)
			s.taskComplete()
		} else {
			close(s.c)
		}
	}
}

func (s *SGuestStreamDisksTask) taskComplete() {
	s.SyncStatus()

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
		if _, ok := storageman.DELETEING_SNAPSHOTS[s.disk.GetId()]; ok {
			time.Sleep(time.Second * 1)
		}
	}

	callback()
	return nil
}

func (s *SGuestReloadDiskTask) Start() {
	s.fetchDisksInfo(s.startReloadDisk)
}

func (s *SGuestReloadDiskTask) fetchDisksInfo(callback func(string)) {
	s.Monitor.GetBlocks(func(res *jsonutils.JSONArray) { s.onGetBlocksSucc(res, callback) })
}

func (s *SGuestReloadDiskTask) onGetBlocksSucc(res *jsonutils.JSONArray, callback func(string)) {
	var device string
	devs, _ := res.GetArray()
	for _, d := range devs {
		device = s.getDiskOfDrive(d)
		if len(device) > 0 {
			callback(device)
			break
		}
	}

	if len(device) == 0 {
		s.taskFailed("Device not found")
	}
}

func (s *SGuestReloadDiskTask) getDiskOfDrive(d jsonutils.JSONObject) string {
	inserted, err := d.Get("inserted")
	if err != nil {
		return ""
	}
	file, err := inserted.GetString("file")
	if err != nil {
		return ""
	}
	if file == s.disk.GetPath() {
		drive, _ := d.GetString("device")
		return drive
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
	hostutils.TaskComplete(s.ctx, nil)
}

func (s *SGuestReloadDiskTask) taskFailed(reason string) {
	log.Errorf("SGuestReloadDiskTask error: %s", reason)
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
		log.Errorf("Monitor reload blkdev error: %s", res)
		cb = s.onSnapshotBlkdevFail
	}
	s.Monitor.SimpleCommand("cont", cb)
}

func (s *SGuestDiskSnapshotTask) onSnapshotBlkdevFail(string) {
	snapshotDir := s.disk.GetSnapshotDir()
	snapshotPath := path.Join(snapshotDir, s.snapshotId)
	_, err := procutils.NewCommand("rm", "-rf", snapshotPath).Run()
	if err != nil {
		log.Errorln(err)
	}
	hostutils.TaskFailed(s.ctx, "Reload blkdev error")
}

func (s *SGuestDiskSnapshotTask) onResumeSucc(res string) {
	log.Infof("guest disk snapshot task resume succ %s", res)
	snapshotDir := s.disk.GetSnapshotDir()
	snapshotLocation := path.Join(snapshotDir, s.snapshotId)
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
	if err = img.Convert2Qcow2To(convertedDisk, true); err != nil {
		log.Errorln(err)
		if fileutils2.Exists(convertedDisk) {
			os.Remove(convertedDisk)
		}
		return err
	}

	s.tmpPath = snapshotPath + ".swap"
	if _, err := procutils.NewCommand("mv", "-f", snapshotPath, s.tmpPath).Run(); err != nil {
		log.Errorln(err)
		if fileutils2.Exists(s.tmpPath) {
			procutils.NewCommand("mv", "-f", s.tmpPath, snapshotPath).Run()
		}
		return err
	}
	if _, err := procutils.NewCommand("mv", "-f", convertedDisk, snapshotPath).Run(); err != nil {
		log.Errorln(err)
		if fileutils2.Exists(s.tmpPath) {
			procutils.NewCommand("mv", "-f", s.tmpPath, snapshotPath).Run()
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
		log.Errorln("Reload blkdev failed: %s", err)
		callback = s.onSnapshotBlkdevFail
	}
	s.Monitor.SimpleCommand("cont", callback)
}

func (s *SGuestSnapshotDeleteTask) onSnapshotBlkdevFail(res string) {
	snapshotPath := path.Join(s.disk.GetSnapshotDir(), s.convertSnapshot)
	if _, err := procutils.NewCommand("rm", "-f", s.tmpPath, snapshotPath).Run(); err != nil {
		log.Errorln(err)
	}
	s.taskFailed("Reload blkdev failed")
}

func (s *SGuestSnapshotDeleteTask) onResumeSucc(res string) {
	log.Infof("guest do new snapshot task resume succ %s", res)
	if len(s.tmpPath) > 0 {
		_, err := procutils.NewCommand("rm", "-f", s.tmpPath).Run()
		if err != nil {
			log.Errorln(err)
		}
	}
	if !s.pendingDelete {
		snapshotDir := s.disk.GetSnapshotDir()
		procutils.NewCommand("rm", "-f", path.Join(snapshotDir, s.deleteSnapshot))
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

	ctx      context.Context
	nbdUri   string
	onSucc   func()
	syncMode string
	index    int
}

func NewDriveMirrorTask(
	ctx context.Context, s *SKVMGuestInstance, nbdUri, syncMode string, onSucc func(),
) *SDriveMirrorTask {
	return &SDriveMirrorTask{
		SKVMGuestInstance: s,
		ctx:               ctx,
		nbdUri:            nbdUri,
		syncMode:          syncMode,
		onSucc:            onSucc,
	}
}

func (s *SDriveMirrorTask) Start() {
	s.startMirror("")
}

func (s *SDriveMirrorTask) startMirror(res string) {
	log.Infof("drive mirror results:%s", res)
	if len(res) > 0 {
		hostutils.TaskFailed(s.ctx, res)
		return
	}
	disks, _ := s.Desc.GetArray("disks")
	if s.index < len(disks) {
		if s.index >= 1 { // data disk
			s.syncMode = "none"
		}
		target := fmt.Sprintf("%s:exportname=drive_%d", s.nbdUri, s.index)
		s.Monitor.DriveMirror(s.startMirror, fmt.Sprintf("drive_%d", s.index),
			target, s.syncMode, true)
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

func (task *SGuestOnlineResizeDiskTask) OnGetBlocksSucc(results *jsonutils.JSONArray) {
	for i := 0; i < results.Size(); i += 1 {
		result, _ := results.GetAt(i)
		fileStr, _ := result.GetString("inserted", "file")
		if len(fileStr) > 0 && strings.HasSuffix(fileStr, task.diskId) {
			driveName, _ := result.GetString("device")
			task.Monitor.ResizeDisk(driveName, task.sizeMB, task.OnResizeSucc)
			return
		}
	}
	hostutils.TaskFailed(task.ctx, fmt.Sprintf("disk %s not found on this guest", task.diskId))
}

func (task *SGuestOnlineResizeDiskTask) OnResizeSucc(result string) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(task.sizeMB), "disk_size")
	hostutils.TaskComplete(task.ctx, params)
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
	params := map[string]string{
		"id":   fmt.Sprintf("mem%d", *task.memSlotNewIndex),
		"size": fmt.Sprintf("%dM", task.addMemSize),
	}
	task.Monitor.ObjectAdd("memory-backend-ram", params, task.onAddMemObject)
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
