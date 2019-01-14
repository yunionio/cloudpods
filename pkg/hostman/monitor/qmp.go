package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
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

type qmpMonitorCallBack func(*Response)

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

	commandQueue  []*Command
	callbackQueue []qmpMonitorCallBack
}

func NewQmpMonitor(OnMonitorDisConnect, OnMonitorTimeout MonitorErrorFunc,
	OnMonitorConnected MonitorSuccFunc) *QmpMonitor {
	m := &QmpMonitor{
		SBaseMonitor:  *NewBaseMonitor(OnMonitorConnected, OnMonitorDisConnect, OnMonitorTimeout),
		commandQueue:  make([]*Command, 0),
		callbackQueue: make([]qmpMonitorCallBack, 0),
	}

	// qmp init must set capabilities
	m.commandQueue = append(m.commandQueue, &Command{Execute: "qmp_capabilities"})
	m.callbackQueue = append(m.callbackQueue, nil)
	return m
}

func (m *QmpMonitor) actionResult(res *Response) string {
	if res.ErrorVal != nil {
		return res.ErrorVal.Error()
	} else {
		return ""
	}
}

func (m *QmpMonitor) callBack(res *Response) {
	m.mutex.Lock()
	if len(m.callbackQueue) == 0 {
		return
	}
	cb := m.callbackQueue[0]
	m.callbackQueue = m.callbackQueue[1:]
	m.mutex.Unlock()
	if cb != nil {
		go cb(res)
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
			res.Return = *val
			if id, ok := objmap["id"]; ok {
				res.Id = string(*id)
			}
			m.callBack(res)
		} else if val, ok := objmap["event"]; ok {
			var event = &Event{}
			event.Event = string(*val)
			json.Unmarshal(*objmap["data"], &event.Data)
			json.Unmarshal(*objmap["timestamp"], event.Timestamp)
			m.watchEvent(event)
		} else if val, ok := objmap["QMP"]; ok {
			var version Version
			json.Unmarshal(*val, &version)
			m.QemuVersion = version.String()
			m.connected = true
			// remove reader timeout
			m.rwc.SetReadDeadline(time.Time{})
			go m.query()
			go m.OnMonitorConnected()
		}
	}

	log.Errorln("Scan over ...")
	if err := scanner.Err(); err != nil {
		log.Errorln(err)
		if m.connected {
			m.connected = false
			m.OnMonitorDisConnect(err)
		} else {
			m.OnMonitorTimeout(err)
		}
	}
	m.reading = false
}

func (m QmpMonitor) watchEvent(event *Event) {
	log.Infof(event.String())
}

func (m *QmpMonitor) write(cmd []byte) error {
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
		//pop
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

func (m *QmpMonitor) Connect(host string, port int) error {
	err := m.SBaseMonitor.Connect(host, port)
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
			callback(status.(string))
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
		callback(jr.(*jsonutils.JSONArray))
	}

	cmd := &Command{Execute: "query-block"}
	m.Query(cmd, cb)
}

func (m *QmpMonitor) ChangeCdrom(dev string, path string, callback StringCallback) {
	var (
		args = map[string]interface{}{
			"arguments": map[string]interface{}{
				"device": dev,
				"target": path,
			},
		}
		cmd = &Command{
			Execute: "change",
			Args:    args,
		}

		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
	)

	m.Query(cmd, cb)
}

func (m *QmpMonitor) EjectCdrom(dev string, callback StringCallback) {
	var (
		args = map[string]interface{}{
			"arguments": map[string]interface{}{
				"device": dev,
				"force":  true,
			},
		}
		cmd = &Command{
			Execute: "eject",
			Args:    args,
		}

		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
	)

	m.Query(cmd, cb)
}

func (m *QmpMonitor) DriveDel(idstr string, callback StringCallback) {
	var (
		args = map[string]interface{}{
			"arguments": map[string]interface{}{
				"device": idstr,
			},
		}
		cmd = &Command{
			Execute: "drive_del",
			Args:    args,
		}

		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
	)

	m.Query(cmd, cb)
}

func (m *QmpMonitor) DeviceDel(idstr string, callback StringCallback) {
	var (
		args = map[string]interface{}{
			"arguments": map[string]interface{}{
				"device": idstr,
			},
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

func (m *QmpMonitor) DriveAdd(bus string, params map[string]string, callback StringCallback) {
	var (
		args = map[string]interface{}{
			"arguments": map[string]interface{}{
				"bus":    bus,
				"params": params,
			},
		}
		cmd = &Command{
			Execute: "drive_add",
			Args:    args,
		}

		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
	)

	m.Query(cmd, cb)
}

func (m *QmpMonitor) DeviceAdd(dev string, params map[string]interface{}, callback StringCallback) {
	var (
		args = map[string]interface{}{
			"arguments": map[string]interface{}{
				"device": dev,
				"params": params,
			},
		}
		cmd = &Command{
			Execute: "device_add",
			Args:    args,
		}

		cb = func(res *Response) {
			callback(m.actionResult(res))
		}
	)

	m.Query(cmd, cb)
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
		Execute: "query-migrate-capabilities",
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

func (m *QmpMonitor) GetBlockJobs(callback func(jobs int)) {
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
