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

package zstack

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	CLOUD_PROVIDER_ZSTACK = api.CLOUD_PROVIDER_ZSTACK
	ZSTACK_DEFAULT_REGION = "ZStack"
	ZSTACK_API_VERSION    = "v1"
)

var (
	SkipEsxi bool = true
)

type ZstackClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	authURL  string
	username string
	password string

	debug bool
}

func NewZstackClientConfig(authURL, username, password string) *ZstackClientConfig {
	cfg := &ZstackClientConfig{
		authURL:  strings.TrimSuffix(authURL, "/"),
		username: username,
		password: password,
	}
	return cfg
}

func (cfg *ZstackClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *ZstackClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *ZstackClientConfig) Debug(debug bool) *ZstackClientConfig {
	cfg.debug = debug
	return cfg
}

type SZStackClient struct {
	*ZstackClientConfig

	httpClient *http.Client

	iregions []cloudprovider.ICloudRegion
}

func getTime() string {
	zone, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(zone).Format("Mon, 02 Jan 2006 15:04:05 MST")
}

func sign(accessId, accessKey, method, date, url string) string {
	h := hmac.New(sha1.New, []byte(accessKey))
	h.Write([]byte(fmt.Sprintf("%s\n%s\n%s", method, date, url)))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func getSignUrl(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(u.Path, "/zstack"), nil
}

func NewZStackClient(cfg *ZstackClientConfig) (*SZStackClient, error) {
	httpClient := cfg.cpcfg.AdaptiveTimeoutHttpClient()
	ts, _ := httpClient.Transport.(*http.Transport)
	httpClient.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		if cfg.cpcfg.ReadOnly {
			if req.Method == "GET" || req.Method == "HEAD" {
				return nil, nil
			}
			// 认证
			if req.Method == "PUT" && req.URL.Path == "/zstack/v1/accounts/login" {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})

	cli := &SZStackClient{
		ZstackClientConfig: cfg,
		httpClient:         httpClient,
	}
	if err := cli.connect(); err != nil {
		return nil, err
	}
	cli.iregions = []cloudprovider.ICloudRegion{&SRegion{client: cli, Name: ZSTACK_DEFAULT_REGION}}
	return cli, nil
}

func (cli *SZStackClient) GetCloudRegionExternalIdPrefix() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_ZSTACK, cli.cpcfg.Id)
}

func (cli *SZStackClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{
		Id:           cli.cpcfg.Id,
		Account:      cli.username,
		Name:         cli.cpcfg.Name,
		HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (cli *SZStackClient) GetIRegions() []cloudprovider.ICloudRegion {
	return cli.iregions
}

func (cli *SZStackClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(cli.iregions); i++ {
		if cli.iregions[i].GetGlobalId() == id {
			return cli.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SZStackClient) getRequestURL(resource string, params url.Values) string {
	return cli.authURL + fmt.Sprintf("/zstack/%s/%s", ZSTACK_API_VERSION, resource) + "?" + params.Encode()
}

func (cli *SZStackClient) testAccessKey() error {
	zones := []SZone{}
	err := cli.listAll("zones", url.Values{}, &zones)
	if err != nil {
		return errors.Wrap(err, "testAccessKey")
	}
	return nil
}

func (cli *SZStackClient) connect() error {
	header := http.Header{}
	header.Add("Content-Type", "application/json")
	authURL := cli.authURL + "/zstack/v1/accounts/login"
	body := jsonutils.Marshal(map[string]interface{}{
		"logInByAccount": map[string]string{
			"accountName": cli.username,
			"password":    fmt.Sprintf("%x", sha512.Sum512([]byte(cli.password))),
		},
	})
	_, _, err := httputils.JSONRequest(cli.httpClient, context.Background(), "PUT", authURL, header, body, cli.debug)
	if err != nil {
		err = cli.testAccessKey()
		if err == nil {
			return nil
		}
		return errors.Wrapf(err, "connect")
	}
	return fmt.Errorf("password auth has been deprecated, please using ak sk auth")
}

func (cli *SZStackClient) listAll(resource string, params url.Values, retVal interface{}) error {
	result := []jsonutils.JSONObject{}
	start, limit := 0, 50
	for {
		resp, err := cli._list(resource, start, limit, params)
		if err != nil {
			return err
		}
		objs, err := resp.GetArray("inventories")
		if err != nil {
			return err
		}
		result = append(result, objs...)
		if start+limit > len(result) {
			inventories := jsonutils.Marshal(map[string][]jsonutils.JSONObject{"inventories": result})
			return inventories.Unmarshal(retVal, "inventories")
		}
		start += limit
	}
}

func (cli *SZStackClient) sign(uri, method string, header http.Header) error {
	url, err := getSignUrl(uri)
	if err != nil {
		return errors.Wrap(err, "sign.getSignUrl")
	}
	date := getTime()
	signature := sign(cli.username, cli.password, method, date, url)
	header.Add("Signature", signature)
	header.Add("Authorization", fmt.Sprintf("ZStack %s:%s", cli.username, signature))
	header.Add("Date", date)
	return nil
}

func (cli *SZStackClient) _list(resource string, start int, limit int, params url.Values) (jsonutils.JSONObject, error) {
	header := http.Header{}
	if params == nil {
		params = url.Values{}
	}
	params.Set("replyWithCount", "true")
	params.Set("start", fmt.Sprintf("%d", start))
	if limit == 0 {
		limit = 50
	}
	params.Set("limit", fmt.Sprintf("%d", limit))
	requestURL := cli.getRequestURL(resource, params)
	err := cli.sign(requestURL, "GET", header)
	if err != nil {
		return nil, err
	}
	_, resp, err := httputils.JSONRequest(cli.httpClient, context.Background(), "GET", requestURL, header, nil, cli.debug)
	if err != nil {
		if e, ok := err.(*httputils.JSONClientError); ok {
			if strings.Contains(e.Details, "wrong accessKey signature") || strings.Contains(e.Details, "access key id") {
				return nil, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, err.Error())
			}
		}
		return nil, err
	}
	return resp, nil
}

func (cli *SZStackClient) getDeleteURL(resource, resourceId, deleteMode string) string {
	if len(resourceId) == 0 {
		return cli.authURL + fmt.Sprintf("/zstack/%s/%s", ZSTACK_API_VERSION, resource)
	}
	url := cli.authURL + fmt.Sprintf("/zstack/%s/%s/%s", ZSTACK_API_VERSION, resource, resourceId)
	if len(deleteMode) > 0 {
		url += "?deleteMode=" + deleteMode
	}
	return url
}

func (cli *SZStackClient) delete(resource, resourceId, deleteMode string) error {
	_, err := cli._delete(resource, resourceId, deleteMode)
	return err
}

func (cli *SZStackClient) _delete(resource, resourceId, deleteMode string) (jsonutils.JSONObject, error) {
	header := http.Header{}
	requestURL := cli.getDeleteURL(resource, resourceId, deleteMode)
	err := cli.sign(requestURL, "DELETE", header)
	if err != nil {
		return nil, err
	}
	_, resp, err := httputils.JSONRequest(cli.httpClient, context.Background(), "DELETE", requestURL, header, nil, cli.debug)
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("DELETE %s %s %s", resource, resourceId, deleteMode))
	}
	if resp.Contains("location") {
		location, _ := resp.GetString("location")
		return cli.wait(header, "delete", requestURL, jsonutils.NewDict(), location)
	}
	return resp, nil
}

func (cli *SZStackClient) getURL(resource, resourceId, spec string) string {
	if len(resourceId) == 0 {
		return cli.authURL + fmt.Sprintf("/zstack/%s/%s", ZSTACK_API_VERSION, resource)
	}
	if len(spec) == 0 {
		return cli.authURL + fmt.Sprintf("/zstack/%s/%s/%s", ZSTACK_API_VERSION, resource, resourceId)
	}
	return cli.authURL + fmt.Sprintf("/zstack/%s/%s/%s/%s", ZSTACK_API_VERSION, resource, resourceId, spec)
}

func (cli *SZStackClient) getPostURL(resource string) string {
	return cli.authURL + fmt.Sprintf("/zstack/%s/%s", ZSTACK_API_VERSION, resource)
}

func (cli *SZStackClient) put(resource, resourceId string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return cli._put(resource, resourceId, params)
}

func (cli *SZStackClient) _put(resource, resourceId string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	header := http.Header{}
	requestURL := cli.getURL(resource, resourceId, "actions")
	err := cli.sign(requestURL, "PUT", header)
	if err != nil {
		return nil, err
	}
	_, resp, err := httputils.JSONRequest(cli.httpClient, context.Background(), "PUT", requestURL, header, params, cli.debug)
	if err != nil {
		return nil, err
	}
	if resp.Contains("location") {
		location, _ := resp.GetString("location")
		return cli.wait(header, "update", requestURL, params, location)
	}
	return resp, nil
}

func (cli *SZStackClient) getResource(resource, resourceId string, retval interface{}) error {
	if len(resourceId) == 0 {
		return cloudprovider.ErrNotFound
	}
	resp, err := cli._get(resource, resourceId, "")
	if err != nil {
		return err
	}
	inventories, err := resp.GetArray("inventories")
	if err != nil {
		return err
	}
	if len(inventories) == 1 {
		return inventories[0].Unmarshal(retval)
	}
	if len(inventories) == 0 {
		return cloudprovider.ErrNotFound
	}
	return cloudprovider.ErrDuplicateId
}

func (cli *SZStackClient) getMonitor(resource string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return cli._getMonitor(resource, params)
}

func (cli *SZStackClient) _getMonitor(resource string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	header := http.Header{}
	requestURL := cli.getPostURL(resource)
	paramDict := params.(*jsonutils.JSONDict)
	if paramDict.Size() > 0 {
		values := url.Values{}
		for _, key := range paramDict.SortedKeys() {
			value, _ := paramDict.GetString(key)
			values.Add(key, value)
		}
		requestURL += fmt.Sprintf("?%s", values.Encode())
	}
	var resp jsonutils.JSONObject
	startTime := time.Now()
	for time.Now().Sub(startTime) < time.Minute*5 {
		err := cli.sign(requestURL, "GET", header)
		if err != nil {
			return nil, err
		}
		_, resp, err = cli.jsonRequest(context.TODO(), "GET", requestURL, header, nil)
		if err != nil {
			if strings.Contains(err.Error(), "exceeded while awaiting headers") {
				time.Sleep(time.Second * 5)
				continue
			}
			return nil, errors.Wrapf(err, fmt.Sprintf("GET %s %s", resource, params))
		}
		break
	}

	if resp.Contains("location") {
		location, _ := resp.GetString("location")
		return cli.wait(header, "get", requestURL, jsonutils.NewDict(), location)
	}
	return resp, nil
}

func (cli *SZStackClient) get(resource, resourceId string, spec string) (jsonutils.JSONObject, error) {
	return cli._get(resource, resourceId, spec)
}

func (cli *SZStackClient) _get(resource, resourceId string, spec string) (jsonutils.JSONObject, error) {
	header := http.Header{}
	requestURL := cli.getURL(resource, resourceId, spec)
	var resp jsonutils.JSONObject
	startTime := time.Now()
	for time.Now().Sub(startTime) < time.Minute*5 {
		err := cli.sign(requestURL, "GET", header)
		if err != nil {
			return nil, err
		}
		_, resp, err = cli.jsonRequest(context.TODO(), "GET", requestURL, header, nil)
		if err != nil {
			if strings.Contains(err.Error(), "exceeded while awaiting headers") {
				time.Sleep(time.Second * 5)
				continue
			}
			return nil, errors.Wrapf(err, fmt.Sprintf("GET %s %s %s", resource, resourceId, spec))
		}
		break
	}

	if resp.Contains("location") {
		location, _ := resp.GetString("location")
		return cli.wait(header, "get", requestURL, jsonutils.NewDict(), location)
	}
	return resp, nil
}

func (cli *SZStackClient) create(resource string, params jsonutils.JSONObject, retval interface{}) error {
	resp, err := cli._post(resource, params)
	if err != nil {
		return err
	}
	if retval == nil {
		return nil
	}
	return resp.Unmarshal(retval, "inventory")
}

func (cli *SZStackClient) post(resource string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return cli._post(resource, params)
}

func (cli *SZStackClient) request(ctx context.Context, method httputils.THttpMethod, urlStr string, header http.Header, body io.Reader) (*http.Response, error) {
	resp, err := httputils.Request(cli.httpClient, ctx, method, urlStr, header, body, cli.debug)
	return resp, err
}

func (cli *SZStackClient) jsonRequest(ctx context.Context, method httputils.THttpMethod, urlStr string, header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	hdr, data, err := httputils.JSONRequest(cli.httpClient, ctx, method, urlStr, header, body, cli.debug)
	return hdr, data, err
}

func (cli *SZStackClient) wait(header http.Header, action string, requestURL string, params jsonutils.JSONObject, location string) (jsonutils.JSONObject, error) {
	startTime := time.Now()
	timeout := time.Minute * 30
	for {
		resp, err := cli.request(context.TODO(), "GET", location, header, nil)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("wait location %s", location))
		}
		_, result, err := httputils.ParseJSONResponse("", resp, err, cli.debug)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, cloudprovider.ErrNotFound
			}
			return nil, err
		}
		if time.Now().Sub(startTime) > timeout {
			return nil, fmt.Errorf("timeout for waitting %s %s params: %s", action, requestURL, params.PrettyString())
		}
		if resp.StatusCode != 200 {
			log.Debugf("wait for job %s %s %s complete", action, requestURL, params.String())
			time.Sleep(5 * time.Second)
			continue
		}
		return result, nil
	}
}

func (cli *SZStackClient) _post(resource string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	header := http.Header{}
	requestURL := cli.getPostURL(resource)
	err := cli.sign(requestURL, "POST", header)
	if err != nil {
		return nil, err
	}
	_, resp, err := cli.jsonRequest(context.TODO(), "POST", requestURL, header, params)
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("POST %s %s", resource, params.String()))
	}
	if resp.Contains("location") {
		location, _ := resp.GetString("location")
		return cli.wait(header, "create", requestURL, params, location)
	}
	return resp, nil
}

func (cli *SZStackClient) list(baseURL string, start int, limit int, params url.Values, retVal interface{}) error {
	resp, err := cli._list(baseURL, start, limit, params)
	if err != nil {
		return err
	}
	return resp.Unmarshal(retVal, "inventories")
}

func (cli *SZStackClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(cli.iregions); i++ {
		if cli.iregions[i].GetId() == regionId {
			return cli.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (cli *SZStackClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(cli.iregions))
	for i := 0; i < len(regions); i++ {
		region := cli.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (cli *SZStackClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SZStackClient) GetCapabilities() []string {
	caps := []string{
		// cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		// cloudprovider.CLOUD_CAPABILITY_EVENT,
	}
	return caps
}
