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

package requests

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
)

type IRequest interface {
	GetScheme() string
	GetMethod() string
	GetDomain() string
	GetPort() string
	GetRegionId() string
	GetProjectId() string
	GetHost() string
	GetURI() string
	GetHeaders() map[string]string
	GetQueryParams() map[string]string
	GetFormParams() map[string]string
	GetContent() []byte
	GetBodyReader() io.Reader
	GetProduct() string
	GetVersion() string

	SetStringToSign(stringToSign string)
	GetStringToSign() string

	SetDomain(domain string)
	SetContent(content []byte)
	SetScheme(scheme string)
	BuildUrl() string
	BuildQueries() string

	AddHeaderParam(key, value string)
	AddQueryParam(key, value string)
	AddFormParam(key, value string)
}

type SRequest struct {
	Scheme   string // HTTP、HTTPS
	Method   string // GET、PUT、DELETE、POST、PATCH
	Domain   string // myhuaweicloud.com
	Port     string // 80
	RegionId string // cn-north-1

	product   string // 弹性云服务 ECS : ecs
	version   string // API版本： v2
	projectId string // 项目ID: 43cbe5e77aaf4665bbb962062dc1fc9d.可以为空。

	resourcePath string // /users/{user_id}/groups
	QueryParams  map[string]string
	Headers      map[string]string
	FormParams   map[string]string
	Content      []byte

	queries string

	stringToSign string
}

func (self *SRequest) GetProjectId() string {
	return self.projectId
}

func (self *SRequest) GetScheme() string {
	return self.Scheme
}

func (self *SRequest) GetMethod() string {
	return self.Method
}

func (self *SRequest) GetDomain() string {
	return self.Domain
}

func (self *SRequest) GetPort() string {
	return self.Port
}

func (self *SRequest) GetRegionId() string {
	return self.RegionId
}

func (self *SRequest) GetHeaders() map[string]string {
	return self.Headers
}

func (self *SRequest) GetQueryParams() map[string]string {
	return self.QueryParams
}

func (self *SRequest) GetFormParams() map[string]string {
	return self.FormParams
}

func (self *SRequest) GetContent() []byte {
	return self.Content
}

func (self *SRequest) GetBodyReader() io.Reader {
	if self.FormParams != nil && len(self.FormParams) > 0 {
		formData := GetUrlFormedMap(self.FormParams)
		return strings.NewReader(formData)
	} else {
		return strings.NewReader(string(self.Content))
	}
}

func (self *SRequest) GetProduct() string {
	return self.product
}

func (self *SRequest) GetVersion() string {
	return self.version
}

func (self *SRequest) SetStringToSign(stringToSign string) {
	self.stringToSign = stringToSign
}

func (self *SRequest) GetStringToSign() string {
	return self.stringToSign
}

func (self *SRequest) SetDomain(domain string) {
	self.Domain = domain
}

func (self *SRequest) SetContent(content []byte) {
	self.Content = content
}

func (self *SRequest) SetScheme(scheme string) {
	self.Scheme = scheme
}

func (self *SRequest) BuildUrl() string {
	scheme := strings.ToLower(self.Scheme)
	baseUrl := fmt.Sprintf("%s://%s", scheme, self.GetHost())
	queries := self.BuildQueries()
	if len(queries) > 0 {
		return baseUrl + self.GetURI() + "?" + queries
	} else {
		return baseUrl + self.GetURI()
	}
}

func (self *SRequest) GetHost() string {
	scheme := strings.ToLower(self.Scheme)
	host := self.getEndpoint()
	if len(self.Port) > 0 {
		if (scheme == "http" && self.Port == "80") || (scheme == "https" && self.Port == "443") {
			host = fmt.Sprintf("%s:%s", host, self.Port)
		}
	}

	return host
}

func (self *SRequest) GetURI() string {
	// URI
	uri := ""
	for _, m := range []string{self.version, self.projectId} {
		if len(m) > 0 {
			uri += fmt.Sprintf("/%s", m)
		}
	}

	if len(self.resourcePath) > 0 {
		s := ""
		if !strings.HasPrefix(self.resourcePath, "/") {
			s = "/"
		}

		if strings.HasSuffix(self.resourcePath, "/") {
			strings.TrimSuffix(s, "/")
		}

		uri = uri + s + self.resourcePath
	}

	return uri
}

func (self *SRequest) getEndpoint() string {
	// ecs.cn-north-1.myhuaweicloud.com
	items := []string{}
	for _, item := range []string{self.product, self.RegionId, self.Domain} {
		if len(item) > 0 {
			items = append(items, item)
		}
	}

	return strings.Join(items, ".")
}

func (self *SRequest) BuildQueries() string {
	self.queries = GetUrlFormedMap(self.QueryParams)
	return self.queries
}

func (self *SRequest) AddHeaderParam(key, value string) {
	self.Headers[key] = value
}

func (self *SRequest) AddQueryParam(key, value string) {
	self.QueryParams[key] = value
}

func (self *SRequest) AddFormParam(key, value string) {
	self.FormParams[key] = value
}

func GetUrlFormedMap(source map[string]string) string {
	// 按key排序后编译
	keys := make([]string, 0)
	for k := range source {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	urlEncoder := url.Values{}
	for _, k := range keys {
		urlEncoder.Add(k, source[k])
	}

	return urlEncoder.Encode()
}

func defaultRequest() (request *SRequest) {
	request = &SRequest{
		Scheme:      "HTTPS",
		Method:      "GET",
		QueryParams: make(map[string]string),
		Headers:     map[string]string{},
		FormParams:  make(map[string]string),
	}
	return
}

func NewResourceRequest(domain, method, product, version, region, project, resourcePath string) *SRequest {
	return &SRequest{
		Scheme:       "HTTPS",
		Method:       method,
		Domain:       domain,
		product:      product,
		RegionId:     region,
		version:      version,
		projectId:    project,
		resourcePath: resourcePath,
		QueryParams:  make(map[string]string),
		Headers:      map[string]string{},
		FormParams:   make(map[string]string),
	}
}
