package modules

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/requests"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

type IRequestHook interface {
	Process(r requests.IRequest)
}

type BaseManager struct {
	signer      auth.Signer
	httpClient  *http.Client
	requestHook IRequestHook // 用于对request做特殊处理。非必要请不要使用！！！。目前只有port接口用到。

	columns []string
	debug   bool
}

func (self *BaseManager) GetColumns() []string {
	return self.columns
}

func (self *BaseManager) _list(request requests.IRequest, responseKey string) (*responses.ListResult, error) {
	_, body, err := self.jsonRequest(request)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("empty response")
	}

	rets, err := body.GetArray(responseKey)
	if err != nil {
		return nil, err
	}
	total, err := body.Int("count")
	if err != nil {
		total = int64(len(rets))
	}

	if total == 0 {
		total = int64(len(rets))
	}

	limit := 0
	v, exists := request.GetQueryParams()["limit"]
	if exists {
		limit, err = strconv.Atoi(v)
	}

	offset := 0
	v, exists = request.GetQueryParams()["offset"]
	if !exists {
		offset, err = strconv.Atoi(v)
	}

	return &responses.ListResult{rets, int(total), limit, offset}, nil
}

func (self *BaseManager) _do(request requests.IRequest, responseKey string) (jsonutils.JSONObject, error) {
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

func (self *BaseManager) _get(request requests.IRequest, responseKey string) (jsonutils.JSONObject, error) {
	return self._do(request, responseKey)
}

func (self *BaseManager) jsonRequest(request requests.IRequest) (http.Header, jsonutils.JSONObject, error) {
	ctx := context.Background()
	// hook request
	if self.requestHook != nil {
		self.requestHook.Process(request)
	}
	// 拼接、编译、签名 requests here。
	err := self.buildRequestWithSigner(request, self.signer)
	if err != nil {
		return nil, nil, fmt.Errorf(err.Error())
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

	// 发送 request。todo: 支持debug
	return httputils.JSONRequest(self.httpClient, ctx, httputils.THttpMethod(request.GetMethod()), request.BuildUrl(), header, jsonBody, self.debug)
}

func (self *BaseManager) rawRequest(request requests.IRequest) (*http.Response, error) {
	ctx := context.Background()
	// 拼接、编译requests here。
	header := http.Header{}
	for k, v := range request.GetHeaders() {
		header.Set(k, v)
	}
	return httputils.Request(self.httpClient, ctx, httputils.THttpMethod(request.GetMethod()), request.BuildUrl(), header, request.GetBodyReader(), self.debug)
}

func (self *BaseManager) buildRequestWithSigner(request requests.IRequest, signer auth.Signer) error {
	return auth.Sign(request, signer)
}
