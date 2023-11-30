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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
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

	if err := ret.reloadNbdDevices(); err != nil {
		return ret, errors.Wrap(err, "reloadNbdDevices")
	}
	if err := ret.findNbdDevices(); err != nil {
		return ret, errors.Wrap(err, "findNbdDevices")
	}
	go ret.cleanupNbdDevices()
	return ret, nil
}

func tryDetachNbd(nbddev string) error {
	const MaxTries = 3
	tried := 0
	var errs []error
	for tried < MaxTries && fileutils2.IsBlockDeviceUsed(nbddev) {
		tried++
		if err := putdownNbdDevice(nbddev); err != nil {
			errs = append(errs, err)
			time.Sleep(time.Second)
			continue
		}
		break

	}
	if tried < MaxTries {
		return nil
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}

func putdownNbdDevice(nbddev string) error {
	nbdDriver := &NBDDriver{nbdDev: nbddev}

	if err := nbdDriver.findPartitions(); err != nil {
		return err
	}

	if _, err := nbdDriver.setupLVMS(); err != nil {
		return err
	}
	for i := range nbdDriver.partitions {
		if nbdDriver.partitions[i].IsMounted() {
			if err := nbdDriver.partitions[i].Umount(); err != nil {
				return errors.Wrapf(err, "umount %s", nbdDriver.partitions[i].GetPartDev())
			}
		}
	}

	if !nbdDriver.putdownLVMs() {
		return errors.Errorf("failed putdown lvms")
	}
	if err := QemuNbdDisconnect(nbddev); err != nil {
		return err
	}
	return nil

}

func (m *SNBDManager) cleanupNbdDevices() {
	var i = 0
	for {
		nbddev := fmt.Sprintf("/dev/nbd%d", i)
		if fileutils2.Exists(nbddev) {
			if fileutils2.IsBlockDeviceUsed(nbddev) {
				log.Infof("nbd device %s is used", nbddev)
				err := tryDetachNbd(nbddev)
				if err != nil {
					log.Errorf("tryDetachNbd fail %s", err)
				} else {
					m.ReleaseNbddev(nbddev)
				}
			}
			i++
		} else {
			break
		}
	}
}

func (m *SNBDManager) reloadNbdDevices() error {
	output, err := procutils.NewRemoteCommandAsFarAsPossible("rmmod", "nbd").Output()
	if err != nil {
		log.Errorf("rmmod error: %s", output)
	}
	output, err = procutils.NewRemoteCommandAsFarAsPossible("modprobe", "nbd", "max_part=16").Output()
	if err != nil {
		return errors.Wrapf(err, "Failed to activate nbd device: %s", output)
	}
	return nil
}

func (m *SNBDManager) findNbdDevices() error {
	var i = 0
	for {
		nbddev := fmt.Sprintf("/dev/nbd%d", i)
		if fileutils2.Exists(nbddev) {
			i++
			if fileutils2.IsBlockDeviceUsed(nbddev) {
				continue
			}
			m.nbdDevs[nbddev] = false
			// https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-class-bdi
			nbdBdi := fmt.Sprintf("/sys/block/nbd%d/bdi/", i-1)
			sysutils.SetSysConfig(nbdBdi+"max_ratio", "0")
			sysutils.SetSysConfig(nbdBdi+"min_ratio", "0")
		} else {
			break
		}
	}
	if len(m.nbdDevs) == 0 {
		return errors.Wrap(errors.ErrNotFound, "No nbd devices found")
	}
	log.Infof("NBD_DEVS: %#v", m.nbdDevs)
	return nil
}

func (m *SNBDManager) AcquireNbddev() string {
	defer m.nbdLock.Unlock()
	m.nbdLock.Lock()
	for nbdDev := range m.nbdDevs {
		if fileutils2.IsBlockDeviceUsed(nbdDev) {
			m.nbdDevs[nbdDev] = true
			continue
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
		if !fileutils2.IsBlockDeviceUsed(nbddev) {
			m.nbdDevs[nbddev] = false
		}
	}
}
