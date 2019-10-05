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

package idrac

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/redfish/generic"
)

type SIDracRedfishApiFactory struct {
}

func (f *SIDracRedfishApiFactory) Name() string {
	return "iDRAC"
}

func (f *SIDracRedfishApiFactory) NewApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	return NewIDracRedfishApi(endpoint, username, password, debug)
}

func init() {
	redfish.RegisterApiFactory(&SIDracRedfishApiFactory{})
}

type SIDracRefishApi struct {
	generic.SGenericRefishApi
}

func NewIDracRedfishApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	api := &SIDracRefishApi{
		SGenericRefishApi: generic.SGenericRefishApi{
			SBaseRedfishClient: redfish.NewBaseRedfishClient(endpoint, username, password, debug),
		},
	}
	api.SetVirtualObject(api)
	return api
}

func (r *SIDracRefishApi) ParseRoot(root jsonutils.JSONObject) error {
	accountUrl, _ := root.GetString("AccountService", "@odata.id")
	if strings.Contains(accountUrl, "iDRAC.Embedded") {
		return nil
	}
	return errors.Error("not iDrac")
}

func (r *SIDracRefishApi) GetVirtualCdromInfo(ctx context.Context) (string, redfish.SCdromInfo, error) {
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

func (r *SIDracRefishApi) MountVirtualCdrom(ctx context.Context, path string, cdromUrl string) error {
	info := jsonutils.NewDict()
	info.Set("Image", jsonutils.NewString(cdromUrl))

	path = httputils.JoinPath(path, "Actions/VirtualMedia.InsertMedia")
	_, _, err := r.Post(ctx, path, info)
	if err != nil {
		return errors.Wrap(err, "r.Post")
	}
	// log.Debugf("%s", resp.PrettyString())
	return nil
}

func (r *SIDracRefishApi) UmountVirtualCdrom(ctx context.Context, path string) error {
	info := jsonutils.NewDict()

	path = httputils.JoinPath(path, "Actions/VirtualMedia.EjectMedia")
	_, _, err := r.Post(ctx, path, info)
	if err != nil {
		return errors.Wrap(err, "r.Post")
	}
	// log.Debugf("%s", resp.PrettyString())
	return nil
}

func (r *SIDracRefishApi) SetNextBootVirtualCdrom(ctx context.Context) error {
	idracConf := iDRACConfig{}
	idracConf.SystemConfiguration.Components = []iDracComponenct{
		{
			FQDD: "iDRAC.Embedded.1",
			Attributes: []iDracAttribute{
				{
					Name:  "ServerBoot.1#BootOnce",
					Value: "Enabled",
				},
				{
					Name:  "ServerBoot.1#FirstBootDevice",
					Value: "VCD-DVD",
				},
			},
		},
	}
	return r.doImportConfig(ctx, idracConf)
}

type iDracComponenct struct {
	FQDD       string           `json:"FQDD"`
	Attributes []iDracAttribute `json:"Attributes"`
}

type iDracAttribute struct {
	Name  string `json:"Name""`
	Value string `json:"Value"`
}

type iDRACConfig struct {
	SystemConfiguration struct {
		Model      string `json:"Model"`
		ServiceTag string `json:"ServiceTag"`
		// TimeStamp  time.Time `json:"TimeStamp"`
		Components []iDracComponenct `json:"Components"`
	} `json:"SystemConfiguration"`
}

func (r iDRACConfig) getConfig(fqdd string, name string) (string, error) {
	for _, comp := range r.SystemConfiguration.Components {
		if comp.FQDD == fqdd {
			for _, attr := range comp.Attributes {
				if attr.Name == name {
					return attr.Value, nil
				}
			}
		}
	}
	return "", httperrors.ErrNotFound
}

func (r iDRACConfig) toXml() string {
	buf := strings.Builder{}
	buf.WriteString("<SystemConfiguration>")
	for _, comp := range r.SystemConfiguration.Components {
		buf.WriteString(fmt.Sprintf(`<Component FQDD="%s">`, comp.FQDD))
		for _, attr := range comp.Attributes {
			buf.WriteString(fmt.Sprintf(`<Attribute Name="%s">`, attr.Name))
			buf.WriteString(attr.Value)
			buf.WriteString("</Attribute>")
		}
		buf.WriteString("</Component>")
	}
	buf.WriteString("</SystemConfiguration>")
	return buf.String()
}

func (r *SIDracRefishApi) GetNTPConf(ctx context.Context) (redfish.SNTPConf, error) {
	ntpConf := redfish.SNTPConf{}
	eConf, err := r.fetchExportConfig(ctx, "IDRAC")
	if err != nil {
		return ntpConf, errors.Wrap(err, "fetchExportConfig")
	}
	ntpConf.NTPServers = make([]string, 0)
	ntp1, _ := eConf.getConfig("iDRAC.Embedded.1", "NTPConfigGroup.1#NTP1")
	if len(ntp1) > 0 {
		ntpConf.NTPServers = append(ntpConf.NTPServers, ntp1)
	}
	ntp2, _ := eConf.getConfig("iDRAC.Embedded.1", "NTPConfigGroup.1#NTP2")
	if len(ntp2) > 0 {
		ntpConf.NTPServers = append(ntpConf.NTPServers, ntp2)
	}
	ntp3, _ := eConf.getConfig("iDRAC.Embedded.1", "NTPConfigGroup.1#NTP3")
	if len(ntp3) > 0 {
		ntpConf.NTPServers = append(ntpConf.NTPServers, ntp3)
	}
	ntpEnable, _ := eConf.getConfig("iDRAC.Embedded.1", "NTPConfigGroup.1#NTPEnable")
	if ntpEnable == "Enabled" {
		ntpConf.ProtocolEnabled = true
	}
	tz, _ := eConf.getConfig("iDRAC.Embedded.1", "Time.1#TimeZone")
	if len(tz) > 0 {
		ntpConf.TimeZone = tz
	}
	return ntpConf, nil
}

func ntpConf2idrac(conf redfish.SNTPConf) iDRACConfig {
	idracConf := iDRACConfig{}
	idracConf.SystemConfiguration.Components = []iDracComponenct{
		{
			FQDD: "iDRAC.Embedded.1",
			Attributes: []iDracAttribute{
				{
					Name:  "Time.1#TimeZone",
					Value: conf.TimeZone,
				},
				{
					Name:  "NTPConfigGroup.1#NTPEnable",
					Value: "Enabled",
				},
			},
		},
	}
	for i, srv := range conf.NTPServers {
		idracConf.SystemConfiguration.Components[0].Attributes = append(idracConf.SystemConfiguration.Components[0].Attributes, iDracAttribute{
			Name:  fmt.Sprintf("NTPConfigGroup.1#NTP%d", i+1),
			Value: srv,
		})
	}
	return idracConf
}

func (r *SIDracRefishApi) SetNTPConf(ctx context.Context, conf redfish.SNTPConf) error {
	iDracConf := ntpConf2idrac(conf)
	return r.doImportConfig(ctx, iDracConf)
}

/*
 * target:  "ALL", "IDRAC", "BIOS", "NIC", "RAID"
 */
func (r *SIDracRefishApi) fetchExportConfig(ctx context.Context, target string) (*iDRACConfig, error) {
	_, manager, err := r.GetResource(ctx, "Managers", "0")
	if err != nil {
		return nil, errors.Wrap(err, "GetResource")
	}
	urlPath, err := manager.GetString("Actions", "Oem", "OemManager.v1_1_0#OemManager.ExportSystemConfiguration", "target")
	if err != nil {
		return nil, errors.Wrap(err, "OemManager.v1_1_0#OemManager.ExportSystemConfiguration")
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("JSON"), "ExportFormat")
	params.Add(jsonutils.NewString("Default"), "IncludeInExport")
	params.Add(jsonutils.NewString(target), "ShareParameters", "Target")
	hdr, _, err := r.Post(ctx, urlPath, params)
	if err != nil {
		return nil, errors.Wrapf(err, "r.Post %s", urlPath)
	}
	jobUrl := hdr.Get("Location")
	if r.IsDebug {
		log.Debugf("%s", jobUrl)
	}
	taskStatus := "Running"
	maxWait := 3600 // 1 hour
	interval := 5
	for waited := 0; taskStatus == "Running" && waited < maxWait; waited += interval {
		time.Sleep(time.Duration(interval) * time.Second)
		resp, err := r.Get(ctx, jobUrl)
		if err != nil {
			return nil, errors.Wrapf(err, "r.Get %s", jobUrl)
		}
		log.Debugf("probe: %d", waited)
		// log.Debugf("%s", resp.PrettyString())
		taskStatus, _ = resp.GetString("TaskState")
		if taskStatus == "" {
			conf := iDRACConfig{}
			err = resp.Unmarshal(&conf)
			if err != nil {
				return nil, errors.Wrap(err, "Unmarshal iDRACConfig")
			}
			// log.Debugf("%s", jsonutils.Marshal(&conf).PrettyString())
			return &conf, nil
		}
	}
	return nil, httperrors.ErrTimeout
}

func (r *SIDracRefishApi) doImportConfig(ctx context.Context, conf iDRACConfig) error {
	_, manager, err := r.GetResource(ctx, "Managers", "0")
	if err != nil {
		return errors.Wrap(err, "GetResource")
	}
	urlPath, err := manager.GetString("Actions", "Oem", "OemManager.v1_1_0#OemManager.ImportSystemConfiguration", "target")
	if err != nil {
		return errors.Wrap(err, "OemManager.v1_1_0#OemManager.ImportSystemConfiguration")
	}
	payload := jsonutils.NewDict()
	payload.Add(jsonutils.NewString(conf.toXml()), "ImportBuffer")
	payload.Add(jsonutils.NewString("ALL"), "ShareParameters", "Target")
	hdr, _, err := r.Post(ctx, urlPath, payload)
	if err != nil {
		return errors.Wrapf(err, "r.Post %s", urlPath)
	}
	jobUrl := hdr.Get("Location")
	if r.IsDebug {
		log.Debugf("%s", jobUrl)
	}
	taskStatus := "Running"
	maxWait := 3600 // 1 hour
	interval := 5
	for waited := 0; taskStatus == "Running" && waited < maxWait; waited += interval {
		time.Sleep(time.Duration(interval) * time.Second)
		resp, err := r.Get(ctx, jobUrl)
		if err != nil {
			return errors.Wrapf(err, "r.Get %s", jobUrl)
		}
		// log.Debugf("%s", resp.PrettyString())
		taskStatus, _ = resp.GetString("TaskState")
	}
	return nil
}

func (r *SIDracRefishApi) GetConsoleJNLP(ctx context.Context) (string, error) {
	loginData := strings.Join([]string{
		"user=" + url.QueryEscape(r.GetUsername()),
		"password=" + url.QueryEscape(r.GetPassword()),
	}, "&")

	// cookie:
	// -http-session-=::http.session::0103fd02ceac2d642361b6fdcd4a5994;
	// sysidledicon=ledIcon%20grayLed;
	// tokenvalue=478be97abdaeb4d454c0418fcca9094d

	cookies := make(map[string]string)
	cookies["-http-session-"] = ""

	// first do html login
	postHdr := http.Header{}
	postHdr.Set("Content-Type", "application/x-www-form-urlencoded")
	redfish.SetCookieHeader(postHdr, cookies)
	hdr, loginResp, err := r.RawRequest(ctx, httputils.POST, "/data/login", postHdr, []byte(loginData))
	if err != nil {
		return "", errors.Wrap(err, "r.FormPost Login")
	}
	for _, cookieHdr := range hdr["Set-Cookie"] {
		parts := strings.Split(cookieHdr, ";")
		if len(parts) > 0 {
			pparts := strings.Split(parts[0], "=")
			if len(pparts) > 1 {
				cookies[pparts[0]] = pparts[1]
			}
		}
	}
	forwardUrlPattern := regexp.MustCompile(`<forwardUrl>(.*)</forwardUrl>`)
	matched := forwardUrlPattern.FindAllStringSubmatch(string(loginResp), -1)
	indexUrlStr := ""
	if len(matched) > 0 && len(matched[0]) > 1 {
		indexUrlStr = matched[0][1]
	}
	if len(indexUrlStr) == 0 {
		return "", errors.Wrapf(httperrors.ErrBadRequest, "no valid forwardUrl")
	}

	tokenPattern := regexp.MustCompile(`ST1=(\w+),ST2=`)
	matched = tokenPattern.FindAllStringSubmatch(indexUrlStr, -1)
	log.Debugf("%s", matched)
	token := ""
	if len(matched) > 0 && len(matched[0]) > 1 {
		token = matched[0][1]
	}
	cookies["tokenvalue"] = token

	getHdr := http.Header{}
	redfish.SetCookieHeader(getHdr, cookies)

	_, sysInfo, err := r.GetSystemInfo(ctx)
	if err != nil {
		return "", errors.Wrap(err, "r.GetSystemInfo")
	}
	sysStr := url.QueryEscape(fmt.Sprintf("idrac-%s, %s, User: %s", sysInfo.SKU, sysInfo.Model, r.GetUsername()))
	path := fmt.Sprintf("viewer.jnlp(%s@0@%s@%d@ST1=%s)", r.GetHost(), sysStr, time.Now().UnixNano()/1000000, token)

	_, rspBody, err := r.RawRequest(ctx, httputils.GET, path, getHdr, nil)
	if err != nil {
		return "", errors.Wrapf(err, "r.RawGet %s", path)
	}
	return string(rspBody), nil
}
