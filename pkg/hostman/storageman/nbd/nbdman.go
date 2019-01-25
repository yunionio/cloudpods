package nbd

import (
	"fmt"
	"sync"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SNBDManager struct {
	nbdDevs map[string]bool
	nbdLock *sync.Mutex
}

var nbdManager *SNBDManager

func init() {
	nbdManager = NewNBDManager()
}

func GetNBDManager() *SNBDManager {
	return nbdManager
}

func NewNBDManager() *SNBDManager {
	var ret = new(SNBDManager)
	ret.nbdDevs = make(map[string]bool, 0)
	ret.nbdLock = new(sync.Mutex)
	ret.findNbdDevices()
	return ret
}

func (m *SNBDManager) findNbdDevices() {
	var i = 0
	for {
		if fileutils2.Exists(fmt.Sprintf("/dev/nbd%d", i)) {
			m.nbdDevs[fmt.Sprintf("/dev/nbd%d", i)] = false
			i++
		} else {
			break
		}
	}
	log.Infof("NBD_DEVS: %#v", m.nbdDevs)
}

func (m *SNBDManager) AcquireNbddev() string {
	defer m.nbdLock.Unlock()
	m.nbdLock.Lock()
	for nbdDev := range m.nbdDevs {
		if fileutils2.IsBlockDeviceUsed(nbdDev) {
			m.nbdDevs[nbdDev] = true
		}
		if !m.nbdDevs[nbdDev] {
			m.nbdDevs[nbdDev] = true
			return nbdDev
		}
	}
	return ""
}

func (m *SNBDManager) ReleaseNbddev(nbddev string) {
	if _, ok := m.nbdDevs[nbddev]; ok {
		defer m.nbdLock.Unlock()
		m.nbdLock.Lock()
		m.nbdDevs[nbddev] = false
	}
}
