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

package proxmox

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_PROXMOX = api.CLOUD_PROVIDER_PROXMOX
	AUTH_ADDR              = "/access/ticket"
)

type SProxmoxClient struct {
	*ProxmoxClientConfig
}

type ProxmoxClientConfig struct {
	cpcfg    cloudprovider.ProviderConfig
	username string
	password string
	host     string
	authURL  string
	opt      string
	port     int

	csrfToken  string
	authTicket string // Combination of user, realm, token ID and UUID

	debug bool
}

func NewProxmoxClientConfig(username, password, host, opt string, port, tasktimeout int) *ProxmoxClientConfig {
	cfg := &ProxmoxClientConfig{
		username: username,
		password: password,
		host:     host,
		authURL:  fmt.Sprintf("https://%s:%s/api2/json", host, port),
		opt:      opt,
		port:     port,
	}
	return cfg
}

func (self *ProxmoxClientConfig) Debug(debug bool) *ProxmoxClientConfig {
	self.debug = debug
	return self
}

func (self *ProxmoxClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *ProxmoxClientConfig {
	self.cpcfg = cpcfg
	return self
}

func NewProxmoxClient(cfg *ProxmoxClientConfig) (*SProxmoxClient, error) {

	client := &SProxmoxClient{
		ProxmoxClientConfig: cfg,
	}

	return client, client.auth()
}

func (self *SProxmoxClient) auth() error {
	params := map[string]interface{}{
		"username": self.username,
		"password": self.password,
		"opt":      self.opt,
	}
	ret, err := self.__jsonRequest(httputils.POST, AUTH_ADDR, params)
	if err != nil {
		return errors.Wrapf(err, "post")
	}

	if ret.Contains("ticket") && ret.Contains("CSRFPreventionToken") {
		self.authTicket, err = ret.GetString("ticket")
		if err != nil {
			return errors.Wrapf(err, "get authTicket")
		}

		self.csrfToken, err = ret.GetString("CSRFPreventionToken")
		if err != nil {
			return errors.Wrapf(err, "get csrfToken")
		}

		return nil
	}

	return fmt.Errorf(ret.String())
}

func (self *SProxmoxClient) GetRegion() (*SRegion, error) {
	region := &SRegion{client: self}
	return region, nil
}

func (self *SProxmoxClient) GetRegions() ([]SRegion, error) {
	ret := []SRegion{}
	ret = append(ret, SRegion{client: self})
	return ret, nil
}

type ProxmoxError struct {
	Message string
	Code    int
	Params  []string
}

func (self ProxmoxError) Error() string {
	return fmt.Sprintf("[%d] %s with params %s", self.Code, self.Message, self.Params)
}

func (ce *ProxmoxError) ParseErrorFromJsonResponse(statusCode int, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(ce)
		log.Errorf("error: %v", body.PrettyString())
	}
	if ce.Code == 0 && statusCode > 0 {
		ce.Code = statusCode
	}
	if ce.Code == 404 || ce.Code == 400 {
		return errors.Wrap(cloudprovider.ErrNotFound, ce.Error())
	}
	return ce
}

func (cli *SProxmoxClient) getDefaultClient() *http.Client {
	client := httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(client, cli.cpcfg.ProxyFunc)
	ts, _ := client.Transport.(*http.Transport)
	ts.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response), error) {
		if cli.cpcfg.ReadOnly {
			if req.Method == "GET" || req.Method == "HEAD" {
				return nil, nil
			}
			// 认证
			if req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/access/ticket") {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	return client
}

func (cli *SProxmoxClient) post(res string, params interface{}) (jsonutils.JSONObject, error) {
	return cli._jsonRequest(httputils.POST, res, params)
}

func (cli *SProxmoxClient) get(res string, params url.Values, retVal interface{}) error {
	if params != nil {
		res = fmt.Sprintf("%s?%s", res, params.Encode())
	}
	resp, err := cli._jsonRequest(httputils.GET, res, nil)
	if err != nil {
		return err
	}
	return resp.Unmarshal(retVal)
}

func (cli *SProxmoxClient) del(res string, params url.Values, retVal interface{}) error {
	if params != nil {
		res = fmt.Sprintf("%s?%s", res, params.Encode())
	}
	resp, err := cli._jsonRequest(httputils.DELETE, res, nil)
	if err != nil {
		return err
	}
	return resp.Unmarshal(retVal)
}

func (cli *SProxmoxClient) _jsonRequest(method httputils.THttpMethod, res string, params interface{}) (jsonutils.JSONObject, error) {
	ret, err := cli.__jsonRequest(method, res, params)
	if err != nil {
		if e, ok := err.(*ProxmoxError); ok && e.Code == 401 {
			cli.auth()
			return cli.__jsonRequest(method, res, params)
		}
		return ret, err
	}
	return ret, nil
}

func (cli *SProxmoxClient) __jsonRequest(method httputils.THttpMethod, res string, params interface{}) (jsonutils.JSONObject, error) {
	client := httputils.NewJsonClient(cli.getDefaultClient())
	url := fmt.Sprintf("%s/%s", cli.authURL, strings.TrimPrefix(res, "/"))
	req := httputils.NewJsonRequest(method, url, params)
	header := http.Header{}
	if len(cli.csrfToken) > 0 && len(cli.csrfToken) > 0 && res != AUTH_ADDR {
		header.Set("Cookie", "PVEAuthCookie="+cli.authTicket)
		header.Set("Cookie", "CSRFPreventionToken"+cli.csrfToken)
	}

	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("Accept", "application/json")

	req.SetHeader(header)
	oe := &ProxmoxError{}
	_, resp, err := client.Send(context.Background(), req, oe, cli.debug)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (self *SProxmoxClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.username
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SProxmoxClient) GetAccountId() string {
	return self.host
}

func (self *SProxmoxClient) GetIRegions() []cloudprovider.ICloudRegion {
	ret := []cloudprovider.ICloudRegion{}
	region, _ := self.GetRegion()
	ret = append(ret, region)
	return ret
}

func (self *SProxmoxClient) GetCapabilities() []string {
	ret := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK + cloudprovider.READ_ONLY_SUFFIX,
	}
	return ret
}
