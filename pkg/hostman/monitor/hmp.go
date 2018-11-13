package monitor

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"time"

	"yunion.io/x/log"
)

type HmpMonitor struct {
	SBaseMonitor

	commandQueue  []string
	callbackQueue []StringCallback
}

func NewHmpMonitor(OnMonitorConnected MonitorSuccFunc, OnMonitorDisConnect, OnMonitorTimeout MonitorErrorFunc) *HmpMonitor {
	return &HmpMonitor{
		SBaseMonitor:  *NewBaseMonitor(OnMonitorConnected, OnMonitorDisConnect, OnMonitorTimeout),
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
		if m.connected {
			go m.callBack(res)
		} else {
			// remove reader timeout
			m.connected = true
			m.rwc.SetReadDeadline(time.Time{})
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

func (m *HmpMonitor) callBack(res string) {
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

func (m *HmpMonitor) write(cmd []byte) error {
	cmd = append(cmd, '\n')
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
			log.Errorf("Write %s to monitor error: %s", cmd, err)
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

func (m *HmpMonitor) Connect(host string, port int) error {
	err := m.SBaseMonitor.Connect(host, port)
	if err != nil {
		return err
	}
	go m.read(m.rwc)
	return nil
}

func (m *HmpMonitor) QueryStatus(callback StringCallback) {
	m.Query("info status", m.parseStatus(callback))
}

func (m *HmpMonitor) parseStatus(callback StringCallback) StringCallback {
	return func(res string) {
		strs := strings.Split(res, "\r\n")
		for _, str := range strs {
			if strings.HasPrefix(str, "VM status:") {
				callback(strings.TrimSpace(str[len("VM status:"):]))
				return
			}
		}
	}
}

func (m *HmpMonitor) SimpleCommand(cmd string, callback StringCallback) {
	m.Query(cmd, callback)
}

func (m *HmpMonitor) GetVersion(callback StringCallback) {
	m.Query("info version", callback)
}
