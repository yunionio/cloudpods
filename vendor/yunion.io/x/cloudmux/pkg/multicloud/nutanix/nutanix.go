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

package nutanix

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	NUTANIX_VERSION_V2   = "v2.0"
	NUTANIX_VERSION_V0_8 = "v0.8"
	NUTANIX_VERSION_V3   = "v3"

	CLOUD_PROVIDER_NUTANIX = api.CLOUD_PROVIDER_NUTANIX
)

type NutanixClientConfig struct {
	cpcfg    cloudprovider.ProviderConfig
	username string
	password string
	host     string
	port     int
	debug    bool
}

func NewNutanixClientConfig(host, username, password string, port int) *NutanixClientConfig {
	cfg := &NutanixClientConfig{
		host:     host,
		username: username,
		password: password,
		port:     port,
	}
	return cfg
}

func (cfg *NutanixClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *NutanixClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *NutanixClientConfig) Debug(debug bool) *NutanixClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg NutanixClientConfig) Copy() NutanixClientConfig {
	return cfg
}

type SNutanixClient struct {
	*NutanixClientConfig
}

func NewNutanixClient(cfg *NutanixClientConfig) (*SNutanixClient, error) {
	client := &SNutanixClient{
		NutanixClientConfig: cfg,
	}
	return client, client.auth()
}

func (self *SNutanixClient) GetRegion() (*SRegion, error) {
	return &SRegion{cli: self}, nil
}

func (self *SNutanixClient) GetAccountId() string {
	return self.host
}

func (self *SNutanixClient) GetCapabilities() []string {
	return []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
	}
}

func (self *SNutanixClient) auth() error {
	_, err := self.list("clusters", nil, nil)
	return err
}

func (self *SNutanixClient) _getBaseDomain(version string) string {
	if len(version) == 0 {
		version = NUTANIX_VERSION_V2
	}
	return fmt.Sprintf("https://%s:%d/api/nutanix/%s", self.host, self.port, version)
}

func (self *SNutanixClient) getBaseDomain() string {
	return self._getBaseDomain("")
}

func (self *SNutanixClient) getBaseDomainV0_8() string {
	return self._getBaseDomain(NUTANIX_VERSION_V0_8)
}

func (cli *SNutanixClient) getDefaultClient(timeout time.Duration) *http.Client {
	client := httputils.GetDefaultClient()
	if timeout > 0 {
		client = httputils.GetTimeoutClient(timeout)
	}
	proxy := func(req *http.Request) (*url.URL, error) {
		req.SetBasicAuth(cli.username, cli.password)
		if cli.cpcfg.ProxyFunc != nil {
			cli.cpcfg.ProxyFunc(req)
		}
		return nil, nil
	}
	httputils.SetClientProxyFunc(client, proxy)

	ts, _ := client.Transport.(*http.Transport)
	client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		if cli.cpcfg.ReadOnly {
			if req.Method == "GET" {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})

	return client
}

func (self *SNutanixClient) _list(res string, params url.Values) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/%s", self.getBaseDomain(), res)
	if len(params) > 0 {
		url = fmt.Sprintf("%s?%s", url, params.Encode())
	}
	return self.jsonRequest(httputils.GET, url, nil)
}

func (self *SNutanixClient) _post(res string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/%s", self.getBaseDomain(), res)
	if body == nil {
		body = jsonutils.NewDict()
	}
	return self.jsonRequest(httputils.POST, url, body)
}

func (self *SNutanixClient) _update(res, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/%s/%s", self.getBaseDomain(), res, id)
	if body == nil {
		body = jsonutils.NewDict()
	}
	return self.jsonRequest(httputils.PUT, url, body)
}

func (self *SNutanixClient) _upload(res, id string, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/%s/%s", self.getBaseDomainV0_8(), res, id)
	return self.rawRequest(httputils.PUT, url, header, body)
}

func (self *SNutanixClient) upload(res, id string, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	return self._upload(res, id, header, body)
}

func (self *SNutanixClient) _delete(res, id string) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/%s/%s", self.getBaseDomain(), res, id)
	return self.jsonRequest(httputils.DELETE, url, nil)
}

func (self *SNutanixClient) list(res string, params url.Values, retVal interface{}) (int, error) {
	resp, err := self._list(res, params)
	if err != nil {
		return 0, errors.Wrapf(err, "get %s", res)
	}
	if retVal != nil {
		err = resp.Unmarshal(retVal, "entities")
		if err != nil {
			return 0, errors.Wrapf(err, "resp.Unmarshal")
		}
	}
	total, err := resp.Int("metadata", "total_entities")
	if err != nil {
		return 0, errors.Wrapf(err, "get metadata total_entities")
	}
	return int(total), nil
}

func (self *SNutanixClient) delete(res, id string) error {
	resp, err := self._delete(res, id)
	if err != nil {
		return errors.Wrapf(err, "delete %s", res)
	}
	if resp != nil && resp.Contains("task_uuid") {
		task := struct {
			TaskUUID string
		}{}
		resp.Unmarshal(&task)
		if len(task.TaskUUID) > 0 {
			_, err = self.wait(task.TaskUUID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SNutanixClient) wait(taskId string) (string, error) {
	resId := ""
	err := cloudprovider.Wait(time.Second*5, time.Minute*10, func() (bool, error) {
		task, err := self.getTask(taskId)
		if err != nil {
			return false, err
		}
		for _, entity := range task.EntityList {
			if len(entity.EntityID) > 0 {
				resId = entity.EntityID
			}
		}
		log.Debugf("task %s %s status: %s", task.OperationType, task.UUID, task.ProgressStatus)
		if task.ProgressStatus == "Succeeded" {
			return true, nil
		}
		if task.ProgressStatus == "Failed" {
			return false, errors.Errorf(jsonutils.Marshal(task.MetaResponse).String())
		}
		return false, nil
	})
	return resId, errors.Wrapf(err, "wait task %s", taskId)
}

func (self *SNutanixClient) update(res, id string, body jsonutils.JSONObject, retVal interface{}) error {
	resp, err := self._update(res, id, body)
	if err != nil {
		return errors.Wrapf(err, "update %s/%s %v", res, id, body)
	}
	task := struct {
		TaskUUID string
	}{}
	resp.Unmarshal(&task)
	if len(task.TaskUUID) > 0 {
		_, err = self.wait(task.TaskUUID)
		if err != nil {
			return err
		}
	}
	if retVal != nil {
		return resp.Unmarshal(retVal)
	}
	return nil
}

func (self *SNutanixClient) post(res string, body jsonutils.JSONObject, retVal interface{}) error {
	resp, err := self._post(res, body)
	if err != nil {
		return errors.Wrapf(err, "post %s %v", res, body)
	}
	if retVal != nil {
		if resp.Contains("entities") {
			err = resp.Unmarshal(retVal, "entities")
		} else {
			err = resp.Unmarshal(retVal)
		}
		return err
	}
	return nil
}

func (self *SNutanixClient) listAll(res string, params url.Values, retVal interface{}) error {
	if len(params) == 0 {
		params = url.Values{}
	}
	entities := []jsonutils.JSONObject{}
	page, count := 1, 1024
	for {
		params.Set("count", fmt.Sprintf("%d", count))
		params.Set("page", fmt.Sprintf("%d", page))
		resp, err := self._list(res, params)
		if err != nil {
			return errors.Wrapf(err, "list %s", res)
		}
		_entities, err := resp.GetArray("entities")
		if err != nil {
			return errors.Wrapf(err, "resp get entities")
		}
		entities = append(entities, _entities...)
		totalEntities, err := resp.Int("metadata", "total_entities")
		if err != nil {
			return errors.Wrapf(err, "get resp total_entities")
		}
		if int64(page*count) >= totalEntities {
			break
		}
		page++
	}
	return jsonutils.Update(retVal, entities)
}

func (self *SNutanixClient) get(res string, id string, params url.Values, retVal interface{}) error {
	url := fmt.Sprintf("%s/%s/%s", self.getBaseDomain(), res, id)
	if len(params) > 0 {
		url = fmt.Sprintf("%s?%s", url, params.Encode())
	}
	resp, err := self.jsonRequest(httputils.GET, url, nil)
	if err != nil {
		return errors.Wrapf(err, "get %s/%s", res, id)
	}
	if retVal != nil {
		return resp.Unmarshal(retVal)
	}
	return nil
}

func (self *SNutanixClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{
		Id:           self.GetAccountId(),
		Account:      self.username,
		Name:         self.cpcfg.Name,
		HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SNutanixClient) jsonRequest(method httputils.THttpMethod, url string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	client := self.getDefaultClient(time.Duration(0))
	return _jsonRequest(client, method, url, nil, body, self.debug)
}

type sNutanixError struct {
	DetailedMessage string
	Message         string
	ErrorCode       struct {
		Code    int
		HelpUrl string
	}
}

func (self *sNutanixError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (self *sNutanixError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(self)
	}
	if self.ErrorCode.Code == 1202 {
		return errors.Wrapf(cloudprovider.ErrNotFound, self.Error())
	}
	return self
}

func _jsonRequest(cli *http.Client, method httputils.THttpMethod, url string, header http.Header, body jsonutils.JSONObject, debug bool) (jsonutils.JSONObject, error) {
	client := httputils.NewJsonClient(cli)
	req := httputils.NewJsonRequest(method, url, body)
	ne := &sNutanixError{}
	_, resp, err := client.Send(context.Background(), req, ne, debug)
	return resp, err
}

func (self *SNutanixClient) rawRequest(method httputils.THttpMethod, url string, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	client := self.getDefaultClient(time.Hour * 5)
	_resp, err := _rawRequest(client, method, url, header, body, false)
	_, resp, err := httputils.ParseJSONResponse("", _resp, err, self.debug)
	return resp, err
}

func _rawRequest(cli *http.Client, method httputils.THttpMethod, url string, header http.Header, body io.Reader, debug bool) (*http.Response, error) {
	return httputils.Request(cli, context.Background(), method, url, header, body, debug)
}

func (self *SNutanixClient) GetIRegions() []cloudprovider.ICloudRegion {
	region := &SRegion{cli: self}
	return []cloudprovider.ICloudRegion{region}
}

func (self *SNutanixClient) getTask(id string) (*STask, error) {
	task := &STask{}
	return task, self.get("tasks", id, nil, task)
}
