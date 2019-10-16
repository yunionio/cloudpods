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
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

var redfishBasePaths = []string{
	"/redfish/v1",
	"/rest/v1",
}

type SBaseRedfishClient struct {
	object.SObject

	client *http.Client

	username string
	password string
	endpoint string

	SessionToken string
	sessionUrl   string

	Version    string
	Registries map[string]string

	IsDebug bool
}

func NewBaseRedfishClient(endpoint string, username, password string, debug bool) SBaseRedfishClient {
	client := httputils.GetDefaultClient()
	cli := SBaseRedfishClient{
		client:   client,
		endpoint: endpoint,
		username: username,
		password: password,

		IsDebug: debug,
	}
	return cli
}

func (r *SBaseRedfishClient) GetHost() string {
	parts, err := url.Parse(r.endpoint)
	if err != nil {
		log.Errorf("urlParse %s fail %s", r.endpoint, err)
		return ""
	}
	return parts.Hostname()
}

func (r *SBaseRedfishClient) GetUsername() string {
	return r.username
}

func (r *SBaseRedfishClient) GetPassword() string {
	return r.password
}

func (r *SBaseRedfishClient) GetEndpoint() string {
	return r.endpoint
}

func (r *SBaseRedfishClient) request(ctx context.Context, method httputils.THttpMethod, path string, header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	urlStr := httputils.JoinPath(r.endpoint, path)
	if header == nil {
		header = http.Header{}
	}
	if len(r.SessionToken) > 0 {
		header.Set("X-Auth-Token", r.SessionToken)
	} else {
		authStr := r.username + ":" + r.password
		header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(authStr)))
	}
	// !!!always close http connection for redfish API server
	header.Set("Connection", "Close")
	header.Set("Odata-Version", "4.0")
	hdr, resp, err := httputils.JSONRequest(r.client, ctx, method, urlStr, header, body, r.IsDebug)
	if err != nil {
		return nil, nil, errors.Wrap(err, "httputils.JSONRequest")
	}
	return hdr, resp, nil
}

/*func SetCookieHeader(hdr http.Header, cookies map[string]string) {
	cookieParts := make([]string, 0)
	for k, v := range cookies {
		cookieParts = append(cookieParts, k+"="+v)
	}
	if len(cookieParts) > 0 {
		hdr.Set("Cookie", strings.Join(cookieParts, "; "))
	}
}

func (r *SBaseRedfishClient) RawRequest(ctx context.Context, method httputils.THttpMethod, path string, header http.Header, body []byte) (http.Header, []byte, error) {
	urlStr := httputils.JoinPath(r.endpoint, path)
	if header == nil {
		header = http.Header{}
	}
	header.Set("Connection", "Close")
	header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.12; rv:69.0) Gecko/20100101 Firefox/69.0")
	resp, err := httputils.Request(r.client, ctx, method, urlStr, header, bytes.NewReader(body), r.IsDebug)
	hdr, rspBody, err := httputils.ParseResponse(resp, err, r.IsDebug)
	if err != nil {
		return nil, nil, errors.Wrap(err, "httputils.Request")
	}
	return hdr, rspBody, nil
}*/

func (r *SBaseRedfishClient) Get(ctx context.Context, path string) (jsonutils.JSONObject, error) {
	_, resp, err := r.request(ctx, httputils.GET, path, nil, nil)
	return resp, err
}

func (r *SBaseRedfishClient) Patch(ctx context.Context, path string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	_, resp, err := r.request(ctx, httputils.PATCH, path, nil, body)
	return resp, err
}

func (r *SBaseRedfishClient) Post(ctx context.Context, path string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return r.request(ctx, httputils.POST, path, nil, body)
}

func (r *SBaseRedfishClient) Delete(ctx context.Context, path string) (http.Header, jsonutils.JSONObject, error) {
	return r.request(ctx, httputils.DELETE, path, nil, nil)
}

func (r *SBaseRedfishClient) IRedfishDriver() IRedfishDriver {
	return r.GetVirtualObject().(IRedfishDriver)
}

func (r *SBaseRedfishClient) ParseRoot(root jsonutils.JSONObject) error {
	return nil
}

func (r *SBaseRedfishClient) Probe(ctx context.Context) error {
	resp, err := r.Get(ctx, r.IRedfishDriver().BasePath())
	if err != nil {
		return errors.Wrap(err, "r.get")
	}
	if r.IsDebug {
		log.Debugf("%s", resp.PrettyString())
	}
	err = r.IRedfishDriver().ParseRoot(resp)
	if err != nil {
		return errors.Wrap(err, "r.IRedfishDriver().ParseRoot(resp)")
	}
	r.Version, err = resp.GetString(r.IRedfishDriver().VersionKey())
	if err != nil {
		return errors.Wrap(err, "Get RedfishVersion fail")
	}
	resp = r.IRedfishDriver().GetParent(resp)
	respMap, err := resp.GetMap()
	if err != nil {
		return errors.Wrap(err, "resp.GetMap")
	}
	r.Registries = make(map[string]string)
	for k := range respMap {
		urlPath, _ := respMap[k].GetString(r.IRedfishDriver().LinkKey())
		if len(urlPath) > 0 {
			r.Registries[k] = urlPath
			log.Debugf("%s %s", k, urlPath)
		}
	}
	if len(r.Registries) == 0 {
		return errors.Wrap(httperrors.ErrNotFound, "no url found")
	}
	return nil
}

func (r *SBaseRedfishClient) Login(ctx context.Context) error {
	sessionurl := httputils.JoinPath(r.IRedfishDriver().BasePath(), "Sessions")
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(r.username), "UserName")
	params.Add(jsonutils.NewString(r.password), "Password")
	hdr, _, err := r.Post(ctx, sessionurl, params)
	if err != nil {
		return errors.Wrap(err, "Login")
	}
	r.SessionToken = hdr.Get("X-Auth-Token")
	r.sessionUrl = hdr.Get("Location")
	pos := strings.Index(r.sessionUrl, r.IRedfishDriver().BasePath())
	if pos > 0 {
		r.sessionUrl = r.sessionUrl[pos:]
	}
	return nil
}

func (r *SBaseRedfishClient) Logout(ctx context.Context) error {
	if len(r.sessionUrl) > 0 {
		_, _, err := r.Delete(ctx, r.sessionUrl)
		if err != nil {
			return errors.Wrap(err, "Logout")
		}
		r.sessionUrl = ""
		r.SessionToken = ""
	}
	return nil
}

func (r *SBaseRedfishClient) GetResource(ctx context.Context, resname ...string) (string, jsonutils.JSONObject, error) {
	return r.getResourceInternal(ctx, nil, resname...)
}

func (r *SBaseRedfishClient) getResourceInternal(ctx context.Context, parent jsonutils.JSONObject, resname ...string) (string, jsonutils.JSONObject, error) {
	var path string
	if parent == nil {
		path = r.Registries[resname[0]]
	} else {
		if parent.Contains(r.IRedfishDriver().MemberKey()) {
			// a list
			paths, err := parent.GetArray(r.IRedfishDriver().MemberKey())
			if err != nil {
				return "", nil, errors.Wrap(err, "find member list fail")
			}
			idx, err := strconv.ParseInt(resname[0], 10, 64)
			if err != nil {
				return "", nil, errors.Wrapf(err, "invalid index %s", resname[0])
			}
			if idx < 0 || idx >= int64(len(paths)) {
				return "", nil, errors.Wrap(httperrors.ErrOutOfRange, resname[0])
			}
			path, err = paths[idx].GetString(r.IRedfishDriver().LinkKey())
			if err != nil {
				return "", nil, errors.Wrapf(err, "fail to find path at index %s", resname[0])
			}
		} else if parent.Contains(resname[0]) {
			// a dict
			var err error
			path, err = parent.GetString(resname[0], r.IRedfishDriver().LinkKey())
			if err != nil {
				return "", nil, errors.Wrapf(err, "fail to find path for %s", resname[0])
			}
		}
	}
	if len(path) == 0 {
		return "", nil, errors.Wrapf(httperrors.ErrNotFound, "resource \"%s\" not found!", resname[0])
	}
	resp, err := r.Get(ctx, path)
	if err != nil {
		return "", nil, errors.Wrapf(err, "r.get %s", path)
	}
	if len(resname) == 1 {
		return path, resp, nil
	}
	resp = r.IRedfishDriver().GetParent(resp)
	return r.getResourceInternal(ctx, resp, resname[1:]...)
}

func (r *SBaseRedfishClient) GetResourceCount(ctx context.Context, resname ...string) (int, error) {
	_, resp, err := r.getResourceInternal(ctx, nil, resname...)
	if err != nil {
		return -1, errors.Wrap(err, "r.get")
	}
	if r.IsDebug {
		log.Debugf("%s", resp.PrettyString())
	}
	resp = r.IRedfishDriver().GetParent(resp)
	members, err := resp.GetArray(r.IRedfishDriver().MemberKey())
	if err != nil {
		return -1, errors.Wrap(err, "find member error")
	}
	return len(members), nil
}

func (r *SBaseRedfishClient) GetVirtualCdromJSON(ctx context.Context) (string, jsonutils.JSONObject, error) {
	_, resp, err := r.GetResource(ctx, "Managers", "0", "VirtualMedia")
	if err != nil {
		return "", nil, errors.Wrap(err, "r.GetResource")
	}
	resp = r.IRedfishDriver().GetParent(resp)
	vmList, err := resp.GetArray(r.IRedfishDriver().MemberKey())
	for i := len(vmList) - 1; i >= 0; i -= 1 {
		path, _ := vmList[i].GetString(r.IRedfishDriver().LinkKey())
		if len(path) == 0 {
			continue
		}
		cdResp, _ := r.Get(ctx, path)
		if cdResp != nil {
			mts, err := cdResp.GetArray("MediaTypes")
			if err == nil {
				for i := range mts {
					mt, _ := mts[i].GetString()
					if strings.Contains(mt, "CD") || strings.Contains(mt, "DVD") {
						// CD,DVD
						return path, cdResp, nil
					}
				}
			}
		}
	}
	return "", nil, httperrors.ErrNotFound
}

func (r *SBaseRedfishClient) GetVirtualCdromInfo(ctx context.Context) (string, SCdromInfo, error) {
	cdInfo := SCdromInfo{}
	path, jsonResp, err := r.GetVirtualCdromJSON(ctx)
	if err != nil {
		return "", cdInfo, errors.Wrap(err, "r.GetVirtualCdromJSON")
	}
	imgPath, _ := jsonResp.GetString("Image")
	if imgPath == "null" {
		imgPath = ""
	}
	cdInfo.Image = imgPath
	return path, cdInfo, nil
}

func (r *SBaseRedfishClient) MountVirtualCdrom(ctx context.Context, path string, cdromUrl string, boot bool) error {
	info := jsonutils.NewDict()
	info.Set("Image", jsonutils.NewString(cdromUrl))

	resp, err := r.Patch(ctx, path, info)
	if err != nil {
		return errors.Wrap(err, "r.Patch")
	}
	log.Debugf("%s", resp.PrettyString())
	return nil
}

func (r *SBaseRedfishClient) UmountVirtualCdrom(ctx context.Context, path string) error {
	info := jsonutils.NewDict()
	info.Set("Image", jsonutils.JSONNull)

	resp, err := r.Patch(ctx, path, info)
	if err != nil {
		return errors.Wrap(err, "r.Patch")
	}
	log.Debugf("%s", resp.PrettyString())
	return nil
}

func (r *SBaseRedfishClient) SetNextBootVirtualCdrom(ctx context.Context) error {
	return httperrors.ErrNotImplemented
}

func (r *SBaseRedfishClient) GetSystemInfo(ctx context.Context) (string, SSystemInfo, error) {
	sysInfo := SSystemInfo{}
	path, resp, err := r.GetResource(ctx, "Systems", "0")
	if err != nil {
		return path, sysInfo, errors.Wrap(err, "r.GetResource Systems")
	}
	err = resp.Unmarshal(&sysInfo)
	if err != nil {
		return path, sysInfo, errors.Wrap(err, "resp.Unmarshal")
	}

	sysInfo.SerialNumber = strings.TrimSpace(sysInfo.SerialNumber)
	sysInfo.SKU = strings.TrimSpace(sysInfo.SKU)
	sysInfo.Model = strings.TrimSpace(sysInfo.Model)
	sysInfo.Manufacturer = strings.TrimSpace(sysInfo.Manufacturer)

	if strings.EqualFold(sysInfo.PowerState, types.POWER_STATUS_ON) {
		sysInfo.PowerState = types.POWER_STATUS_ON
	} else {
		sysInfo.PowerState = types.POWER_STATUS_OFF
	}

	memGBStr, _ := resp.GetString("MemorySummary", "TotalSystemMemoryGiB")
	memGB, _ := strconv.ParseInt(memGBStr, 10, 64)
	if memGB > 0 {
		sysInfo.MemoryGB = int(memGB)
	} else {
		memGB, _ := strconv.ParseFloat(memGBStr, 64)
		sysInfo.MemoryGB = int(memGB)
	}

	nodeCount, _ := resp.Int("ProcessorSummary", "Count")
	cpuDesc, _ := resp.GetString("ProcessorSummary", "Model")
	sysInfo.NodeCount = int(nodeCount)
	sysInfo.CpuDesc = strings.TrimSpace(cpuDesc)

	nextBootDev, _ := resp.GetString("Boot", "BootSourceOverrideTarget")
	sysInfo.NextBootDev = strings.TrimSpace(nextBootDev)
	nextBootDevSupports, _ := resp.GetArray("Boot", "BootSourceOverrideTarget@Redfish.AllowableValues")
	if len(nextBootDevSupports) == 0 {
		nextBootDevSupports, _ = resp.GetArray("Boot", "BootSourceOverrideSupported")
	}
	sysInfo.NextBootDevSupported = make([]string, len(nextBootDevSupports))
	for i := range nextBootDevSupports {
		devStr, _ := nextBootDevSupports[i].GetString()
		sysInfo.NextBootDevSupported[i] = devStr
	}

	resetTypeValues, _ := resp.GetArray("Actions", "#ComputerSystem.Reset", "ResetType@Redfish.AllowableValues")
	sysInfo.ResetTypeSupported = make([]string, len(resetTypeValues))
	for i := range resetTypeValues {
		resetTypeStr, _ := resetTypeValues[i].GetString()
		sysInfo.ResetTypeSupported[i] = resetTypeStr
	}

	nicListPath, _ := resp.GetString("EthernetInterfaces", "@odata.id")
	nicListInfo, err := r.Get(ctx, nicListPath)
	if err != nil {
		return path, sysInfo, errors.Wrap(err, "Get EthernetInterfaces fail")
	}
	nicList, _ := nicListInfo.GetArray("Members")
	if len(nicList) > 0 {
		sysInfo.EthernetNICs = make([]string, len(nicList))
		for i := range nicList {
			nicPath, _ := nicList[i].GetString("@odata.id")
			nicInfo, err := r.Get(ctx, nicPath)
			if err != nil {
				return path, sysInfo, errors.Wrapf(err, "Get EthernetInterface[%d] error", i)
			}
			var macAddr string
			for _, key := range []string{
				"MacAddress",
				"MACAddress",
				"PermanentMACAddress",
			} {
				macAddr, _ = nicInfo.GetString(key)
				if len(macAddr) > 0 {
					break
				}
			}
			sysInfo.EthernetNICs[i] = netutils.FormatMacAddr(macAddr)
		}
	}

	return path, sysInfo, nil
}

func (r *SBaseRedfishClient) SetNextBootDev(ctx context.Context, dev string) error {
	path, sysInfo, err := r.IRedfishDriver().GetSystemInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "r.GetSystemInfo")
	}
	if sysInfo.NextBootDev == dev {
		return nil
	}
	if !utils.IsInStringArray(dev, sysInfo.NextBootDevSupported) {
		return errors.Wrapf(httperrors.ErrBadRequest, "%s not supported: %s", dev, sysInfo.NextBootDevSupported)
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(dev), "Boot", "BootSourceOverrideTarget")
	patchResp, err := r.Patch(ctx, path, params)
	if err != nil {
		return errors.Wrap(err, "r.Patch")
	}
	if r.IsDebug {
		log.Debugf("%s", patchResp.PrettyString())
	}
	return nil
}

func (r *SBaseRedfishClient) Reset(ctx context.Context, action string) error {
	_, system, err := r.GetResource(ctx, "Systems", "0")
	if err != nil {
		return errors.Wrap(err, "r.GetSystemInfo")
	}
	urlPath, err := system.GetString("Actions", "#ComputerSystem.Reset", "target")
	if err != nil {
		return errors.Wrap(err, "Actions.#ComputerSystem.Reset.target")
	}
	resetTypes, _ := jsonutils.GetStringArray(system, "Actions", "#ComputerSystem.Reset", "ResetType@Redfish.AllowableValues")
	if !utils.IsInStringArray(action, resetTypes) {
		return errors.Wrapf(httperrors.ErrBadRequest, "%s not supported: %s", action, resetTypes)
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(action), "ResetType")
	_, _, err = r.Post(ctx, urlPath, params)
	if err != nil {
		return errors.Wrap(err, "Actions/ComputerSystem.Reset")
	}
	return nil
}

func (r *SBaseRedfishClient) BmcReset(ctx context.Context) error {
	_, manager, err := r.GetResource(ctx, "Managers", "0")
	if err != nil {
		return errors.Wrap(err, "r.GetSystemInfo")
	}
	urlPath, err := manager.GetString("Actions", "#Manager.Reset", "target")
	if err != nil {
		return errors.Wrap(err, "Actions.#Manager.Reset.target")
	}
	params := jsonutils.NewDict()
	restTypes, _ := jsonutils.GetStringArray(manager, "Actions", "#Manager.Reset", "ResetType@Redfish.AllowableValues")
	if len(restTypes) > 0 {
		params.Add(jsonutils.NewString("GracefulRestart"), "ResetType")
	}
	_, _, err = r.Post(ctx, urlPath, params)
	if err != nil {
		return errors.Wrap(err, "Actions/Manager.Reset")
	}
	return nil
}

func Int2str(i int) string {
	return strconv.FormatInt(int64(i), 10)
}

func (r *SBaseRedfishClient) ReadLogs(ctx context.Context, subsys string, index int) ([]SEvent, error) {
	_, manager, err := r.GetResource(ctx, subsys, "0", "LogServices", Int2str(index))
	if err != nil {
		return nil, errors.Wrap(err, "GetResource Managers 0")
	}
	manager = r.IRedfishDriver().GetParent(manager)
	path, _ := manager.GetString("Entries", r.IRedfishDriver().LinkKey())
	if len(path) == 0 {
		return nil, errors.Wrap(err, "no entry???")
	}
	events := make([]SEvent, 0)
	for {
		resp, err := r.Get(ctx, path)
		if err != nil {
			return nil, errors.Wrap(err, path)
		}
		tmpEvents := make([]SEvent, 0)
		err = resp.Unmarshal(&tmpEvents, r.IRedfishDriver().LogItemsKey())
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		events = append(events, tmpEvents...)
		nextPage, _ := resp.GetString("@odata.nextLink")
		if len(nextPage) > 0 {
			path = nextPage
		} else {
			break
		}
	}
	return events, nil
}

func (r *SBaseRedfishClient) ReadSystemLogs(ctx context.Context) ([]SEvent, error) {
	return r.ReadLogs(ctx, "Managers", 0)
}

func (r *SBaseRedfishClient) ReadManagerLogs(ctx context.Context) ([]SEvent, error) {
	return r.ReadLogs(ctx, "Managers", 1)
}

func (r *SBaseRedfishClient) ClearLogs(ctx context.Context, subsys string, index int) error {
	_, logInfo, err := r.GetResource(ctx, subsys, "0", "LogServices", Int2str(index))
	if err != nil {
		return errors.Wrap(err, "GetResource Managers 0")
	}
	urlPath, err := logInfo.GetString("Actions", "#LogService.ClearLog", "target")
	if err != nil {
		return errors.Wrap(err, "Actions.#LogService.ClearLog.target")
	}
	_, _, err = r.Post(ctx, urlPath, jsonutils.NewDict())
	if err != nil {
		return errors.Wrap(err, "r.Post")
	}
	return nil
}

func (r *SBaseRedfishClient) ClearSystemLogs(ctx context.Context) error {
	return r.ClearLogs(ctx, "Managers", 0)
}

func (r *SBaseRedfishClient) ClearManagerLogs(ctx context.Context) error {
	return r.ClearLogs(ctx, "Managers", 1)
}

func (r *SBaseRedfishClient) GetBiosInfo(ctx context.Context) (SBiosInfo, error) {
	biosInfo := SBiosInfo{}
	path, _, err := r.GetResource(ctx, "Systems", "0")
	if err != nil {
		return biosInfo, errors.Wrap(err, "r.GetResource Systems 0")
	}
	biosPath := httputils.JoinPath(path, "Bios/")
	resp, err := r.Get(ctx, biosPath)
	if err != nil {
		return biosInfo, errors.Wrapf(err, "r.Get %s", biosPath)
	}
	log.Debugf("%s", resp.PrettyString())
	return biosInfo, nil
}

func (r *SBaseRedfishClient) GetIndicatorLEDInternal(ctx context.Context, subsys string) (string, string, error) {
	path, resp, err := r.GetResource(ctx, subsys, "0")
	if err != nil {
		return path, "", errors.Wrap(err, "GetResource")
	}
	val, err := resp.GetString("IndicatorLED")
	if err != nil {
		return path, "", errors.Wrap(err, "GetString IndicatorLED")
	}
	return path, val, nil

}

func (r *SBaseRedfishClient) GetIndicatorLED(ctx context.Context) (bool, error) {
	_, val, err := r.GetIndicatorLEDInternal(ctx, "Chassis")
	if err != nil {
		return false, errors.Wrap(err, "r.GetIndicatorLEDInternal")
	}
	if strings.EqualFold(val, "Off") {
		return false, nil
	} else {
		return true, nil
	}
}

// possible IndicatorLED values are: "Lit" or "Blinking" or "Off"
func (r *SBaseRedfishClient) SetIndicatorLEDInternal(ctx context.Context, subsys string, val string) error {
	path, led, err := r.GetIndicatorLEDInternal(ctx, subsys)
	if err != nil {
		return errors.Wrap(err, "GetIndicatorLEDInternal")
	}
	if led == val {
		// no need to set
		return nil
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(val), "IndicatorLED")
	resp, err := r.Patch(ctx, path, params)
	if err != nil {
		return errors.Wrap(err, "r.Patch")
	}
	log.Debugf("%s", resp)
	return nil
}

func (r *SBaseRedfishClient) SetIndicatorLED(ctx context.Context, on bool) error {
	valStr := "Off"
	if on {
		valStr = "Blinking"
	}
	return r.SetIndicatorLEDInternal(ctx, "Chassis", valStr)
}

func (r *SBaseRedfishClient) GetPower(ctx context.Context) ([]SPower, error) {
	_, resp, err := r.GetResource(ctx, "Chassis", "0", "Power")
	if err != nil {
		return nil, errors.Wrap(err, "GetResource")
	}
	powers := make([]SPower, 0)
	err = resp.Unmarshal(&powers, "PowerControl")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return powers, nil
}

func (r *SBaseRedfishClient) GetThermal(ctx context.Context) ([]STemperature, error) {
	_, resp, err := r.GetResource(ctx, "Chassis", "0", "Thermal")
	if err != nil {
		return nil, errors.Wrap(err, "GetResource")
	}
	temps := make([]STemperature, 0)
	err = resp.Unmarshal(&temps, "Temperatures")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return temps, nil
}

func (r *SBaseRedfishClient) GetNTPConf(ctx context.Context) (SNTPConf, error) {
	ntpConf := SNTPConf{}
	_, resp, err := r.GetResource(ctx, "Managers", "0", "NetworkProtocol")
	if err != nil {
		return ntpConf, errors.Wrap(err, "GetResource")
	}
	err = resp.Unmarshal(&ntpConf, "NTP")
	if err != nil {
		return ntpConf, errors.Wrap(err, "resp.Unmarshal NTP")
	}
	return ntpConf, nil
}

func (r *SBaseRedfishClient) SetNTPConf(ctx context.Context, conf SNTPConf) error {
	path, _, err := r.GetResource(ctx, "Managers", "0", "NetworkProtocol")
	if err != nil {
		return errors.Wrap(err, "GetResource")
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.Marshal(conf), "NTP")
	resp, err := r.Patch(ctx, path, params)
	if err != nil {
		return errors.Wrap(err, "r.Patch")
	}
	if r.IsDebug {
		log.Debugf("%s", resp.PrettyString())
	}
	return nil
}

func (r *SBaseRedfishClient) GetConsoleJNLP(ctx context.Context) (string, error) {
	return "", httperrors.ErrNotImplemented
}

func (r *SBaseRedfishClient) GetLanConfigs(ctx context.Context) ([]types.SIPMILanConfig, error) {
	_, ethIfsJson, err := r.GetResource(ctx, "Managers", "0", "EthernetInterfaces")
	if err != nil {
		return nil, errors.Wrap(err, "GetResource Managers 0 EthernetInterfaces")
	}
	ethIfs, err := ethIfsJson.GetArray(r.IRedfishDriver().MemberKey())
	if err != nil {
		return nil, errors.Wrap(err, "GetArray")
	}
	ret := make([]types.SIPMILanConfig, 0)
	for i := range ethIfs {
		ethLink, _ := ethIfs[i].GetString(r.IRedfishDriver().LinkKey())
		if len(ethLink) == 0 {
			continue
		}
		ethJson, err := r.Get(ctx, ethLink)
		if err != nil {
			continue
		}
		v4Addrs, err := ethJson.GetArray("IPv4Addresses")
		if err != nil {
			continue
		}
		if len(v4Addrs) == 0 {
			continue
		}
		for i := range v4Addrs {
			addr, err := v4Addrs[i].GetString("Address")
			if err != nil {
				continue
			}
			if len(addr) > 0 && addr != "0.0.0.0" {
				// find a config
				conf := types.SIPMILanConfig{}
				conf.IPAddr = addr
				mask, _ := v4Addrs[i].GetString("SubnetMask")
				conf.Netmask = mask
				gw, _ := v4Addrs[i].GetString("Gateway")
				conf.Gateway = gw
				src, _ := v4Addrs[i].GetString("AddressOrigin")
				if len(src) == 0 || src == "null" {
					src = "static"
				}
				conf.IPSrc = strings.ToLower(src)
				mac, _ := ethJson.GetString("MACAddress")
				conf.Mac, _ = net.ParseMAC(mac)
				var vlanId int64
				if ethJson.Contains("VLAN") {
					vlanId, _ = ethJson.Int("VLAN", "VLANId")
				} else {
					vlanId, _ = ethJson.Int("VLANId")
				}
				speed, _ := ethJson.Int("SpeedMbps")
				conf.SpeedMbps = int(speed)
				conf.VlanId = int(vlanId)
				ret = append(ret, conf)
			}
		}
	}
	return ret, nil
}
