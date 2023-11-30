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

	"golang.org/x/net/context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/hostutils"
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

type NetworkModify struct {
	Device  string `json:"device"`
	Ipmask  string `json:"ipmask"`
	Gateway string `json:"gateway"`
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
	jobs          map[string]BlockJob
}

func NewQmpMonitor(server, sid string, OnMonitorDisConnect, OnMonitorTimeout MonitorErrorFunc,
	OnMonitorConnected MonitorSuccFunc, qmpEventFunc qmpEventCallback) *QmpMonitor {
	m := &QmpMonitor{
		SBaseMonitor:  *NewBaseMonitor(server, sid, OnMonitorConnected, OnMonitorDisConnect, OnMonitorTimeout),
		qmpEventFunc:  qmpEventFunc,
		commandQueue:  make([]*Command, 0),
		callbackQueue: make([]qmpMonitorCallBack, 0),
		jobs:          map[string]BlockJob{},
	}

	// On qmp init must set capabilities
	m.commandQueue = append(m.commandQueue, &Command{Execute: "qmp_capabilities"})
	m.callbackQueue = append(m.callbackQueue, nil)

	return m
}

func (m *QmpMonitor) actionResult(res *Response) string {
	if res.ErrorVal != nil {
		log.Errorf("Qmp Monitor action result %s: %s", m.server, res.ErrorVal.Error())
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
					log.Errorf("PANIC %s %s:\n%s", m.server, debug.Stack(), r)
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
			log.Errorf("Unmarshal %s error: %s", m.server, err.Error())
			continue
		}
		if val, ok := objmap["event"]; !ok || !utils.IsInStringArray(string(*val), ignoreEvents) {
			log.Infof("QMP Read %s: %s", m.server, string(b))
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
			if m.OnMonitorConnected != nil {
				go m.OnMonitorConnected()
			}
		}
	}

	log.Infof("Scan over %s ...", m.server)
	err := scanner.Err()
	if err != nil {
		log.Infof("QMP Disconnected %s: %s", m.server, err)
	}
	if m.timeout {
		if m.OnMonitorTimeout != nil {
			m.OnMonitorTimeout(err)
		}
	} else if m.connected {
		m.connected = false
		if m.OnMonitorDisConnect != nil {
			m.OnMonitorDisConnect(err)
		}
	}
	m.reading = false
}

func (m *QmpMonitor) watchEvent(event *Event) {
	if !utils.IsInStringArray(event.Event, ignoreEvents) {
		log.Infof("QMP event %s: %s", m.server, event.String())
	}
	if m.qmpEventFunc != nil {
		go m.qmpEventFunc(event)
	}
}

func (m *QmpMonitor) write(cmd []byte) error {
	log.Infof("QMP Write %s: %s", m.server, string(cmd))
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
			log.Errorf("Write %s to monitor %s error: %s", c, m.server, err)
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

func (m *QmpMonitor) ConnectWithSocket(address string, timeout time.Duration) error {
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

func (m *QmpMonitor) QemuMonitorCommand(rawCmd string, callback StringCallback) error {
	c := Command{}
	if err := json.Unmarshal([]byte(rawCmd), &c); err != nil {
		return errors.Errorf("unsupport command format: %s", err)
	}

	cb := func(res *Response) {
		log.Debugf("Monitor %s ret: %s", m.server, res.Return)
		if res.ErrorVal != nil {
			callback(res.ErrorVal.Error())
		} else {
			callback(strings.Trim(string(res.Return), `""`))
		}
	}

	m.Query(&c, cb)
	return nil
}

func (m *QmpMonitor) HumanMonitorCommand(cmd string, callback StringCallback) {
	var (
		c = &Command{
			Execute: "human-monitor-command",
			Args:    map[string]string{"command-line": cmd},
		}

		cb = func(res *Response) {
			log.Debugf("Monitor %s ret: %s", m.server, res.Return)
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
	m.HumanMonitorCommand("info status", m.parseStatus(callback))
}

// func (m *QmpMonitor) parseStatus(callback StringCallback) qmpMonitorCallBack {
// 	return func(res *Response) {
// 		if res.ErrorVal != nil {
// 			callback("unknown")
// 			return
// 		}
// 		var val map[string]interface{}
// 		err := json.Unmarshal(res.Return, &val)
// 		if err != nil {
// 			callback("unknown")
// 			return
// 		}
// 		if status, ok := val["status"]; !ok {
// 			callback("unknown")
// 		} else {
// 			str, _ := status.(string)
// 			callback(str)
// 		}
// 	}
// }

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

func (m *QmpMonitor) GetBlocks(callback func([]QemuBlock)) {
	var cb = func(res *Response) {
		if res.ErrorVal != nil {
			log.Errorf("GetBlocks error %s", res.ErrorVal)
			callback(nil)
			return
		}
		jr, err := jsonutils.Parse(res.Return)
		if err != nil {
			log.Errorf("Get %s block error %s", m.server, err)
			callback(nil)
			return
		}
		blocks := []QemuBlock{}
		jr.Unmarshal(&blocks)
		callback(blocks)
	}

	cmd := &Command{Execute: "query-block"}
	m.Query(cmd, cb)
}

func (m *QmpMonitor) ChangeCdrom(dev string, path string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("change %s %s", dev, path), callback)
}

func (m *QmpMonitor) EjectCdrom(dev string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("eject -f %s", dev), callback)
}

func (m *QmpMonitor) DriveDel(idstr string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("drive_del %s", idstr), callback)
}

func (m *QmpMonitor) DeviceDel(idstr string, callback StringCallback) {
	//m.HumanMonitorCommand(fmt.Sprintf("device_del %s", idstr), callback)
	var (
		args = map[string]interface{}{
			"id": idstr,
		}
		cmd = &Command{
			Execute: "device_del",
			Args:    args,
		}

		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) ObjectDel(idstr string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("object_del %s", idstr), callback)
}

func (m *QmpMonitor) XBlockdevChange(parent, node, child string, callback StringCallback) {
	cb := func(res *Response) {
		callback(m.actionResult(res))
	}
	cmd := &Command{
		Execute: "x-blockdev-change",
	}
	args := map[string]interface{}{
		"parent": parent,
	}
	if len(node) > 0 {
		args["node"] = node
	}
	if len(child) > 0 {
		args["child"] = child
	}
	cmd.Args = args
	m.Query(cmd, cb)
}

func (m *QmpMonitor) DriveAdd(bus, node string, params map[string]string, callback StringCallback) {
	var paramsKvs = []string{}
	for k, v := range params {
		paramsKvs = append(paramsKvs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd := "drive_add"
	if len(node) > 0 {
		cmd = fmt.Sprintf("drive_add -n %s", node)
	}

	cmd = fmt.Sprintf("%s %s %s", cmd, bus, strings.Join(paramsKvs, ","))
	m.HumanMonitorCommand(cmd, callback)
}

func (m *QmpMonitor) DeviceAdd(dev string, params map[string]string, callback StringCallback) {
	args := map[string]interface{}{
		"driver": dev,
	}

	for k, v := range params {
		args[k] = v
	}

	cmd := &Command{
		Execute: "device_add",
		Args:    args,
	}

	cb := func(res *Response) {
		callback(m.actionResult(res))
	}

	m.Query(cmd, cb)
}

func (m *QmpMonitor) MigrateSetDowntime(dtSec float64, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("migrate_set_downtime %f", dtSec), callback)
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

func (m *QmpMonitor) MigrateSetParameter(key string, val interface{}, callback StringCallback) {
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		cmd = &Command{
			Execute: "migrate-set-parameters",
			Args: map[string]interface{}{
				key: val,
			},
		}
	)

	m.Query(cmd, cb)
}

func (m *QmpMonitor) MigrateIncoming(address string, callback StringCallback) {
	cmd := fmt.Sprintf("migrate_incoming %s", address)
	m.HumanMonitorCommand(cmd, callback)
}

func (m *QmpMonitor) MigrateContinue(state string, callback StringCallback) {
	cmd := fmt.Sprintf("migrate_continue %s", state)
	m.HumanMonitorCommand(cmd, callback)
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
				/*
					{"expected-downtime":300,"ram":{"dirty-pages-rate":0,"dirty-sync-count":1,"duplicate":2966538,"mbps":268.5672,"normal":148629,"normal-bytes":608784384,"page-size":4096,"postcopy-requests":0,"remaining":142815232,"skipped":0,"total":12902539264,"transferred":636674057},"setup-time":65,"status":"active","total-time":20002}
					{"disk":{"dirty-pages-rate":0,"dirty-sync-count":0,"duplicate":0,"mbps":0,"normal":0,"normal-bytes":0,"page-size":0,"postcopy-requests":0,"remaining":0,"skipped":0,"total":139586437120,"transferred":139586437120},"expected-downtime":300,"ram":{"dirty-pages-rate":0,"dirty-sync-count":1,"duplicate":193281,"mbps":268.44264,"normal":62311,"normal-bytes":255225856,"page-size":4096,"postcopy-requests":0,"remaining":44474368,"skipped":0,"total":1091379200,"transferred":257555032},"setup-time":15,"status":"active","total-time":10002}
				*/
				ret, err := jsonutils.Parse(res.Return)
				if err != nil {
					log.Errorf("Parse qmp res error %s: %s", m.server, err)
					callback("")
				} else {
					log.Infof("Query migrate status %s: %s", m.server, ret.String())

					status, _ := ret.GetString("status")
					if status == "active" {
						ramTotal, _ := ret.Int("ram", "total")
						ramRemain, _ := ret.Int("ram", "remaining")
						ramMbps, _ := ret.Float("ram", "mbps")
						diskTotal, _ := ret.Int("disk", "total")
						diskRemain, _ := ret.Int("disk", "remaining")
						diskMbps, _ := ret.Float("disk", "mbps")
						if diskRemain > 0 {
							status = "migrate_disk_copy"
						} else if ramRemain > 0 {
							status = "migrate_ram_copy"
						}
						mbps := ramMbps + diskMbps
						progress := (1 - float64(diskRemain+ramRemain)/float64(diskTotal+ramTotal)) * 100.0
						log.Debugf("progress: %f mbps: %f", progress, mbps)
						hostutils.UpdateServerProgress(context.Background(), m.sid, progress, mbps)
					}

					callback(status)
				}
			}
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) GetMigrateStats(callback MigrateStatsCallback) {
	var (
		cmd = &Command{Execute: "query-migrate"}
		cb  = func(res *Response) {
			if res.ErrorVal != nil {
				callback(nil, errors.Errorf(res.ErrorVal.Error()))
			} else {
				migStats := new(MigrationInfo)
				err := json.Unmarshal(res.Return, migStats)
				if err != nil {
					callback(nil, err)
					return
				}
				callback(migStats, nil)
			}
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) MigrateCancel(cb StringCallback) {
	m.HumanMonitorCommand("migrate_cancel", cb)
}

func (m *QmpMonitor) MigrateStartPostcopy(callback StringCallback) {
	var (
		cmd = &Command{Execute: "migrate-start-postcopy"}
		cb  = func(res *Response) {
			if res.ErrorVal != nil {
				callback(res.ErrorVal.Error())
			} else {
				ret, err := jsonutils.Parse(res.Return)
				if err != nil {
					log.Errorf("Parse qmp res error %s: %s", m.server, err)
					callback("MigrateStartPostcopy error")
				} else {
					log.Infof("MigrateStartPostcopy %s: %s", m.server, ret.String())
					callback("")
				}
			}
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) blockJobs(res *Response) ([]BlockJob, error) {
	if res.ErrorVal != nil {
		return nil, errors.Errorf("GetBlockJobs for %s %s", m.server, jsonutils.Marshal(res.ErrorVal).String())
	}
	ret, err := jsonutils.Parse(res.Return)
	if err != nil {
		return nil, errors.Wrapf(err, "GetBlockJobs for %s parse %s", m.server, res.Return)
	}
	log.Debugf("blockJobs response %s", ret)
	jobs := []BlockJob{}
	if err = ret.Unmarshal(&jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (m *QmpMonitor) GetBlockJobCounts(callback func(jobs int)) {
	var cb = func(res *Response) {
		jobs, err := m.blockJobs(res)
		if err != nil {
			callback(-1)
			return
		}
		callback(len(jobs))
	}
	m.Query(&Command{Execute: "query-block-jobs"}, cb)
}

func (m *QmpMonitor) GetBlockJobs(callback func([]BlockJob)) {
	var cb = func(res *Response) {
		jobs, _ := m.blockJobs(res)
		callback(jobs)
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

func (m *QmpMonitor) DriveMirror(callback StringCallback, drive, target, syncMode, format string, unmap, blockReplication bool, speed int64) {
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		args = map[string]interface{}{
			"device": drive,
			"target": target,
			"mode":   "existing",
			"sync":   syncMode,
			"unmap":  unmap,
		}
	)
	if speed > 0 {
		args["speed"] = speed
	}
	if blockReplication {
		args["block-replication"] = true
	}
	cmd := &Command{
		Execute: "drive-mirror",
		Args:    args,
	}

	m.Query(cmd, cb)
}

func (m *QmpMonitor) DriveBackup(callback StringCallback, drive, target, syncMode, format string) {
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		args = map[string]interface{}{
			"device": drive,
			"target": target,
			"mode":   "existing",
			"sync":   syncMode,
			"format": format,
		}
	)
	cmd := &Command{
		Execute: "drive-backup",
		Args:    args,
	}

	m.Query(cmd, cb)
}

func (m *QmpMonitor) BlockStream(drive string, callback StringCallback) {
	var (
		speed = 5 * 100 * 1024 * 1024 // limit 500 MB/s
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

func (m *QmpMonitor) StopNbdServer(callback StringCallback) {
	m.HumanMonitorCommand("nbd_server_stop", callback)
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

func (m *QmpMonitor) GetMemoryDevicesInfo(callback QueryMemoryDevicesCallback) {
	var (
		cb = func(res *Response) {
			if res.ErrorVal != nil {
				callback(nil, res.ErrorVal.Error())
			} else {
				memDevices := make([]MemoryDeviceInfo, 0)
				err := json.Unmarshal(res.Return, &memDevices)
				if err != nil {
					callback(nil, err.Error())
				} else {
					callback(memDevices, "")
				}
			}
		}
		cmd = &Command{
			Execute: "query-memory-devices",
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) GetMemdevList(callback MemdevListCallback) {
	var (
		cb = func(res *Response) {
			if res.ErrorVal != nil {
				callback(nil, res.ErrorVal.Error())
			} else {
				memdevList := make([]Memdev, 0)
				err := json.Unmarshal(res.Return, &memdevList)
				if err != nil {
					callback(nil, err.Error())
				} else {
					callback(memdevList, "")
				}

			}
		}
		cmd = &Command{
			Execute: "query-memdev",
		}
	)
	m.Query(cmd, cb)
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

func (m *QmpMonitor) BlockJobComplete(drive string, callback StringCallback) {
	m.HumanMonitorCommand(fmt.Sprintf("block_job_complete %s", drive), callback)
}

func (m *QmpMonitor) BlockReopenImage(drive, newImagePath, format string, callback StringCallback) {
	var cb = func(res *Response) {
		callback(m.actionResult(res))
	}

	var cmd = &Command{
		Execute: "block_reopen_image",
		Args: map[string]interface{}{
			"device":    drive,
			"new_image": newImagePath,
			"format":    format,
		},
	}

	m.Query(cmd, cb)
}

func (m *QmpMonitor) SnapshotBlkdev(drive, newImagePath, format string, reuse bool, callback StringCallback) {
	var cmd = "snapshot_blkdev"
	if reuse {
		cmd += " -n"
	}
	cmd += fmt.Sprintf(" %s %s %s", drive, newImagePath, format)

	m.HumanMonitorCommand(cmd, callback)
}

func (m *QmpMonitor) NetdevAdd(id, netType string, params map[string]string, callback StringCallback) {
	cmd := fmt.Sprintf("netdev_add %s,id=%s", netType, id)
	for k, v := range params {
		cmd += fmt.Sprintf(",%s=%s", k, v)
	}
	m.HumanMonitorCommand(cmd, callback)
}

func (m *QmpMonitor) NetdevDel(id string, callback StringCallback) {
	cmd := fmt.Sprintf("netdev_del %s", id)
	m.HumanMonitorCommand(cmd, callback)
}

func (m *QmpMonitor) SaveState(stateFilePath string, callback StringCallback) {
	var (
		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
		cmd = &Command{
			Execute: "migrate",
			Args: map[string]interface{}{
				"uri": getSaveStatefileUri(stateFilePath),
			},
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) QueryPci(callback QueryPciCallback) {
	var (
		cb = func(res *Response) {
			if res.ErrorVal != nil {
				callback(nil, res.ErrorVal.Error())
			} else {
				pciInfoList := make([]PCIInfo, 0)
				err := json.Unmarshal(res.Return, &pciInfoList)
				if err != nil {
					callback(nil, err.Error())
				} else {
					callback(pciInfoList, "")
				}
			}
		}
		cmd = &Command{
			Execute: "query-pci",
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) QueryMachines(callback QueryMachinesCallback) {
	var (
		cb = func(res *Response) {
			if res.ErrorVal != nil {
				callback(nil, res.ErrorVal.Error())
			} else {
				machineInfoList := make([]MachineInfo, 0)
				err := json.Unmarshal(res.Return, &machineInfoList)
				if err != nil {
					callback(nil, err.Error())
				} else {
					callback(machineInfoList, "")
				}
			}
		}
		cmd = &Command{
			Execute: "query-machines",
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) Quit(callback StringCallback) {
	var (
		cb = func(res *Response) {
			if res.ErrorVal != nil {
				callback(res.ErrorVal.Error())
			} else {
				callback(string(res.Return))
			}
		}
		cmd = &Command{
			Execute: "quit",
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) InfoQtree(cb StringCallback) {
	m.HumanMonitorCommand("info qtree", cb)
}

func (m *QmpMonitor) GetHotPluggableCpus(callback HotpluggableCPUListCallback) {
	var (
		cb = func(res *Response) {
			if res.ErrorVal != nil {
				callback(nil, res.ErrorVal.Error())
			} else {
				cpuList := make([]HotpluggableCPU, 0)
				err := json.Unmarshal(res.Return, &cpuList)
				if err != nil {
					callback(nil, err.Error())
				} else {
					callback(cpuList, "")
				}
			}
		}
		cmd = &Command{
			Execute: "query-hotpluggable-cpus",
		}
	)
	m.Query(cmd, cb)
}
