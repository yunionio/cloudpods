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

package qingcloud

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_QINGCLOUD_CN = "青云"
	QINGCLOUD_DEFAULT_REGION    = "pek3"
	ISO8601                     = "2006-01-02T15:04:05Z"
)

type QingCloudClientConfig struct {
	cpcfg           cloudprovider.ProviderConfig
	accessKeyId     string
	accessKeySecret string

	debug bool
}

type SQingCloudClient struct {
	*QingCloudClientConfig

	client *http.Client
	lock   sync.Mutex
	ctx    context.Context

	ownerId string
}

func NewQingCloudClientConfig(accessKeyId, accessKeySecret string) *QingCloudClientConfig {
	cfg := &QingCloudClientConfig{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
	}
	return cfg
}

func (self *QingCloudClientConfig) Debug(debug bool) *QingCloudClientConfig {
	self.debug = debug
	return self
}

func (self *QingCloudClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *QingCloudClientConfig {
	self.cpcfg = cpcfg
	return self
}

func NewQingCloudClient(cfg *QingCloudClientConfig) (*SQingCloudClient, error) {
	client := &SQingCloudClient{
		QingCloudClientConfig: cfg,
		ctx:                   context.Background(),
	}
	client.ctx = context.WithValue(client.ctx, "time", time.Now())
	_, err := client.getOwnerId()
	return client, err
}

func (self *SQingCloudClient) GetRegions() []SRegion {
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

func (self *SQingCloudClient) GetRegion(id string) (*SRegion, error) {
	regions := self.GetRegions()
	for i := range regions {
		if regions[i].Region == id {
			regions[i].client = self
			return &regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SQingCloudClient) getUrl(service string) (string, error) {
	switch service {
	case "ec2":
		return fmt.Sprintf("https://api.qingcloud.com/iaas/"), nil
	default:
		return "", errors.Wrapf(cloudprovider.ErrNotSupported, service)
	}
}

func (cli *SQingCloudClient) getDefaultClient() *http.Client {
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

type sQingCloudError struct {
	StatusCode int    `json:"statusCode"`
	RequestId  string `json:"requestId"`
	Code       string
	Message    string
}

func (self *sQingCloudError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (self *sQingCloudError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(self)
	}
	self.StatusCode = statusCode
	return self
}

func (self *SQingCloudClient) sign(req *http.Request) (string, error) {
	keys := []string{}
	for k := range req.URL.Query() {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := []string{}
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s=%s`, url.QueryEscape(key), url.QueryEscape(req.URL.Query().Get(key))))
	}

	signStr := fmt.Sprintf("%s\n%s\n%s", req.Method, req.URL.Path, strings.Join(parts, "&"))
	hashed := hmac.New(sha256.New, []byte(self.accessKeySecret))
	hashed.Write([]byte(signStr))
	return base64.StdEncoding.EncodeToString(hashed.Sum(nil)), nil
}

func (self *SQingCloudClient) Do(req *http.Request) (*http.Response, error) {
	client := self.getDefaultClient()

	signature, err := self.sign(req)
	if err != nil {
		return nil, errors.Wrapf(err, "sign")
	}

	req.URL.RawQuery += fmt.Sprintf("&signature=%s", url.QueryEscape(signature))
	return client.Do(req)
}

func (self *SQingCloudClient) ec2Request(action, regionId string, params map[string]string) (jsonutils.JSONObject, error) {
	return self.request("ec2", action, regionId, params)
}

func (self *SQingCloudClient) request(service, action, regionId string, params map[string]string) (jsonutils.JSONObject, error) {
	uri, err := self.getUrl(service)
	if err != nil {
		return nil, err
	}
	if len(regionId) == 0 {
		regionId = QINGCLOUD_DEFAULT_REGION
	}
	if params == nil {
		params = map[string]string{}
	}
	params["action"] = action
	if regionId == "ap2" {
		regionId = "ap2a"
	}
	params["zone"] = regionId
	params["time_stamp"] = time.Now().Format(ISO8601)
	params["access_key_id"] = self.accessKeyId
	params["version"] = "1"
	params["signature_method"] = "HmacSHA256"
	params["signature_version"] = "1"
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	uri = fmt.Sprintf("%s?%s", uri, values.Encode())
	req := httputils.NewJsonRequest(httputils.GET, uri, nil)
	bErr := &sQingCloudError{}
	client := httputils.NewJsonClient(self)
	_, resp, err := client.Send(self.ctx, req, bErr, self.debug)
	if err != nil {
		return nil, err
	}
	retCode, _ := resp.Int("ret_code")
	if retCode > 0 {
		// https://docs.qingcloud.com/product/api/common/error_code.html
		if retCode == 1200 {
			return nil, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, resp.String())
		}
		return nil, errors.Errorf(resp.String())
	}
	return resp, nil
}

func (self *SQingCloudClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = self.GetAccountId()
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKeyId
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SQingCloudClient) getOwnerId() (string, error) {
	if len(self.ownerId) > 0 {
		return self.ownerId, nil
	}
	_, err := self.QueryBalance()
	return self.ownerId, err
}

func (self *SQingCloudClient) GetAccountId() string {
	ownerId, _ := self.getOwnerId()
	return ownerId
}

type Balance struct {
	Balance    float64
	RootUserId string
}

func (self *SQingCloudClient) QueryBalance() (*Balance, error) {
	resp, err := self.ec2Request("GetBalance", "", nil)
	if err != nil {
		return nil, err
	}
	ret := &Balance{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	self.ownerId = ret.RootUserId
	return ret, nil
}

func (self *SQingCloudClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}
