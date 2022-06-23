package incloudsphere

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_INCLOUD_SPHERE = api.CLOUD_PROVIDER_INCLOUD_SPHERE
)

type SphereClient struct {
	*SphereClientConfig
}

type SphereClientConfig struct {
	cpcfg        cloudprovider.ProviderConfig
	accessKey    string
	accessSecret string
	host         string
	authURL      string

	sessionId string

	debug bool
}

func NewSphereClientConfig(host, accessKey, accessSecret string) *SphereClientConfig {
	return &SphereClientConfig{
		host:         host,
		authURL:      fmt.Sprintf("https://%s", host),
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
}

func (self *SphereClientConfig) Debug(debug bool) *SphereClientConfig {
	self.debug = debug
	return self
}

func (self *SphereClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *SphereClientConfig {
	self.cpcfg = cpcfg
	return self
}

func NewSphereClient(cfg *SphereClientConfig) (*SphereClient, error) {
	client := &SphereClient{
		SphereClientConfig: cfg,
	}
	return client, client.auth()
}

func (self *SphereClient) auth() error {
	params := map[string]interface{}{
		"username": self.accessKey,
		"password": self.accessSecret,
		"domain":   "internal",
		"locale":   "cn",
	}
	ret, err := self.post("system/user/login", params)
	if err != nil {
		return errors.Wrapf(err, "post")
	}
	if ret.Contains("sessonId") {
		self.sessionId, err = ret.GetString("sessonId")
		if err != nil {
			return errors.Wrapf(err, "get sessionId")
		}
		return nil
	}
	return fmt.Errorf(ret.String())
}

func (self *SphereClient) GetRegion() (*SRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SphereClient) GetRegions() ([]SRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}

type SphereError struct {
	httputils.JSONClientError
}

func (ce *SphereError) ParseErrorFromJsonResponse(statusCode int, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(ce)
	}
	if ce.Code == 0 {
		ce.Code = statusCode
	}
	if len(ce.Details) == 0 && body != nil {
		ce.Details = body.String()
	}
	if len(ce.Class) == 0 {
		ce.Class = http.StatusText(statusCode)
	}
	if statusCode == 404 {
		return errors.Wrap(cloudprovider.ErrNotFound, ce.Error())
	}
	return ce
}

func (cli *SphereClient) getDefaultClient() *http.Client {
	client := httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(client, cli.cpcfg.ProxyFunc)
	ts, _ := client.Transport.(*http.Transport)
	client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response), error) {
		if cli.cpcfg.ReadOnly {
			if req.Method == "GET" || req.Method == "HEAD" {
				return nil, nil
			}
			// 认证
			if req.Method == "POST" && (strings.HasSuffix(req.URL.Path, "/authentication") || strings.HasSuffix(req.URL.Path, "/system/user/login")) {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	return client
}

func (cli *SphereClient) post(res string, params interface{}) (jsonutils.JSONObject, error) {
	return cli._jsonRequest(httputils.POST, res, params)
}

func (cli *SphereClient) list(res string, params url.Values) (jsonutils.JSONObject, error) {
	if params != nil {
		res = fmt.Sprintf("%s?%s", res, params.Encode())
	}
	return cli._jsonRequest(httputils.GET, res, nil)
}

func (cli *SphereClient) _jsonRequest(method httputils.THttpMethod, res string, params interface{}) (jsonutils.JSONObject, error) {
	client := httputils.NewJsonClient(cli.getDefaultClient())
	url := fmt.Sprintf("%s/%s", cli.authURL, res)
	req := httputils.NewJsonRequest(method, url, params)
	header := http.Header{}
	if len(cli.sessionId) > 0 {
		header.Set("Authorization", cli.sessionId)
	}
	header.Set("Version", "5.8")
	req.SetHeader(header)
	oe := &SphereError{}
	_, resp, err := client.Send(context.Background(), req, oe, cli.debug)
	return resp, err
}

func (self *SphereClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SphereClient) GetAccountId() string {
	return self.host
}

func (self *SphereClient) GetIRegions() []cloudprovider.ICloudRegion {
	ret := []cloudprovider.ICloudRegion{}
	_, err := self.GetRegions()
	if err != nil {
		return nil
	}
	return ret
}

func (self *SphereClient) GetCapabilities() []string {
	ret := []string{}
	return ret
}
