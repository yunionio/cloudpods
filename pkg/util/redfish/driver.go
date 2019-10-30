// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redfish

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
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
	MountVirtualCdrom(ctx context.Context, path string, cdromUrl string, boot bool) error
	UmountVirtualCdrom(ctx context.Context, path string) error

	GetLanConfigs(ctx context.Context) ([]types.SIPMILanConfig, error)

	GetSystemInfo(ctx context.Context) (string, SSystemInfo, error)
	SetNextBootDev(ctx context.Context, dev string) error
	// SetNextBootVirtualCdrom(ctx context.Context) error

	Reset(ctx context.Context, action string) error

	GetSystemLogsPath() string
	GetManagerLogsPath() string
	GetClearSystemLogsPath() string
	GetClearManagerLogsPath() string
	ReadSystemLogs(ctx context.Context, since time.Time) ([]SEvent, error)
	ReadManagerLogs(ctx context.Context, since time.Time) ([]SEvent, error)
	ClearSystemLogs(ctx context.Context) error
	ClearManagerLogs(ctx context.Context) error

	BmcReset(ctx context.Context) error

	GetBiosInfo(ctx context.Context) (SBiosInfo, error)

	GetIndicatorLED(ctx context.Context) (bool, error)
	SetIndicatorLED(ctx context.Context, on bool) error

	GetPowerPath() string
	GetThermalPath() string
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
	if defaultFactory == nil {
		return nil
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
