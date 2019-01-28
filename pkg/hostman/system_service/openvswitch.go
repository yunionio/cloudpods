package system_service

type SOpenvswitch struct {
	*SBaseSystemService
}

func NewOpenvswitchService() *SOpenvswitch {
	return &SOpenvswitch{&SBaseSystemService{"openvswitch", nil}}
}

func (s *SOpenvswitch) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SOpenvswitch) BgReload(kwargs map[string]interface{}) {
	go s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}
