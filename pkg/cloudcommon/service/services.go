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

package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type IService interface {
	InitService()
	RunService()
	OnExitService()
}

type SServiceBase struct {
	Service IService

	O *options.BaseOptions
}

func NewBaseService(service IService) *SServiceBase {
	return &SServiceBase{Service: service}
}

func (s *SServiceBase) StartService() {
	defer s.Service.OnExitService()
	defer s.RemovePid()

	s.Service.InitService()
	if err := s.CreatePid(); err != nil {
		log.Fatalln(err)
	}

	s.Service.RunService()
}

func (s *SServiceBase) CreatePid() error {
	if s.O == nil || len(s.O.PidFile) == 0 {
		return nil
	}
	absPath, err := filepath.Abs(s.O.PidFile)
	if err != nil {
		return fmt.Errorf("Get pidfile %s absolute path failed: %s", s.O.PidFile, err)
	}
	s.O.PidFile = absPath
	pidDir := filepath.Dir(s.O.PidFile)
	if !fileutils2.Exists(pidDir) {
		output, err := procutils.NewCommand("mkdir", "-p", pidDir).Output()
		if err != nil {
			return fmt.Errorf("Make pid dir %s failed: %s", pidDir, output)
		}
	}
	err = fileutils2.FilePutContents(s.O.PidFile, strconv.Itoa(os.Getpid()), false)
	if err != nil {
		return fmt.Errorf("Write pidfile %s failed: %s", s.O.PidFile, err)
	}
	return nil
}

func (s *SServiceBase) RemovePid() {
	if s.O != nil && len(s.O.PidFile) > 0 && fileutils2.Exists(s.O.PidFile) {
		os.Remove(s.O.PidFile)
	}
}

func (s *SServiceBase) SignalTrap(onExit func()) {
	// dump goroutine stack
	signalutils.RegisterSignal(func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
	}, syscall.SIGUSR1)
	if onExit != nil {
		quitSignals := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM}
		signalutils.RegisterSignal(onExit, quitSignals...)
	}
	signalutils.StartTrap()
}
