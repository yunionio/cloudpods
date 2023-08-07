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

package ksyun

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
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
	CLOUD_PROVIDER_KSYUN_CN = "金山云"
	KSYUN_DEFAULT_REGION    = "cn-beijing-6"
)

type KsyunClientConfig struct {
	cpcfg           cloudprovider.ProviderConfig
	accessKeyId     string
	accessKeySecret string

	debug bool
}

type SKsyunClient struct {
	*KsyunClientConfig

	client *http.Client
	lock   sync.Mutex
	ctx    context.Context

	customerId string

	regions []SRegion
}

func NewKsyunClientConfig(accessKeyId, accessKeySecret string) *KsyunClientConfig {
	cfg := &KsyunClientConfig{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
	}
	return cfg
}

func (self *KsyunClientConfig) Debug(debug bool) *KsyunClientConfig {
	self.debug = debug
	return self
}

func (self *KsyunClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *KsyunClientConfig {
	self.cpcfg = cpcfg
	return self
}

func NewKsyunClient(cfg *KsyunClientConfig) (*SKsyunClient, error) {
	client := &SKsyunClient{
		KsyunClientConfig: cfg,
		ctx:               context.Background(),
	}
	client.ctx = context.WithValue(client.ctx, "time", time.Now())
	var err error
	client.regions, err = client.GetRegions()
	return client, err
}

func (self *SKsyunClient) GetRegions() ([]SRegion, error) {
	resp, err := self.ec2Request("", "DescribeRegions", nil)
	if err != nil {
		return nil, err
	}
	ret := struct {
		RegionSet []SRegion
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	for i := range ret.RegionSet {
		ret.RegionSet[i].client = self
	}
	return ret.RegionSet, nil
}

func (self *SKsyunClient) GetRegion(id string) (*SRegion, error) {
	for i := range self.regions {
		if self.regions[i].Region == id {
			self.regions[i].client = self
			return &self.regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SKsyunClient) getUrl(service, regionId string) string {
	if len(regionId) == 0 {
		regionId = KSYUN_DEFAULT_REGION
	}
	switch service {
	case "kingpay":
		return "http://kingpay.api.ksyun.com"
	case "kec":
		return fmt.Sprintf("https://kec.%s.api.ksyun.com", regionId)
	}
	return ""
}

func (cli *SKsyunClient) getDefaultClient() *http.Client {
	cli.lock.Lock()
	defer cli.lock.Unlock()
	if !gotypes.IsNil(cli.client) {
		return cli.client
	}
	cli.client = httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(cli.client, cli.cpcfg.ProxyFunc)
	ts, _ := cli.client.Transport.(*http.Transport)
	ts.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	cli.client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response), error) {
		params, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			return nil, errors.Wrapf(err, "ParseQuery(%s)", req.URL.RawQuery)
		}
		action := params.Get("Action")

		for _, prefix := range []string{"Get", "List", "Describe", "Query"} {
			if strings.HasPrefix(action, prefix) {
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

type sKsyunError struct {
	StatusCode int    `json:"StatusCode"`
	RequestId  string `json:"RequestId"`
	ErrorMsg   struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
		Type    string `json:"Type"`
	} `json:"Error"`
}

func (self *sKsyunError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (self *sKsyunError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(self)
	}
	self.StatusCode = statusCode
	return self
}

func (self *SKsyunClient) sign(req *http.Request) (string, error) {
	query, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return "", err
	}
	keys := []string{}
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer

	for i := range keys {
		k := keys[i]
		buf.WriteString(strings.Replace(url.QueryEscape(k), "+", "%20", -1))
		buf.WriteString("=")
		buf.WriteString(strings.Replace(url.QueryEscape(query.Get(k)), "+", "%20", -1))
		buf.WriteString("&")
	}
	buf.Truncate(buf.Len() - 1)

	hashed := hmac.New(sha256.New, []byte(self.accessKeySecret))
	hashed.Write([]byte(buf.String()))
	return hex.EncodeToString(hashed.Sum(nil)), nil
}

func (self *SKsyunClient) Do(req *http.Request) (*http.Response, error) {
	client := self.getDefaultClient()

	signature, err := self.sign(req)
	if err != nil {
		return nil, errors.Wrapf(err, "sign")
	}

	query, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	query.Set("Signature", signature)
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Accept", "application/json")
	return client.Do(req)
}

func (self *SKsyunClient) ec2Request(regionId, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return self.request("kec", regionId, apiName, "2016-03-04", params)
}

func (self *SKsyunClient) request(service, regionId, apiName, apiVersion string, params map[string]string) (jsonutils.JSONObject, error) {
	uri := self.getUrl(service, regionId)
	if params == nil {
		params = map[string]string{}
	}
	params["Action"] = apiName
	params["Version"] = apiVersion
	params["Accesskey"] = self.accessKeyId
	params["SignatureMethod"] = "HMAC-SHA256"
	params["Service"] = service
	params["Format"] = "json"
	params["SignatureVersion"] = "1.0"
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	uri = fmt.Sprintf("%s?%s", uri, values.Encode())
	req := httputils.NewJsonRequest(httputils.GET, uri, nil)
	ksErr := &sKsyunError{}
	client := httputils.NewJsonClient(self)
	_, resp, err := client.Send(self.ctx, req, ksErr, self.debug)
	return resp, err
}

func (self *SKsyunClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKeyId
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SKsyunClient) GetAccountId() string {
	if len(self.customerId) > 0 {
		return self.customerId
	}
	self.QueryCashWalletAction()
	return self.customerId
}

type CashWalletDetail struct {
	CustomerId      string
	AvailableAmount float64
	RewardAmount    string
	FrozenAmount    string
	Currency        string
}

func (self *SKsyunClient) QueryCashWalletAction() (*CashWalletDetail, error) {
	resp, err := self.request("kingpay", "", "QueryCashWalletAction", "V1", nil)
	if err != nil {
		return nil, err
	}
	ret := &CashWalletDetail{}
	err = resp.Unmarshal(ret, "data")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	self.customerId = ret.CustomerId
	return ret, nil
}

func (self *SKsyunClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}
