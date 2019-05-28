package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

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

	O *options.CommonOptions
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
		output, err := procutils.NewCommand("mkdir", "-p", pidDir).Run()
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
