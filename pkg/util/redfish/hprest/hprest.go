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

package hprest

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/redfish"
)

const (
	basePath   = "/rest/v1"
	linkKey    = "href"
	memberKey  = "Member"
	versionKey = "ServiceVersion"
)

type SHpRestApiFactory struct {
}

func (f *SHpRestApiFactory) Name() string {
	return "HpRest"
}

func (f *SHpRestApiFactory) NewApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	return NewHpRestApi(endpoint, username, password, debug)
}

func init() {
	redfish.RegisterApiFactory(&SHpRestApiFactory{})
}

type SHpRestApi struct {
	redfish.SBaseRedfishClient
}

func NewHpRestApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	api := &SHpRestApi{
		SBaseRedfishClient: redfish.NewBaseRedfishClient(endpoint, username, password, debug),
	}
	api.SetVirtualObject(api)
	return api
}

func (r *SHpRestApi) BasePath() string {
	return basePath
}

func (r *SHpRestApi) GetParent(parent jsonutils.JSONObject) jsonutils.JSONObject {
	obj, _ := parent.Get("links")
	if obj != nil {
		return obj
	}
	return jsonutils.NewDict()
}

func (r *SHpRestApi) VersionKey() string {
	return versionKey
}

func (r *SHpRestApi) LinkKey() string {
	return linkKey
}

func (r *SHpRestApi) MemberKey() string {
	return memberKey
}

func (r *SHpRestApi) LogItemsKey() string {
	return "Items"
}

func (r *SHpRestApi) Probe(ctx context.Context) error {
	err := r.SBaseRedfishClient.Probe(ctx)
	if err != nil {
		return errors.Wrap(err, "r.SBaseRedfishClient.Probe")
	}
	resp, err := r.Get(ctx, "/rest/v1/resourcedirectory")
	if err != nil {
		return errors.Wrap(err, "Get /rest/v1/resourcedirectory")
	}
	if r.IsDebug {
		log.Debugf("%s", resp.PrettyString())
	}
	return nil
}

func (r *SHpRestApi) GetVirtualCdromInfo(ctx context.Context) (string, redfish.SCdromInfo, error) {
	path, cdInfo, err := r.SBaseRedfishClient.GetVirtualCdromInfo(ctx)
	if err == nil {
		cdInfo.SupportAction = true
	}
	return path, cdInfo, err
}

func (r *SHpRestApi) GetSystemInfo(ctx context.Context) (string, redfish.SSystemInfo, error) {
	sysInfo := redfish.SSystemInfo{}

	path, resp, err := r.GetResource(ctx, "Systems", "0")
	if err != nil {
		return path, sysInfo, errors.Wrap(err, "r.GetResource Systems")
	}
	err = resp.Unmarshal(&sysInfo)
	if err != nil {
		return path, sysInfo, errors.Wrap(err, "resp.Unmarshal")
	}

	memGB, _ := resp.Int("Memory", "TotalSystemMemoryGB")
	sysInfo.MemoryGB = int(memGB)
	cpuCnt, _ := resp.Int("Processors", "Count")
	cpuDesc, _ := resp.GetString("Processors", "ProcessorFamily")
	sysInfo.NodeCount = int(cpuCnt)
	sysInfo.CpuDesc = strings.TrimSpace(cpuDesc)
	power, _ := resp.GetString("Power")
	sysInfo.PowerState = strings.TrimSpace(power)

	sysInfo.SerialNumber = strings.TrimSpace(sysInfo.SerialNumber)
	sysInfo.SKU = strings.TrimSpace(sysInfo.SKU)
	sysInfo.Model = strings.TrimSpace(sysInfo.Model)
	sysInfo.Manufacturer = strings.TrimSpace(sysInfo.Manufacturer)

	nextBootDev, _ := resp.GetString("Boot", "BootSourceOverrideTarget")
	sysInfo.NextBootDev = strings.TrimSpace(nextBootDev)
	nextBootDevSupports, _ := resp.GetArray("Boot", "BootSourceOverrideSupported")
	sysInfo.NextBootDevSupported = make([]string, len(nextBootDevSupports))
	for i := range nextBootDevSupports {
		devStr, _ := nextBootDevSupports[i].GetString()
		sysInfo.NextBootDevSupported[i] = devStr
	}

	AvailableActions, _ := resp.GetArray("AvailableActions")
	if AvailableActions != nil {
		resetCapabilities, _ := AvailableActions[0].GetArray("Capabilities")
		if resetCapabilities != nil {
			resetTypeValues, _ := resetCapabilities[0].GetArray("AllowableValues")
			sysInfo.ResetTypeSupported = make([]string, len(resetTypeValues))
			for i := range resetTypeValues {
				resetTypeStr, _ := resetTypeValues[i].GetString()
				sysInfo.ResetTypeSupported[i] = resetTypeStr
			}
		}
	}

	macArray, _ := resp.GetArray("HostCorrelation", "HostMACAddress")
	if len(macArray) > 0 {
		sysInfo.EthernetNICs = make([]string, len(macArray))
		for i := range macArray {
			macAddr, _ := macArray[i].GetString()
			sysInfo.EthernetNICs[i] = netutils.FormatMacAddr(macAddr)
		}
	}

	return path, sysInfo, nil
}

func (r *SHpRestApi) Reset(ctx context.Context, action string) error {
	path, sysInfo, err := r.IRedfishDriver().GetSystemInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "r.GetSystemInfo")
	}
	if !utils.IsInStringArray(action, sysInfo.ResetTypeSupported) {
		return errors.Wrapf(httperrors.ErrBadRequest, "%s not supported: %s", action, sysInfo.ResetTypeSupported)
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(action), "ResetType")
	params.Add(jsonutils.NewString("Reset"), "Action")
	_, resp, err := r.Post(ctx, path, params)
	if err != nil {
		return errors.Wrap(err, "Action.Reset")
	}
	if r.IsDebug {
		log.Debugf("%s", resp)
	}
	return nil
}

func (r *SHpRestApi) BmcReset(ctx context.Context) error {
	path, _, err := r.GetResource(ctx, "Managers", "0")
	if err != nil {
		return errors.Wrap(err, "r.GetSystemInfo")
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("Reset"), "Action")
	_, resp, err := r.Post(ctx, path, params)
	if err != nil {
		return errors.Wrap(err, "Actions/Manager.Reset")
	}
	log.Debugf("%s", resp)
	return nil
}

func (r *SHpRestApi) readLogs(ctx context.Context, subsys string) ([]redfish.SEvent, error) {
	_, manager, err := r.GetResource(ctx, subsys, "0", "Logs", "0")
	if err != nil {
		return nil, errors.Wrap(err, "GetResource Managers 0")
	}
	manager = r.IRedfishDriver().GetParent(manager)
	entries, _ := manager.GetArray("Entries")
	if len(entries) == 0 {
		return nil, errors.Error("no log entries")
	}
	path, _ := entries[0].GetString(r.IRedfishDriver().LinkKey())
	if len(path) == 0 {
		return nil, errors.Wrap(err, "no valid url???")
	}
	nextPath := path
	events := make([]redfish.SEvent, 0)
	for {
		resp, err := r.Get(ctx, nextPath)
		if err != nil {
			return nil, errors.Wrap(err, nextPath)
		}
		tmpEvents := make([]redfish.SEvent, 0)
		err = resp.Unmarshal(&tmpEvents, "Items")
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		events = append(events, tmpEvents...)
		nextPage, _ := resp.Int("links", "NextPage", "page")
		if nextPage > 0 {
			nextPath = fmt.Sprintf("%s?page=%d", path, nextPage)
		} else {
			break
		}
	}
	return events, nil
}

func (r *SHpRestApi) ReadSystemLogs(ctx context.Context) ([]redfish.SEvent, error) {
	return r.readLogs(ctx, "Systems")
}

func (r *SHpRestApi) ReadManagerLogs(ctx context.Context) ([]redfish.SEvent, error) {
	return r.readLogs(ctx, "Managers")
}

func (r *SHpRestApi) clearLogs(ctx context.Context, subsys string) error {
	path, _, err := r.GetResource(ctx, subsys, "0", "Logs", "0")
	if err != nil {
		return errors.Wrap(err, "GetResource Managers 0")
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("ClearLog"), "Action")
	_, resp, err := r.Post(ctx, path, params)
	if err != nil {
		return errors.Wrap(err, "r.Post")
	}
	if r.IsDebug {
		log.Debugf("%s", resp.PrettyString())
	}
	return nil
}

func (r *SHpRestApi) ClearSystemLogs(ctx context.Context) error {
	return r.clearLogs(ctx, "Systems")
}

func (r *SHpRestApi) ClearManagerLogs(ctx context.Context) error {
	return r.clearLogs(ctx, "Managers")
}
