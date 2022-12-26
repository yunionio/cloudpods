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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/auth"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/manager"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/requests"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/responses"
)

type IRequestHook interface {
	Process(r requests.IRequest)
}

type SBaseManager struct {
	cfg         manager.IManagerConfig
	httpClient  *http.Client
	requestHook IRequestHook // 用于对request做特殊处理。非必要请不要使用！！！。目前只有port接口用到。

	columns []string
	debug   bool
}

type sThrottlingThreshold struct {
	locked   bool
	lockTime time.Time
}

func (t *sThrottlingThreshold) CheckingLock() {
	if !t.locked {
		return
	}

	for {
		if t.lockTime.Sub(time.Now()).Seconds() < 0 {
			return
		}
		log.Debugf("throttling threshold has been reached. release at %s", t.lockTime)
		time.Sleep(5 * time.Second)
	}
}

func (t *sThrottlingThreshold) Lock() {
	// 锁定至少15秒
	t.locked = true
	t.lockTime = time.Now().Add(15 * time.Second)
}

var ThrottlingLock = sThrottlingThreshold{locked: false, lockTime: time.Time{}}

func NewBaseManager2(cfg manager.IManagerConfig, requesthk IRequestHook) SBaseManager {
	return SBaseManager{
		cfg:         cfg,
		httpClient:  httputils.GetDefaultClient(),
		debug:       cfg.GetDebug(),
		requestHook: requesthk,
	}
}

func NewBaseManager(cfg manager.IManagerConfig) SBaseManager {
	return NewBaseManager2(cfg, nil)
}

func (self *SBaseManager) GetEndpoint() string {
	return self.cfg.GetEndpoint()
}

func (self *SBaseManager) GetColumns() []string {
	return self.columns
}

func (self *SBaseManager) SetHttpClient(httpClient *http.Client) {
	self.httpClient = httpClient
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
		return nil, errors.Wrapf(err, "body.GetArray %s", responseKey)
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

	return &responses.ListResult{
		Data:   rets,
		Total:  int(total),
		Limit:  limit,
		Offset: offset,
	}, nil
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

type HuaweiClientError struct {
	Code      int
	Errorcode []string
	err       error
	Details   string
	ErrorCode string
}

func (ce *HuaweiClientError) Error() string {
	return jsonutils.Marshal(ce).String()
}

func (ce *HuaweiClientError) ParseErrorFromJsonResponse(statusCode int, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(ce)
	}
	if ce.Code == 0 {
		ce.Code = statusCode
	}
	if len(ce.Details) == 0 && body != nil {
		ce.Details = body.String()
	}
	return ce
}

func (self *SBaseManager) jsonRequest(request requests.IRequest) (http.Header, jsonutils.JSONObject, error) {
	ThrottlingLock.CheckingLock()
	ctx := context.Background()
	// hook request
	if self.requestHook != nil {
		self.requestHook.Process(request)
	}
	// 拼接、编译、签名 requests here。
	err := self.buildRequestWithSigner(request, self.cfg.GetSigner())
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

	client := httputils.NewJsonClient(self.httpClient)
	req := httputils.NewJsonRequest(httputils.THttpMethod(request.GetMethod()), request.BuildUrl(), jsonBody)
	req.SetHeader(header)
	resp := &HuaweiClientError{}
	const MAX_RETRY = 3
	retry := MAX_RETRY
	for {
		h, b, e := client.Send(ctx, req, resp, self.debug)
		if e == nil {
			return h, b, nil
		}

		log.Errorf("[%s] %s body: %v error: %v", req.GetHttpMethod(), req.GetUrl(), jsonBody, e)

		switch err := e.(type) {
		case *HuaweiClientError:
			if err.ErrorCode == "APIGW.0301" {
				return h, b, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, e.Error())
			} else if err.Code == 499 && retry > 0 && request.GetMethod() == "GET" {
				retry -= 1
				time.Sleep(3 * time.Second * time.Duration(MAX_RETRY-retry))
			} else if (err.Code == 404 || strings.Contains(err.Details, "could not be found") ||
				strings.Contains(err.Error(), "Not Found") ||
				strings.Contains(err.Details, "does not exist")) && request.GetMethod() != "POST" {
				return h, b, errors.Wrap(cloudprovider.ErrNotFound, err.Error())
			} else if err.Code == 429 && retry > 0 {
				// 当前请求过多。
				ThrottlingLock.Lock()
				retry -= 1
				time.Sleep(15 * time.Second)
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
