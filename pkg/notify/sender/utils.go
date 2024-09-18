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
package sender

import (
	"context"
	"net/http"
	"net/url"

	"yunion.io/x/jsonutils"
	utilerr "yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/notify/options"
)

const (
	LARK_MESSAGE_SUCCESS = "success"
)

var (
	cli = &http.Client{
		Transport: httputils.GetTransport(true),
	}
)

// 通知请求
func sendRequest(ctx context.Context, url string, method httputils.THttpMethod, header http.Header, params url.Values, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if header == nil {
		header = http.Header{}
	}
	if len(header.Values("Content-Type")) == 0 {
		header.Set("Content-Type", "application/json")
	}
	if params != nil {
		url += params.Encode()
	}
	_, resp, err := httputils.JSONRequest(cli, ctx, method, url, header, body, options.Options.DebugRequest)
	if err != nil {
		return resp, utilerr.Wrap(err, "http request")
	}
	return resp, nil
}
