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

//go:build !windows
// +build !windows

package procutils

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"yunion.io/x/log"
)

func WaitZombieLoop(ctx context.Context) {
	myPid := os.Getpid()
	if myPid != 1 {
		log.Infof("My pid is not 1 and no need to wait zombies")
		return
	}
	const myPidStr = "1"

	tick := time.NewTicker(31 * time.Second)
	for {
		dirs, err := ioutil.ReadDir("/proc")
		if err != nil {
			log.Errorf("read /proc dir: %v", err)
		}
		for _, dir := range dirs {
			// /proc/<pid>/
			if !dir.IsDir() {
				continue
			}
			name := dir.Name()
			allDigits := true
			for _, c := range name {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
			if !allDigits {
				continue
			}

			// read /proc/<pid>/stat
			statPath := filepath.Join("/proc", name, "stat")
			data, err := ioutil.ReadFile(statPath)
			if err != nil {
				log.Errorf("read %s: %v", statPath, err)
				continue
			}
			dataStr := string(data)
			items := strings.Split(dataStr, " ")
			const (
				idxPid = iota
				idxName
				idxState
				idxPpid
				idxMinLen
			)
			if len(items) < idxMinLen {
				log.Errorf("%s contains less than %d items: %s", statPath, idxMinLen, dataStr)
			}

			// my zombie subprocesses
			state := items[idxState]
			if state != "Z" {
				continue
			}
			ppidStr := items[idxPpid]
			if ppidStr != myPidStr {
				continue
			}

			// wait it
			pname := items[idxName]
			pidStr := items[idxPid]
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				log.Errorf("%s: %s has invalid pid number %q: %v", pname, statPath, pidStr, err)
				continue
			}
			pid1, err := syscall.Wait4(pid, nil, 0, nil)
			if err != nil {
				log.Errorf("%s: %s: wait error: %v", pname, statPath, err)
				continue
			}
			if pid1 == pid {
				log.Infof("%s: pid %d: wait done", pname, pid)
			}
		}
		select {
		case <-ctx.Done():
		case <-tick.C:
		}
	}
}
