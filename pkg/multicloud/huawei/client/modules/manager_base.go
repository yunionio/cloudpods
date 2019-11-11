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

package modules

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/requests"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/responses"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type IRequestHook interface {
	Process(r requests.IRequest)
}

type SBaseManager struct {
	signer      auth.Signer
	httpClient  *http.Client
	requestHook IRequestHook // 用于对request做特殊处理。非必要请不要使用！！！。目前只有port接口用到。

	columns []string
	debug   bool
}

func NewBaseManager2(signer auth.Signer, debug bool, requesthk IRequestHook) SBaseManager {
	return SBaseManager{
		signer:      signer,
		httpClient:  httputils.GetDefaultClient(),
		debug:       debug,
		requestHook: requesthk,
	}
}

func NewBaseManager(signer auth.Signer, debug bool) SBaseManager {
	return NewBaseManager2(signer, debug, nil)
}

func (self *SBaseManager) GetColumns() []string {
	return self.columns
}

func (self *SBaseManager) _list(request requests.IRequest, responseKey string) (*responses.ListResult, error) {
	_, body, err := self.jsonRequest(request)
	if err != nil {
		return nil, err
	}
	if body == nil {
		log.Warningf("empty response")
		return &responses.ListResult{}, nil
	}

	rets, err := body.GetArray(responseKey)
	if err != nil {
		return nil, err
	}
	total, _ := body.Int("count")
	// if err != nil {
	//	total = int64(len(rets))
	//}

	//if total == 0 {
	//	total = int64(len(rets))
	//}

	limit := 0
	if v, exists := request.GetQueryParams()["limit"]; exists {
		limit, _ = strconv.Atoi(v)
	}

	offset := 0
	if v, exists := request.GetQueryParams()["offset"]; exists {
		offset, _ = strconv.Atoi(v)
	}

	return &responses.ListResult{rets, int(total), limit, offset}, nil
}

func (self *SBaseManager) _do(request requests.IRequest, responseKey string) (jsonutils.JSONObject, error) {
	_, resp, e := self.jsonRequest(request)
	if e != nil {
		return nil, e
	}

	if resp == nil { // no reslt
		return jsonutils.NewDict(), nil
	}

	if len(responseKey) == 0 {
		return resp, nil
	}

	ret, e := resp.Get(responseKey)
	if e != nil {
		return nil, e
	}

	return ret, nil
}

func (self *SBaseManager) _get(request requests.IRequest, responseKey string) (jsonutils.JSONObject, error) {
	return self._do(request, responseKey)
}

func (self *SBaseManager) jsonRequest(request requests.IRequest) (http.Header, jsonutils.JSONObject, error) {
	ctx := context.Background()
	// hook request
	if self.requestHook != nil {
		self.requestHook.Process(request)
	}
	// 拼接、编译、签名 requests here。
	err := self.buildRequestWithSigner(request, self.signer)
	if err != nil {
		return nil, nil, err
	}
	header := http.Header{}
	for k, v := range request.GetHeaders() {
		header.Set(k, v)
	}

	var jsonBody jsonutils.JSONObject
	content := request.GetContent()
	if len(content) > 0 {
		jsonBody, err = jsonutils.Parse(content)
		if err != nil {
			return nil, nil, fmt.Errorf("not a json body")
		}
	}

	if self.debug {
		log.Debugf("url: %s", request.BuildUrl())
	}

	// 发送 request。todo: 支持debug
	const MAX_RETRY = 3
	retry := MAX_RETRY
	for {
		h, b, e := httputils.JSONRequest(self.httpClient, ctx, httputils.THttpMethod(request.GetMethod()), request.BuildUrl(), header, jsonBody, self.debug)
		if e == nil {
			if self.debug {
				log.Debugf("response: %s body: %s", h, b)
			}
			return h, b, e
		}

		switch err := e.(type) {
		case *httputils.JSONClientError:
			if err.Code == 499 && retry > 0 && request.GetMethod() == "GET" {
				retry -= 1
				time.Sleep(time.Second * time.Duration(MAX_RETRY-retry))
			} else if (err.Code == 404 || strings.Index(err.Details, "could not be found") > 0) && request.GetMethod() != "POST" {
				return h, b, cloudprovider.ErrNotFound
			} else {
				return h, b, e
			}
		default:
			return h, b, e
		}
	}
}

func (self *SBaseManager) rawRequest(request requests.IRequest) (*http.Response, error) {
	ctx := context.Background()
	// 拼接、编译requests here。
	header := http.Header{}
	for k, v := range request.GetHeaders() {
		header.Set(k, v)
	}
	return httputils.Request(self.httpClient, ctx, httputils.THttpMethod(request.GetMethod()), request.BuildUrl(), header, request.GetBodyReader(), self.debug)
}

func (self *SBaseManager) buildRequestWithSigner(request requests.IRequest, signer auth.Signer) error {
	return auth.Sign(request, signer)
}
