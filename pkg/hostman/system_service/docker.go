package system_service

import (
	"os"

	"yunion.io/x/jsonutils"
)

type SDocker struct {
	*SBaseSystemService
}

func NewDockerService() *SDocker {
	return &SDocker{&SBaseSystemService{"docker", nil}}
}

func (s *SDocker) GetConfig(kwargs map[string]interface{}) string {
	return jsonutils.Marshal(kwargs).PrettyString()
}

func (s *SDocker) GetConfigFile() string {
	os.MkdirAll("/etc/docker", 0755)
	return "/etc/docker/daemon.json"
}

func (s *SDocker) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SDocker) BgReload(kwargs map[string]interface{}) {
	go s.Reload(kwargs)
}
