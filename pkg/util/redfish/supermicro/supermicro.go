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

package supermicro

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/redfish/bmconsole"
	"yunion.io/x/onecloud/pkg/util/redfish/generic"
)

type SSupermicroRedfishApiFactory struct {
}

func (f *SSupermicroRedfishApiFactory) Name() string {
	return "Supermicro"
}

func (f *SSupermicroRedfishApiFactory) NewApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	return NewSupermicroRedfishApi(endpoint, username, password, debug)
}

func init() {
	redfish.RegisterApiFactory(&SSupermicroRedfishApiFactory{})
}

type SSupermicroRefishApi struct {
	generic.SGenericRefishApi
}

func NewSupermicroRedfishApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	api := &SSupermicroRefishApi{
		SGenericRefishApi: generic.SGenericRefishApi{
			SBaseRedfishClient: redfish.NewBaseRedfishClient(endpoint, username, password, debug),
		},
	}
	api.SetVirtualObject(api)
	return api
}

func (r *SSupermicroRefishApi) ParseRoot(root jsonutils.JSONObject) error {
	if root.Contains("UUID") && root.Contains("UpdateService") && root.Contains("Oem") {
		return nil
	}
	return errors.Error("not iDrac")
}

func (r *SSupermicroRefishApi) GetVirtualCdromJSON(ctx context.Context) (string, jsonutils.JSONObject, error) {
	_, resp, err := r.GetResource(ctx, "Managers", "0", "VirtualMedia")
	if err != nil {
		return "", nil, errors.Wrap(err, "r.GetResource")
	}
	log.Debugf("resp %s", resp)
	path, err := resp.GetString("Oem", "Supermicro", "VirtualMediaConfig", "@odata.id")
	if err != nil {
		return "", nil, errors.Wrap(err, "GetString Oem Supermicro VirtualMediaConfig @odata.id")
	}
	cdResp, err := r.Get(ctx, path)
	if err != nil {
		return "", nil, errors.Wrap(err, "Get Cdrom info fail")
	}
	if cdResp != nil {
		return path, cdResp, nil
	}
	return "", nil, httperrors.ErrNotFound
}

func (r *SSupermicroRefishApi) GetVirtualCdromInfo(ctx context.Context) (string, redfish.SCdromInfo, error) {
	cdInfo := redfish.SCdromInfo{}
	path, jsonResp, err := r.GetVirtualCdromJSON(ctx)
	if err != nil {
		return "", cdInfo, errors.Wrap(err, "r.GetVirtualCdromJSON")
	}
	imgPath, _ := jsonResp.GetString("Image")
	if imgPath == "null" {
		imgPath = ""
	}
	cdInfo.Image = imgPath
	if jsonResp.Contains("Actions") {
		cdInfo.SupportAction = true
	}
	return path, cdInfo, nil
}

func (r *SSupermicroRefishApi) MountVirtualCdrom(ctx context.Context, path string, cdromUrl string, boot bool) error {
	info := jsonutils.NewDict()
	info.Set("Image", jsonutils.NewString(cdromUrl))

	path = httputils.JoinPath(path, "Actions/VirtualMedia.InsertMedia")
	_, _, err := r.Post(ctx, path, info)
	if err != nil {
		return errors.Wrap(err, "r.Post")
	}
	// log.Debugf("%s", resp.PrettyString())
	if boot {
		err = r.SetNextBootVirtualCdrom(ctx)
		if err != nil {
			return errors.Wrap(err, "r.SetNextBootVirtualCdrom")
		}
	}
	return nil
}

func (r *SSupermicroRefishApi) UmountVirtualCdrom(ctx context.Context, path string) error {
	info := jsonutils.NewDict()

	path = httputils.JoinPath(path, "Actions/VirtualMedia.EjectMedia")
	_, _, err := r.Post(ctx, path, info)
	if err != nil {
		return errors.Wrap(err, "r.Post")
	}
	// log.Debugf("%s", resp.PrettyString())
	return nil
}

func (r *SSupermicroRefishApi) GetConsoleJNLP(ctx context.Context) (string, error) {
	bmc := bmconsole.NewBMCConsole(r.GetHost(), r.GetUsername(), r.GetPassword(), r.IsDebug)
	return bmc.GetSupermicroConsoleJNLP(ctx)
}

func (r *SSupermicroRefishApi) ReadSystemLogs(ctx context.Context, since time.Time) ([]redfish.SEvent, error) {
	_, resp, err := r.GetResource(ctx, "Managers", "0", "LogServices", "0", "Entries")
	if err != nil {
		return nil, errors.Wrap(err, "GetResource Managers 0 LogServices 0 Entries")
	}
	members, err := resp.GetArray("Members")
	if err != nil {
		return nil, errors.Wrap(err, "find Members")
	}
	events := make([]redfish.SEvent, 0)
	for i := range members {
		path, _ := members[i].GetString("@odata.id")
		resp, err := r.Get(ctx, path)
		if err != nil {
			log.Errorf("Get url for event %s error %s", path, err)
			continue
		}
		event := redfish.SEvent{}
		err = resp.Unmarshal(&event)
		if err == nil {
			event.Type, _ = resp.GetString("EntryType")
			event.Severity, _ = resp.GetString("EntryCode")
			events = append(events, event)
		} else {
			log.Errorf("unmarshal event fail %s %s", resp, err)
		}
	}
	return events, nil
}

func (r *SSupermicroRefishApi) ReadManagerLogs(ctx context.Context, since time.Time) ([]redfish.SEvent, error) {
	return nil, httperrors.ErrNotSupported
}

func (r *SSupermicroRefishApi) GetClearSystemLogsPath() string {
	return "/redfish/v1/Managers/1/LogServices/Log1/Actions/LogService.Reset"
}

func (r *SSupermicroRefishApi) GetClearManagerLogsPath() string {
	return "/redfish/v1/Managers/1/LogServices/Log1/Actions/LogService.Reset"
}

func (r *SSupermicroRefishApi) GetPowerPath() string {
	return "/redfish/v1/Chassis/1/Power"
}

func (r *SSupermicroRefishApi) GetThermalPath() string {
	return "/redfish/v1/Chassis/1/Thermal"
}

type sNtpConf struct {
	NTPEnable          bool   `json:"NTPEnable"`
	PrimaryNTPServer   string `json:"PrimaryNTPServer"`
	SecondaryNTPServer string `json:"SecondaryNTPServer"`
}

func (r *SSupermicroRefishApi) GetNTPConf(ctx context.Context) (redfish.SNTPConf, error) {
	ntpConf := redfish.SNTPConf{}
	_, resp, err := r.GetResource(ctx, "Managers", "0")
	if err != nil {
		return ntpConf, errors.Wrap(err, "GetResource")
	}
	path, err := resp.GetString("Oem", "Supermicro", "NTP", "@odata.id")
	if err != nil {
		return ntpConf, errors.Wrap(err, "GetString \"Oem\", \"Supermicro\", \"NTP\", \"@odata.id\"")
	}
	resp, err = r.Get(ctx, path)
	if err != nil {
		return ntpConf, errors.Wrapf(err, "Get %s fail %s", path, err)
	}
	conf := sNtpConf{}
	err = resp.Unmarshal(&conf)
	if err != nil {
		return ntpConf, errors.Wrap(err, "resp.Unmarshal NTP")
	}
	ntpConf.ProtocolEnabled = conf.NTPEnable
	if len(conf.PrimaryNTPServer) > 0 {
		ntpConf.NTPServers = append(ntpConf.NTPServers, conf.PrimaryNTPServer)
	}
	if len(conf.SecondaryNTPServer) > 0 {
		ntpConf.NTPServers = append(ntpConf.NTPServers, conf.SecondaryNTPServer)
	}
	return ntpConf, nil
}

func (r *SSupermicroRefishApi) SetNTPConf(ctx context.Context, conf redfish.SNTPConf) error {
	_, resp, err := r.GetResource(ctx, "Managers", "0")
	if err != nil {
		return errors.Wrap(err, "GetResource")
	}
	path, err := resp.GetString("Oem", "Supermicro", "NTP", "@odata.id")
	if err != nil {
		return errors.Wrap(err, "GetString \"Oem\", \"Supermicro\", \"NTP\", \"@odata.id\"")
	}
	nconf := sNtpConf{
		NTPEnable: conf.ProtocolEnabled,
	}
	if len(conf.NTPServers) > 0 {
		nconf.PrimaryNTPServer = conf.NTPServers[0]
	}
	if len(conf.NTPServers) > 1 {
		nconf.SecondaryNTPServer = conf.NTPServers[1]
	}
	resp, err = r.Patch(ctx, path, jsonutils.Marshal(nconf))
	if err != nil {
		return errors.Wrap(err, "r.Patch")
	}
	if r.IsDebug {
		log.Debugf("%s", resp.PrettyString())
	}
	return nil
}
