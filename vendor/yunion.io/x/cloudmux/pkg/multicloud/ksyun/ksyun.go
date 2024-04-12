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
	CLOUD_PROVIDER_KSYUN_CN   = "金山云"
	KSYUN_DEFAULT_REGION      = "cn-beijing-6"
	KSYUN_DEFAULT_API_VERSION = "2016-03-04"
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

func (cli *KsyunClientConfig) Debug(debug bool) *KsyunClientConfig {
	cli.debug = debug
	return cli
}

func (cli *KsyunClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *KsyunClientConfig {
	cli.cpcfg = cpcfg
	return cli
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

func (cli *SKsyunClient) GetRegions() ([]SRegion, error) {
	resp, err := cli.ec2Request("", "DescribeRegions", nil)
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
		ret.RegionSet[i].client = cli
	}
	return ret.RegionSet, nil
}

func (cli *SKsyunClient) GetRegion(id string) (*SRegion, error) {
	for i := range cli.regions {
		if cli.regions[i].GetGlobalId() == id || cli.regions[i].GetId() == id {
			cli.regions[i].client = cli
			return &cli.regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SKsyunClient) getUrl(service, regionId string) (string, error) {
	if len(regionId) == 0 {
		regionId = KSYUN_DEFAULT_REGION
	}
	switch service {
	case "kingpay", "iam", "vpc", "ebs", "eip":
		return fmt.Sprintf("http://%s.api.ksyun.com", service), nil
	case "kec", "tag":
		return fmt.Sprintf("https://%s.%s.api.ksyun.com", service, regionId), nil
	}
	return "", errors.Wrapf(cloudprovider.ErrNotSupported, "service %s", service)
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
	cli.client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
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

func (cli *sKsyunError) Error() string {
	return jsonutils.Marshal(cli).String()
}

func (cli *sKsyunError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(cli)
	}
	cli.StatusCode = statusCode
	return cli
}

func (cli *SKsyunClient) sign(req *http.Request) (string, error) {
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

	hashed := hmac.New(sha256.New, []byte(cli.accessKeySecret))
	hashed.Write(buf.Bytes())
	return hex.EncodeToString(hashed.Sum(nil)), nil
}

func (cli *SKsyunClient) Do(req *http.Request) (*http.Response, error) {
	client := cli.getDefaultClient()

	signature, err := cli.sign(req)
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

func (cli *SKsyunClient) ec2Request(regionId, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return cli.request("kec", regionId, apiName, KSYUN_DEFAULT_API_VERSION, params)
}

func (cli *SKsyunClient) iamRequest(regionId, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return cli.request("iam", regionId, apiName, "2015-11-01", params)
}

func (cli *SKsyunClient) tagRequest(regionId, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return cli.request("tag", regionId, apiName, KSYUN_DEFAULT_API_VERSION, params)
}

func (cli *SKsyunClient) eipRequest(regionId, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return cli.request("eip", regionId, apiName, KSYUN_DEFAULT_API_VERSION, params)
}

func (cli *SKsyunClient) ebsRequest(regionId, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return cli.request("ebs", regionId, apiName, KSYUN_DEFAULT_API_VERSION, params)
}

func (cli *SKsyunClient) vpcRequest(regionId, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return cli.request("vpc", regionId, apiName, KSYUN_DEFAULT_API_VERSION, params)
}

func (cli *SKsyunClient) request(service, regionId, apiName, apiVersion string, params map[string]string) (jsonutils.JSONObject, error) {
	uri, err := cli.getUrl(service, regionId)
	if err != nil {
		return nil, errors.Wrapf(err, "getUrl")
	}
	if params == nil {
		params = map[string]string{}
	}
	if len(regionId) > 0 {
		params["Region"] = regionId
	}
	params["Action"] = apiName
	params["Version"] = apiVersion
	params["Accesskey"] = cli.accessKeyId
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
	method := httputils.GET
	if !strings.HasPrefix(apiName, "Describe") && !strings.HasPrefix(apiName, "Get") {
		method = httputils.POST
	}
	req := httputils.NewJsonRequest(method, uri, nil)
	ksErr := &sKsyunError{}
	client := httputils.NewJsonClient(cli)
	_, resp, err := client.Send(cli.ctx, req, ksErr, cli.debug)
	if err != nil {
		return nil, err
	}
	if info, err := resp.GetMap(); err == nil {
		for k, v := range info {
			if strings.HasSuffix(k, "Result") {
				return v, nil
			}
		}
	}
	return resp, nil
}

func (cli *SKsyunClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = cli.GetAccountId()
	subAccount.Name = cli.cpcfg.Name
	subAccount.Account = cli.accessKeyId
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (cli *SKsyunClient) GetAccountId() string {
	if len(cli.customerId) > 0 {
		return cli.customerId
	}
	cli.QueryCashWalletAction()
	return cli.customerId
}

type CashWalletDetail struct {
	CustomerId      string
	AvailableAmount float64
	RewardAmount    string
	FrozenAmount    string
	Currency        string
}

func (cli *SKsyunClient) QueryCashWalletAction() (*CashWalletDetail, error) {
	resp, err := cli.request("kingpay", "", "QueryCashWalletAction", "V1", nil)
	if err != nil {
		return nil, err
	}
	ret := &CashWalletDetail{}
	err = resp.Unmarshal(ret, "data")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	cli.customerId = ret.CustomerId
	return ret, nil
}

func (cli *SKsyunClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_PROJECT + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
	}
	return caps
}
