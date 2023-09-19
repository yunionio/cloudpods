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

	start     time.Time
	preOffset int64
	now       time.Time
	speedMbps float64
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

func (self *BlockJob) PreOffset(preOffset int64) {
	if self.start.IsZero() {
		self.start = time.Now()
		self.now = time.Now()
		self.preOffset = preOffset
		return
	}
	second := time.Now().Sub(self.now).Seconds()
	if second > 0 {
		speed := float64(self.Offset-preOffset) / second
		self.speedMbps = speed / 1024 / 1024
		avgSpeed := float64(self.Offset) / time.Now().Sub(self.start).Seconds()
		log.Infof(`[%s / %s] server %s block job for %s speed: %s/s(avg: %s/s)`, blockSizeByte(self.Offset).String(), blockSizeByte(self.Len).String(), self.server, self.Device, blockSizeByte(speed).String(), blockSizeByte(avgSpeed).String())
	}
	self.preOffset = preOffset
	self.now = time.Now()
	return
}

// Memdev -> Memdev (struct)

// Memdev implements the "Memdev" QMP API type.
type Memdev struct {
	ID        *string  `json:"id,omitempty"`
	Size      uint64   `json:"size"`
	Merge     bool     `json:"merge"`
	Dump      bool     `json:"dump"`
	Prealloc  bool     `json:"prealloc"`
	HostNodes []uint16 `json:"host-nodes"`
	Policy    string   `json:"policy"`
}

type MemdevListCallback func(res []Memdev, err string)

// PciBridgeInfo -> PCIBridgeInfo (struct)

// PCIBridgeInfo implements the "PciBridgeInfo" QMP API type.
type PCIBridgeInfo struct {
	Bus     PCIBusInfo      `json:"bus"`
	Devices []PCIDeviceInfo `json:"devices,omitempty"`
}

// PciBusInfo -> PCIBusInfo (struct)

// PCIBusInfo implements the "PciBusInfo" QMP API type.
type PCIBusInfo struct {
	Number            int64          `json:"number"`
	Secondary         int64          `json:"secondary"`
	Subordinate       int64          `json:"subordinate"`
	IORange           PCIMemoryRange `json:"io_range"`
	MemoryRange       PCIMemoryRange `json:"memory_range"`
	PrefetchableRange PCIMemoryRange `json:"prefetchable_range"`
}

// PciDeviceClass -> PCIDeviceClass (struct)

// PCIDeviceClass implements the "PciDeviceClass" QMP API type.
type PCIDeviceClass struct {
	Desc  *string `json:"desc,omitempty"`
	Class int64   `json:"class"`
}

// PciDeviceId -> PCIDeviceID (struct)

// PCIDeviceID implements the "PciDeviceId" QMP API type.
type PCIDeviceID struct {
	Device int64 `json:"device"`
	Vendor int64 `json:"vendor"`
}

// PciDeviceInfo -> PCIDeviceInfo (struct)

// PCIDeviceInfo implements the "PciDeviceInfo" QMP API type.
type PCIDeviceInfo struct {
	Bus       int64             `json:"bus"`
	Slot      int64             `json:"slot"`
	Function  int64             `json:"function"`
	ClassInfo PCIDeviceClass    `json:"class_info"`
	ID        PCIDeviceID       `json:"id"`
	Irq       *int64            `json:"irq,omitempty"`
	QdevID    string            `json:"qdev_id"`
	PCIBridge *PCIBridgeInfo    `json:"pci_bridge,omitempty"`
	Regions   []PCIMemoryRegion `json:"regions"`
}

// PciInfo -> PCIInfo (struct)

// PCIInfo implements the "PciInfo" QMP API type.
type PCIInfo struct {
	Bus     int64           `json:"bus"`
	Devices []PCIDeviceInfo `json:"devices"`
}

// PciMemoryRange -> PCIMemoryRange (struct)

// PCIMemoryRange implements the "PciMemoryRange" QMP API type.
type PCIMemoryRange struct {
	Base  int64 `json:"base"`
	Limit int64 `json:"limit"`
}

// PciMemoryRegion -> PCIMemoryRegion (struct)

// PCIMemoryRegion implements the "PciMemoryRegion" QMP API type.
type PCIMemoryRegion struct {
	Bar       int64  `json:"bar"`
	Type      string `json:"type"`
	Address   int64  `json:"address"`
	Size      int64  `json:"size"`
	Prefetch  *bool  `json:"prefetch,omitempty"`
	MemType64 *bool  `json:"mem_type_64,omitempty"`
}

type QueryPciCallback func(pciInfoList []PCIInfo, err string)

type Monitor interface {
	Connect(host string, port int) error
	ConnectWithSocket(address string) error
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

	GetCpuCount(func(count int))
	AddCpu(cpuIndex int, callback StringCallback)
	GeMemtSlotIndex(func(index int))
	GetMemdevList(MemdevListCallback)
	GetScsiNumQueues(func(int64))
	QueryPci(callback QueryPciCallback)

	GetBlocks(callback func([]QemuBlock))
	EjectCdrom(dev string, callback StringCallback)
	ChangeCdrom(dev string, path string, callback StringCallback)

	DriveDel(idstr string, callback StringCallback)
	DeviceDel(idstr string, callback StringCallback)
	ObjectDel(idstr string, callback StringCallback)

	ObjectAdd(objectType string, params map[string]string, callback StringCallback)
	DriveAdd(bus, node string, params map[string]string, callback StringCallback)
	DeviceAdd(dev string, params map[string]interface{}, callback StringCallback)

	XBlockdevChange(parent, node, child string, callback StringCallback)
	BlockStream(drive string, idx, blkCnt int, callback StringCallback)
	DriveMirror(callback StringCallback, drive, target, syncMode, format string, unmap, blockReplication bool)
	DriveBackup(callback StringCallback, drive, target, syncMode, format string)
	BlockJobComplete(drive string, cb StringCallback)
	BlockReopenImage(drive, newImagePath, format string, cb StringCallback)
	SnapshotBlkdev(drive, newImagePath, format string, reuse bool, cb StringCallback)

	MigrateSetDowntime(dtSec float64, callback StringCallback)
	MigrateSetCapability(capability, state string, callback StringCallback)
	MigrateSetParameter(key string, val interface{}, callback StringCallback)
	MigrateIncoming(address string, callback StringCallback)
	Migrate(destStr string, copyIncremental, copyFull bool, callback StringCallback)
	GetMigrateStatus(callback StringCallback)
	MigrateStartPostcopy(callback StringCallback)
	GetMigrateStats(callback MigrateStatsCallback)
	MigrateCancel(cb StringCallback)

	ReloadDiskBlkdev(device, path string, callback StringCallback)
	SetVncPassword(proto, password string, callback StringCallback)
	StartNbdServer(port int, exportAllDevice, writable bool, callback StringCallback)

	ResizeDisk(driveName string, sizeMB int64, callback StringCallback)
	BlockIoThrottle(driveName string, bps, iops int64, callback StringCallback)
	CancelBlockJob(driveName string, force bool, callback StringCallback)

	NetdevAdd(id, netType string, params map[string]string, callback StringCallback)
	NetdevDel(id string, callback StringCallback)

	SaveState(statFilePath string, callback StringCallback)
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
