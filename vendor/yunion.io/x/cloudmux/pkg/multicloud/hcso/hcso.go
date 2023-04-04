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

package hcso

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go/auth/aksk"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type akClient struct {
	client *http.Client
	aksk   aksk.SignOptions
}

func (self *akClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Del("Host")
	req.Header.Del("Authorization")
	req.Header.Del("X-Sdk-Date")
	req.Header.Del("Accept")
	if req.Method == string(httputils.GET) || req.Method == string(httputils.DELETE) || req.Method == string(httputils.PATCH) {
		req.Header.Del("Content-Length")
	}
	aksk.Sign(req, self.aksk)
	return self.client.Do(req)
}

func (self *SHuaweiClient) getAkClient() *akClient {
	return &akClient{
		client: self.getDefaultClient(),
		aksk: aksk.SignOptions{
			AccessKey: self.accessKey,
			SecretKey: self.accessSecret,
		},
	}
}

func (self *SHuaweiClient) getDefaultClient() *http.Client {
	if self.httpClient != nil {
		return self.httpClient
	}
	self.httpClient = self.cpcfg.AdaptiveTimeoutHttpClient()
	ts, _ := self.httpClient.Transport.(*http.Transport)
	self.httpClient.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response), error) {
		service, method, path := strings.Split(req.URL.Host, ".")[0], req.Method, req.URL.Path
		respCheck := func(resp *http.Response) {
			if resp.StatusCode == 403 {
				if self.cpcfg.UpdatePermission != nil {
					self.cpcfg.UpdatePermission(service, fmt.Sprintf("%s %s", method, path))
				}
			}
		}
		if self.cpcfg.ReadOnly {
			if req.Method == "GET" {
				return respCheck, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return respCheck, nil
	})
	return self.httpClient
}

func (self *SHuaweiClient) request(method httputils.THttpMethod, url string, query url.Values, params map[string]interface{}) (jsonutils.JSONObject, error) {
	client := self.getAkClient()
	if len(query) > 0 {
		url = fmt.Sprintf("%s?%s", url, query.Encode())
	}
	var body jsonutils.JSONObject = nil
	if len(params) > 0 {
		body = jsonutils.Marshal(params)
	}
	header := http.Header{}
	if len(self.projectId) > 0 {
		header.Set("X-Project-Id", self.projectId)
	}
	if strings.Contains(url, "/OS-CREDENTIAL/credentials") && len(self.ownerId) > 0 {
		header.Set("X-Domain-Id", self.ownerId)
	}
	var resp jsonutils.JSONObject
	var err error

	for i := 0; i < 3; i++ {
		_, resp, err = requestWithRetry(client, context.Background(), method, url, header, body, self.debug)
		if method == httputils.GET && needRetry(err) {
			time.Sleep(time.Second * 15)
			continue
		}
		break
	}
	return resp, err
}

func requestWithRetry(client *akClient, ctx context.Context, method httputils.THttpMethod, urlStr string, header http.Header, body jsonutils.JSONObject, debug bool) (http.Header, jsonutils.JSONObject, error) {
	var bodystr string
	if !gotypes.IsNil(body) {
		bodystr = body.String()
	}
	jbody := strings.NewReader(bodystr)
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Length", strconv.FormatInt(int64(len(bodystr)), 10))
	header.Set("Content-Type", "application/json")
	resp, err := httputils.RequestWithRetry(client, ctx, method, urlStr, header, jbody, debug)
	return httputils.ParseJSONResponse(bodystr, resp, err, debug)
}

func (self *SHuaweiClient) resetEndpoint(endpoint, serviceName string) string {
	if len(endpoint) == 0 {
		domain := self.HuaweiClientConfig.endpoints.EndpointDomain
		regionId := self.HuaweiClientConfig.cpcfg.DefaultRegion
		if len(regionId) == 0 {
			regionId = self.GetRegions()[0].ID
		}
		endpoint = fmt.Sprintf("%s.%s.%s", serviceName, regionId, domain)
	}
	return endpoint
}

func (self *SHuaweiClient) getAKSKList(userId string) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Iam, "iam-pub")
	uri := fmt.Sprintf("https://%s/v3.0/OS-CREDENTIAL/credentials", endpoint)
	query := url.Values{}
	query.Set("user_id", userId)
	return self.request(httputils.GET, uri, query, nil)
}

func (self *SHuaweiClient) createAKSK(params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Iam, "iam-pub")
	uri := fmt.Sprintf("https://%s/v3.0/OS-CREDENTIAL/credentials", endpoint)
	return self.request(httputils.POST, uri, nil, params)
}

func (self *SHuaweiClient) deleteAKSK(accessKey string) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Iam, "iam-pub")
	uri := fmt.Sprintf("https://%s/v3.0/OS-CREDENTIAL/credentials/%s", endpoint, accessKey)
	return self.request(httputils.DELETE, uri, nil, nil)
}

func (self *SHuaweiClient) modelartsPoolNetworkList(params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v1/%s/networks", endpoint, self.projectId)
	return self.request(httputils.GET, uri, url.Values{}, params)
}

func (cli *SHuaweiClient) modelartsPoolNetworkDetail(networkName string) (jsonutils.JSONObject, error) {
	endpoint := cli.resetEndpoint(cli.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v1/%s/networks/%s", endpoint, cli.projectId, networkName)
	return cli.request(httputils.GET, uri, url.Values{}, nil)
}

func (cli *SHuaweiClient) modelartsPoolNetworkDelete(networkName string) (jsonutils.JSONObject, error) {
	endpoint := cli.resetEndpoint(cli.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v1/%s/networks/%s", endpoint, cli.projectId, networkName)
	return cli.request(httputils.DELETE, uri, url.Values{}, nil)
}

func (self *SHuaweiClient) modelartsPoolNetworkCreate(params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v1/%s/networks", endpoint, self.projectId)
	return self.request(httputils.POST, uri, url.Values{}, params)
}

func (self *SHuaweiClient) modelartsPoolById(poolName string) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v2/%s/pools/%s", endpoint, self.projectId, poolName)
	return self.request(httputils.GET, uri, url.Values{}, nil)
}

func (cli *SHuaweiClient) modelartsPoolListWithStatus(resource, status string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := cli.resetEndpoint(cli.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v2/%s/pools", endpoint, cli.projectId)
	value := url.Values{}
	value.Add("status", status)
	return cli.request(httputils.GET, uri, value, params)
}

func (self *SHuaweiClient) modelartsPoolList(params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v2/%s/pools", endpoint, self.projectId)
	return self.request(httputils.GET, uri, url.Values{}, params)
}

func (self *SHuaweiClient) modelartsPoolCreate(params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v2/%s/pools", endpoint, self.projectId)
	return self.request(httputils.POST, uri, url.Values{}, params)
}

func (self *SHuaweiClient) modelartsPoolDelete(poolName string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v2/%s/pools/%s", endpoint, self.projectId, poolName)
	return self.request(httputils.DELETE, uri, url.Values{}, params)
}

func (self *SHuaweiClient) modelartsPoolUpdate(poolName string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v2/%s/pools/%s", endpoint, self.projectId, poolName)
	urlValue := url.Values{}
	urlValue.Add("time_range", "")
	urlValue.Add("statistics", "")
	urlValue.Add("period", "")
	return self.patchRequest(httputils.PATCH, uri, urlValue, params)
}

func (self *SHuaweiClient) modelartsPoolMonitor(poolName string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v2/%s/pools/%s/monitor", endpoint, self.projectId, poolName)
	return self.request(httputils.GET, uri, url.Values{}, params)
}

func (self *SHuaweiClient) modelartsResourceflavors(params map[string]interface{}) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Modelarts, "modelarts")
	uri := fmt.Sprintf("https://%s/v1/%s/resourceflavors", endpoint, self.projectId)
	return self.request(httputils.GET, uri, url.Values{}, params)
}

func (self *SHuaweiClient) commonMonitor(params map[string]string) (jsonutils.JSONObject, error) {
	endpoint := self.resetEndpoint(self.endpoints.Ces, "ces")
	uri := fmt.Sprintf("https://%s/V1.0/%s/metric-data", endpoint, self.projectId)
	url := url.Values{}
	for k, v := range params {
		url.Set(k, v)
	}
	return self.request(httputils.GET, uri, url, nil)
}

func (self *SHuaweiClient) patchRequest(method httputils.THttpMethod, url string, query url.Values, params map[string]interface{}) (jsonutils.JSONObject, error) {
	client := self.getAkClient()
	if len(query) > 0 {
		url = fmt.Sprintf("%s?%s", url, query.Encode())
	}
	var body jsonutils.JSONObject = nil
	if len(params) > 0 {
		body = jsonutils.Marshal(params)
	}
	header := http.Header{}
	if len(self.projectId) > 0 {
		header.Set("X-Project-Id", self.projectId)
	}
	var bodystr string
	if !gotypes.IsNil(body) {
		bodystr = body.String()
	}
	jbody := strings.NewReader(bodystr)
	header.Set("Content-Length", strconv.FormatInt(int64(len(bodystr)), 10))
	header.Set("Content-Type", "application/merge-patch+json")
	resp, err := httputils.Request(client, context.Background(), method, url, header, jbody, self.debug)
	_, respValue, err := httputils.ParseJSONResponse(bodystr, resp, err, self.debug)
	if err != nil {
		if e, ok := err.(*httputils.JSONClientError); ok && e.Code == 404 {
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
		}
		return nil, err
	}
	return respValue, err
}

func needRetry(err error) bool {
	if err == nil {
		return false
	}
	switch e := err.(type) {
	case *url.Error:
		switch e.Err.(type) {
		case *net.DNSError, *net.OpError, net.UnknownNetworkError:
			return true
		}
		if strings.Contains(err.Error(), "The throttling threshold has been reached: policy ip over ratelimit") {
			return true
		}
	}
	return false
}
