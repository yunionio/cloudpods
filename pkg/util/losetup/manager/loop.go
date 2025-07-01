package manager

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/losetup"
	"yunion.io/x/onecloud/pkg/util/losetup/ioctl"
	"yunion.io/x/onecloud/pkg/util/mountutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type ILoopDevice interface {
	// GetDevicePath 返回设备路径
	GetDevicePath() string
	// IsUsed 检查设备是否在使用中
	IsUsed() bool
	// SetUsed 设置设备使用状态
	SetUsed(used bool)
}

type ILoopManager interface {
	AttachDevice(filePath string, partScan bool) (*losetup.Device, error)
	DetachDevice(devPath string) error
}

// loopDevice Loop设备实现
type loopDevice struct {
	devicePath string
	used       bool
	lock       *sync.Mutex
}

// NewLoopDevice 创建新的Loop设备
func NewLoopDevice(devicePath string) ILoopDevice {
	return &loopDevice{
		devicePath: devicePath,
		used:       fileutils2.IsBlockDeviceUsed(devicePath),
		lock:       &sync.Mutex{},
	}
}

func (d *loopDevice) GetDevicePath() string {
	return d.devicePath
}

func (d *loopDevice) IsUsed() bool {
	return d.used
}

func (d *loopDevice) SetUsed(used bool) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.used = used
}

func (d *loopDevice) Detach() error {
	// 获取所有挂载点
	mountPoints, err := d.getMountPoints()
	if err != nil {
		return errors.Wrap(err, "getMountPoints")
	}

	// 卸载所有挂载点
	for _, mountPoint := range mountPoints {
		if err := mountutils.Unmount(mountPoint, false); err != nil {
			return errors.Wrapf(err, "umount %s", mountPoint)
		}
	}

	// 断开loop设备
	if err := losetup.DetachDevice(d.GetDevicePath()); err != nil {
		return errors.Wrapf(err, "detach loop device %s", d.GetDevicePath())
	}

	return nil
}

func (d *loopDevice) getMountPoints() ([]string, error) {
	cmd := fmt.Sprintf("mount | grep %sp1 | awk '{print $3}'", d.GetDevicePath())
	output, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "exec cmd %s: %s", cmd, output)
	}
	return strings.Split(string(output), "\n"), nil
}

func (m *loopManager) initDevices() error {
	// 使用 ls 和 grep 命令列出所有以数字结尾的 /dev/loop* 设备
	cmd := "ls /dev/loop* | grep -E 'loop[0-9]+$'"
	output, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		log.Errorf("list loop devices error: %v", err)
		output = make([]byte, 0)
	}

	// 按行分割输出
	devices := strings.Split(string(output), "\n")
	for _, dev := range devices {
		dev = strings.TrimSpace(dev)
		if dev == "" {
			continue
		}
		m.devices[dev] = NewLoopDevice(dev)
	}

	return nil
}

// loopManager Loop设备管理器
type loopManager struct {
	devices    map[string]ILoopDevice
	actionLock *sync.Mutex
	mapLock    *sync.Mutex
}

// NewLoopManager 创建新的Loop管理器
func newLoopManager() (ILoopManager, error) {
	ret := &loopManager{
		devices:    make(map[string]ILoopDevice),
		actionLock: &sync.Mutex{},
		mapLock:    new(sync.Mutex),
	}

	if err := ret.initDevices(); err != nil {
		return ret, errors.Wrap(err, "initDevices")
	}
	return ret, nil
}

const (
	MAX_LOOPDEV_COUNT = 512
)

func (m *loopManager) findNewDeviceNumber() (int, error) {
	// 获取所有已使用的设备号
	usedNumbers := make(map[int]bool)
	for name := range m.devices {
		// 从设备名称中提取数字,例如从 /dev/loop0 中提取 0
		numStr := strings.TrimPrefix(name, "/dev/loop")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return -1, errors.Wrapf(err, "parse device number from %s", name)
		}
		usedNumbers[num] = true
	}

	// 从0开始查找第一个未使用的设备号
	for i := 0; i < MAX_LOOPDEV_COUNT; i++ {
		if !usedNumbers[i] {
			// 创建新的loop设备
			return i, nil
		}
	}
	return -1, errors.Wrap(errors.ErrNotFound, "No available device found")
}

func (m *loopManager) AttachDevice(filePath string, partScan bool) (*losetup.Device, error) {
	m.actionLock.Lock()
	defer m.actionLock.Unlock()

	dev, err := m.acquireDevice()
	if err != nil {
		return nil, errors.Wrap(err, "AcquireDevice")
	}
	loDev, err := losetup.AttachDeviceWithPath(dev.GetDevicePath(), filePath, partScan)
	if err != nil {
		return nil, errors.Wrapf(err, "AttachDeviceWithPath: %s, filePath: %s", dev.GetDevicePath(), filePath)
	}

	return loDev, nil
}

func (m *loopManager) DetachDevice(devPath string) error {
	m.actionLock.Lock()
	defer m.actionLock.Unlock()

	if err := losetup.DetachDevice(devPath); err != nil {
		return errors.Wrapf(err, "DetachDevice: %s", devPath)
	}
	m.releaseDevice(devPath)
	return nil
}

func (m *loopManager) acquireDevice() (ILoopDevice, error) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()

	freeDevices := make([]ILoopDevice, 0)
	for _, device := range m.devices {
		if !device.IsUsed() {
			freeDevices = append(freeDevices, device)
		}
	}
	if len(freeDevices) > 0 {
		device := freeDevices[rand.Intn(len(freeDevices))]
		device.SetUsed(true)
		return device, nil
	}

	// create new device
	num, err := m.findNewDeviceNumber()
	if err != nil {
		return nil, errors.Wrap(err, "findNewDeviceNumber")
	}
	devPath, err := ioctl.AddDevice(num)
	if err != nil {
		return nil, errors.Wrapf(err, "add device %d", num)
	}
	device := NewLoopDevice(devPath)
	m.devices[devPath] = device
	device.SetUsed(true)

	return device, nil
}

func (m *loopManager) releaseDevice(devPath string) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()

	if dev, ok := m.devices[devPath]; ok {
		dev.SetUsed(false)
	}
}
