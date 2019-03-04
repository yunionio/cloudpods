package system_service

type SLxcfs struct {
	*SBaseSystemService
}

func NewLxcfsService() *SLxcfs {
	return &SLxcfs{&SBaseSystemService{"lxcfs", nil}}
}

func (s *SLxcfs) Reload(kwargs map[string]interface{}) error {
	return nil
}

func (s *SLxcfs) BgReload(kwargs map[string]interface{}) {
}
