package system_service

type SHostDeployer struct {
	*SBaseSystemService
}

func NewHostDeployerService() *SHostDeployer {
	return &SHostDeployer{NewBaseSystemService("yunion-host-deployer", nil)}
}

func (s *SHostDeployer) Reload(kwargs map[string]interface{}) error {
	return s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}

func (s *SHostDeployer) BgReload(kwargs map[string]interface{}) {
	go s.reload(s.GetConfig(kwargs), s.GetConfigFile())
}
