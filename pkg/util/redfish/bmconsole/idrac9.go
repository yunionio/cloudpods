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
	"context"
	"net/http"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
)

const (
	IDRAC9_LOGIN_URL     = "/sysmgmt/2015/bmc/session"
	IDRAC9_VCONSOLE_JAVA = "/sysmgmt/2015/server/vconsole?type=Java"
)

func (r *SBMCConsole) GetIdrac9ConsoleJNLP(ctx context.Context) (string, error) {
	cookies := make(map[string]string)
	cookies["-http-session-"] = ""

	// first do html login
	postHdr := http.Header{}
	postHdr.Set("password", r.password)
	postHdr.Set("user", r.username)
	postHdr.Set("Content-Length", "0")
	setCookieHeader(postHdr, cookies)
	hdr, _, err := r.RawRequest(ctx, httputils.POST, IDRAC9_LOGIN_URL, postHdr, nil)
	if err != nil {
		return "", errors.Wrap(err, "r.FormPost Login")
	}
	for _, cookieHdr := range hdr["Set-Cookie"] {
		parts := strings.Split(cookieHdr, ";")
		if len(parts) > 0 {
			pparts := strings.Split(parts[0], "=")
			if len(pparts) > 1 {
				cookies[pparts[0]] = pparts[1]
			}
		}
	}
	// XSRF-TOKEN: a636651fcc8841a3c146cea89730389c
	xsrfToken := hdr.Get("Xsrf-Token")

	getHdr := http.Header{}
	setCookieHeader(getHdr, cookies)
	getHdr.Set("Xsrf-Token", xsrfToken)
	getHdr.Set("Sec-Fetch-Dest", "empty")
	getHdr.Set("Sec-Fetch-Mode", "cors")
	getHdr.Set("Sec-Fetch-Site", "same-origin")
	_, rspBody, err := r.RawRequest(ctx, httputils.GET, IDRAC9_VCONSOLE_JAVA, getHdr, nil)
	if err != nil {
		return "", errors.Wrap(err, "r.RawGet")
	}

	return string(rspBody), nil
}
