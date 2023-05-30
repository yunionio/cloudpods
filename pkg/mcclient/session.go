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
	"io/ioutil"
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

func (this *ClientSession) GetEndpointType() string {
	return this.endpointType
}

func (this *ClientSession) GetClient() *Client {
	return this.client
}

func (this *ClientSession) SetZone(zone string) {
	this.zone = zone
}

func getApiVersionByServiceType(serviceType string) string {
	switch serviceType {
	case "compute":
		return "v2"
	}
	return ""
}

func (this *ClientSession) getServiceName(service string) string {
	apiVersion := getApiVersionByServiceType(service)
	if len(apiVersion) > 0 && apiVersion != DEFAULT_API_VERSION {
		service = fmt.Sprintf("%s_%s", service, apiVersion)
	}
	return service
}

func (this *ClientSession) GetServiceURL(service, endpointType string) (string, error) {
	return this.GetServiceVersionURL(service, endpointType)
}

func (this *ClientSession) SetServiceCatalog(catalog IServiceCatalog) {
	this.catalog = catalog
}

func (this *ClientSession) GetServiceCatalog() IServiceCatalog {
	if this.catalog != nil {
		return this.catalog
	}
	return this.client.GetServiceCatalog()
}

func (this *ClientSession) GetServiceVersionURL(service, endpointType string) (string, error) {
	urls, err := this.GetServiceVersionURLs(service, endpointType)
	if err != nil {
		return "", errors.Wrap(err, "GetServiceVersionURLs")
	}
	return urls[rand.Intn(len(urls))], nil
}

func (this *ClientSession) GetServiceURLs(service, endpointType string) ([]string, error) {
	return this.GetServiceVersionURLs(service, endpointType)
}

func (this *ClientSession) GetServiceVersionURLs(service, endpointType string) ([]string, error) {
	if len(this.endpointType) > 0 {
		// session specific endpoint type should override the input endpointType, which is supplied by manager
		endpointType = this.endpointType
	}
	service = this.getServiceName(service)
	if endpointType == api.EndpointInterfaceApigateway {
		return this.getApigatewayServiceURLs(service, this.region, this.zone, endpointType)
	} else {
		return this.getServiceVersionURLs(service, this.region, this.zone, endpointType)
	}
}

func (this *ClientSession) getApigatewayServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	urls, err := this.getServiceVersionURLs(service, region, zone, "")
	if err != nil {
		return nil, errors.Wrap(err, "getServiceVersionURLs")
	}
	// replace URLs with authUrl prefix
	// find the common prefix
	prefix := this.client.authUrl
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

func (this *ClientSession) getServiceVersionURLs(service, region, zone, endpointType string) ([]string, error) {
	catalog := this.GetServiceCatalog()
	if gotypes.IsNil(catalog) {
		return []string{this.client.authUrl}, nil
	}
	urls, err := catalog.getServiceURLs(service, region, zone, endpointType)
	// HACK! in case of fail to get kestone url or schema of keystone changed, always trust authUrl
	if service == api.SERVICE_TYPE && (err != nil || len(urls) == 0 || (len(this.client.authUrl) != 0 && this.client.authUrl[:5] != urls[0][:5])) {
		var msg string
		if err != nil {
			msg = fmt.Sprintf("fail to retrieve keystone urls: %s", err)
		} else if len(urls) == 0 {
			msg = fmt.Sprintf("empty keystone url")
		} else {
			msg = fmt.Sprintf("Schema of keystone authUrl and endpoint mismatch: %s!=%s", this.client.authUrl, urls)
		}
		log.Warningln(msg)
		return []string{this.client.authUrl}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "catalog.GetServiceURLs")
	}
	return urls, err
}

func (this *ClientSession) getBaseUrl(service, endpointType string) (string, error) {
	if len(service) > 0 {
		if strings.HasPrefix(service, "http://") || strings.HasPrefix(service, "https://") {
			return service, nil
		} else if url, ok := this.customizeServiceUrl[service]; ok {
			return url, nil
		} else {
			return this.GetServiceVersionURL(service, endpointType)
		}
	} else {
		return "", fmt.Errorf("Empty service type or baseURL")
	}
}

func (this *ClientSession) RawBaseUrlRequest(
	service, endpointType string,
	method httputils.THttpMethod, url string,
	headers http.Header, body io.Reader,
	baseurlFactory func(string) string,
) (*http.Response, error) {
	baseurl, err := this.getBaseUrl(service, endpointType)
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
	populateHeader(&tmpHeader, this.Header)
	appctx.SetHTTPLangHeader(this.ctx, tmpHeader)
	ctx := this.ctx
	if this.ctx == nil {
		ctx = context.Background()
	}
	return this.client.rawRequest(ctx, baseurl,
		this.token.GetTokenString(),
		method, url, tmpHeader, body)
}

func (this *ClientSession) RawVersionRequest(
	service, endpointType string, method httputils.THttpMethod, url string,
	headers http.Header, body io.Reader,
) (*http.Response, error) {
	return this.RawBaseUrlRequest(service, endpointType, method, url, headers, body, nil)
}

func (this *ClientSession) RawRequest(service, endpointType string, method httputils.THttpMethod, url string, headers http.Header, body io.Reader) (*http.Response, error) {
	return this.RawVersionRequest(service, endpointType, method, url, headers, body)
}

func (this *ClientSession) JSONVersionRequest(
	service, endpointType string, method httputils.THttpMethod, url string,
	headers http.Header, body jsonutils.JSONObject,
) (http.Header, jsonutils.JSONObject, error) {
	baseUrl, err := this.getBaseUrl(service, endpointType)
	if err != nil {
		return headers, nil, err
	}
	tmpHeader := http.Header{}
	if headers != nil {
		populateHeader(&tmpHeader, headers)
	}
	populateHeader(&tmpHeader, this.Header)
	appctx.SetHTTPLangHeader(this.ctx, tmpHeader)
	ctx := this.ctx
	if this.ctx == nil {
		ctx = context.Background()
	}
	return this.client.jsonRequest(ctx, baseUrl,
		this.token.GetTokenString(),
		method, url, tmpHeader, body)
}

func (this *ClientSession) JSONRequest(service, endpointType string, method httputils.THttpMethod, url string, headers http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return this.JSONVersionRequest(service, endpointType, method, url, headers, body)
}

func (this *ClientSession) ParseJSONResponse(reqBody string, resp *http.Response, err error) (http.Header, jsonutils.JSONObject, error) {
	return httputils.ParseJSONResponse(reqBody, resp, err, this.client.debug)
}

func (this *ClientSession) HasSystemAdminPrivilege() bool {
	return this.token.HasSystemAdminPrivilege()
}

func (this *ClientSession) GetRegion() string {
	return this.region
}

func (this *ClientSession) GetUserId() string {
	return this.token.GetUserId()
}

func (this *ClientSession) GetTenantId() string {
	return this.token.GetTenantId()
}

func (this *ClientSession) GetTenantName() string {
	return this.token.GetTenantName()
}

func (this *ClientSession) GetProjectId() string {
	return this.GetTenantId()
}

func (this *ClientSession) GetProjectName() string {
	return this.GetTenantName()
}

func (this *ClientSession) GetProjectDomain() string {
	return this.token.GetProjectDomain()
}

func (this *ClientSession) GetProjectDomainId() string {
	return this.token.GetProjectDomainId()
}

func (this *ClientSession) GetDomainId() string {
	return this.token.GetDomainId()
}

func (this *ClientSession) GetDomainName() string {
	return this.token.GetDomainName()
}

func (this *ClientSession) SetTaskNotifyUrl(url string) {
	this.Header.Add(TASK_NOTIFY_URL, url)
}

func (this *ClientSession) RemoveTaskNotifyUrl() {
	this.Header.Del(TASK_NOTIFY_URL)
}

func (this *ClientSession) SetServiceUrl(service, url string) {
	this.customizeServiceUrl[service] = url
}

func (this *ClientSession) PrepareTask() {
	// start a random htttp server
	this.notifyChannel = make(chan string)

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	port := 55000 + r1.Intn(1000)
	ip := utils.GetOutboundIP()
	addr := fmt.Sprintf("%s:%d", ip.String(), port)
	url := fmt.Sprintf("http://%s", addr)
	this.SetTaskNotifyUrl(url)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
		body, err := ioutil.ReadAll(r.Body)
		var msg string
		if err != nil {
			msg = fmt.Sprintf("Read request data error: %s", err)
		} else {
			msg = string(body)
		}
		this.notifyChannel <- msg
	})

	go func() {
		fmt.Println("List on address: ", url)
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Printf("Task notify server error: %s\n", err)
		}
	}()
}

func (this *ClientSession) WaitTaskNotify() {
	if this.notifyChannel != nil {
		msg := <-this.notifyChannel
		fmt.Println("---------------Task complete -------------")
		fmt.Println(msg)
	}
}

/*func (this *ClientSession) SetApiVersion(version string) {
	this.defaultApiVersion = version
}

func (this *ClientSession) GetApiVersion() string {
	apiVersion := this.getApiVersion("")
	if len(apiVersion) == 0 {
		return DEFAULT_API_VERSION
	}
	return apiVersion
}*/

func (this *ClientSession) ToJson() jsonutils.JSONObject {
	params := jsonutils.NewDict()
	simpleToken := SimplifyToken(this.token)
	tokenJson := jsonutils.Marshal(simpleToken)
	params.Update(tokenJson)
	// params.Add(jsonutils.NewString(this.GetApiVersion()), "api_version")
	if len(this.endpointType) > 0 {
		params.Add(jsonutils.NewString(this.endpointType), "endpoint_type")
	}
	if len(this.region) > 0 {
		params.Add(jsonutils.NewString(this.region), "region")
	}
	if len(this.zone) > 0 {
		params.Add(jsonutils.NewString(this.zone), "zone")
	}
	if tokenV3, ok := this.token.(*TokenCredentialV3); ok {
		params.Add(jsonutils.NewStringArray(tokenV3.Token.Policies.Project), "project_policies")
		params.Add(jsonutils.NewStringArray(tokenV3.Token.Policies.Domain), "domain_policies")
		params.Add(jsonutils.NewStringArray(tokenV3.Token.Policies.System), "system_policies")
	}
	return params
}

func (cs *ClientSession) GetToken() TokenCredential {
	return cs.token
}

func (cs *ClientSession) GetContext() context.Context {
	if cs.ctx == nil {
		return context.Background()
	}
	return cs.ctx
}

func (cs *ClientSession) GetCommonEtcdEndpoint() (*api.EndpointDetails, error) {
	return cs.GetClient().GetCommonEtcdEndpoint(cs.GetToken(), cs.region, cs.endpointType)
}
