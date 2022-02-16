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

package bingocloud

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	xj "github.com/basgys/goxml2json"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_BINGO_CLOUD = api.CLOUD_PROVIDER_BINGO_CLOUD
)

type BingoCloudConfig struct {
	cpcfg     cloudprovider.ProviderConfig
	endpoint  string
	accessKey string
	secretKey string

	debug bool
}

func NewBingoCloudClientConfig(endpoint, accessKey, secretKey string) *BingoCloudConfig {
	cfg := &BingoCloudConfig{
		endpoint:  endpoint,
		accessKey: accessKey,
		secretKey: secretKey,
	}
	return cfg
}

func (cfg *BingoCloudConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *BingoCloudConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *BingoCloudConfig) Debug(debug bool) *BingoCloudConfig {
	cfg.debug = debug
	return cfg
}

type SBingoCloudClient struct {
	*BingoCloudConfig

	regions []SRegion
}

func NewBingoCloudClient(cfg *BingoCloudConfig) (*SBingoCloudClient, error) {
	client := &SBingoCloudClient{BingoCloudConfig: cfg}
	var err error
	client.regions, err = client.GetRegions()
	if err != nil {
		return nil, err
	}
	for i := range client.regions {
		client.regions[i].client = client
	}
	return client, nil
}

func (self *SBingoCloudClient) GetAccountId() string {
	return self.endpoint
}

func (self *SBingoCloudClient) GetRegion(id string) (*SRegion, error) {
	for i := range self.regions {
		if self.regions[i].RegionId == id {
			return &self.regions[i], nil
		}
	}
	if len(id) == 0 {
		return &self.regions[0], nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SBingoCloudClient) getDefaultClient(timeout time.Duration) *http.Client {
	client := httputils.GetDefaultClient()
	if timeout > 0 {
		client = httputils.GetTimeoutClient(timeout)
	}
	proxy := func(req *http.Request) (*url.URL, error) {
		if cli.cpcfg.ProxyFunc != nil {
			cli.cpcfg.ProxyFunc(req)
		}
		return nil, nil
	}
	httputils.SetClientProxyFunc(client, proxy)
	return client
}

func (self *SBingoCloudClient) sign(query string) string {
	uri, _ := url.Parse(self.endpoint)
	items := strings.Split(query, "&")
	sort.Slice(items, func(i, j int) bool {
		x0, y0 := strings.Split(items[i], "=")[0], strings.Split(items[j], "=")[0]
		return x0 < y0
	})
	path := "/"
	if len(uri.Path) > 0 {
		path = uri.Path
	}
	stringToSign := fmt.Sprintf("POST\n%s\n%s\n", uri.Host, path) + strings.Join(items, "&")
	hmac := hmac.New(sha256.New, []byte(self.secretKey))
	hmac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(hmac.Sum(nil))
}

func (self *SBingoCloudClient) invoke(action string, params map[string]string) (jsonutils.JSONObject, error) {
	var encode = func(k, v string) string {
		d := url.Values{}
		d.Set(k, v)
		return d.Encode()
	}
	query := encode("Action", action)
	for k, v := range params {
		query += "&" + encode(k, v)
	}
	// 2022-02-11T03:57:37.000Z
	timeStamp := time.Now().Format("2006-01-02T15:04:05.000Z")
	query += "&" + encode("Timestamp", timeStamp)
	query += "&" + encode("AWSAccessKeyId", self.accessKey)
	query += "&" + encode("Version", "2009-08-15")
	query += "&" + encode("SignatureVersion", "2")
	query += "&" + encode("SignatureMethod", "HmacSHA256")
	query += "&" + encode("Signature", self.sign(query))
	client := self.getDefaultClient(0)
	resp, err := httputils.Request(client, context.Background(), httputils.POST, self.endpoint, nil, strings.NewReader(query), self.debug)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := xj.Convert(resp.Body)
	if err != nil {
		return nil, err
	}

	if self.debug {
		log.Debugf("response: %s", result.String())
	}

	obj, err := jsonutils.Parse([]byte(result.String()))
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Parse")
	}

	respKey := action + "Response"
	if obj.Contains(respKey) {
		obj, err = obj.Get(respKey)
		if err != nil {
			return nil, err
		}
	}

	// 处理请求单个资源情况
	if strings.HasPrefix(action, "Describe") && strings.HasSuffix(action, "s") {
		objDict := obj.(*jsonutils.JSONDict)
		for k, v := range objDict.Value() {
			if (strings.HasSuffix(k, "Set") || k == "regionInfo") && v.Contains("item") {
				value := v.(*jsonutils.JSONDict)
				item, _ := v.Get("item")
				_, ok := item.(*jsonutils.JSONArray)
				if !ok {
					value.Set("item", jsonutils.NewArray(item))
					objDict.Set(k, value)
				}
			}
		}
	}

	return obj, nil
}

func (self *SBingoCloudClient) GetCapabilities() []string {
	return []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
	}
}

///////////

type sBingoError struct {
	DetailedMessage string
	Message         string
	ErrorCode       struct {
		Code    int
		HelpUrl string
	}
}

func (self *sBingoError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (self *sBingoError) ParseErrorFromJsonResponse(statusCode int, body jsonutils.JSONObject) error {
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
	ne := &sBingoError{}
	_, resp, err := client.Send(context.Background(), req, ne, debug)
	return resp, err
}

func (self *SBingoCloudClient) jsonRequest(method httputils.THttpMethod, url string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	client := self.getDefaultClient(time.Duration(0))
	return _jsonRequest(client, method, url, nil, body, self.debug)
}

func (self *SBingoCloudClient) _list(res string, params url.Values) (jsonutils.JSONObject, error) {
	// url := fmt.Sprintf("%s/%s", self.getBaseDomain(), res)
	// return fmt.Sprintf("https://%s:%d/api/nutanix/%s", self.host, self.port, version)

	url := "http://10.1.33.25:8663/main.yaws"

	if len(params) > 0 {
		url = fmt.Sprintf("%s?%s", url, params.Encode())
	}
	return self.jsonRequest(httputils.GET, url, nil)
}

func (self *SBingoCloudClient) list(res string, params url.Values, retVal interface{}) (int, error) {
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

func (self *SBingoCloudClient) get(res string, id string, params url.Values, retVal interface{}) error {
	// url := fmt.Sprintf("%s/%s/%s", self.getBaseDomain(), res, id)
	url := "http://10.1.33.25:8663/main.yaws"
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

func (self *SBingoCloudClient) listAll(res string, params url.Values, retVal interface{}) error {
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

func (self *SBingoCloudClient) _post(res string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// url := fmt.Sprintf("%s/%s", self.getBaseDomain(), res)
	url := "http://10.1.33.25:8663/main.yaws"

	if body == nil {
		body = jsonutils.NewDict()
	}
	return self.jsonRequest(httputils.POST, url, body)
}

func (self *SBingoCloudClient) post(res string, body jsonutils.JSONObject, retVal interface{}) error {
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

func (self *SBingoCloudClient) _delete(res, id string) (jsonutils.JSONObject, error) {
	// url := fmt.Sprintf("%s/%s/%s", self.getBaseDomain(), res, id)
	url := "http://10.1.33.25:8663/main.yaws"

	return self.jsonRequest(httputils.DELETE, url, nil)
}

func (self *SBingoCloudClient) wait(taskId string) (string, error) {
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

func (self *SBingoCloudClient) getTask(id string) (*STask, error) {
	task := &STask{}
	return task, self.get("tasks", id, nil, task)
}

func (self *SBingoCloudClient) delete(res, id string) error {
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

func (self *SBingoCloudClient) update(res, id string, body jsonutils.JSONObject, retVal interface{}) error {
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

func (self *SBingoCloudClient) _update(res, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// url := fmt.Sprintf("%s/%s/%s", self.getBaseDomain(), res, id)
	url := "http://10.1.33.25:8663/main.yaws"

	if body == nil {
		body = jsonutils.NewDict()
	}
	return self.jsonRequest(httputils.PUT, url, body)
}

func (self *SBingoCloudClient) _upload(res, id string, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	// url := fmt.Sprintf("%s/%s/%s", self.getBaseDomainV0_8(), res, id)
	url := "http://10.1.33.25:8663/main.yaws"

	return self.rawRequest(httputils.PUT, url, header, body)
}

func (self *SBingoCloudClient) rawRequest(method httputils.THttpMethod, url string, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	client := self.getDefaultClient(time.Hour * 5)
	_resp, err := _rawRequest(client, method, url, header, body, false)
	_, resp, err := httputils.ParseJSONResponse("", _resp, err, self.debug)
	return resp, err
}

func _rawRequest(cli *http.Client, method httputils.THttpMethod, url string, header http.Header, body io.Reader, debug bool) (*http.Response, error) {
	return httputils.Request(cli, context.Background(), method, url, header, body, debug)
}

func (self *SBingoCloudClient) upload(res, id string, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	return self._upload(res, id, header, body)
}
