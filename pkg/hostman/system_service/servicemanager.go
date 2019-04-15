package system_service

type SServiceStatus struct {
	Loaded bool
	Active bool
}

type IServiceManager interface {
	Detect() bool
	Start(srvname string) error
	Enable(srvname string) error
	Stop(srvname string) error
	Disable(srvname string) error
	GetStatus(srvname string) SServiceStatus
}
