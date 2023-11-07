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

package cucloud

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/s3cli"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_CUCLOUD_CN = "联通云"
	CUCLOUD_DEFAULT_REGION    = "cn-langfang-2"
)

type ChinaUnionClientConfig struct {
	cpcfg           cloudprovider.ProviderConfig
	accessKeyId     string
	accessKeySecret string

	debug bool
}

type SChinaUnionClient struct {
	*ChinaUnionClientConfig

	client *http.Client
	lock   sync.Mutex
	ctx    context.Context

	regions []SRegion
	ownerId string
}

func NewChinaUnionClientConfig(accessKeyId, accessKeySecret string) *ChinaUnionClientConfig {
	cfg := &ChinaUnionClientConfig{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
	}
	return cfg
}

func (self *ChinaUnionClientConfig) Debug(debug bool) *ChinaUnionClientConfig {
	self.debug = debug
	return self
}

func (self *ChinaUnionClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *ChinaUnionClientConfig {
	self.cpcfg = cpcfg
	return self
}

func NewChinaUnionClient(cfg *ChinaUnionClientConfig) (*SChinaUnionClient, error) {
	client := &SChinaUnionClient{
		ChinaUnionClientConfig: cfg,
		ctx:                    context.Background(),
	}
	client.ctx = context.WithValue(client.ctx, "time", time.Now())
	var err error
	client.regions, err = client.GetRegions()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (self *SChinaUnionClient) GetRegions() ([]SRegion, error) {
	if len(self.regions) > 0 {
		return self.regions, nil
	}
	resp, err := self.list("/instance/v1/product/cloudregions", nil)
	if err != nil {
		return nil, err
	}
	ret := struct {
		Result struct {
			Total int
			List  []SRegion
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	self.regions = []SRegion{}
	for i := range ret.Result.List {
		ret.Result.List[i].client = self
		self.regions = append(self.regions, ret.Result.List[i])
	}
	return self.regions, nil
}

func (self *SChinaUnionClient) GetRegion(id string) (*SRegion, error) {
	regions, err := self.GetRegions()
	if err != nil {
		return nil, err
	}
	for i := range regions {
		if regions[i].GetId() == id || regions[i].GetGlobalId() == id {
			regions[i].client = self
			return &regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SChinaUnionClient) getUrl(resource string) string {
	return fmt.Sprintf("https://gateway.cucloud.cn/%s", strings.TrimPrefix(resource, "/"))
}

func (cli *SChinaUnionClient) getDefaultClient() *http.Client {
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

type sChinaUnionError struct {
	StatusCode int `json:"statusCode"`
	Status     string
	Code       string
	Message    string
}

func (self *sChinaUnionError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (self *sChinaUnionError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(self)
	}
	self.StatusCode = statusCode
	return self
}

func (self *SChinaUnionClient) sign(req *http.Request) (string, error) {
	keys := []string{}
	keyMap := map[string]string{}
	for k := range req.Header {
		key, ok := map[string]string{
			"Accesskey":   "accessKey",
			"Algorithm":   "algorithm",
			"Requesttime": "requestTime",
		}[k]
		if ok {
			keys = append(keys, key)
			keyMap[key] = req.Header.Get(k)
		}
	}
	params, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return "", errors.Wrapf(err, "ParseQuery")
	}
	for k := range params {
		keys = append(keys, k)
		keyMap[k] = params.Get(k)
	}
	if req.Method == "POST" {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return "", errors.Wrapf(err, "read body")
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		obj, err := jsonutils.Parse(body)
		if err != nil {
			return "", errors.Wrapf(err, "params req body")
		}
		objMap, err := obj.GetMap()
		if err != nil {
			return "", errors.Wrapf(err, "req body map")
		}
		for k := range objMap {
			keys = append(keys, k)
			keyMap[k], _ = objMap[k].GetString()
		}
	}
	sort.Strings(keys)
	signStrs := []string{}
	for _, k := range keys {
		signStrs = append(signStrs, fmt.Sprintf(`%s="%s"`, k, keyMap[k]))
	}

	hasher := hmac.New(sha256.New, []byte(self.accessKeySecret))
	hasher.Write([]byte(strings.Join(signStrs, "&")))
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (self *SChinaUnionClient) Do(req *http.Request) (*http.Response, error) {
	client := self.getDefaultClient()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("algorithm", "HmacSHA256")
	req.Header.Set("requestTime", fmt.Sprintf("%d", time.Now().UTC().UnixMilli()))
	req.Header.Set("accessKey", self.accessKeyId)

	signature, err := self.sign(req)
	if err != nil {
		return nil, errors.Wrapf(err, "sign")
	}

	req.Header.Set("sign", signature)
	return client.Do(req)
}

func (self *SChinaUnionClient) list(resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.request(httputils.GET, resource, params)
}

func (self *SChinaUnionClient) post(resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.request(httputils.POST, resource, params)
}

func (self *SChinaUnionClient) request(method httputils.THttpMethod, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := self.getUrl(resource)
	if params == nil {
		params = map[string]interface{}{}
	}
	var body jsonutils.JSONObject = jsonutils.NewDict()
	switch method {
	case httputils.GET:
		values := url.Values{}
		for k, v := range params {
			values.Set(k, v.(string))
		}
		if len(values) > 0 {
			uri = fmt.Sprintf("%s?%s", uri, values.Encode())
		}
	case httputils.POST:
		body = jsonutils.Marshal(params)
	}
	req := httputils.NewJsonRequest(method, uri, body)
	bErr := &sChinaUnionError{}
	client := httputils.NewJsonClient(self)
	_, resp, err := client.Send(self.ctx, req, bErr, self.debug)
	if err != nil {
		return nil, err
	}
	if gotypes.IsNil(resp) {
		return nil, fmt.Errorf("empty response")
	}
	code, _ := resp.GetString("code")
	if code != "200" {
		return nil, errors.Errorf(resp.String())
	}
	return resp, nil
}

func (self *SChinaUnionClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = self.GetAccountId()
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKeyId
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SChinaUnionClient) getOwnerId() (string, error) {
	if len(self.ownerId) > 0 {
		return self.ownerId, nil
	}
	client, err := self.getS3Client()
	if err != nil {
		return "", err
	}
	buckets, err := client.ListBuckets()
	if err != nil {
		return "", err
	}
	self.ownerId = buckets.Owner.ID
	return self.ownerId, nil
}

func (self *SChinaUnionClient) getS3Client() (*s3cli.Client, error) {
	client, err := s3cli.New("obs-helf.cucloud.cn", self.accessKeyId, self.accessKeySecret, true, self.debug)

	tr := httputils.GetTransport(true)
	tr.Proxy = self.cpcfg.ProxyFunc
	return client, err
}

func (self *SChinaUnionClient) GetAccountId() string {
	ownerId, _ := self.getOwnerId()
	return ownerId
}

type CashBalance struct {
	CashBalance float64
}

// 接口不可用
func (self *SChinaUnionClient) QueryBalance() (*CashBalance, error) {
	ret := &CashBalance{}
	resp, err := self.post("bill-manage-console/bill/manage/balance/queryAvailableBalanceDetail", nil)
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SChinaUnionClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}
