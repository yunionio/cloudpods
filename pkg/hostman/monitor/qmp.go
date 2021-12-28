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

var ignoreEvents = []string{"RTC_CHANGE"}

type qmpMonitorCallBack func(*Response)
type qmpEventCallback func(*Event)

type Response struct {
	Return   jsonutils.JSONObject
	ErrorVal *Error
	Id       string
}

type Event struct {
	Event string `json:"event"`
	Data  struct {
		// ACPI_DEVICE_OST
		Device   *string
		Slot     *string
		SlotType *string `json:"slot-type"`
		Source   *string
		Status   *string

		// BALLOON_CHANGE
		Actual *int64 // "actual": actual level of the guest memory balloon in bytes (json-number)

		// BLOCK_IMAGE_CORRUPTED
		/*
			- "device":    Device name (json-string)
			- "node-name": Node name (json-string, optional)
			- "msg":       Informative message (e.g., reason for the corruption)
			               (json-string)
			- "offset":    If the corruption resulted from an image access, this
			               is the host's access offset into the image
			               (json-int, optional)
			- "size":      If the corruption resulted from an image access, this
			               is the access size (json-int, optional)
		*/
		NodeName *string `json:"node-name"`
		Msg      *string `json:"msg"`
		Offset   *int64
		Size     *int64

		// BLOCK_IO_ERROR
		/*
			- "device": device name (json-string)
			- "operation": I/O operation (json-string, "read" or "write")
			- "action": action that has been taken, it's one of the following (json-string):
			    "ignore": error has been ignored
				"report": error has been reported to the device
				"stop": the VM is going to stop because of the error
		*/
		Operation *string `json:"operation"`
		Action    *string `json:"action"`

		// BLOCK_JOB_CANCELLED
		/*
			- "type":     Job type (json-string; "stream" for image streaming
			                                     "commit" for block commit)
			- "device":   Job identifier. Originally the device name but other
			              values are allowed since QEMU 2.7 (json-string)
			- "len":      Maximum progress value (json-int)
			- "offset":   Current progress value (json-int)
			              On success this is equal to len.
			              On failure this is less than len.
			- "speed":    Rate limit, bytes per second (json-int)
		*/
		Type  *string
		Len   *int64
		Speed *int64

		// BLOCK_JOB_COMPLETED
		/*
			- "type":     Job type (json-string; "stream" for image streaming
			                                     "commit" for block commit)
			- "device":   Job identifier. Originally the device name but other
			              values are allowed since QEMU 2.7 (json-string)
			- "len":      Maximum progress value (json-int)
			- "offset":   Current progress value (json-int)
			              On success this is equal to len.
			              On failure this is less than len.
			- "speed":    Rate limit, bytes per second (json-int)
			- "error":    Error message (json-string, optional)
			              Only present on failure.  This field contains a human-readable
			              error message.  There are no semantics other than that streaming
			              has failed and clients should not try to interpret the error
			              string.
		*/
		Error *string

		// BLOCK_JOB_ERROR
		/*
			- "device": Job identifier. Originally the device name but other
			            values are allowed since QEMU 2.7 (json-string)
			- "operation": I/O operation (json-string, "read" or "write")
			- "action": action that has been taken, it's one of the following (json-string):
			    "ignore": error has been ignored, the job may fail later
			    "report": error will be reported and the job canceled
			    "stop": error caused job to be paused
		*/
		// BLOCK_JOB_READY
		/*
			- "type":     Job type (json-string; "stream" for image streaming
			                                     "commit" for block commit)
			- "device":   Job identifier. Originally the device name but other
			              values are allowed since QEMU 2.7 (json-string)
			- "len":      Maximum progress value (json-int)
			- "offset":   Current progress value (json-int)
			              On success this is equal to len.
			              On failure this is less than len.
			- "speed":    Rate limit, bytes per second (json-int)
		*/
		// DEVICE_DELETED
		/*
			- "device": device name (json-string, optional)
			- "path": device path (json-string)
		*/
		Path *string

		// DEVICE_TRAY_MOVED
		/*
			- "device": device name (json-string)
			- "tray-open": true if the tray has been opened or false if it has been closed
			               (json-bool)
		*/
		TryOpen *bool `json:"try-open"`

		// DUMP_COMPLETED
		/*
			- "result": DumpQueryResult type described in qapi-schema.json
			- "error": Error message when dump failed. This is only a
			  human-readable string provided when dump failed. It should not be
			  parsed in any way (json-string, optional)
		*/
		Result *struct {
			Total     int64
			Status    string
			completed int64
		}

		// GUEST_PANICKED
		/*
			- "action": Action that has been taken (json-string, currently always "pause").
		*/

		Info *string //????

		// MEM_UNPLUG_ERROR
		/*
			- "device": device name (json-string)
			- "msg": Informative message (e.g., reason for the error) (json-string)
		*/

		// NIC_RX_FILTER_CHANGED
		/*
			- "name": net client name (json-string)
			- "path": device path (json-string)
		*/
		Name *string

		// POWERDOWN

		// QUORUM_FAILURE
		/*
			- "reference":     device name if defined else node name.
			- "sector-num":    Number of the first sector of the failed read operation.
			- "sectors-count": Failed read operation sector count.
		*/
		Reference    *string
		SectorNum    *int64 `json:"sector-num"`
		SectorsCount *int64 `json:"sectors-count"`

		// QUORUM_REPORT_BAD
		/*
			- "type":          Quorum operation type
			- "error":         Error message (json-string, optional)
			                   Only present on failure.  This field contains a human-readable
			                   error message.  There are no semantics other than that the
			                   block layer reported an error and clients should not try to
			                   interpret the error string.
			- "node-name":     The graph node name of the block driver state.
			- "sector-num":    Number of the first sector of the failed read operation.
			- "sectors-count": Failed read operation sector count.
		*/

		// RESET

		// RESUME

		// RTC_CHANGE
		/*
			- "offset": Offset between base RTC clock (as specified by -rtc base), and
			new RTC clock value (json-number)
		*/

		// SHUTDOWN

		// SPICE_DISCONNECTED
		/*
			- "server": Server information (json-object)
			  - "host": IP address (json-string)
			  - "port": port number (json-string)
			  - "family": address family (json-string, "ipv4" or "ipv6")
			- "client": Client information (json-object)
			  - "host": IP address (json-string)
			  - "port": port number (json-string)
			  - "family": address family (json-string, "ipv4" or "ipv6")
		*/
		Server *struct {
			Port   string
			Host   string
			Family string
			Auth   string
		}
		Client *struct {
			Port         string
			Host         string
			Family       string
			ConnectionId string `json:"connection-id"`
			ChannelType  string `json:"channel-type"`
			ChannelId    string `json:"channel-id"`
			Tls          bool
		}

		// SPICE_INITIALIZED

		// SPICE_INITIALIZED
		/*
			- "server": Server information (json-object)
			  - "host": IP address (json-string)
			  - "port": port number (json-string)
			  - "family": address family (json-string, "ipv4" or "ipv6")
			  - "auth": authentication method (json-string, optional)
			- "client": Client information (json-object)
			  - "host": IP address (json-string)
			  - "port": port number (json-string)
			  - "family": address family (json-string, "ipv4" or "ipv6")
			  - "connection-id": spice connection id.  All channels with the same id
			                     belong to the same spice session (json-int)
			  - "channel-type": channel type.  "1" is the main control channel, filter for
			                    this one if you want track spice sessions only (json-int)
			  - "channel-id": channel id.  Usually "0", might be different needed when
			                  multiple channels of the same type exist, such as multiple
			                  display channels in a multihead setup (json-int)
			  - "tls": whevener the channel is encrypted (json-bool)
		*/

		// SPICE_MIGRATE_COMPLETED

		// MIGRATION
		/*
			- "status": migration status
				See MigrationStatus in ~/qapi-schema.json for possible values
		*/

		// MIGRATION_PASS

		// STOP

		// SUSPEND

		// SUSPEND_DISK

		// VNC_CONNECTED

		// VNC_DISCONNECTED

		// VNC_INITIALIZED

		// VSERPORT_CHANGE

		// WAKEUP

		// WATCHDOG
	} `json:"data"`
	Timestamp *Timestamp `json:"timestamp"`
}

func (e *Event) String() string {
	return fmt.Sprintf("QMP Event %s result: %s", e.Event, jsonutils.Marshal(e.Data).String())
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
		obj := struct {
			Error  *Error
			Return jsonutils.JSONObject
			Id     string
			Event  string
			Qmp    *struct {
				Version Version
			}
		}{}
		b := scanner.Bytes()
		log.Debugf("QMP Read %s: %s", m.server, string(b))
		jobj, err := jsonutils.Parse(b)
		if err != nil {
			log.Errorf("Parse %s %s error: %s", m.server, string(b), err.Error())
			continue
		}
		jobj.Unmarshal(&obj)
		if obj.Error != nil || obj.Return != nil {
			var res = &Response{
				Return:   obj.Return,
				Id:       obj.Id,
				ErrorVal: obj.Error,
			}
			m.callBack(res)
		} else if len(obj.Event) > 0 {
			event := &Event{}
			jobj.Unmarshal(event)
			m.watchEvent(event)
		} else if obj.Qmp != nil {
			// On qmp connected
			m.QemuVersion = obj.Qmp.Version.String()

			// remove reader timeout
			m.rwc.SetReadDeadline(time.Time{})
			m.connected = true
			m.timeout = false
			go m.query()
			go m.OnMonitorConnected()
		}
	}

	log.Infof("Scan over %s ...", m.server)
	err := scanner.Err()
	if err != nil {
		log.Infof("QMP Disconnected %s: %s", m.server, err)
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
				callback(res.Return.String())
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
			log.Debugf("Monitor %s ret: %s", m.server, res.Return)
			if res.ErrorVal != nil {
				callback(res.ErrorVal.Error())
			} else {
				callback(strings.Trim(res.Return.String(), `""`))
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
		status := struct {
			Status string
		}{}
		res.Return.Unmarshal(&status)
		if len(status.Status) > 0 {
			callback(status.Status)
			return
		}
		callback("unknown")
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
		version := Version{}
		res.Return.Unmarshal(&version)
		if version.QEMU.Major > 0 {
			callback(version.String())
			return
		}
		callback("")
	}
}

func (m *QmpMonitor) GetBlocks(callback func(*jsonutils.JSONArray)) {
	var cb = func(res *Response) {
		if res.ErrorVal != nil {
			callback(nil)
		}
		jra, _ := res.Return.(*jsonutils.JSONArray)
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
				return
			}
			/*
				{"expected-downtime":300,"ram":{"dirty-pages-rate":0,"dirty-sync-count":1,"duplicate":2966538,"mbps":268.5672,"normal":148629,"normal-bytes":608784384,"page-size":4096,"postcopy-requests":0,"remaining":142815232,"skipped":0,"total":12902539264,"transferred":636674057},"setup-time":65,"status":"active","total-time":20002}
				{"disk":{"dirty-pages-rate":0,"dirty-sync-count":0,"duplicate":0,"mbps":0,"normal":0,"normal-bytes":0,"page-size":0,"postcopy-requests":0,"remaining":0,"skipped":0,"total":139586437120,"transferred":139586437120},"expected-downtime":300,"ram":{"dirty-pages-rate":0,"dirty-sync-count":1,"duplicate":193281,"mbps":268.44264,"normal":62311,"normal-bytes":255225856,"page-size":4096,"postcopy-requests":0,"remaining":44474368,"skipped":0,"total":1091379200,"transferred":257555032},"setup-time":15,"status":"active","total-time":10002}
			*/

			ret := struct {
				Status string
				Ram    struct {
					Total     int64
					Remaining int64
					Mbps      float64
				}
				Disk struct {
					Total     int64
					Remaining int64
					Mbps      float64
				}
			}{}
			res.Return.Unmarshal(&ret)

			log.Infof("Query migrate status %s: %s", m.server, ret.Status)

			status := ret.Status
			if status == "active" {
				if ret.Disk.Remaining > 0 {
					status = "migrate_disk_copy"
				} else if ret.Ram.Remaining > 0 {
					status = "migrate_ram_copy"
				}
				mbps := ret.Disk.Mbps + ret.Ram.Mbps
				progress := 50 + (1.0-float64(ret.Disk.Remaining+ret.Ram.Remaining)/float64(ret.Disk.Total+ret.Ram.Total))*100.0*0.5
				log.Debugf("progress: %f mbps: %f", progress, mbps)
				hostutils.UpdateServerProgress(context.Background(), m.sid, progress, mbps)
			}
			callback(status)
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) MigrateStartPostcopy(callback StringCallback) {
	var (
		cmd = &Command{Execute: "migrate-start-postcopy"}
		cb  = func(res *Response) {
			if res.ErrorVal != nil {
				callback(res.ErrorVal.Error())
				return
			}
			callback("")
		}
	)
	m.Query(cmd, cb)
}

func (m *QmpMonitor) blockJobs(res *Response) ([]BlockJob, error) {
	if res.ErrorVal != nil {
		return nil, errors.Errorf("GetBlockJobs for %s %s", m.server, jsonutils.Marshal(res.ErrorVal).String())
	}
	jobs := []BlockJob{}
	res.Return.Unmarshal(&jobs)
	defer func() {
		mbps, progress := 0.0, 0.0
		totalSize, totalOffset := int64(1), int64(0)
		for _, job := range m.jobs {
			mbps += job.speedMbps
			totalSize += job.Len
			totalOffset += job.Offset
		}
		if len(m.jobs) == 0 && len(jobs) == 0 {
			progress = 100.0
		} else {
			progress = float64(totalOffset) / float64(totalSize) * 100
		}
		hostutils.UpdateServerProgress(context.Background(), m.sid, progress, mbps)
	}()
	for i := range jobs {
		job := jobs[i]
		job.server = m.server
		_job, ok := m.jobs[job.Device]
		if !ok {
			job.PreOffset(0)
			m.jobs[job.Device] = job
			continue
		}
		if _job.Status == "ready" {
			delete(m.jobs, _job.Device)
			continue
		}
		job.start, job.now = _job.start, _job.now
		job.PreOffset(_job.Offset)
		m.jobs[job.Device] = job
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

func (m *QmpMonitor) DriveMirror(callback StringCallback, drive, target, syncMode string, unmap, blockReplication bool) {
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
	if blockReplication {
		args["block-replication"] = true
	}
	cmd := &Command{
		Execute: "drive-mirror",
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
