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

package ctyun

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_CTYUN    = api.CLOUD_PROVIDER_CTYUN
	CLOUD_PROVIDER_CTYUN_CN = "天翼云"
	CLOUD_PROVIDER_CTYUN_EN = CLOUD_PROVIDER_CTYUN

	SERVICE_ECS   = "ecs"
	SERVICE_VPC   = "vpc"
	SERVICE_IMAGE = "image"
	SERVICE_ACCT  = "acct"
	SERVICE_EBS   = "ebs"
)

type CtyunClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	projectId    string
	accessKey    string
	accessSecret string

	debug bool
}

func NewSCtyunClientConfig(accessKey, accessSecret string) *CtyunClientConfig {
	cfg := &CtyunClientConfig{
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
	return cfg
}

func (cfg *CtyunClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *CtyunClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *CtyunClientConfig) Debug(debug bool) *CtyunClientConfig {
	cfg.debug = debug
	return cfg
}

type SCtyunClient struct {
	*CtyunClientConfig

	regions []SRegion

	lock   sync.Mutex
	client *http.Client
	ctx    context.Context
}

func NewSCtyunClient(cfg *CtyunClientConfig) (*SCtyunClient, error) {
	client := &SCtyunClient{
		CtyunClientConfig: cfg,
		ctx:               context.Background(),
	}
	client.ctx = context.WithValue(client.ctx, "time", time.Now())

	var err error
	client.regions, err = client.GetRegions()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (self *SCtyunClient) getUrl(service, resource string) (string, error) {
	switch service {
	case SERVICE_ECS, SERVICE_VPC, SERVICE_IMAGE:
		return fmt.Sprintf("https://ct%s-global.ctapi.ctyun.cn/%s", service, strings.TrimPrefix(resource, "/")), nil
	case SERVICE_ACCT, SERVICE_EBS:
		return fmt.Sprintf("https://%s-global.ctapi.ctyun.cn/%s", service, strings.TrimPrefix(resource, "/")), nil
	default:
		return "", errors.Wrapf(cloudprovider.ErrNotSupported, "service %s", service)
	}
}

type sCtyunError struct {
	StatusCode string
	Code       string
	EopErrCode string
	RequestId  string
	Message    string
}

func (self *sCtyunError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (self *sCtyunError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(self)
	}
	if strings.Contains(self.Message, "signature verification failed") {
		return errors.Wrapf(cloudprovider.ErrInvalidAccessKey, jsonutils.Marshal(self).String())
	}
	return self
}

func (cli *SCtyunClient) getDefaultClient() *http.Client {
	cli.lock.Lock()
	defer cli.lock.Unlock()
	if !gotypes.IsNil(cli.client) {
		return cli.client
	}
	cli.client = httputils.GetTimeoutClient(time.Minute)
	httputils.SetClientProxyFunc(cli.client, cli.cpcfg.ProxyFunc)
	ts, _ := cli.client.Transport.(*http.Transport)
	ts.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	cli.client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		if req.Method == "GET" {
			return nil, nil
		}
		for _, prefix := range []string{"list", "query", "info", "get", "detail", "show"} {
			if strings.Contains(req.URL.Path, prefix) {
				return nil, nil
			}
		}
		if cli.cpcfg.ReadOnly {
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	return cli.client
}

func (self *SCtyunClient) sign(req *http.Request) (string, error) {
	eopDate := req.Header.Get("eop-date")
	requestId := req.Header.Get("ctyun-eop-request-id")
	headerStr := fmt.Sprintf("ctyun-eop-request-id:%s\neop-date:%s\n",
		requestId,
		eopDate,
	)

	keys := []string{}
	for key := range req.URL.Query() {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := []string{}
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(req.URL.Query().Get(key))))
	}

	body := []byte{}
	if req.Method == "POST" {
		var err error
		body, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return "", errors.Wrapf(err, "read body")
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	}

	var hmacSha256 = func(secret, data []byte) []byte {
		hasher := hmac.New(sha256.New, []byte(secret))
		hasher.Write(data)
		return hasher.Sum(nil)
	}

	hash := sha256.New()
	hash.Write(body)
	bodyHash := hex.EncodeToString(hash.Sum(nil))

	signStr := fmt.Sprintf("%s\n%s\n%s", headerStr, strings.Join(parts, "&"), bodyHash)

	kTime := hmacSha256([]byte(self.accessSecret), []byte(eopDate))
	kAk := hmacSha256(kTime, []byte(self.accessKey))
	t := strings.Split(eopDate, "T")[0]
	kDate := hmacSha256(kAk, []byte(t))
	signBase64 := base64.StdEncoding.EncodeToString(hmacSha256(kDate, []byte(signStr)))

	return fmt.Sprintf("%s Headers=ctyun-eop-request-id;eop-date Signature=%s", self.accessKey, signBase64), nil
}

func (self *SCtyunClient) Do(req *http.Request) (*http.Response, error) {
	client := self.getDefaultClient()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ctyun-eop-request-id", utils.GenRequestId(20))
	sh, _ := time.LoadLocation("Asia/Shanghai")
	req.Header.Set("eop-date", time.Now().In(sh).Format("20060102T150405Z"))

	signature, err := self.sign(req)
	if err != nil {
		return nil, errors.Wrapf(err, "sign")
	}

	req.Header.Set("Eop-Authorization", signature)
	return client.Do(req)
}

func (self *SCtyunClient) list(service, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.request(httputils.GET, service, resource, params)
}

func (self *SCtyunClient) GetRegions() ([]SRegion, error) {
	resp, err := self.list(SERVICE_ECS, "/v4/region/list-regions", nil)
	if err != nil {
		return nil, err
	}
	ret := struct {
		ReturnObj struct {
			RegionList []SRegion
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	for i := range ret.ReturnObj.RegionList {
		ret.ReturnObj.RegionList[i].client = self
	}
	return ret.ReturnObj.RegionList, nil
}

func (self *SCtyunClient) post(service, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.request(httputils.POST, service, resource, params)
}

func (self *SCtyunClient) request(method httputils.THttpMethod, service, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri, err := self.getUrl(service, resource)
	if err != nil {
		return nil, err
	}
	if params == nil {
		params = map[string]interface{}{}
	}
	var body jsonutils.JSONObject = nil
	switch method {
	case httputils.GET:
		values := url.Values{}
		for k, v := range params {
			value := ""
			switch v.(type) {
			case string:
				value = fmt.Sprintf("%s", v)
			case int:
				value = fmt.Sprintf("%d", v)
			default:
				value = fmt.Sprintf("%d", v)
			}
			values.Set(k, value)
		}
		if len(params) > 0 {
			uri = fmt.Sprintf("%s?%s", uri, values.Encode())
		}
	case httputils.POST:
		body = jsonutils.Marshal(params)
	}
	req := httputils.NewJsonRequest(method, uri, body)
	ctErr := &sCtyunError{}
	client := httputils.NewJsonClient(self)
	_, resp, err := client.Send(self.ctx, req, ctErr, self.debug)
	if err != nil {
		return nil, err
	}
	ret := struct {
		Message     string
		Description string
		StatusCode  int
		ErrorCode   string
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	if ret.StatusCode == 800 || (ret.StatusCode == 900 && resp.Contains("returnObj")) {
		return resp, nil
	}
	if strings.HasSuffix(ret.ErrorCode, "NotFound") || ret.ErrorCode == "ebs.ebsInfo.get volume resourceId failed" {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, resp.String())
	}
	log.Errorf("request %s with params %s error: %s", uri, jsonutils.Marshal(body).String(), resp.String())
	return nil, fmt.Errorf(resp.String())
}

func (self *SCtyunClient) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	ret := []cloudprovider.ICloudRegion{}
	for i := range self.regions {
		self.regions[i].client = self
		ret = append(ret, &self.regions[i])
	}
	return ret, nil
}

func (self *SCtyunClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccounts := make([]cloudprovider.SSubAccount, 0)
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = self.GetAccountId()
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	subAccounts = append(subAccounts, subAccount)
	return subAccounts, nil
}

func (client *SCtyunClient) GetAccountId() string {
	return client.accessKey
}

func (self *SCtyunClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion(id)
	if err != nil {
		return nil, err
	}
	return region, nil
}

func (self *SCtyunClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return []cloudprovider.ICloudProject{}, nil
}

func (self *SCtyunClient) GetAccessEnv() string {
	return api.CLOUD_ACCESS_ENV_CTYUN_CHINA
}

func (self *SCtyunClient) GetRegion(id string) (*SRegion, error) {
	for i := range self.regions {
		self.regions[i].client = self
		if self.regions[i].GetId() == id || self.regions[i].GetGlobalId() == id || self.regions[i].RegionId == id {
			return &self.regions[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SCtyunClient) GetCloudRegionExternalIdPrefix() string {
	return CLOUD_PROVIDER_CTYUN
}

func (self *SCtyunClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP,
		cloudprovider.CLOUD_CAPABILITY_EIP,
	}
	return caps
}

func (self *SCtyunClient) GetBalance() {
	_, err := self.post(SERVICE_ACCT, "/bill_queryBalance", nil)
	if err != nil {
		return
	}
}
