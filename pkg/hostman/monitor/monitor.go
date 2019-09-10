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
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

type StringCallback func(string)

type Monitor interface {
	Connect(host string, port int) error
	ConnectWithSocket(address string) error
	Disconnect()
	IsConnected() bool

	// The callback function will be called in another goroutine
	SimpleCommand(cmd string, callback StringCallback)
	HumanMonitorCommand(cmd string, callback StringCallback)

	QueryStatus(StringCallback)
	GetVersion(StringCallback)
	GetBlockJobCounts(func(jobs int))
	GetBlockJobs(func(*jsonutils.JSONArray))

	GetCpuCount(func(count int))
	AddCpu(cpuIndex int, callback StringCallback)
	GeMemtSlotIndex(func(index int))

	GetBlocks(callback func(*jsonutils.JSONArray))
	EjectCdrom(dev string, callback StringCallback)
	ChangeCdrom(dev string, path string, callback StringCallback)

	DriveDel(idstr string, callback StringCallback)
	DeviceDel(idstr string, callback StringCallback)
	ObjectDel(idstr string, callback StringCallback)

	ObjectAdd(objectType string, params map[string]string, callback StringCallback)
	DriveAdd(bus string, params map[string]string, callback StringCallback)
	DeviceAdd(dev string, params map[string]interface{}, callback StringCallback)

	BlockStream(drive string, callback StringCallback)
	DriveMirror(callback StringCallback, drive, target, syncMode string, unmap bool)

	MigrateSetCapability(capability, state string, callback StringCallback)
	Migrate(destStr string, copyIncremental, copyFull bool, callback StringCallback)
	GetMigrateStatus(callback StringCallback)

	ReloadDiskBlkdev(device, path string, callback StringCallback)
	SetVncPassword(proto, password string, callback StringCallback)
	StartNbdServer(port int, exportAllDevice, writable bool, callback StringCallback)

	ResizeDisk(driveName string, sizeMB int64, callback StringCallback)
	CancelBlockJob(driveName string, force bool, callback StringCallback)
}

type MonitorErrorFunc func(error)
type MonitorSuccFunc func()

type SBaseMonitor struct {
	OnMonitorDisConnect MonitorErrorFunc
	OnMonitorConnected  MonitorSuccFunc
	OnMonitorTimeout    MonitorErrorFunc

	QemuVersion string
	connected   bool
	timeout     bool
	rwc         net.Conn

	mutex   *sync.Mutex
	writing bool
	reading bool
}

func NewBaseMonitor(OnMonitorConnected MonitorSuccFunc, OnMonitorDisConnect, OnMonitorTimeout MonitorErrorFunc) *SBaseMonitor {
	return &SBaseMonitor{
		OnMonitorConnected:  OnMonitorConnected,
		OnMonitorDisConnect: OnMonitorDisConnect,
		OnMonitorTimeout:    OnMonitorTimeout,
		timeout:             true,
		mutex:               &sync.Mutex{},
	}
}

func (m *SBaseMonitor) connect(protocol, address string) error {
	conn, err := net.Dial(protocol, address)
	if err != nil {
		return errors.Errorf("Connect to %s %s failed %s", protocol, address, err)
	}
	log.Infof("Connect %s %s success", protocol, address)
	m.onConnectSuccess(conn)
	return nil
}

func (m *SBaseMonitor) onConnectSuccess(conn net.Conn) {
	// Setup reader timeout
	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	// set rwc hand
	m.rwc = conn
}

func (m *SBaseMonitor) Connect(host string, port int) error {
	return m.connect("tcp", fmt.Sprintf("%s:%d", host, port))
}

func (m *SBaseMonitor) ConnectWithSocket(address string) error {
	return m.connect("unix", address)
}

func (m *SBaseMonitor) Disconnect() {
	if m.connected {
		m.connected = false
		m.rwc.Close()
	}
}

func (m *SBaseMonitor) IsConnected() bool {
	return m.connected
}

func (m *SBaseMonitor) checkReading() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.reading {
		return false
	} else {
		m.reading = true
	}
	return true
}

func (m *SBaseMonitor) checkWriting() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.writing {
		return false
	} else {
		m.writing = true
	}
	return true
}
