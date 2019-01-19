package system_service

type SHostSdnagent struct {
	*SBaseSystemService
}

func NewHostSdnagentService() *SHostSdnagent {
	return &SHostSdnagent{&SBaseSystemService{"yunion-host-sdnagent", nil}}
}

func (s *SHostSdnagent) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SHostSdnagent) BgReload(kwargs map[string]interface{}) {
	go s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}
