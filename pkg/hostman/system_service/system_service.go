package system_service

import (
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type ISystemService interface {
	IsInstalled() bool
	Start(enable bool) error
	Stop(disable bool) error
	IsActive() bool
	GetConfig(map[string]interface{}) string
	SetConf(interface{})
	GetConf() interface{}
	BgReload(kwargs map[string]interface{})
	Enable() error
	Disable() error
	GetStatus() map[string]string
	Reload(kwargs map[string]interface{}) error
}

type NewServiceFunc func()

var serviceMap = map[string]ISystemService{
	"ntpd":          NewNtpdService(),
	"telegraf":      NewTelegrafService(),
	"host_sdnagent": NewHostSdnagentService(),
	"openvswitch":   NewOpenvswitchService(),
	"fluentbit":     NewFluentbitService(),
	"kube_agent":    NewKubeAgentService(),
	"lxcfs":         NewLxcfsService(),
	"docker":        NewDockerService(),
}

func GetService(name string) ISystemService {
	if service, ok := serviceMap[name]; ok {
		return service
	} else {
		return nil
	}
}

type SBaseSystemService struct {
	name string
	urls interface{}
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
	status := s.GetStatus()
	if loaded, ok := status["loaded"]; ok && loaded == "loaded" {
		return true
	}
	return false
}

func (s *SBaseSystemService) GetStatus() map[string]string {
	res, _ := procutils.NewCommand("systemctl", "status", s.name).Run()
	var ret = make(map[string]string, 0)
	lines := strings.Split(string(res), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			if strings.HasPrefix(line, "Loaded:") {
				ret["loaded"] = strings.Split(line, " ")[1]
			} else if strings.HasPrefix(line, "Active:") {
				ret["active"] = strings.Split(line, " ")[1]
			}
		}
	}
	return ret
}

func (s *SBaseSystemService) Start(enable bool) error {
	if enable {
		if err := s.Enable(); err != nil {
			return err
		}
	}
	_, err := procutils.NewCommand("systemctl", "restart", s.name).Run()
	return err
}

func (s *SBaseSystemService) Stop(disable bool) error {
	if disable {
		if err := s.Disable(); err != nil {
			return err
		}
	}
	_, err := procutils.NewCommand("systemctl", "stop", s.name).Run()
	return err
}

func (s *SBaseSystemService) IsActive() bool {
	status := s.GetStatus()
	if active, ok := status["active"]; ok && active == "active" {
		return true
	}
	return false
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
	_, err := procutils.NewCommand("systemctl", "enable", s.name).Run()
	return err
}

func (s *SBaseSystemService) Disable() error {
	_, err := procutils.NewCommand("systemctl", "disable", s.name).Run()
	return err
}
