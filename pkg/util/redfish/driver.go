package redfish

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

type IRedfishDriverFactory interface {
	Name() string
	NewApi(endpoint, username, password string, debug bool) IRedfishDriver
}

type IRedfishDriver interface {
	Login(ctx context.Context) error
	Logout(ctx context.Context) error

	Probe(ctx context.Context) error

	BasePath() string
	VersionKey() string
	LinkKey() string
	MemberKey() string
	LogItemsKey() string

	ParseRoot(root jsonutils.JSONObject) error

	GetParent(parent jsonutils.JSONObject) jsonutils.JSONObject

	GetResource(ctx context.Context, resname ...string) (string, jsonutils.JSONObject, error)
	GetResourceCount(ctx context.Context, resname ...string) (int, error)

	GetVirtualCdromInfo(ctx context.Context) (string, SCdromInfo, error)
	MountVirtualCdrom(ctx context.Context, path string, cdromUrl string) error
	UmountVirtualCdrom(ctx context.Context, path string) error

	GetSystemInfo(ctx context.Context) (string, SSystemInfo, error)
	SetNextBootDev(ctx context.Context, dev string) error
	SetNextBootVirtualCdrom(ctx context.Context) error

	Reset(ctx context.Context, action string) error

	ReadSystemLogs(ctx context.Context) ([]SEvent, error)
	ReadManagerLogs(ctx context.Context) ([]SEvent, error)
	ClearSystemLogs(ctx context.Context) error
	ClearManagerLogs(ctx context.Context) error

	BmcReset(ctx context.Context) error

	GetBiosInfo(ctx context.Context) (SBiosInfo, error)

	GetIndicatorLED(ctx context.Context) (bool, error)
	SetIndicatorLED(ctx context.Context, on bool) error

	GetPower(ctx context.Context) ([]SPower, error)
	GetThermal(ctx context.Context) ([]STemperature, error)

	GetNTPConf(ctx context.Context) (SNTPConf, error)
	SetNTPConf(ctx context.Context, conf SNTPConf) error

	GetConsoleJNLP(ctx context.Context) (string, error)
}

var defaultFactory IRedfishDriverFactory
var factories map[string]IRedfishDriverFactory

func init() {
	factories = make(map[string]IRedfishDriverFactory)
}

func RegisterApiFactory(f IRedfishDriverFactory) {
	factories[f.Name()] = f
}

func RegisterDefaultApiFactory(f IRedfishDriverFactory) {
	defaultFactory = f
}

func NewRedfishDriver(ctx context.Context, endpoint string, username, password string, debug bool) IRedfishDriver {
	for k, factory := range factories {
		drv := factory.NewApi(endpoint, username, password, debug)
		err := drv.Probe(ctx)
		if err == nil {
			log.Infof("Found %s Redfish REST Api Driver", k)
			return drv
		}
	}
	drv := defaultFactory.NewApi(endpoint, username, password, debug)
	err := drv.Probe(ctx)
	if err == nil {
		log.Infof("Use generic Redfish REST Api Driver")
		return drv
	}
	log.Errorf("No Redfish driver found")
	return nil
}
