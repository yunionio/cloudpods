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
	"strings"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type StringCallback func(string)

type BlockJob struct {
	server string

	Busy bool
	// commit|
	Type   string
	Len    int64
	Paused bool
	Ready  bool
	// running|ready
	Status string
	// ok|
	IoStatus string `json:"io-status"`
	Offset   int64
	Device   string
	Speed    int64

	Start     time.Time
	PreOffset int64
	Now       time.Time
	SpeedMbps float64
}

type QemuBlock struct {
	IoStatus  string `json:"io-status"`
	Device    string
	Locked    bool
	Removable bool
	Qdev      string
	TrayOpen  bool
	Type      string
	Inserted  struct {
		Ro               bool
		Drv              string
		Encrypted        bool
		File             string
		BackingFile      string
		BackingFileDepth int
		Bps              int64
		BpsRd            int64
		BpsWr            int
		Iops             int64
		IopsRd           int
		IopsWr           int
		BpsMax           int64
		BpsRdMax         int64
		BpsWrMax         int64
		IopsMax          int
		IopsRdMax        int
		IopsWrMax        int
		IopsSize         int64
		DetectZeroes     string
		WriteThreshold   int
		Image            struct {
			Filename              string
			Format                string
			VirtualSize           int64 `json:"virtual-size"`
			BackingFile           string
			FullBackingFilename   string `json:"full-backing-filename"`
			BackingFilenameFormat string `json:"backing-filename-format"`
			Snapshots             []struct {
				Id          string
				Name        string
				VmStateSize int   `json:"vm-state-size"`
				DateSec     int64 `json:"date-sec"`
				DateNsec    int   `json:"date-nsec"`
				VmClockSec  int   `json:"vm-clock-sec"`
				VmClockNsec int   `json:"vm-clock-nsec"`
			}
			BackingImage struct {
				filename    string
				format      string
				VirtualSize int64 `json:"virtual-size"`
			} `json:"backing-image"`
		}
	}
}

type MigrationInfo struct {
	Status                *MigrationStatus  `json:"status,omitempty"`
	RAM                   *MigrationStats   `json:"ram,omitempty"`
	Disk                  *MigrationStats   `json:"disk,omitempty"`
	XbzrleCache           *XbzrleCacheStats `json:"xbzrle-cache,omitempty"`
	TotalTime             *int64            `json:"total-time,omitempty"`
	ExpectedDowntime      *int64            `json:"expected-downtime,omitempty"`
	Downtime              *int64            `json:"downtime,omitempty"`
	SetupTime             *int64            `json:"setup-time,omitempty"`
	CPUThrottlePercentage *int64            `json:"cpu-throttle-percentage,omitempty"`
	ErrorDesc             *string           `json:"error-desc,omitempty"`
}

type MigrationStats struct {
	Transferred      int64   `json:"transferred"`
	Remaining        int64   `json:"remaining"`
	Total            int64   `json:"total"`
	Duplicate        int64   `json:"duplicate"`
	Skipped          int64   `json:"skipped"`
	Normal           int64   `json:"normal"`
	NormalBytes      int64   `json:"normal-bytes"`
	DirtyPagesRate   int64   `json:"dirty-pages-rate"`
	Mbps             float64 `json:"mbps"`
	DirtySyncCount   int64   `json:"dirty-sync-count"`
	PostcopyRequests int64   `json:"postcopy-requests"`
	PageSize         int64   `json:"page-size"`
}

// XbzrleCacheStats implements the "XBZRLECacheStats" QMP API type.
type XbzrleCacheStats struct {
	CacheSize     int64   `json:"cache-size"`
	Bytes         int64   `json:"bytes"`
	Pages         int64   `json:"pages"`
	CacheMiss     int64   `json:"cache-miss"`
	CacheMissRate float64 `json:"cache-miss-rate"`
	Overflow      int64   `json:"overflow"`
}

// MigrationStatus implements the "MigrationStatus" QMP API type.
type MigrationStatus string

type MigrateStatsCallback func(*MigrationInfo, error)

type blockSizeByte int64

func (self blockSizeByte) String() string {
	size := map[string]float64{
		"Kb": 1024,
		"Mb": 1024 * 1024,
		"Gb": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}
	for _, unit := range []string{"TB", "Gb", "Mb", "Kb"} {
		if int64(self)/int64(size[unit]) > 0 {
			return fmt.Sprintf("%.2f%s", float64(self)/size[unit], unit)
		}
	}
	return fmt.Sprintf("%d", int64(self))
}

func (self *BlockJob) CalcOffset(preOffset int64) {
	if self.Start.IsZero() {
		self.Start = time.Now()
		self.Now = time.Now()
		self.PreOffset = preOffset
		return
	}
	second := time.Now().Sub(self.Now).Seconds()
	if second > 0 {
		speed := float64(self.Offset-preOffset) / second
		self.SpeedMbps = speed / 1024 / 1024
		avgSpeed := float64(self.Offset) / time.Now().Sub(self.Start).Seconds()
		log.Infof(`[%s / %s] server %s block job for %s speed: %s/s(avg: %s/s)`, blockSizeByte(self.Offset).String(), blockSizeByte(self.Len).String(), self.server, self.Device, blockSizeByte(speed).String(), blockSizeByte(avgSpeed).String())
	}
	self.PreOffset = preOffset
	self.Now = time.Now()
	return
}

type Monitor interface {
	Connect(host string, port int) error
	ConnectWithSocket(address string, timeout time.Duration) error
	Disconnect()
	IsConnected() bool

	// The callback function will be called in another goroutine
	SimpleCommand(cmd string, callback StringCallback)
	HumanMonitorCommand(cmd string, callback StringCallback)
	QemuMonitorCommand(cmd string, callback StringCallback) error

	QueryStatus(StringCallback)
	GetVersion(StringCallback)
	GetBlockJobCounts(func(jobs int))
	GetBlockJobs(func([]BlockJob))
	QueryPci(callback QueryPciCallback)
	InfoQtree(cb StringCallback)

	GetCpuCount(func(count int))
	AddCpu(cpuIndex int, callback StringCallback)
	GetHotPluggableCpus(HotpluggableCPUListCallback)
	GeMemtSlotIndex(func(index int))
	GetMemoryDevicesInfo(QueryMemoryDevicesCallback)
	GetMemdevList(MemdevListCallback)

	GetBlocks(callback func([]QemuBlock))
	EjectCdrom(dev string, callback StringCallback)
	ChangeCdrom(dev string, path string, callback StringCallback)

	DriveDel(idstr string, callback StringCallback)
	DeviceDel(idstr string, callback StringCallback)
	ObjectDel(idstr string, callback StringCallback)

	ObjectAdd(objectType string, params map[string]string, callback StringCallback)
	DriveAdd(bus, node string, params map[string]string, callback StringCallback)
	DeviceAdd(dev string, params map[string]string, callback StringCallback)

	XBlockdevChange(parent, node, child string, callback StringCallback)
	BlockStream(drive string, callback StringCallback)
	DriveMirror(callback StringCallback, drive, target, syncMode, format string, unmap, blockReplication bool, speed int64)
	DriveBackup(callback StringCallback, drive, target, syncMode, format string)
	BlockJobComplete(drive string, cb StringCallback)
	BlockReopenImage(drive, newImagePath, format string, cb StringCallback)
	SnapshotBlkdev(drive, newImagePath, format string, reuse bool, cb StringCallback)

	MigrateSetDowntime(dtSec float64, callback StringCallback)
	MigrateSetCapability(capability, state string, callback StringCallback)
	MigrateSetParameter(key string, val interface{}, callback StringCallback)
	MigrateIncoming(address string, callback StringCallback)
	Migrate(destStr string, copyIncremental, copyFull bool, callback StringCallback)
	MigrateContinue(state string, callback StringCallback)
	GetMigrateStatus(callback StringCallback)
	MigrateStartPostcopy(callback StringCallback)
	GetMigrateStats(callback MigrateStatsCallback)
	MigrateCancel(cb StringCallback)

	ReloadDiskBlkdev(device, path string, callback StringCallback)
	SetVncPassword(proto, password string, callback StringCallback)
	StartNbdServer(port int, exportAllDevice, writable bool, callback StringCallback)
	StopNbdServer(callback StringCallback)

	ResizeDisk(driveName string, sizeMB int64, callback StringCallback)
	BlockIoThrottle(driveName string, bps, iops int64, callback StringCallback)
	CancelBlockJob(driveName string, force bool, callback StringCallback)

	NetdevAdd(id, netType string, params map[string]string, callback StringCallback)
	NetdevDel(id string, callback StringCallback)

	SaveState(statFilePath string, callback StringCallback)
	QueryMachines(callback QueryMachinesCallback)
	Quit(StringCallback)
}

type MonitorErrorFunc func(error)
type MonitorSuccFunc func()

type SBaseMonitor struct {
	OnMonitorDisConnect MonitorErrorFunc
	OnMonitorConnected  MonitorSuccFunc
	OnMonitorTimeout    MonitorErrorFunc

	server string
	sid    string

	QemuVersion string
	connected   bool
	timeout     bool
	rwc         net.Conn

	mutex   *sync.Mutex
	writing bool
	reading bool
}

func NewBaseMonitor(server, sid string, OnMonitorConnected MonitorSuccFunc, OnMonitorDisConnect, OnMonitorTimeout MonitorErrorFunc) *SBaseMonitor {
	return &SBaseMonitor{
		OnMonitorConnected:  OnMonitorConnected,
		OnMonitorDisConnect: OnMonitorDisConnect,
		OnMonitorTimeout:    OnMonitorTimeout,
		server:              server,
		sid:                 sid,
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

func (m *SBaseMonitor) SetReadDeadlineTimeout(duration time.Duration) {
	if m.rwc != nil {
		m.rwc.SetReadDeadline(time.Now().Add(duration))
	}
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

func (m *SBaseMonitor) QemuMonitorCommand(cmd string, callback StringCallback) error {
	return errors.ErrNotSupported
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

func (m *SBaseMonitor) parseStatus(callback StringCallback) StringCallback {
	return func(output string) {
		strs := strings.Split(output, "\r\n")
		for _, str := range strs {
			if strings.HasPrefix(str, "VM status:") {
				callback(strings.TrimSpace(
					strings.Trim(str[len("VM status:"):], "\\r\\n"),
				))
				return
			}
		}
	}
}

func getSaveStatefileUri(stateFilePath string) string {
	if strings.HasSuffix(stateFilePath, ".gz") {
		return fmt.Sprintf("exec:gzip -c > %s", stateFilePath)
	}
	return fmt.Sprintf("exec:cat > %s", stateFilePath)
}
