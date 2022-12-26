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

package oscli

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
)

const (
	AUTH_TOKEN     = "X-Auth-Token"
	REGION_VERSION = "X-Region-Version"

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

func (this *ClientSession) GetEndpointType() string {
	return this.endpointType
}

func (this *ClientSession) GetClient() *Client {
	return this.client
}

func (this *ClientSession) SetZone(zone string) {
	this.zone = zone
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
	return this.getServiceVersionURLs(service, this.region, this.zone, endpointType)
}

func (this *ClientSession) getServiceVersionURLs(service, region, zone, endpointType string) ([]string, error) {
	catalog := this.GetServiceCatalog()
	if gotypes.IsNil(catalog) {
		return []string{this.client.authUrl}, nil
	}
	urls, err := catalog.GetServiceURLs(service, region, zone, endpointType)
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

type ctxLang uintptr

const (
	ctxLangKey = ctxLang(0)
)

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
	langv := this.ctx.Value(ctxLangKey)
	langTag, ok := langv.(language.Tag)
	if ok {
		tmpHeader.Set("X-Yunion-Lang", langTag.String())
	}
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
	langv := this.ctx.Value(ctxLangKey)
	langTag, ok := langv.(language.Tag)
	if ok {
		tmpHeader.Set("X-Yunion-Lang", langTag.String())
	}
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

func (this *ClientSession) SetServiceUrl(service, url string) {
	this.customizeServiceUrl[service] = url
}

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
