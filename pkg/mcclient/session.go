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

package mcclient

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
)

const (
	TASK_ID         = "X-Task-Id"
	TASK_NOTIFY_URL = "X-Task-Notify-Url"
	AUTH_TOKEN      = api.AUTH_TOKEN_HEADER //  "X-Auth-Token"
	REGION_VERSION  = "X-Region-Version"

	DEFAULT_API_VERSION = "v1"
	V2_API_VERSION      = "v2"
)

type ClientSession struct {
	ctx context.Context

	client        *Client
	region        string
	zone          string
	endpointType  string
	token         TokenCredential
	Header        http.Header /// headers for this session
	notifyChannel chan string

	customizeServiceUrl map[string]string

	catalog IServiceCatalog
}

func populateHeader(self *http.Header, update http.Header) {
	for k, v := range update {
		for _, vv := range v {
			// self.Add(k, vv)
			self.Set(k, vv)
		}
	}
}

func GetTokenHeaders(userCred TokenCredential) http.Header {
	headers := http.Header{}
	headers.Set(AUTH_TOKEN, userCred.GetTokenString())
	headers.Set(REGION_VERSION, V2_API_VERSION)
	return headers
}

func SplitVersionedURL(url string) (string, string) {
	endidx := len(url) - 1
	for ; endidx >= 0 && url[endidx] == '/'; endidx-- {
	}
	lastslash := strings.LastIndexByte(url[0:endidx+1], '/')
	if lastslash >= 0 {
		if strings.EqualFold(url[lastslash+1:endidx+1], "latest") {
			return url[0:lastslash], ""
		}
		match, err := regexp.MatchString(`^v\d+\.?\d*`, url[lastslash+1:endidx+1])
		if err == nil && match {
			return url[0:lastslash], url[lastslash+1 : endidx+1]
		}
	}
	return url[0 : endidx+1], ""
}

/* func stripURLVersion(url string) string {
	base, _ := SplitVersionedURL(url)
	log.Debugf("stripURLVersion %s => %s", url, base)
	return base
}*/

func (cliss *ClientSession) GetEndpointType() string {
	return cliss.endpointType
}

func (cliss *ClientSession) GetClient() *Client {
	return cliss.client
}

func (cliss *ClientSession) SetZone(zone string) {
	cliss.zone = zone
}

func getApiVersionByServiceType(serviceType string) string {
	switch serviceType {
	case "compute":
		return "v2"
	}
	return ""
}

func (cliss *ClientSession) GetServiceName(service string) string {
	apiVersion := getApiVersionByServiceType(service)
	if len(apiVersion) > 0 && apiVersion != DEFAULT_API_VERSION {
		service = fmt.Sprintf("%s_%s", service, apiVersion)
	}
	return service
}

func (cliss *ClientSession) GetServiceURL(service, endpointType string, method httputils.THttpMethod) (string, error) {
	return cliss.GetServiceVersionURL(service, endpointType, method)
}

func (cliss *ClientSession) SetServiceCatalog(catalog IServiceCatalog) {
	cliss.catalog = catalog
}

func (cliss *ClientSession) GetServiceCatalog() IServiceCatalog {
	if cliss.catalog != nil {
		return cliss.catalog
	}
	return cliss.client.GetServiceCatalog()
}

func (cliss *ClientSession) GetServiceVersionURL(service, endpointType string, method httputils.THttpMethod) (string, error) {
	urls, err := cliss.GetServiceVersionURLs(service, endpointType, method)
	if err != nil {
		return "", errors.Wrap(err, "GetServiceVersionURLs")
	}
	return urls[rand.Intn(len(urls))], nil
}

func (cliss *ClientSession) GetServiceURLs(service, endpointType string, method httputils.THttpMethod) ([]string, error) {
	return cliss.GetServiceVersionURLs(service, endpointType, method)
}

func (cliss *ClientSession) GetServiceVersionURLs(service, endpointType string, method httputils.THttpMethod) ([]string, error) {
	return cliss.GetServiceVersionURLsByMethod(service, endpointType, method)
}

func (cliss *ClientSession) GetServiceVersionURLsByMethod(service, endpointType string, method httputils.THttpMethod) ([]string, error) {
	if len(cliss.endpointType) > 0 {
		// session specific endpoint type should override the input endpointType, which is supplied by manager
		endpointType = cliss.endpointType
	}
	service = cliss.GetServiceName(service)
	if endpointType == api.EndpointInterfaceApigateway {
		return cliss.getApigatewayServiceURLs(service, cliss.region, cliss.zone)
	} else {
		if (endpointType == "" || endpointType == api.EndpointInterfaceInternal) && (method == httputils.GET || method == httputils.HEAD) {
			urls, _ := cliss.getServiceVersionURLs(service, cliss.region, cliss.zone, api.EndpointInterfaceSlave)
			if len(urls) > 0 {
				return urls, nil
			}
		}
		return cliss.getServiceVersionURLs(service, cliss.region, cliss.zone, endpointType)
	}
}

func (cliss *ClientSession) getApigatewayServiceURLs(service, region, zone string) ([]string, error) {
	urls, err := cliss.getServiceVersionURLs(service, region, zone, api.EndpointInterfaceInternal)
	if err != nil {
		return nil, errors.Wrap(err, "getServiceVersionURLs")
	}
	// replace URLs with authUrl prefix
	// find the common prefix
	prefix := cliss.client.authUrl
	lastSlashPos := strings.LastIndex(prefix, "/api/s/identity")
	if lastSlashPos <= 0 {
		return nil, errors.Wrapf(errors.ErrInvalidFormat, "invalue auth_url %s, should be url of apigateway endpoint, e.g. https://<apigateway-host>/api/s/identity/v3", prefix)
	}
	prefix = httputils.JoinPath(prefix[:lastSlashPos], "api/s", service)
	if len(region) > 0 {
		prefix = httputils.JoinPath(prefix, "r", region)
		if len(zone) > 0 {
			prefix = httputils.JoinPath(prefix, "z", zone)
		}
	}
	rets := make([]string, len(urls))
	for i, url := range urls {
		if len(url) < 9 {
			// len("https://") == 8
			log.Errorf("invalid url %s: shorter than 9 bytes", url)
			continue
		}
		slashPos := strings.IndexByte(url[9:], '/')
		if slashPos > 0 {
			url = url[9+slashPos:]
			rets[i] = httputils.JoinPath(prefix, url)
		} else {
			rets[i] = prefix
		}
	}
	return rets, nil
}

func (cliss *ClientSession) getServiceVersionURLs(service, region, zone, endpointType string) ([]string, error) {
	catalog := cliss.GetServiceCatalog()
	if gotypes.IsNil(catalog) {
		return []string{cliss.client.authUrl}, nil
	}
	urls, err := catalog.getServiceURLs(service, region, zone, endpointType)
	// HACK! in case of fail to get keystone url or schema of keystone changed, always trust authUrl
	if service == api.SERVICE_TYPE && (err != nil || len(urls) == 0 || (len(cliss.client.authUrl) != 0 && cliss.client.authUrl[:5] != urls[0][:5])) {
		var msg string
		if err != nil {
			msg = fmt.Sprintf("fail to retrieve keystone urls: %s", err)
		} else if len(urls) == 0 {
			msg = "empty keystone url"
		} else {
			msg = fmt.Sprintf("Schema of keystone authUrl and endpoint mismatch: %s!=%s", cliss.client.authUrl, urls)
		}
		log.Warningln(msg)
		return []string{cliss.client.authUrl}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "catalog.GetServiceURLs")
	}
	return urls, err
}

func (cliss *ClientSession) GetBaseUrl(service, endpointType string, method httputils.THttpMethod) (string, error) {
	if len(service) > 0 {
		if strings.HasPrefix(service, "http://") || strings.HasPrefix(service, "https://") {
			return service, nil
		} else if url, ok := cliss.customizeServiceUrl[service]; ok {
			return url, nil
		} else {
			return cliss.GetServiceVersionURL(service, endpointType, method)
		}
	} else {
		return "", fmt.Errorf("Empty service type or baseURL")
	}
}

func (cliss *ClientSession) RawBaseUrlRequest(
	service, endpointType string,
	method httputils.THttpMethod, url string,
	headers http.Header, body io.Reader,
	baseurlFactory func(string) string,
) (*http.Response, error) {
	baseurl, err := cliss.GetBaseUrl(service, endpointType, method)
	if err != nil {
		return nil, err
	}
	if baseurlFactory != nil {
		baseurl = baseurlFactory(baseurl)
	}
	tmpHeader := http.Header{}
	if headers != nil {
		populateHeader(&tmpHeader, headers)
	}
	populateHeader(&tmpHeader, cliss.Header)
	appctx.SetHTTPLangHeader(cliss.ctx, tmpHeader)
	ctx := cliss.ctx
	if cliss.ctx == nil {
		ctx = context.Background()
	}
	return cliss.client.rawRequest(ctx, baseurl,
		cliss.token.GetTokenString(),
		method, url, tmpHeader, body)
}

func (cliss *ClientSession) RawVersionRequest(
	service, endpointType string, method httputils.THttpMethod, url string,
	headers http.Header, body io.Reader,
) (*http.Response, error) {
	return cliss.RawBaseUrlRequest(service, endpointType, method, url, headers, body, nil)
}

func (cliss *ClientSession) RawRequest(service, endpointType string, method httputils.THttpMethod, url string, headers http.Header, body io.Reader) (*http.Response, error) {
	return cliss.RawVersionRequest(service, endpointType, method, url, headers, body)
}

func (cliss *ClientSession) JSONVersionRequest(
	service, endpointType string, method httputils.THttpMethod, url string,
	headers http.Header, body jsonutils.JSONObject,
) (http.Header, jsonutils.JSONObject, error) {
	baseUrl, err := cliss.GetBaseUrl(service, endpointType, method)
	if err != nil {
		return headers, nil, err
	}
	tmpHeader := http.Header{}
	if headers != nil {
		populateHeader(&tmpHeader, headers)
	}
	populateHeader(&tmpHeader, cliss.Header)
	appctx.SetHTTPLangHeader(cliss.ctx, tmpHeader)
	ctx := cliss.ctx
	if cliss.ctx == nil {
		ctx = context.Background()
	}
	return cliss.client.jsonRequest(ctx, baseUrl,
		cliss.token.GetTokenString(),
		method, url, tmpHeader, body)
}

func (cliss *ClientSession) JSONRequest(service, endpointType string, method httputils.THttpMethod, url string, headers http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return cliss.JSONVersionRequest(service, endpointType, method, url, headers, body)
}

func (cliss *ClientSession) ParseJSONResponse(reqBody string, resp *http.Response, err error) (http.Header, jsonutils.JSONObject, error) {
	return httputils.ParseJSONResponse(reqBody, resp, err, cliss.client.debug)
}

func (cliss *ClientSession) HasSystemAdminPrivilege() bool {
	return cliss.token.HasSystemAdminPrivilege()
}

func (cliss *ClientSession) GetRegion() string {
	return cliss.region
}

func (cliss *ClientSession) GetUserId() string {
	return cliss.token.GetUserId()
}

func (cliss *ClientSession) GetTenantId() string {
	return cliss.token.GetTenantId()
}

func (cliss *ClientSession) GetTenantName() string {
	return cliss.token.GetTenantName()
}

func (cliss *ClientSession) GetProjectId() string {
	return cliss.GetTenantId()
}

func (cliss *ClientSession) GetProjectName() string {
	return cliss.GetTenantName()
}

func (cliss *ClientSession) GetProjectDomain() string {
	return cliss.token.GetProjectDomain()
}

func (cliss *ClientSession) GetProjectDomainId() string {
	return cliss.token.GetProjectDomainId()
}

func (cliss *ClientSession) GetDomainId() string {
	return cliss.token.GetDomainId()
}

func (cliss *ClientSession) GetDomainName() string {
	return cliss.token.GetDomainName()
}

func (cliss *ClientSession) SetTaskNotifyUrl(url string) {
	cliss.Header.Add(TASK_NOTIFY_URL, url)
}

func (cliss *ClientSession) RemoveTaskNotifyUrl() {
	cliss.Header.Del(TASK_NOTIFY_URL)
}

func (cliss *ClientSession) SetServiceUrl(service, url string) {
	cliss.customizeServiceUrl[service] = url
}

func (cliss *ClientSession) WithTaskCallback(taskId string, req func() error) error {
	baseUrl, err := cliss.GetBaseUrl(consts.GetServiceType(), api.EndpointInterfacePublic, httputils.POST)
	if err != nil {
		log.Errorf("GetServiceURLs error: %s", err)
		return errors.Wrap(err, "GetServiceURLs")
	}
	if len(baseUrl) == 0 {
		return errors.Wrap(errors.ErrInvalidFormat, "empty service url")
	}
	taskUrl := joinUrl(baseUrl, fmt.Sprintf("/tasks/%s", taskId))
	log.Infof("SetTaskNotifyUrl: %s service: %s", taskUrl, consts.GetServiceType())
	cliss.SetTaskNotifyUrl(taskUrl)
	defer cliss.RemoveTaskNotifyUrl()

	{
		err := req()
		if err != nil {
			return errors.Wrap(err, "Request")
		}
	}

	return nil
}

func (cliss *ClientSession) PrepareTask() {
	// start a random htttp server
	cliss.notifyChannel = make(chan string)

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	port := 55000 + r1.Intn(1000)
	ip := utils.GetOutboundIP()
	addr := fmt.Sprintf("%s:%d", ip.String(), port)
	url := fmt.Sprintf("http://%s", addr)
	cliss.SetTaskNotifyUrl(url)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
		body, err := io.ReadAll(r.Body)
		var msg string
		if err != nil {
			msg = fmt.Sprintf("Read request data error: %s", err)
		} else {
			msg = string(body)
		}
		cliss.notifyChannel <- msg
	})

	go func() {
		fmt.Println("List on address: ", url)
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Printf("Task notify server error: %s\n", err)
		}
	}()
}

func (cliss *ClientSession) WaitTaskNotify() {
	if cliss.notifyChannel != nil {
		msg := <-cliss.notifyChannel
		fmt.Println("---------------Task complete -------------")
		fmt.Println(msg)
	}
}

/*func (cliss *ClientSession) SetApiVersion(version string) {
	cliss.defaultApiVersion = version
}

func (cliss *ClientSession) GetApiVersion() string {
	apiVersion := cliss.getApiVersion("")
	if len(apiVersion) == 0 {
		return DEFAULT_API_VERSION
	}
	return apiVersion
}*/

func (cliss *ClientSession) ToJson() jsonutils.JSONObject {
	params := jsonutils.NewDict()
	simpleToken := SimplifyToken(cliss.token)
	tokenJson := jsonutils.Marshal(simpleToken)
	params.Update(tokenJson)
	// params.Add(jsonutils.NewString(cliss.GetApiVersion()), "api_version")
	if len(cliss.endpointType) > 0 {
		params.Add(jsonutils.NewString(cliss.endpointType), "endpoint_type")
	}
	if len(cliss.region) > 0 {
		params.Add(jsonutils.NewString(cliss.region), "region")
	}
	if len(cliss.zone) > 0 {
		params.Add(jsonutils.NewString(cliss.zone), "zone")
	}
	if tokenV3, ok := cliss.token.(*TokenCredentialV3); ok {
		params.Add(jsonutils.NewStringArray(tokenV3.Token.Policies.Project), "project_policies")
		params.Add(jsonutils.NewStringArray(tokenV3.Token.Policies.Domain), "domain_policies")
		params.Add(jsonutils.NewStringArray(tokenV3.Token.Policies.System), "system_policies")
	}
	return params
}

func (cliss *ClientSession) GetToken() TokenCredential {
	return cliss.token
}

func (cliss *ClientSession) GetContext() context.Context {
	if cliss.ctx == nil {
		return context.Background()
	}
	return cliss.ctx
}

func (cliss *ClientSession) GetCommonEtcdEndpoint() (*api.EndpointDetails, error) {
	return cliss.GetClient().GetCommonEtcdEndpoint(cliss.GetToken(), cliss.region, cliss.endpointType)
}
