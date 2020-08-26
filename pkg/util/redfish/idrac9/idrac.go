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

package idrac9

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/redfish/bmconsole"
	"yunion.io/x/onecloud/pkg/util/redfish/generic"
	"yunion.io/x/onecloud/pkg/util/redfish/idrac"
)

type SIDrac9RedfishApiFactory struct {
}

func (f *SIDrac9RedfishApiFactory) Name() string {
	return "iDRAC9"
}

func (f *SIDrac9RedfishApiFactory) NewApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	return NewIDrac9RedfishApi(endpoint, username, password, debug)
}

func init() {
	redfish.RegisterApiFactory(&SIDrac9RedfishApiFactory{})
}

type SIDrac9RefishApi struct {
	idrac.SIDracRefishApi
}

func NewIDrac9RedfishApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	api := &SIDrac9RefishApi{
		SIDracRefishApi: idrac.SIDracRefishApi{
			SGenericRefishApi: generic.SGenericRefishApi{
				SBaseRedfishClient: redfish.NewBaseRedfishClient(endpoint, username, password, debug),
			},
		},
	}
	api.SetVirtualObject(api)
	return api
}

func (r *SIDrac9RefishApi) ParseRoot(root jsonutils.JSONObject) error {
	// log.Debugf("ParseRoot %s", root.PrettyString())
	oem, _ := root.Get("Oem", "Dell")
	if oem != nil {
		return nil
	}
	return errors.Error("not iDrac")
}

func (r *SIDrac9RefishApi) GetNTPConf(ctx context.Context) (redfish.SNTPConf, error) {
	return r.SGenericRefishApi.GetNTPConf(ctx)
}

func (r *SIDrac9RefishApi) SetNTPConf(ctx context.Context, conf redfish.SNTPConf) error {
	return r.SGenericRefishApi.SetNTPConf(ctx, conf)
}

func (r *SIDrac9RefishApi) GetConsoleJNLP(ctx context.Context) (string, error) {
	bmc := bmconsole.NewBMCConsole(r.GetHost(), r.GetUsername(), r.GetPassword(), r.IsDebug)
	return bmc.GetIdrac9ConsoleJNLP(ctx)
}

func (r *SIDrac9RefishApi) GetSystemLogsPath() string {
	return "/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/Sel/Entries"
}

func (r *SIDrac9RefishApi) GetManagerLogsPath() string {
	return "/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/Lclog/Entries"
}

func (r *SIDrac9RefishApi) GetClearSystemLogsPath() string {
	return "/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/Sel/Actions/LogService.ClearLog"
}

func (r *SIDrac9RefishApi) GetClearManagerLogsPath() string {
	return "/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/Lclog/Actions/LogService.ClearLog"
}

func (r *SIDrac9RefishApi) GetPowerPath() string {
	return "/redfish/v1/Chassis/System.Embedded.1/Power"
}

func (r *SIDrac9RefishApi) GetThermalPath() string {
	return "/redfish/v1/Chassis/System.Embedded.1/Thermal"
}
