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

package libguestfs

import (
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs/guestfish"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type ErrorFish string

func (e ErrorFish) Error() string {
	return string(e)
}

const (
	ErrFishsMismatch = ErrorFish("fishs mismatch")
	ErrFishsWorking  = ErrorFish("fishs working")
	ErrFishsDied     = ErrorFish("fishs died")
)

var guestfsManager *GuestfsManager

func Init(count int) error {
	if guestfsManager == nil {
		if err := procutils.NewRemoteCommandAsFarAsPossible("modprobe", "dm-mod").Run(); err != nil {
			return errors.Wrap(err, "modprobe dm-mod")
		}
		log.Infof("guestfish count %v", count)
		guestfsManager = NewGuestfsManager(count)
		time.AfterFunc(time.Minute*3, guestfsManager.fishsRecycle)
	}
	return nil
}

type GuestfsManager struct {
	fishMaximum      int
	happyFishCount   int
	workingFishCount int

	fishs    map[*guestfish.Guestfish]bool
	fishChan chan *guestfish.Guestfish
	fishlock sync.Mutex

	lastTimeFishing time.Time
}

func NewGuestfsManager(count int) *GuestfsManager {
	if count < 1 {
		count = 1
	}
	return &GuestfsManager{
		fishMaximum: count,
		fishs:       make(map[*guestfish.Guestfish]bool, count),
		fishChan:    make(chan *guestfish.Guestfish, count),
	}
}

func (m *GuestfsManager) AcquireFish() (*guestfish.Guestfish, error) {
	fisher := func() (*guestfish.Guestfish, error) {
		m.fishlock.Lock()
		defer m.fishlock.Unlock()

		m.lastTimeFishing = time.Now()
		fish, err := m.acquireFish()
		if err == ErrFishsWorking {
			return nil, err
		} else if err != nil {
			return nil, err
		}
		if !fish.IsAlive() {
			go func() {
				if e := m.ReleaseFish(fish); e != nil {
					log.Errorf("release fish failed %s", e)
				}
			}()
			return nil, nil
		}
		return fish, nil
	}
	for i := 0; i < 3; i++ {
		fish, err := fisher()
		if err == ErrFishsWorking {
			return m.waitingFishFinish(), nil
		}
		if err != nil {
			return nil, err
		}
		if fish != nil {
			return fish, nil
		}
	}
	return nil, ErrFishsDied
}

func (m *GuestfsManager) acquireFish() (*guestfish.Guestfish, error) {
	if m.happyFishCount == 0 {
		if m.workingFishCount < m.fishMaximum {
			fish, err := guestfish.NewGuestfish()
			if err != nil {
				return nil, err
			}
			m.fishs[fish] = true
			m.workingFishCount++
			return fish, nil
		} else {
			return nil, ErrFishsWorking
		}
	} else {
		for fish, working := range m.fishs {
			if !working {
				m.fishs[fish] = true
				m.workingFishCount++
				m.happyFishCount--
				return fish, nil
			}
		}
		return nil, ErrFishsMismatch
	}
}

func (m *GuestfsManager) waitingFishFinish() *guestfish.Guestfish {
	select {
	case fish := <-m.fishChan:
		return fish
	}
}

func (m *GuestfsManager) ReleaseFish(fish *guestfish.Guestfish) error {
	err := m.washfish(fish)
	m.fishlock.Lock()
	defer m.fishlock.Unlock()
	if err != nil {
		errQ := fish.Quit()
		if errQ != nil {
			log.Errorf("fish quit failed: %s", errQ)
		}
		delete(m.fishs, fish)
		m.workingFishCount--
	}
	m.fishChan <- fish
	return err
}

func (m *GuestfsManager) washfish(fish *guestfish.Guestfish) error {
	return fish.RemoveDrive()
}

func (m *GuestfsManager) fishsRecycle() {
	defer time.AfterFunc(time.Minute*10, m.fishsRecycle)
	m.fishlock.Lock()
	defer m.fishlock.Unlock()
	if m.lastTimeFishing.IsZero() || time.Now().Sub(m.lastTimeFishing) < time.Minute*10 {
		return
	}
	log.Infof("start release fishs ...")

Loop:
	for {
		select {
		case fish := <-m.fishChan:
			m.fishs[fish] = false
			m.happyFishCount++
			m.workingFishCount--
		default:
			break Loop
		}
	}

	for fish, working := range m.fishs {
		if !working {
			err := fish.Quit()
			if err != nil {
				log.Errorf("fish quit failed: %s", err)
			}
			m.happyFishCount--
			delete(m.fishs, fish)
		}
	}
}
