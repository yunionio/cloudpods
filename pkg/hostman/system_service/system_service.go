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

package system_service

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type ISystemService interface {
	IsInstalled() bool
	IsActive() bool
	IsEnabled() bool

	Start(enable bool) error
	Stop(disable bool) error
	GetConfig(map[string]interface{}) string
	SetConf(interface{})
	GetConf() interface{}
	BgReload(kwargs map[string]interface{})
	Enable() error
	Disable() error
	Reload(kwargs map[string]interface{}) error
}

type NewServiceFunc func()

var serviceMap map[string]ISystemService

func Init() {
	serviceMap = map[string]ISystemService{
		"ntpd":          NewNtpdService(),
		"telegraf":      NewTelegrafService(),
		"host_sdnagent": NewHostSdnagentService(),
		"openvswitch":   NewOpenvswitchService(),
		"fluentbit":     NewFluentbitService(),
		"kube_agent":    NewKubeAgentService(),
		"lxcfs":         NewLxcfsService(),
		"docker":        NewDockerService(),
		"host-deployer": NewHostDeployerService(),
	}
}

func GetService(name string) ISystemService {
	if service, ok := serviceMap[name]; ok {
		return service
	} else {
		return nil
	}
}

type SBaseSystemService struct {
	manager IServiceManager
	name    string
	urls    interface{}
}

func NewBaseSystemService(name string, urls interface{}) *SBaseSystemService {
	ss := SBaseSystemService{}
	if SystemdServiceManager.Detect() {
		ss.manager = SystemdServiceManager
	} else {
		ss.manager = SysVServiceManager
	}
	ss.name = name
	ss.urls = urls
	return &ss
}

func (s *SBaseSystemService) reload(conf, conFile string) error {
	oldConf, err := fileutils2.FileGetContents(conFile)
	if err != nil {
		return err
	}
	if conf != oldConf {
		log.Infof("Reload service %s ...", s.name)
		err := fileutils2.FilePutContents(conFile, conf, false)
		if err != nil {
			return err
		}
		return s.Start(false)
	}
	return nil
}

func (s *SBaseSystemService) IsInstalled() bool {
	status := s.manager.GetStatus(s.name)
	return status.Loaded
}

func (s *SBaseSystemService) Start(enable bool) error {
	if enable {
		if err := s.Enable(); err != nil {
			return err
		}
	}
	return s.manager.Start(s.name)
}

func (s *SBaseSystemService) Stop(disable bool) error {
	if disable {
		if err := s.Disable(); err != nil {
			return err
		}
	}
	return s.manager.Stop(s.name)
}

func (s *SBaseSystemService) IsActive() bool {
	status := s.manager.GetStatus(s.name)
	return status.Active
}

func (s *SBaseSystemService) IsEnabled() bool {
	status := s.manager.GetStatus(s.name)
	return status.Enabled
}

func (s *SBaseSystemService) GetConfig(map[string]interface{}) string {
	return ""
}

func (s *SBaseSystemService) GetConfigFile() string {
	return ""
}

func (s *SBaseSystemService) SetConf(urls interface{}) {
	s.urls = urls
}

func (s *SBaseSystemService) GetConf() interface{} {
	return s.urls
}

func (s *SBaseSystemService) Enable() error {
	return s.manager.Enable(s.name)
}

func (s *SBaseSystemService) Disable() error {
	return s.manager.Disable(s.name)
}
