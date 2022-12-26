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

package ecloud

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type IRequest interface {
	GetScheme() string
	GetMethod() string
	SetMethod(string)
	GetEndpoint() string
	GetServerPath() string
	GetPort() string
	GetRegionId() string
	GetHeaders() map[string]string
	GetQueryParams() map[string]string
	GetBodyReader() io.Reader
	GetVersion() string
	GetTimestamp() string
	GetReadTimeout() time.Duration
	GetConnectTimeout() time.Duration
	GetHTTPSInsecure() bool
	SetHTTPSInsecure(bool)
	GetUserAgent() map[string]string
	ForMateResponseBody(jrbody jsonutils.JSONObject) (jsonutils.JSONObject, error)
}

type SJSONRequest struct {
	SBaseRequest
	Data jsonutils.JSONObject
}

func NewJSONRequest(data jsonutils.JSONObject) *SJSONRequest {
	jr := &SJSONRequest{
		Data:         data,
		SBaseRequest: *NewBaseRequest(),
	}
	headers := jr.GetHeaders()
	headers["Content-Type"] = "application/json"
	return jr
}

func (jr *SJSONRequest) GetBodyReader() io.Reader {
	if jr.Data == nil {
		return nil
	}
	return strings.NewReader(jr.Data.String())
}

type SApiRequest struct {
	SJSONRequest
	RegionId string
}

func NewApiRequest(regionId string, serverPath string, query map[string]string, data jsonutils.JSONObject) *SApiRequest {
	r := SApiRequest{
		RegionId:     regionId,
		SJSONRequest: *NewJSONRequest(data),
	}
	r.ServerPath = serverPath
	mergeMap(r.GetQueryParams(), query)
	return &r
}

func (rr *SApiRequest) GetScheme() string {
	return "https"
}

func (rr *SApiRequest) GetPort() string {
	return "8443"
}

func (rr *SApiRequest) GetEndpoint() string {
	return fmt.Sprintf("api-%s.cmecloud.cn", rr.RegionId)
}

type SConsoleRequest struct {
	SJSONRequest
	RegionId string
}

func NewConsoleRequest(regionId string, serverPath string, query map[string]string, data jsonutils.JSONObject) *SConsoleRequest {
	r := SConsoleRequest{
		RegionId:     regionId,
		SJSONRequest: *NewJSONRequest(data),
	}
	r.ServerPath = serverPath
	mergeMap(r.GetQueryParams(), query)
	return &r
}

func (rr *SConsoleRequest) GetScheme() string {
	return "https"
}

func (rr *SConsoleRequest) GetPort() string {
	return "8443"
}

func (rr *SConsoleRequest) GetEndpoint() string {
	return fmt.Sprintf("console-%s.cmecloud.cn", rr.RegionId)
}

func mergeMap(m1, m2 map[string]string) {
	if m2 == nil {
		return
	}
	for k, v := range m2 {
		m1[k] = v
	}
}

type SBaseRequest struct {
	Scheme         string
	Method         string
	Endpoint       string
	ServerPath     string
	Port           string
	RegionId       string
	ReadTimeout    time.Duration
	ConnectTimeout time.Duration
	isInsecure     *bool

	QueryParams map[string]string
	Headers     map[string]string
	Content     []byte
}

func NewBaseRequest() *SBaseRequest {
	return &SBaseRequest{
		QueryParams: map[string]string{},
		Headers:     map[string]string{},
	}
}

func (br *SBaseRequest) GetScheme() string {
	return br.Scheme
}

func (br *SBaseRequest) GetMethod() string {
	return br.Method
}

func (br *SBaseRequest) SetMethod(method string) {
	br.Method = method
}

func (br *SBaseRequest) GetEndpoint() string {
	return br.Endpoint
}

func (br *SBaseRequest) GetServerPath() string {
	return br.ServerPath
}

func (br *SBaseRequest) GetPort() string {
	return br.Port
}

func (br *SBaseRequest) GetRegionId() string {
	return br.RegionId
}

func (br *SBaseRequest) GetHeaders() map[string]string {
	return br.Headers
}

func (br *SBaseRequest) GetQueryParams() map[string]string {
	return br.QueryParams
}

func (br *SBaseRequest) GetBodyReader() io.Reader {
	if len(br.Content) > 0 {
		return bytes.NewReader(br.Content)
	}
	return nil
}

func (br *SBaseRequest) GetVersion() string {
	return "2016-12-05"
}

func (br *SBaseRequest) GetTimestamp() string {
	sh, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(sh).Format("2006-01-02T15:04:05Z")
}

func (br *SBaseRequest) GetReadTimeout() time.Duration {
	return br.ReadTimeout
}

func (br *SBaseRequest) GetConnectTimeout() time.Duration {
	return br.ConnectTimeout
}

func (br *SBaseRequest) GetHTTPSInsecure() bool {
	if br.isInsecure == nil {
		return false
	}
	return *br.isInsecure
}

func (br *SBaseRequest) SetHTTPSInsecure(isInsecure bool) {
	br.isInsecure = &isInsecure
}

func (br *SBaseRequest) GetUserAgent() map[string]string {
	return nil
}

func (br *SBaseRequest) ForMateResponseBody(jrbody jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if jrbody == nil || !jrbody.Contains("state") {
		return nil, ErrMissKey{
			Key: "state",
			Jo:  jrbody,
		}
	}
	state, _ := jrbody.GetString("state")
	switch state {
	case "OK":
		if !jrbody.Contains("body") {
			return nil, ErrMissKey{
				Key: "body",
				Jo:  jrbody,
			}
		}
		body, _ := jrbody.Get("body")
		return body, nil
	default:
		if jrbody.Contains("errorMessage") {
			msg, _ := jrbody.GetString("errorMessage")
			if strings.Contains(msg, "Invalid parameter AccessKey") {
				return nil, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, msg)
			}
			return nil, &httputils.JSONClientError{Code: 400, Details: msg}
		}
		return nil, &httputils.JSONClientError{Code: 400, Details: jrbody.String()}
	}
}
