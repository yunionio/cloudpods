package ilo

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/redfish/generic"
)

type SILORedfishApiFactory struct {
}

func (f *SILORedfishApiFactory) Name() string {
	return "iLO"
}

func (f *SILORedfishApiFactory) NewApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	return NewILORedfishApi(endpoint, username, password, debug)
}

func init() {
	redfish.RegisterApiFactory(&SILORedfishApiFactory{})
}

type SILORefishApi struct {
	generic.SGenericRefishApi
}

func NewILORedfishApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	api := &SILORefishApi{
		SGenericRefishApi: generic.SGenericRefishApi{
			SBaseRedfishClient: redfish.NewBaseRedfishClient(endpoint, username, password, debug),
		},
	}
	api.SetVirtualObject(api)
	return api
}

func (r *SILORefishApi) ParseRoot(root jsonutils.JSONObject) error {
	oemHp, _ := root.Get("Oem", "Hp")
	if oemHp != nil {
		return nil
	}
	return errors.Error("not iLO")
}

func (r *SILORefishApi) Probe(ctx context.Context) error {
	err := r.SGenericRefishApi.Probe(ctx)
	if err != nil {
		return errors.Wrap(err, "r.SGenericRefishApi.Probe")
	}
	if r.IsDebug {
		resp, err := r.Get(ctx, "/redfish/v1/ResourceDirectory/")
		if err != nil {
			return errors.Wrap(err, "r.Get /redfish/v1/ResourceDirectory/")
		}
		log.Debugf("%s", resp.PrettyString())
	}
	return nil
}

func (r *SILORefishApi) GetVirtualCdromInfo(ctx context.Context) (string, redfish.SCdromInfo, error) {
	path, cdInfo, err := r.SGenericRefishApi.GetVirtualCdromInfo(ctx)
	if err == nil {
		cdInfo.SupportAction = true
	}
	return path, cdInfo, err
}

func (r *SILORefishApi) SetNextBootVirtualCdrom(ctx context.Context) error {
	path, cdInfo, err := r.GetVirtualCdromJSON(ctx)
	if err != nil {
		return errors.Wrap(err, "GetVirtualCdromJSON")
	}
	var oemKey string
	nextBoot, err := cdInfo.Bool("Oem", "Hp", "BootOnNextServerReset")
	if err != nil {
		nextBoot, err = cdInfo.Bool("Oem", "Hpe", "BootOnNextServerReset")
		if err != nil {
			return errors.Wrap(err, "no BootOnNextServerReset found???")
		}
		oemKey = "Hpe"
	} else {
		oemKey = "Hp"
	}
	if !nextBoot {
		params := jsonutils.NewDict()
		params.Add(jsonutils.JSONTrue, "Oem", oemKey, "BootOnNextServerReset")
		resp, err := r.Patch(ctx, path, params)
		if err != nil {
			return errors.Wrap(err, "r.Patch")
		}
		if r.IsDebug {
			log.Debugf("%s", resp.PrettyString())
		}
	}
	return nil
}

func (r *SILORefishApi) ReadSystemLogs(ctx context.Context) ([]redfish.SEvent, error) {
	return r.ReadLogs(ctx, "Systems", 0)
}

func (r *SILORefishApi) ReadManagerLogs(ctx context.Context) ([]redfish.SEvent, error) {
	return r.ReadLogs(ctx, "Managers", 0)
}

func (r *SILORefishApi) ClearSystemLogs(ctx context.Context) error {
	return r.ClearLogs(ctx, "Systems", 0)
}

func (r *SILORefishApi) ClearManagerLogs(ctx context.Context) error {
	return r.ClearLogs(ctx, "Managers", 0)
}

func (r *SILORefishApi) LogItemsKey() string {
	return "Items"
}

func (r *SILORefishApi) GetIndicatorLED(ctx context.Context) (bool, error) {
	_, val, err := r.GetIndicatorLEDInternal(ctx, "Systems")
	if err != nil {
		return false, errors.Wrap(err, "r.GetIndicatorLEDInternal")
	}
	if strings.EqualFold(val, "Off") {
		return false, nil
	} else {
		return true, nil
	}
}

func (r *SILORefishApi) SetIndicatorLED(ctx context.Context, on bool) error {
	valStr := "Off"
	if on {
		valStr = "Lit"
	}
	return r.SetIndicatorLEDInternal(ctx, "Systems", valStr)
}

func (r *SILORefishApi) GetNTPConf(ctx context.Context) (redfish.SNTPConf, error) {
	ntpConf := redfish.SNTPConf{}
	path, _, err := r.GetResource(ctx, "Managers", "0")
	if err != nil {
		return ntpConf, errors.Wrap(err, "r.GetResource Managers 0")
	}
	dateUrl := httputils.JoinPath(path, "DateTime")
	resp, err := r.Get(ctx, dateUrl)
	if err != nil {
		return ntpConf, errors.Wrapf(err, "r.Get %s", dateUrl)
	}
	if r.IsDebug {
		log.Debugf("%s", resp.PrettyString())
	}
	ntpSrvs, _ := jsonutils.GetStringArray(resp, "StaticNTPServers")
	tz, _ := resp.GetString("TimeZone", "Name")
	ntpConf.NTPServers = make([]string, 0)
	for _, ntpsrv := range ntpSrvs {
		if len(ntpsrv) > 0 {
			ntpConf.NTPServers = append(ntpConf.NTPServers, ntpsrv)
		}
	}
	if len(ntpConf.NTPServers) > 0 {
		ntpConf.ProtocolEnabled = true
	}
	ntpConf.TimeZone = tz
	return ntpConf, nil
}

func (r *SILORefishApi) SetNTPConf(ctx context.Context, conf redfish.SNTPConf) error {
	path, _, err := r.GetResource(ctx, "Managers", "0")
	if err != nil {
		return errors.Wrap(err, "r.GetResource Managers 0")
	}
	dateUrl := httputils.JoinPath(path, "DateTime")
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewStringArray(conf.NTPServers), "StaticNTPServers")
	params.Add(jsonutils.NewString(conf.TimeZone), "TimeZone", "Name")
	resp, err := r.Patch(ctx, dateUrl, params)
	if err != nil {
		return errors.Wrapf(err, "r.Patch %s", dateUrl)
	}
	if r.IsDebug {
		log.Debugf("%s", resp.PrettyString())
	}
	return nil
}

func (r *SILORefishApi) GetConsoleJNLP(ctx context.Context) (string, error) {
	loginData := jsonutils.NewDict()
	loginData.Add(jsonutils.NewString("login"), "method")
	loginData.Add(jsonutils.NewString(r.GetUsername()), "user_login")
	loginData.Add(jsonutils.NewString(r.GetPassword()), "password")

	postHdr := http.Header{}
	postHdr.Set("Content-Type", "application/json")
	_, loginRespBytes, err := r.RawRequest(ctx, httputils.POST, "/json/login_session", postHdr, []byte(loginData.String()))
	if err != nil {
		return "", errors.Wrap(err, "r.FormPost Login")
	}

	loginRespJson, err := jsonutils.Parse(loginRespBytes)
	if err != nil {
		return "", errors.Wrap(err, "jsonutils.Parse loginRespBytes")
	}

	sessionKey, err := loginRespJson.GetString("session_key")
	if err != nil {
		return "", errors.Wrap(err, "Get session_key")
	}

	endpoint := r.GetEndpoint()
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}

	cookies := make(map[string]string)
	cookies["sessionKey"] = sessionKey
	cookies["sessionLang"] = "en"
	cookies["sessionUrl"] = endpoint

	getHdr := http.Header{}
	redfish.SetCookieHeader(getHdr, cookies)
	_, tempBytes, err := r.RawRequest(ctx, httputils.GET, "/html/jnlp_template.html", getHdr, nil)
	if err != nil {
		return "", errors.Wrap(err, "request template")
	}

	startToken := []byte("<![CDATA[\n")
	endToken := []byte("]]>")
	pos := bytes.Index(tempBytes, startToken)
	if pos < 0 {
		return "", errors.Wrapf(err, "invalid template content %s: no start token", tempBytes)
	}
	tempBytes = tempBytes[pos+len(startToken):]
	pos = bytes.Index(tempBytes, endToken)
	if pos < 0 {
		return "", errors.Wrapf(err, "invalid template content %s: no end token", tempBytes)
	}
	template := string(tempBytes[:pos])

	// replace variables
	template = strings.ReplaceAll(template, "<%= this.baseUrl %>", endpoint)
	template = strings.ReplaceAll(template, "<%= this.sessionKey %>", sessionKey)
	template = strings.ReplaceAll(template, "<%= this.langId %>", "en")

	return template, nil
}
