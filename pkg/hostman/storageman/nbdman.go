package storageman

import (
	"fmt"
	"os"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/hostman"
)

type SNBDManager struct {
	// find local /dev/nbdx device, map key is devs, value is it in use
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
		_, err := os.Stat(fmt.Sprintf("/dev/nbd%d", i))
		if !os.IsNotExist(err) {
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
		if hostman.IsBlockDeviceUsed(nbdDev) {
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
