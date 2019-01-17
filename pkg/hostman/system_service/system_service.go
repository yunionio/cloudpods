package system_service

type ISystemService interface {
	IsInstalled() bool
	Start() error
	IsActive() bool
	GetConf(map[string]interface{}) []string
	SetConf([]string)
	BgReload(servers []string)
	Enable() error
	Disable() error
	GetStatus() string
	Reload()
}

func GetService(name string) ISystemService {
	return nil
}
