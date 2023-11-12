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

package monitor

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/regutils2"
)

type HmpMonitor struct {
	SBaseMonitor

	commandQueue  []string
	callbackQueue []StringCallback
}

func NewHmpMonitor(server, sid string, OnMonitorDisConnect, OnMonitorTimeout MonitorErrorFunc, OnMonitorConnected MonitorSuccFunc) *HmpMonitor {
	return &HmpMonitor{
		SBaseMonitor:  *NewBaseMonitor(server, sid, OnMonitorConnected, OnMonitorDisConnect, OnMonitorTimeout),
		commandQueue:  make([]string, 0),
		callbackQueue: make([]StringCallback, 0),
	}
}

var hmpMark = []byte("(qemu) ")

func (m *HmpMonitor) hmpSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	index := bytes.Index(data, hmpMark)
	if index >= 0 {
		return index + len(hmpMark), data[0:index], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func (m *HmpMonitor) actionResult(res string) string {
	return res
}

func (m *HmpMonitor) read(r io.Reader) {
	if !m.checkReading() {
		return
	}
	scanner := bufio.NewScanner(r)
	scanner.Split(m.hmpSplitFunc)
	for scanner.Scan() {
		res := scanner.Text()
		if len(res) == 0 {
			continue
		}
		log.Infof("HMP Read %s: %s", m.server, res)
		if m.connected {
			go m.callBack(res)
		} else {
			// remove reader timeout
			m.connected = true
			m.timeout = false
			m.rwc.SetReadDeadline(time.Time{})
			go m.query()
			go m.OnMonitorConnected()
		}
	}
	log.Infof("Scan over %s ...", m.server)
	err := scanner.Err()
	if err != nil {
		log.Infof("HMP Disconnected %s: %s", m.server, err)
	}
	if m.timeout {
		m.OnMonitorTimeout(err)
	} else if m.connected {
		m.connected = false
		m.OnMonitorDisConnect(err)
	}
	m.reading = false
}

func (m *HmpMonitor) callBack(res string) {
	m.mutex.Lock()
	if len(m.callbackQueue) == 0 {
		m.mutex.Unlock()
		return
	}
	cb := m.callbackQueue[0]
	m.callbackQueue = m.callbackQueue[1:]
	m.mutex.Unlock()
	if cb != nil {
		pos := strings.Index(res, "\r\n")
		if pos > 0 {
			res = res[pos+2:]
		}
		go cb(res)
	}
}

func (m *HmpMonitor) write(cmd []byte) error {
	cmd = append(cmd, '\n')
	log.Infof("HMP Write %s: %s", m.server, string(cmd))
	length, index := len(cmd), 0
	for index < length {
		i, err := m.rwc.Write(cmd)
		if err != nil {
			return err
		}
		index += i
	}
	return nil
}

func (m *HmpMonitor) query() {
	if !m.checkWriting() {
		return
	}
	for {
		if len(m.commandQueue) == 0 {
			break
		}
		//pop
		m.mutex.Lock()
		cmd := m.commandQueue[0]
		m.commandQueue = m.commandQueue[1:]
		err := m.write([]byte(cmd))
		m.mutex.Unlock()
		if err != nil {
			log.Errorf("Write %s to monitor error %s: %s", cmd, m.server, err)
			break
		}
	}
	m.writing = false
}

func (m *HmpMonitor) Query(cmd string, cb StringCallback) {
	// push
	m.mutex.Lock()
	m.commandQueue = append(m.commandQueue, cmd)
	m.callbackQueue = append(m.callbackQueue, cb)
	m.mutex.Unlock()
	if m.connected {
		if !m.writing {
			go m.query()
		}
		if !m.reading {
			go m.read(m.rwc)
		}
	}
}

func (m *HmpMonitor) ConnectWithSocket(address string) error {
	err := m.SBaseMonitor.connect("unix", address)
	if err != nil {
		return err
	}
	go m.read(m.rwc)
	return nil
}

func (m *HmpMonitor) Connect(host string, port int) error {
	err := m.SBaseMonitor.connect("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	go m.read(m.rwc)
	return nil
}

func (m *HmpMonitor) QueryStatus(callback StringCallback) {
	m.Query("info status", m.parseStatus(callback))
}

func (m *HmpMonitor) SimpleCommand(cmd string, callback StringCallback) {
	m.Query(cmd, callback)
}

func (m *HmpMonitor) HumanMonitorCommand(cmd string, callback StringCallback) {
	m.Query(cmd, callback)
}

func (m *HmpMonitor) GetVersion(callback StringCallback) {
	_cb := func(versionStr string) {
		versionStr = strings.TrimSpace(versionStr)
		callback(versionStr)
	}
	m.Query("info version", _cb)
}

func (m *HmpMonitor) GetBlocks(callback func([]QemuBlock)) {
	var cb = func(output string) {
		var lines = strings.Split(strings.TrimSuffix(output, "\r\n"), "\r\n")
		var mergedOutput = []string{}

		// merge output
		for _, line := range lines {
			parts := regexp.MustCompile(`\s+`).Split(line, -1)
			if strings.HasSuffix(parts[0], ":") {
				mergedOutput = append(mergedOutput, "")
			} else if regexp.MustCompile(`\(#block\d+\):`).MatchString(line) {
				mergedOutput = append(mergedOutput, "")
			}
			mergedOutput[len(mergedOutput)-1] = mergedOutput[len(mergedOutput)-1] + " " + line
			mergedOutput[len(mergedOutput)-1] = strings.TrimSpace(mergedOutput[len(mergedOutput)-1])
		}

		// parse to json
		ret := []QemuBlock{}
		for _, line := range mergedOutput {
			parts := regexp.MustCompile(`\s+`).Split(line, -1)
			if strings.HasSuffix(parts[0], ":") ||
				regexp.MustCompile(`\(#block\d+\):`).MatchString(parts[1]) {
				block := QemuBlock{}
				if strings.HasSuffix(parts[0], ":") {
					block.Device = parts[0][:len(parts[0])-1]
				} else {
					block.Device = parts[0]
				}
				if regexp.MustCompile(`\(#block\d+\):`).MatchString(parts[1]) {
					block.Inserted.File = parts[2]
					for i := 0; i < len(parts)-2; i++ {
						if parts[i] == "Backing" && parts[i+1] == "file:" {
							block.Inserted.BackingFile = parts[i+2]
							break
						}
					}
				}
				ret = append(ret, block)
			}
		}
		callback(ret)
	}

	m.Query("info block", cb)
}

func (m *HmpMonitor) EjectCdrom(dev string, callback StringCallback) {
	m.Query(fmt.Sprintf("eject -f %s", dev), callback)
}

func (m *HmpMonitor) ChangeCdrom(dev string, path string, callback StringCallback) {
	m.Query(fmt.Sprintf("change %s %s", dev, path), callback)
}

func (m *HmpMonitor) DriveDel(idstr string, callback StringCallback) {
	m.Query(fmt.Sprintf("drive_del %s", idstr), callback)
}

func (m *HmpMonitor) DeviceDel(idstr string, callback StringCallback) {
	m.Query(fmt.Sprintf("device_del %s", idstr), callback)
}

func (m *HmpMonitor) ObjectDel(idstr string, callback StringCallback) {
	m.Query(fmt.Sprintf("object_del %s", idstr), callback)
}

func (m *HmpMonitor) XBlockdevChange(parent, node, child string, callback StringCallback) {
	go callback("hmp not support command x-blockdev-change")
}

func (m *HmpMonitor) DriveAdd(bus, node string, params map[string]string, callback StringCallback) {
	var paramsKvs = []string{}
	for k, v := range params {
		paramsKvs = append(paramsKvs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd := "drive_add"
	if len(node) > 0 {
		cmd = fmt.Sprintf("drive_add -n %s", node)
	}
	m.Query(fmt.Sprintf("%s %s %s", cmd, bus, strings.Join(paramsKvs, ",")), callback)
}

func (m *HmpMonitor) DeviceAdd(dev string, params map[string]string, callback StringCallback) {
	var paramsKvs = []string{}
	for k, v := range params {
		paramsKvs = append(paramsKvs, fmt.Sprintf("%s=%s", k, v))
	}
	m.Query(fmt.Sprintf("device_add %s,%s", dev, strings.Join(paramsKvs, ",")), callback)
}

func (m *HmpMonitor) MigrateSetDowntime(dtSec float64, callback StringCallback) {
	m.Query(fmt.Sprintf("migrate_set_downtime %f", dtSec), callback)
}

func (m *HmpMonitor) MigrateSetCapability(capability, state string, callback StringCallback) {
	m.Query(fmt.Sprintf("migrate_set_capability %s %s", capability, state), callback)
}

func (m *HmpMonitor) MigrateSetParameter(key string, val interface{}, callback StringCallback) {
	cmd := fmt.Sprintf("migrate_set_parameter %s %s", key, val)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) MigrateIncoming(address string, callback StringCallback) {
	cmd := fmt.Sprintf("migrate_incoming %s", address)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) MigrateContinue(state string, callback StringCallback) {
	cmd := fmt.Sprintf("migrate_continue %s", state)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) Migrate(
	destStr string, copyIncremental, copyFull bool, callback StringCallback,
) {
	cmd := "migrate -d"
	if copyIncremental {
		cmd += " -i"
	} else if copyFull {
		cmd += " -b"
	}
	cmd += " " + destStr
	m.Query(cmd, callback)
}

func (m *HmpMonitor) GetMigrateStatus(callback StringCallback) {
	cb := func(output string) {
		log.Infof("Query migrate status %s: %s", m.server, output)

		var status string
		for _, line := range strings.Split(strings.TrimSuffix(output, "\r\n"), "\r\n") {
			if strings.HasPrefix(line, "Migration status") {
				status = line[strings.LastIndex(line, " ")+1:]
				break
			}
		}
		callback(status)
	}

	m.Query("info migrate", cb)
}

func (m *HmpMonitor) GetMigrateStats(callback MigrateStatsCallback) {
	go callback(nil, errors.Errorf("unsupport get migrate stats"))
}

func (m *HmpMonitor) MigrateCancel(cb StringCallback) {
	m.Query("migrate_cancel", cb)
}

func (m *HmpMonitor) MigrateStartPostcopy(callback StringCallback) {
	cb := func(output string) {
		log.Infof("MigrateStartPostcopy %s: %s", m.server, output)
		callback(output)
	}
	m.Query("migrate_start_postcopy", cb)
}

func (m *HmpMonitor) GetBlockJobCounts(callback func(jobs int)) {
	cb := func(output string) {
		lines := strings.Split(strings.TrimSuffix(output, "\r\n"), "\r\n")
		if lines[0] == "No active jobs" {
			callback(0)
		} else {
			callback(len(lines))
		}
	}

	m.Query("info block-jobs", cb)
}

func (m *HmpMonitor) GetBlockJobs(callback func([]BlockJob)) {
	cb := func(output string) {
		lines := strings.Split(strings.TrimSuffix(output, "\r\n"), "\r\n")
		if lines[0] == "No active jobs" {
			callback(nil)
			return
		}
		jobs := []BlockJob{}
		re := regexp.MustCompile(`Type (?P<type>\w+), device (?P<device>\w+)`)
		for i := 0; i < len(lines); i++ {
			m := regutils2.GetParams(re, lines[i])
			if len(m) > 0 {
				job := BlockJob{}
				job.Type, _ = m["type"]
				job.Device, _ = m["device"]
				jobs = append(jobs, job)
			}
		}
		callback(jobs)
	}
	m.Query("info block-jobs", cb)
}

func (m *HmpMonitor) ReloadDiskBlkdev(device, path string, callback StringCallback) {
	m.Query(fmt.Sprintf("reload_disk_snapshot_blkdev -n %s %s", device, path), callback)
}

func (m *HmpMonitor) DriveMirror(callback StringCallback, drive, target, syncMode, format string, unmap, blockReplication bool, speed int64) {
	cmd := "drive_mirror -n"
	if blockReplication {
		cmd += " -c"
	}
	if syncMode == "full" {
		cmd += " -f"
	}
	cmd += fmt.Sprintf(" %s %s %s", drive, target, format)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) DriveBackup(callback StringCallback, drive, target, syncMode, format string) {
	cmd := "drive_backup -n"
	if syncMode == "full" {
		cmd += " -f"
	}
	cmd += fmt.Sprintf(" %s %s %s", drive, target, format)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) BlockStream(drive string, callback StringCallback) {
	var (
		speed = 500 // limit 500 MB/s
		cmd   = fmt.Sprintf("block_stream %s %d", drive, speed)
	)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) BlockJobComplete(drive string, callback StringCallback) {
	m.Query("block_job_complete", callback)
}

func (m *HmpMonitor) BlockReopenImage(drive, newImagePath, format string, cb StringCallback) {
	m.Query(fmt.Sprintf("block_reopen_image %s %s %s", drive, newImagePath, format), cb)
}

func (m *HmpMonitor) SnapshotBlkdev(drive, newImagePath, format string, reuse bool, cb StringCallback) {
	var cmd = "snapshot_blkdev"
	if reuse {
		cmd += " -n"
	}
	cmd += fmt.Sprintf(" %s %s %s", drive, newImagePath, format)
	m.Query(cmd, cb)
}

func (m *HmpMonitor) SetVncPassword(proto, password string, callback StringCallback) {
	if len(password) > 8 {
		password = password[:8]
	}
	m.Query(fmt.Sprintf("set_password %s %s", proto, password), callback)
}

func (m *HmpMonitor) StartNbdServer(port int, exportAllDevice, writable bool, callback StringCallback) {
	var cmd = "nbd_server_start"
	if exportAllDevice {
		cmd += " -a"
	}
	if writable {
		cmd += " -w"
	}
	cmd += fmt.Sprintf(" 0.0.0.0:%d", port)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) StopNbdServer(callback StringCallback) {
	m.Query("nbd_server_stop", callback)
}

func (m *HmpMonitor) ResizeDisk(driveName string, sizeMB int64, callback StringCallback) {
	cmd := fmt.Sprintf("block_resize %s %d", driveName, sizeMB)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) GetCpuCount(callback func(count int)) {
	var cb = func(output string) {
		cpus := strings.Split(strings.TrimSuffix(output, "\r\n"), "\r\n")
		callback(len(cpus))
	}
	m.Query("info cpus", cb)
}

func (m *HmpMonitor) AddCpu(cpuIndex int, callback StringCallback) {
	m.Query(fmt.Sprintf("cpu-add %d", cpuIndex), callback)
}

func (m *HmpMonitor) GeMemtSlotIndex(callback func(index int)) {
	var cb = func(output string) {
		memInfos := strings.Split(strings.TrimSuffix(output, "\r\n"), "\r\n")
		var count int
		for _, line := range memInfos {
			if strings.HasPrefix(line, "slot:") {
				count += 1
			}
		}
		callback(count)
	}
	m.Query("info memory-devices", cb)
}

func (m *HmpMonitor) GetMemoryDevicesInfo(cb QueryMemoryDevicesCallback) {
	go cb(nil, "hmp unsupport get memory devices info")
}

func (m *HmpMonitor) GetMemdevList(callback MemdevListCallback) {
	go callback(nil, "hmp unsupport get memdev list")
}

func (m *HmpMonitor) ObjectAdd(objectType string, params map[string]string, callback StringCallback) {
	var paramsKvs = []string{}
	for k, v := range params {
		paramsKvs = append(paramsKvs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd := fmt.Sprintf("object_add %s,%s", objectType, strings.Join(paramsKvs, ","))
	m.Query(cmd, callback)
}

func (m *HmpMonitor) BlockIoThrottle(driveName string, bps, iops int64, callback StringCallback) {
	cmd := fmt.Sprintf("block_set_io_throttle %s %d 0 0 %d 0 0", driveName, bps, iops)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) CancelBlockJob(driveName string, force bool, callback StringCallback) {
	cmd := "block_job_cancel "
	if force {
		cmd += "-f "
	}
	cmd += driveName
	m.Query(cmd, callback)
}

func (m *HmpMonitor) NetdevAdd(id, netType string, params map[string]string, callback StringCallback) {
	cmd := fmt.Sprintf("netdev_add %s,id=%s", netType, id)
	for k, v := range params {
		cmd += fmt.Sprintf(",%s=%s", k, v)
	}
	m.Query(cmd, callback)
}

func (m *HmpMonitor) NetdevDel(id string, callback StringCallback) {
	cmd := fmt.Sprintf("netdev_del %s", id)
	m.Query(cmd, callback)
}

func (m *HmpMonitor) SaveState(stateFilePath string, callback StringCallback) {
	cmd := fmt.Sprintf(`migrate -d "%s"`, getSaveStatefileUri(stateFilePath))
	m.Query(cmd, callback)
}

func (m *HmpMonitor) QueryPci(callback QueryPciCallback) {
	go callback(nil, "unsupported query pci for hmp")
}

func (m *HmpMonitor) QueryMachines(callback QueryMachinesCallback) {
	go callback(nil, "unsupported query machines for hmp")
}

func (m *HmpMonitor) Quit(cb StringCallback) {
	m.Query("quit", cb)
}

func (m *HmpMonitor) InfoQtree(cb StringCallback) {
	m.Query("info qtree", cb)
}

func (m *HmpMonitor) GetHotPluggableCpus(callback HotpluggableCPUListCallback) {
	go callback(nil, "unsupported get hotpluggable cpu list for hmp")
}
