// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package smsdriver

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	utilerr "yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
)

const (
	LARK_MESSAGE_SUCCESS = "success"
)

var (
	cli = &http.Client{
		Transport: httputils.GetTransport(true),
	}
	ctx = context.Background()
)

// 通知请求
func sendRequest(url string, method httputils.THttpMethod, header http.Header, params url.Values, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var bodystr string
	if !gotypes.IsNil(body) {
		bodystr = body.String()
	}
	jbody := strings.NewReader(bodystr)
	if header == nil {
		header = http.Header{}
	}
	if header == nil {
		header = http.Header{}
	}
	if len(header.Values("Content-Type")) == 0 {
		header.Set("Content-Type", "application/json")
	}
	if params != nil {
		url += params.Encode()
	}
	resp, err := httputils.Request(cli, ctx, method, url, header, jbody, true)
	if err != nil {
		return nil, utilerr.Wrap(err, "http request")
	}
	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	rbody = bytes.TrimSpace(rbody)

	var jrbody jsonutils.JSONObject = nil
	if len(rbody) > 0 && (rbody[0] == '{' || rbody[0] == '[') {
		jrbody, _ = jsonutils.Parse(rbody)
	} else {
		return nil, errors.Wrap(err, "resource not json")
	}
	return jrbody, nil
}
