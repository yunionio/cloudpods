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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
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
	port     int

	csrfToken  string
	authTicket string // Combination of user, realm, token ID and UUID

	debug bool
}

func NewProxmoxClientConfig(username, password, host string, port int) *ProxmoxClientConfig {

	cfg := &ProxmoxClientConfig{
		username: username,
		password: password,
		host:     host,
		authURL:  fmt.Sprintf("https://%s:%d/api2/json", host, port),
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
	}
	ret, err := self.__jsonRequest(httputils.POST, AUTH_ADDR, params)
	if err != nil {
		return errors.Wrapf(err, "post")
	}

	dat, err := ret.Get("data")
	if err != nil {
		return errors.Wrapf(err, "decode data")
	}

	if ticket, err := dat.GetString("ticket"); err != nil {
		return errors.Wrapf(err, "get ticket")
	} else {
		self.authTicket = ticket
	}

	if token, err := dat.GetString("CSRFPreventionToken"); err != nil {
		return errors.Wrapf(err, "get Token")
	} else {
		self.csrfToken = token
	}

	return nil
}

func (self *SProxmoxClient) GetRegion() *SRegion {
	region := &SRegion{client: self}
	return region
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
	Errors  string
	Status  string
}

func (self ProxmoxError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (ce *ProxmoxError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	ce.Status = status
	if body != nil {
		body.Unmarshal(ce)
		log.Errorf("error: %v", body.PrettyString())
	}
	if ce.Code == 0 && statusCode > 0 {
		ce.Code = statusCode
	}
	if ce.Code == 404 {
		return errors.Wrap(cloudprovider.ErrNotFound, ce.Error())
	}
	return ce
}

func (cli *SProxmoxClient) getDefaultClient() *http.Client {
	client := httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(client, cli.cpcfg.ProxyFunc)
	ts, _ := client.Transport.(*http.Transport)
	ts.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
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
	resp, err := cli._jsonRequest(httputils.POST, res, params)
	if err != nil {
		return resp, err
	}
	ret := struct {
		TaskId string
	}{}
	resp.Unmarshal(&ret)
	if len(ret.TaskId) > 0 && ret.TaskId != "null" {
		_, err = cli.waitTask(ret.TaskId)
		return resp, err
	}
	return resp, nil
}

func (cli *SProxmoxClient) put(res string, params url.Values, body jsonutils.JSONObject) error {
	if params != nil {
		res = fmt.Sprintf("%s?%s", res, params.Encode())
	}
	resp, err := cli._jsonRequest(httputils.PUT, res, body)
	if err != nil {
		return err
	}
	ret := struct {
		TaskId string
	}{}
	resp.Unmarshal(&ret)
	if len(ret.TaskId) > 0 && ret.TaskId != "null" {
		_, err = cli.waitTask(ret.TaskId)
		return err
	}
	return nil
}

func (cli *SProxmoxClient) get(res string, params url.Values, retVal interface{}) error {
	if len(params) > 0 {
		res = fmt.Sprintf("%s?%s", res, params.Encode())
	}
	resp, err := cli._jsonRequest(httputils.GET, res, nil)
	if err != nil {
		return err
	}
	dat, err := resp.Get("data")
	if err != nil {
		return errors.Wrapf(err, "decode data")
	}

	return dat.Unmarshal(retVal)
}

func (cli *SProxmoxClient) getAgent(res string, params url.Values, retVal interface{}) error {
	resp, err := cli._jsonRequest(httputils.GET, res, nil)
	if err != nil {
		return err
	}
	dat, err := resp.Get("data")
	if err != nil {
		return errors.Wrapf(err, "decode data")
	}
	ret, err := dat.Get("result")
	if err != nil {
		return errors.Wrapf(err, "decode data")
	}

	return ret.Unmarshal(retVal)
}

func (cli *SProxmoxClient) del(res string, params url.Values, retVal interface{}) error {
	if params != nil {
		res = fmt.Sprintf("%s?%s", res, params.Encode())
	}
	resp, err := cli._jsonRequest(httputils.DELETE, res, nil)
	if err != nil {
		return err
	}
	taskId, err := resp.GetString("data")
	if err != nil {
		return err
	}
	_, err = cli.waitTask(taskId)

	return err

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
		header.Set("CSRFPreventionToken", cli.csrfToken)
	}

	req.SetHeader(header)
	oe := &ProxmoxError{}
	_, resp, err := client.Send(context.Background(), req, oe, cli.debug)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (cli *SProxmoxClient) upload(node, storageName, filename string, reader io.Reader) (*SImage, error) {
	if !strings.HasSuffix(filename, ".iso") {
		filename = filename + ".iso"
	}
	filename = filepath.Base(filename)
	client := cli.getDefaultClient()
	res := fmt.Sprintf("/nodes/%s/storage/%s/upload", node, storageName)
	url := fmt.Sprintf("%s/%s", cli.authURL, strings.TrimPrefix(res, "/"))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	err := writer.WriteField("content", "iso")
	if err != nil {
		return nil, err
	}
	part, err := writer.CreateFormFile("filename", filename)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, reader)
	if err != nil {
		return nil, errors.Wrapf(err, "io.Copy")
	}
	writer.Close()
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	if len(cli.csrfToken) > 0 && len(cli.csrfToken) > 0 && res != AUTH_ADDR {
		req.Header.Set("Cookie", "PVEAuthCookie="+cli.authTicket)
		req.Header.Set("CSRFPreventionToken", cli.csrfToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	obj, err := jsonutils.Parse(data)
	if err != nil {
		return nil, err
	}
	if obj.Contains("errors") {
		return nil, fmt.Errorf(string(data))
	}

	now := time.Now()
	for now.Sub(time.Now()) < time.Minute*1 {
		images, err := cli.GetImages(node, storageName)
		if err != nil {
			return nil, errors.Wrapf(err, "GetImageStatus")
		}
		for i := range images {
			if strings.HasSuffix(images[i].Volid, filename) {
				return &images[i], nil
			}
		}
		time.Sleep(time.Second * 10)
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after upload")
}

func (self *SProxmoxClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = self.host
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
	region := self.GetRegion()
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
