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

package baidu

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_BAIDU_CN = "百度云"
	BAIDU_DEFAULT_REGION    = "bj"
	ISO8601                 = "2006-01-02T15:04:05Z"
)

type BaiduClientConfig struct {
	cpcfg           cloudprovider.ProviderConfig
	accessKeyId     string
	accessKeySecret string

	debug bool
}

type SBaiduClient struct {
	*BaiduClientConfig

	client *http.Client
	lock   sync.Mutex
	ctx    context.Context

	ownerId string
}

func NewBaiduClientConfig(accessKeyId, accessKeySecret string) *BaiduClientConfig {
	cfg := &BaiduClientConfig{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
	}
	return cfg
}

func (self *BaiduClientConfig) Debug(debug bool) *BaiduClientConfig {
	self.debug = debug
	return self
}

func (self *BaiduClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *BaiduClientConfig {
	self.cpcfg = cpcfg
	return self
}

func NewBaiduClient(cfg *BaiduClientConfig) (*SBaiduClient, error) {
	client := &SBaiduClient{
		BaiduClientConfig: cfg,
		ctx:               context.Background(),
	}
	client.ctx = context.WithValue(client.ctx, "time", time.Now())
	_, err := client.getOwnerId()
	return client, err
}

func (self *SBaiduClient) GetRegions() []SRegion {
	ret := []SRegion{}
	for k, v := range regions {
		ret = append(ret, SRegion{
			client:     self,
			Region:     k,
			RegionName: v,
		})
	}
	return ret
}

func (self *SBaiduClient) GetRegion(id string) (*SRegion, error) {
	regions := self.GetRegions()
	for i := range regions {
		if regions[i].Region == id {
			regions[i].client = self
			return &regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SBaiduClient) getUrl(service, regionId, resource string) (string, error) {
	if len(regionId) == 0 {
		regionId = BAIDU_DEFAULT_REGION
	}
	switch service {
	case "bbc":
		return fmt.Sprintf("https://bbc.%s.baidubce.com/%s", regionId, strings.TrimPrefix(resource, "/")), nil
	case "bos":
		return fmt.Sprintf("https://%s.bcebos.com", regionId), nil
	case "billing":
		return fmt.Sprintf("https://billing.baidubce.com/%s", strings.TrimPrefix(resource, "/")), nil
	default:
		return "", errors.Wrapf(cloudprovider.ErrNotSupported, service)
	}
}

func (cli *SBaiduClient) getDefaultClient() *http.Client {
	cli.lock.Lock()
	defer cli.lock.Unlock()
	if !gotypes.IsNil(cli.client) {
		return cli.client
	}
	cli.client = httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(cli.client, cli.cpcfg.ProxyFunc)
	ts, _ := cli.client.Transport.(*http.Transport)
	ts.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	cli.client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		if cli.cpcfg.ReadOnly {
			if req.Method == "GET" {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	return cli.client
}

type sBaiduError struct {
	StatusCode int    `json:"statusCode"`
	RequestId  string `json:"requestId"`
	Code       string
	Message    string
}

func (self *sBaiduError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (self *sBaiduError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(self)
	}
	self.StatusCode = statusCode
	return self
}

func (self *SBaiduClient) Do(req *http.Request) (*http.Response, error) {
	client := self.getDefaultClient()

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("x-bce-date", time.Now().UTC().Format(ISO8601))
	req.Header.Set("host", req.Host)

	signature, err := self.sign(req)
	if err != nil {
		return nil, errors.Wrapf(err, "sign")
	}

	req.Header.Set("Authorization", signature)
	return client.Do(req)
}

func (self *SBaiduClient) list(service, regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.request(httputils.GET, service, regionId, resource, params)
}

func (self *SBaiduClient) post(service, regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.request(httputils.POST, service, regionId, resource, params)
}

func (self *SBaiduClient) request(method httputils.THttpMethod, service, regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri, err := self.getUrl(service, regionId, resource)
	if err != nil {
		return nil, err
	}
	if params == nil {
		params = map[string]interface{}{}
	}
	values := url.Values{}
	if method == httputils.GET && len(params) > 0 {
		for k, v := range params {
			values.Set(k, v.(string))
		}
		uri = fmt.Sprintf("%s?%s", uri, values.Encode())
	}
	req := httputils.NewJsonRequest(method, uri, params)
	bErr := &sBaiduError{}
	client := httputils.NewJsonClient(self)
	_, resp, err := client.Send(self.ctx, req, bErr, self.debug)
	return resp, err
}

func (self *SBaiduClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = self.GetAccountId()
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKeyId
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SBaiduClient) getOwnerId() (string, error) {
	if len(self.ownerId) > 0 {
		return self.ownerId, nil
	}
	resp, err := self.list("bos", "bj", "/", nil)
	if err != nil {
		return "", err
	}
	self.ownerId, err = resp.GetString("owner", "id")
	return self.ownerId, err
}

func (self *SBaiduClient) GetAccountId() string {
	ownerId, _ := self.getOwnerId()
	return ownerId
}

type CashBalance struct {
	CashBalance float64
}

func (self *SBaiduClient) QueryBalance() (*CashBalance, error) {
	resp, err := self.post("billing", "", "/v1/finance/cash/balance", nil)
	if err != nil {
		return nil, err
	}
	ret := &CashBalance{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SBaiduClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}
