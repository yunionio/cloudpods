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
	region := &SRegion{client: self}
	return region, nil
}

func (self *SphereClient) GetRegions() ([]SRegion, error) {
	ret := []SRegion{}
	ret = append(ret, SRegion{client: self})
	return ret, nil
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

func (cli *SphereClient) list(res string, params url.Values, retVal interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	page, items := 1, jsonutils.NewArray()
	params.Set("pageSize", "100")
	for {
		params.Set("currentPage", fmt.Sprintf("%d", page))
		resp, err := cli._list(res, params)
		if err != nil {
			return errors.Wrapf(err, "list(%s)", res)
		}
		totalSize, _ := resp.Int("totalSize")
		if resp.Contains("items") {
			array, err := resp.GetArray("items")
			if err != nil {
				return errors.Wrapf(err, "get items")
			}
			items.Add(array...)
		}
		if totalSize <= int64(items.Length()) {
			break
		}
		page++
	}
	return items.Unmarshal(retVal)
}

func (cli *SphereClient) _list(res string, params url.Values) (jsonutils.JSONObject, error) {
	if params != nil {
		res = fmt.Sprintf("%s?%s", res, params.Encode())
	}
	return cli._jsonRequest(httputils.GET, res, nil)
}

func (cli *SphereClient) _jsonRequest(method httputils.THttpMethod, res string, params interface{}) (jsonutils.JSONObject, error) {
	client := httputils.NewJsonClient(cli.getDefaultClient())
	url := fmt.Sprintf("%s/%s", cli.authURL, strings.TrimPrefix(res, "/"))
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
	region, _ := self.GetRegion()
	ret = append(ret, region)
	return ret
}

func (self *SphereClient) GetCapabilities() []string {
	ret := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_NETWORK + cloudprovider.READ_ONLY_SUFFIX,
	}
	return ret
}
