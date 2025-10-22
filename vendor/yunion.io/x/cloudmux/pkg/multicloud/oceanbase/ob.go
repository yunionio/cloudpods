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

package oceanbase

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/icholy/digest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	OB_DEFAULT_REGION_NAME = "OceanBase Cloud"
)

type OceanBaseClientConfig struct {
	cpcfg           cloudprovider.ProviderConfig
	accessKeyId     string
	accessKeySecret string

	debug bool
}

type SOceanBaseClient struct {
	*OceanBaseClientConfig

	client *http.Client
	lock   sync.Mutex
	ctx    context.Context
}

func NewOceanBaseClientConfig(accessKeyId, accessKeySecret string) *OceanBaseClientConfig {
	cfg := &OceanBaseClientConfig{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
	}
	return cfg
}

func (cfg *OceanBaseClientConfig) Debug(debug bool) *OceanBaseClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg *OceanBaseClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *OceanBaseClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func NewOceanBaseClient(cfg *OceanBaseClientConfig) (*SOceanBaseClient, error) {
	client := &SOceanBaseClient{
		OceanBaseClientConfig: cfg,
		ctx:                   context.Background(),
	}
	client.ctx = context.WithValue(client.ctx, "time", time.Now())
	_, err := client.list("/api/v2/instances", nil)
	return client, err
}

func (cli *SOceanBaseClient) GetRegion() *SRegion {
	return &SRegion{
		client: cli,
	}
}

func (cli *SOceanBaseClient) getDefaultClient() *http.Client {
	cli.lock.Lock()
	defer cli.lock.Unlock()
	if !gotypes.IsNil(cli.client) {
		return cli.client
	}
	cli.client = httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(cli.client, cli.cpcfg.ProxyFunc)
	ts, _ := cli.client.Transport.(*http.Transport)
	ts.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	t := &digest.Transport{
		Username: cli.accessKeyId,
		Password: cli.accessKeySecret,
		Transport: cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
			if cli.cpcfg.ReadOnly {
				if req.Method == "GET" {
					return nil, nil
				}
				return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
			}
			return nil, nil
		}),
	}
	cli.client.Transport = t
	return cli.client
}

type sObError struct {
	StatusCode int `json:"statusCode"`
	method     httputils.THttpMethod
	url        string
	body       jsonutils.JSONObject
}

func (e *sObError) Error() string {
	return jsonutils.Marshal(e).String()
}

func (e *sObError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(e)
	}
	e.StatusCode = statusCode
	log.Infof("%s %s body: %s error: %v", e.method, e.url, e.body, e.Error())
	if e.StatusCode == 404 {
		return errors.Wrapf(cloudprovider.ErrNotFound, "%s", e.Error())
	}
	return e
}

func (cli *SOceanBaseClient) Do(req *http.Request) (*http.Response, error) {
	client := cli.getDefaultClient()

	return client.Do(req)
}

func (cli *SOceanBaseClient) list(resource string, params url.Values) (jsonutils.JSONObject, error) {
	return cli.request(httputils.GET, resource, params, nil)
}

func (cli *SOceanBaseClient) delete(resource string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return cli.request(httputils.DELETE, resource, nil, body)
}

func (cli *SOceanBaseClient) put(resource string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return cli.request(httputils.PUT, resource, nil, body)
}

func (cli *SOceanBaseClient) request(method httputils.THttpMethod, resource string, params url.Values, body map[string]interface{}) (jsonutils.JSONObject, error) {
	if body == nil {
		body = map[string]interface{}{}
	}
	uri := fmt.Sprintf("https://api-cloud-cn.oceanbase.com/%s", strings.TrimPrefix(resource, "/"))
	if len(params) > 0 {
		uri = fmt.Sprintf("%s?%s", uri, params.Encode())
	}
	req := httputils.NewJsonRequest(method, uri, body)
	bErr := &sObError{method: method, url: uri, body: jsonutils.Marshal(body)}
	client := httputils.NewJsonClient(cli)
	_, resp, err := client.Send(cli.ctx, req, bErr, cli.debug)
	if err != nil {
		return nil, err
	}
	if !jsonutils.QueryBoolean(resp, "success", true) {
		return nil, fmt.Errorf("request failed: %s", resp.String())
	}
	return resp, nil
}

func (cli *SOceanBaseClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = cli.GetAccountId()
	subAccount.Name = cli.cpcfg.Name
	subAccount.Account = cli.accessKeyId
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (cli *SOceanBaseClient) GetAccountId() string {
	return ""
}

func (cli *SOceanBaseClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_RDS,
	}
	return caps
}
