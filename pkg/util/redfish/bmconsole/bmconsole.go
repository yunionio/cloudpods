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

package bmconsole

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SBMCConsole struct {
	client *http.Client

	username string
	password string
	host     string

	isDebug bool
}

func NewBMCConsole(host, username, password string, isDebug bool) *SBMCConsole {
	client := httputils.GetDefaultClient()
	return &SBMCConsole{
		client:   client,
		host:     host,
		username: username,
		password: password,
		isDebug:  isDebug,
	}
}

func setCookieHeader(hdr http.Header, cookies map[string]string) {
	cookieParts := make([]string, 0)
	for k, v := range cookies {
		cookieParts = append(cookieParts, k+"="+v)
	}
	if len(cookieParts) > 0 {
		hdr.Set("Cookie", strings.Join(cookieParts, "; "))
	}
}

func (r *SBMCConsole) RawRequest(ctx context.Context, method httputils.THttpMethod, path string, header http.Header, body []byte) (http.Header, []byte, error) {
	urlStr := httputils.JoinPath(fmt.Sprintf("https://%s", r.host), path)
	if header == nil {
		header = http.Header{}
	}
	header.Set("Connection", "Close")
	header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.12; rv:69.0) Gecko/20100101 Firefox/69.0")
	resp, err := httputils.Request(r.client, ctx, method, urlStr, header, bytes.NewReader(body), r.isDebug)
	hdr, rspBody, err := httputils.ParseResponse(resp, err, r.isDebug)
	if err != nil {
		return nil, nil, errors.Wrap(err, "httputils.Request")
	}
	return hdr, rspBody, nil
}
