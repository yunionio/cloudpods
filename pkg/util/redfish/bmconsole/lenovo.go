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
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/httperrors"
)

func (r *SBMCConsole) GetLenovoConsoleJNLP(ctx context.Context) (string, error) {
	loginData := strings.Join([]string{
		"user=" + url.QueryEscape(r.username),
		"password=" + url.QueryEscape(r.password),
	}, "&")

	// cookie:
	// _appwebSessionId_=09eb9a178d520d2c9fa1430dd355dc27; path=/; httponly; secure

	cookies := make(map[string]string)
	cookies["_appwebSessionId_"] = ""

	// first do html login
	postHdr := http.Header{}
	postHdr.Set("Content-Type", "application/x-www-form-urlencoded")
	postHdr.Set("Referer", fmt.Sprintf("https://%s/login.html", r.host))
	setCookieHeader(postHdr, cookies)
	hdr, _, err := r.RawRequest(ctx, httputils.POST, "/data/login", postHdr, []byte(loginData))
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

	getHdr := http.Header{}
	getHdr.Set("Referer", fmt.Sprintf("https://%s/bmctree.html", r.host))
	setCookieHeader(getHdr, cookies)
	_, launchResp, err := r.RawRequest(ctx, httputils.GET, "/vkvmLaunch.html", getHdr, nil)
	if err != nil {
		return "", errors.Wrap(err, "Get vkvmLauch.html")
	}

	var token string
	st3Pattern := regexp.MustCompile(`<input type="hidden" name="ST3" value="(\w+)">`)
	matched := st3Pattern.FindAllStringSubmatch(string(launchResp), -1)
	if len(matched) > 0 && len(matched[0]) > 1 {
		token = matched[0][1]
	}

	if len(token) == 0 {
		return "", errors.Wrap(httperrors.ErrBadRequest, "no valid ST3 token")
	}

	path := fmt.Sprintf("viewer.jnlp(%s@0@%d)", r.host, time.Now().UnixNano()/1000000)
	body := "ST3=" + url.QueryEscape(token)
	getHdr.Set("Referer", fmt.Sprintf("https://%s/vkvmLaunch.html", r.host))
	_, rspBody, err := r.RawRequest(ctx, httputils.POST, path, getHdr, []byte(body))
	if err != nil {
		return "", errors.Wrapf(err, "r.RawGet %s", path)
	}
	return string(rspBody), nil
}
