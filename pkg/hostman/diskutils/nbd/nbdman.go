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

func Init() error {
	var err error
	nbdManager, err = NewNBDManager()
	return err
}

func GetNBDManager() *SNBDManager {
	return nbdManager
}

func NewNBDManager() (*SNBDManager, error) {
	var ret = new(SNBDManager)
	ret.nbdDevs = make(map[string]bool, 0)
	ret.nbdLock = new(sync.Mutex)
	if err := ret.findNbdDevices(); err != nil {
		return ret, err
	}
	return ret, nil
}

func (m *SNBDManager) findNbdDevices() error {
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
	if len(m.nbdDevs) == 0 {
		return fmt.Errorf("No nbd devices found")
	}
	return nil
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
