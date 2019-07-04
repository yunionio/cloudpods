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

package signalutils

import (
	"os"
	"os/signal"
	"runtime/debug"
	"sync"

	"yunion.io/x/log"
)

type SSignalManager struct {
	sig     chan os.Signal
	traps   map[os.Signal]Trap
	mtx     *sync.RWMutex
	started bool
}

var signalManager *SSignalManager

type Trap func()

func RegisterSignal(trap Trap, sigs ...os.Signal) {
	signalManager.mtx.Lock()
	defer signalManager.mtx.Unlock()
	if !signalManager.started {
		for i := 0; i < len(sigs); i++ {
			signalManager.traps[sigs[i]] = trap
		}
	} else {
		for i := 0; i < len(sigs); i++ {
			signalManager.traps[sigs[i]] = trap
			signal.Notify(signalManager.sig, sigs[i])
		}
	}
}

func StartTrap() {
	if signalManager.started {
		return
	} else {
		signalManager.started = true
	}
	for sig, _ := range signalManager.traps {
		signal.Notify(signalManager.sig, sig)
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("Call trap func error: %#v", err)
				debug.PrintStack()
			}
		}()
		signalManager.mtx.RLock()
		defer signalManager.mtx.RUnlock()
		for {
			s := <-signalManager.sig
			trapFunc := signalManager.traps[s]
			trapFunc()
		}
	}()
}

func init() {
	signalManager = &SSignalManager{
		sig:   make(chan os.Signal, 1),
		traps: make(map[os.Signal]Trap, 0),
		mtx:   &sync.RWMutex{},
	}
}
