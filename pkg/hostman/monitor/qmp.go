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
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
)

// https://github.com/qemu/qemu/blob/master/docs/interop/qmp-spec.txt
// https://wiki.qemu.org/QMP
/*
Not support oob yet
1. response error message
    { "error": { "class": json-string, "desc": json-string }, "id": json-value }
2. response event message
    { "event": json-string, "data": json-object,
  "timestamp": { "seconds": json-number, "microseconds": json-number } }
3. response cmd result
    { "return": json-value, "id": json-value }
4. response qmp init information
    { "QMP": {"version": {"qemu": {"micro": 0, "minor": 0, "major": 3},
     "package": "v3.0.0"}, "capabilities": [] } }
*/

var ignoreEvents = []string{`"RTC_CHANGE"`}

type qmpMonitorCallBack func(*Response)
type qmpEventCallback func(*Event)

type Response struct {
	Return   []byte
	ErrorVal *Error
	Id       string
}

type Event struct {
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data"`
	Timestamp *Timestamp             `json:"timestamp"`
}

func (e *Event) String() string {
	return fmt.Sprintf("QMP Event result: %#v", e)
}

type Timestamp struct {
	Seconds       int64 `json:"seconds"`
	Microsenconds int64 `json:"microsenconds"`
}

type Command struct {
	Execute string      `json:"execute"`
	Args    interface{} `json:"arguments,omitempty"`
}

type Version struct {
	Package string `json:"package"`
	QEMU    struct {
		Major int `json:"major"`
		Micro int `json:"micro"`
		Minor int `json:"minor"`
	} `json:"qemu"`
}

func (v *Version) String() string {
	q := v.QEMU
	return fmt.Sprintf("%d.%d.%d", q.Major, q.Minor, q.Micro)
}

type Error struct {
	Class string `json:"class"`
	Desc  string `json:"desc"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Class, e.Desc)
}

type QmpMonitor struct {
	SBaseMonitor

	qmpEventFunc  qmpEventCallback
	commandQueue  []*Command
	callbackQueue []qmpMonitorCallBack
}

func NewQmpMonitor(OnMonitorDisConnect, OnMonitorTimeout MonitorErrorFunc,
	OnMonitorConnected MonitorSuccFunc, qmpEventFunc qmpEventCallback) *QmpMonitor {
	m := &QmpMonitor{
		SBaseMonitor:  *NewBaseMonitor(OnMonitorConnected, OnMonitorDisConnect, OnMonitorTimeout),
		qmpEventFunc:  qmpEventFunc,
		commandQueue:  make([]*Command, 0),
		callbackQueue: make([]qmpMonitorCallBack, 0),
	}

	// On qmp init must set capabilities
	m.commandQueue = append(m.commandQueue, &Command{Execute: "qmp_capabilities"})
	m.callbackQueue = append(m.callbackQueue, nil)

	return m
}

func (m *QmpMonitor) actionResult(res *Response) string {
	if res.ErrorVal != nil {
		log.Errorf("Qmp Monitor action result %s", res.ErrorVal.Error())
		return res.ErrorVal.Error()
	} else {
		return ""
	}
}

func (m *QmpMonitor) callBack(res *Response) {
	m.mutex.Lock()
	if len(m.callbackQueue) == 0 {
		m.mutex.Unlock()
		return
	}
	cb := m.callbackQueue[0]
	m.callbackQueue = m.callbackQueue[1:]
	m.mutex.Unlock()
	if cb != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("PANIC %s:\n%s", debug.Stack(), r)
				}
			}()
			cb(res)
		}()
	}
}

func (m *QmpMonitor) read(r io.Reader) {
	if !m.checkReading() {
		return
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var objmap map[string]*json.RawMessage
		b := scanner.Bytes()
		if err := json.Unmarshal(b, &objmap); err != nil {
			log.Errorln("Error, ", err.Error())
			continue
		}
		if val, ok := objmap["error"]; ok {
			var res = &Response{}
			res.ErrorVal = &Error{}
			json.Unmarshal(*val, res.ErrorVal)
			if id, ok := objmap["id"]; ok {
				res.Id = string(*id)
			}
			m.callBack(res)
		} else if val, ok := objmap["return"]; ok {
			var res = &Response{}
			res.Return = []byte(*val)
			if id, ok := objmap["id"]; ok {
				res.Id = string(*id)
			}
			m.callBack(res)
		} else if val, ok := objmap["event"]; ok {
			var event = &Event{
				Event:     string(*val),
				Data:      make(map[string]interface{}, 0),
				Timestamp: new(Timestamp),
			}
			if data, ok := objmap["data"]; ok {
				json.Unmarshal(*data, &event.Data)
			}
			if timestamp, ok := objmap["timestamp"]; ok {
				json.Unmarshal(*timestamp, event.Timestamp)
			}
			m.watchEvent(event)
		} else if val, ok := objmap["QMP"]; ok {
			// On qmp connected
			json.Unmarshal(*val, &objmap)
			if val, ok = objmap["version"]; ok {
				var version Version
				json.Unmarshal(*val, &version)
				m.QemuVersion = version.String()
			}

			// remove reader timeout
			m.rwc.SetReadDeadline(time.Time{})
			m.connected = true
			m.timeout = false
			go m.query()
			go m.OnMonitorConnected()
		}
	}

	log.Infof("Scan over ...")
	err := scanner.Err()
	if err != nil {
		log.Infof("QMP Disconnected: %s", err)
	}
	if m.timeout {
		m.OnMonitorTimeout(err)
	} else if m.connected {
		m.connected = false
		m.OnMonitorDisConnect(err)
	}
	m.reading = false
}

func (m *QmpMonitor) watchEvent(event *Event) {
	if !utils.IsInStringArray(event.Event, ignoreEvents) {
		log.Infof(event.String())
	}
	if m.qmpEventFunc != nil {
		go m.qmpEventFunc(event)
	}
}

func (m *QmpMonitor) write(cmd []byte) error {
	log.Infof("QMP Write: %s", string(cmd))
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

func (m *QmpMonitor) query() {
	if !m.checkWriting() {
		return
	}
	for {
		if len(m.commandQueue) == 0 {
			break
		}

		// pop cmd
		m.mutex.Lock()
		cmd := m.commandQueue[0]
		m.commandQueue = m.commandQueue[1:]

		c, _ := json.Marshal(cmd)
		err := m.write(c)
		m.mutex.Unlock()
		if err != nil {
			log.Errorf("Write %s to monitor error: %s", c, err)
			break
		}
	}
	m.writing = false
}

func (m *QmpMonitor) Query(cmd *Command, cb qmpMonitorCallBack) {
	// push cmd
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

func (m *QmpMonitor) ConnectWithSocket(address string) error {
	err := m.SBaseMonitor.connect("unix", address)
	if err != nil {
		return err
	}
	go m.read(m.rwc)
	return nil
}

func (m *QmpMonitor) Connect(host string, port int) error {
	err := m.SBaseMonitor.connect("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	go m.read(m.rwc)
	return nil
}

func (m *QmpMonitor) parseCmd(cmd string) string {
	re := regexp.MustCompile(`\s+`)
	parts := re.Split(strings.TrimSpace(cmd), -1)
	if parts[0] == "info" && len(parts) > 1 {
		return "query-" + parts[1]
	} else {
		return parts[0]
	}
}

func (m *QmpMonitor) SimpleCommand(cmd string, callback StringCallback) {
	cmd = m.parseCmd(cmd)
	var cb func(res *Response)
	if callback != nil {
		cb = func(res *Response) {
			if res.ErrorVal != nil {
				callback(res.ErrorVal.Error())
			} else {
				callback(string(res.Return))
			}
		}
	}
	c := &Command{Execute: cmd}
	m.Query(c, cb)
}

func (m *QmpMonitor) HumanMonitorCommand(cmd string, callback StringCallback) {
	var (
		c = &Command{
			Execute: "human-monitor-command",
			Args:    map[string]string{"command-line": cmd},
		}

		cb = func(res *Response) {
			log.Debugf("Monitor ret: %s", res.Return)
			if res.ErrorVal != nil {
				callback(res.ErrorVal.Error())
			} else {
				callback(strings.Trim(string(res.Return), `""`))
			}
		}
	)
	m.Query(c, cb)
}

func (m *QmpMonitor) QueryStatus(callback StringCallback) {
	cmd := &Command{Execute: "query-status"}
	m.Query(cmd, m.parseStatus(callback))
}

func (m *QmpMonitor) parseStatus(callback StringCallback) qmpMonitorCallBack {
	return func(res *Response) {
		if res.ErrorVal != nil {
			callback("unknown")
			return
		}
		var val map[string]interface{}
		err := json.Unmarshal(res.Return, &val)
		if err != nil {
			callback("unknown")
			return
		}
		if status, ok := val["status"]; !ok {
			callback("unknown")
		} else {
			str, _ := status.(string)
			callback(str)
		}
	}
}

// If get version error, callback with empty string
func (m *QmpMonitor) GetVersion(callback StringCallback) {
	cmd := &Command{Execute: "query-version"}
	m.Query(cmd, m.parseVersion(callback))
}

func (m *QmpMonitor) parseVersion(callback StringCallback) qmpMonitorCallBack {
	return func(res *Response) {
		if res.ErrorVal != nil {
			callback("")
			return
		}
		var version Version
		err := json.Unmarshal(res.Return, &version)
		if err != nil {
			callback("")
		} else {
			callback(version.String())
		}
	}
}

func (m *QmpMonitor) GetBlocks(callback func(*jsonutils.JSONArray)) {
	var cb = func(res *Response) {
		if res.ErrorVal != nil {
			callback(nil)
		}
		jr, err := jsonutils.Parse(res.Return)
		if err != nil {
			log.Errorf("Get block error %s", err)
			callback(nil)
		}
		jra, _ := jr.(*jsonutils.JSONArray)
		callback(jra)
	}

	cmd := &Command{Execute: "query-block"}
	m.Query(cmd, cb)
}

func (m *QmpMonitor) ChangeCdrom(dev string, path string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("change %s %s", dev, path), callback)
	// var (
	// 	args = map[string]interface{}{
	// 		"arguments": map[string]interface{}{
	// 			"device": dev,
	// 			"target": path,
	// 		},
	// 	}
	// 	cmd = &Command{
	// 		Execute: "change",
	// 		Args:    args,
	// 	}

	// 	cb = func(res *Response) {
	// 		callback(m.actionResult(res))
	// 	}
	// )

	// m.Query(cmd, cb)
}

func (m *QmpMonitor) EjectCdrom(dev string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("eject -f %s", dev), callback)
	// XXX: 同下
	// var (
	// 	args = map[string]interface{}{
	// 		"arguments": map[string]interface{}{
	// 			"device": dev,
	// 			"force":  true,
	// 		},
	// 	}
	// 	cmd = &Command{
	// 		Execute: "eject",
	// 		Args:    args,
	// 	}

	// 	cb = func(res *Response) {
	// 		callback(m.actionResult(res))
	// 	}
	// )

	// m.Query(cmd, cb)
}

func (m *QmpMonitor) DriveDel(idstr string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("drive_del %s", idstr), callback)
	// XXX: 同下
	// var (
	// 	args = map[string]interface{}{
	// 		"arguments": map[string]interface{}{
	// 			"device": idstr,
	// 		},
	// 	}
	// 	cmd = &Command{
	// 		Execute: "drive_del",
	// 		Args:    args,
	// 	}

	// 	cb = func(res *Response) {
	// 		callback(m.actionResult(res))
	// 	}
	// )

	// m.Query(cmd, cb)
}

func (m *QmpMonitor) DeviceDel(idstr string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("device_del %s", idstr), callback)
	// XXX: 同下
	// var (
	// 	args = map[string]interface{}{
	// 		"arguments": map[string]interface{}{
	// 			"device": idstr,
	// 		},
	// 	}
	// 	cmd = &Command{
	// 		Execute: "device_del",
	// 		Args:    args,
	// 	}

	// 	cb = func(res *Response) {
	// 		callback(m.actionResult(res))
	// 	}
	// )

	// m.Query(cmd, cb)
}

func (m *QmpMonitor) ObjectDel(idstr string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("object_del %s", idstr), callback)
}

func (m *QmpMonitor) DriveAdd(bus string, params map[string]string, callback StringCallback) {
	var paramsKvs = []string{}
	for k, v := range params {
		paramsKvs = append(paramsKvs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd := fmt.Sprintf("drive_add %s %s", bus, strings.Join(paramsKvs, ","))
	m.HumanMonitorCommand(cmd, callback)
	// XXX: 同下
	// var (
	// 	args = map[string]interface{}{
	// 		"arguments": map[string]interface{}{
	// 			"bus":    bus,
	// 			"params": params,
	// 		},
	// 	}
	// 	cmd = &Command{
	// 		Execute: "drive_add",
	// 		Args:    args,
	// 	}

	// 	cb = func(res *Response) {
	// 		callback(m.actionResult(res))
	// 	}
	// )

	// m.Query(cmd, cb)
}

func (m *QmpMonitor) DeviceAdd(dev string, params map[string]interface{}, callback StringCallback) {
	var paramsKvs = []string{}
	for k, v := range params {
		paramsKvs = append(paramsKvs, fmt.Sprintf("%s=%v", k, v))
	}
	cmd := fmt.Sprintf("device_add %s,%s", dev, strings.Join(paramsKvs, ","))
	m.HumanMonitorCommand(cmd, callback)

	// XXX: 参数不对，之后再调，先用着hmp的参数
	// var (
	// 	args = map[string]interface{}{
	// 		"arguments": map[string]interface{}{
	// 			"driver": dev,
	// 			"params": params,
	// 		},
	// 	}
	// 	cmd = &Command{
	// 		Execute: "device_add",
	// 		Args:    args,
	// 	}

	// 	cb = func(res *Response) {
	// 		callback(m.actionResult(res))
	// 	}
	// )

	// m.Query(cmd, cb)
}

func (m *QmpMonitor) MigrateSetCapability(capability, state string, callback StringCallback) {
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		st = false
	)
	if state == "on" {
		st = true
	}

	cmd := &Command{
		Execute: "migrate-set-capabilities",
		Args: map[string]interface{}{
			"capabilities": []interface{}{
				map[string]interface{}{
					"capability": capability,
					"state":      st,
				},
			},
		},
	}

	m.Query(cmd, cb)
}

func (m *QmpMonitor) Migrate(
	destStr string, copyIncremental, copyFull bool, callback StringCallback,
) {
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		cmd = &Command{
			Execute: "migrate",
			Args: map[string]interface{}{
				"uri": destStr,
				"blk": copyFull,
				"inc": copyIncremental,
			},
		}
	)

	m.Query(cmd, cb)
}

func (m *QmpMonitor) GetMigrateStatus(callback StringCallback) {
	var (
		cmd = &Command{Execute: "query-migrate"}
		cb  = func(res *Response) {
			if res.ErrorVal != nil {
				callback(res.ErrorVal.Error())
			} else {
				ret, err := jsonutils.Parse(res.Return)
				if err != nil {
					log.Errorf("Parse qmp res error: %s", err)
					callback("")
				} else {
					log.Infof("Query migrate status: %s", ret.String())
					status, _ := ret.GetString("status")
					callback(status)
				}
			}
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) GetBlockJobCounts(callback func(jobs int)) {
	var cb = func(res *Response) {
		if res.ErrorVal != nil {
			log.Errorln(res.ErrorVal.Error())
			callback(-1)
		} else {
			ret, err := jsonutils.Parse(res.Return)
			if err != nil {
				log.Errorf("Parse qmp res error: %s", err)
				callback(-1)
			} else {
				jobs, _ := ret.GetArray()
				callback(len(jobs))
			}
		}
	}
	m.Query(&Command{Execute: "query-block-jobs"}, cb)
}

func (m *QmpMonitor) GetBlockJobs(callback func(*jsonutils.JSONArray)) {
	var cb = func(res *Response) {
		if res.ErrorVal != nil {
			log.Errorln(res.ErrorVal.Error())
			callback(nil)
		} else {
			ret, err := jsonutils.Parse(res.Return)
			if err != nil {
				log.Errorf("Parse qmp res error: %s", err)
				callback(nil)
			} else {
				jobs, _ := ret.(*jsonutils.JSONArray)
				callback(jobs)
			}
		}
	}
	m.Query(&Command{Execute: "query-block-jobs"}, cb)
}

func (m *QmpMonitor) ReloadDiskBlkdev(device, path string, callback StringCallback) {
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		cmd = &Command{
			Execute: "reload-disk-snapshot-blkdev-sync",
			Args: map[string]string{
				"device":        device,
				"snapshot-file": path,
				"mode":          "existing",
				"format":        "qcow2",
			},
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) DriveMirror(callback StringCallback, drive, target, syncMode string, unmap bool) {
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		cmd = &Command{
			Execute: "drive-mirror",
			Args: map[string]interface{}{
				"device": drive,
				"target": target,
				"mode":   "existing",
				"sync":   syncMode,
				"unmap":  unmap,
			},
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) BlockStream(drive string, callback StringCallback) {
	var (
		speed = 30 * 1024 * 1024 // qmp speed default unit is byte
		cb    = func(res *Response) {
			callback(m.actionResult(res))
		}
		cmd = &Command{
			Execute: "block-stream",
			Args: map[string]interface{}{
				"device": drive,
				"speed":  speed,
			},
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) SetVncPassword(proto, password string, callback StringCallback) {
	if len(password) > 8 {
		password = password[:8]
	}
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		cmd = &Command{
			Execute: "set_password",
			Args: map[string]interface{}{
				"protocol": proto,
				"password": password,
			},
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) StartNbdServer(port int, exportAllDevice, writable bool, callback StringCallback) {
	var cmd = "nbd_server_start"
	if exportAllDevice {
		cmd += " -a"
	}
	if writable {
		cmd += " -w"
	}
	cmd += fmt.Sprintf(" 0.0.0.0:%d", port)
	m.HumanMonitorCommand(cmd, callback)
}

func (m *QmpMonitor) ResizeDisk(driveName string, sizeMB int64, callback StringCallback) {
	cmd := fmt.Sprintf("block_resize %s %d", driveName, sizeMB)
	m.HumanMonitorCommand(cmd, callback)
}

func (m *QmpMonitor) GetCpuCount(callback func(count int)) {
	var cb = func(res string) {
		cpus := strings.Split(res, "\\n")
		count := 0
		for _, cpuInfo := range cpus {
			if len(strings.TrimSpace(cpuInfo)) > 0 {
				count += 1
			}
		}
		callback(count)
	}
	m.HumanMonitorCommand("info cpus", cb)
}

func (m *QmpMonitor) AddCpu(cpuIndex int, callback StringCallback) {
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		cmd = &Command{
			Execute: "cpu-add",
			Args:    map[string]interface{}{"id": cpuIndex},
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) ObjectAdd(objectType string, params map[string]string, callback StringCallback) {
	var paramsKvs = []string{}
	for k, v := range params {
		paramsKvs = append(paramsKvs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd := fmt.Sprintf("object_add %s,%s", objectType, strings.Join(paramsKvs, ","))
	m.HumanMonitorCommand(cmd, callback)
}

func (m *QmpMonitor) GeMemtSlotIndex(callback func(index int)) {
	var cb = func(res string) {
		memInfos := strings.Split(res, "\\n")
		var count int
		for _, line := range memInfos {
			if strings.HasPrefix(strings.TrimSpace(line), "slot:") {
				count += 1
			}
		}
		callback(count)
	}
	m.HumanMonitorCommand("info memory-devices", cb)
}

func (m *QmpMonitor) BlockIoThrottle(driveName string, bps, iops int64, callback StringCallback) {
	cmd := fmt.Sprintf("block_set_io_throttle %s %d 0 0 %d 0 0", driveName, bps, iops)
	m.HumanMonitorCommand(cmd, callback)
}

func (m *QmpMonitor) CancelBlockJob(driveName string, force bool, callback StringCallback) {
	cmd := "block_job_cancel "
	if force {
		cmd += "-f "
	}
	cmd += driveName
	m.HumanMonitorCommand(cmd, callback)
}
