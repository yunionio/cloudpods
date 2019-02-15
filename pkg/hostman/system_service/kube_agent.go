package system_service

import "fmt"

type SKubeAgent struct {
	*SBaseSystemService
}

func NewKubeAgentService() *SKubeAgent {
	return &SKubeAgent{&SBaseSystemService{"yunion-kube-agent", nil}}
}

func (s *SKubeAgent) GetConfig(kwargs map[string]interface{}) string {
	conf := ""
	conf += fmt.Sprintf("server = \"%s\"\n", kwargs["serverUrl"])
	conf += fmt.Sprintf("token = \"%s\"\n", kwargs["token"])
	conf += fmt.Sprintf("id = \"%s\"\n", kwargs["nodeId"])
	return conf
}

func (s *SKubeAgent) GetConfigFile() string {
	return "/etc/yunion/kube-agent.conf"
}

func (s *SKubeAgent) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SKubeAgent) BgReload(kwargs map[string]interface{}) {
	go s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}
