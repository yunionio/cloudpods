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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

func (r *SBMCConsole) GetIdracConsoleJNLP(ctx context.Context, sku, model string) (string, error) {
	loginData := strings.Join([]string{
		"user=" + url.QueryEscape(r.username),
		"password=" + url.QueryEscape(r.password),
	}, "&")

	// cookie:
	// -http-session-=::http.session::0103fd02ceac2d642361b6fdcd4a5994;
	// sysidledicon=ledIcon%20grayLed;
	// tokenvalue=478be97abdaeb4d454c0418fcca9094d

	cookies := make(map[string]string)
	cookies["-http-session-"] = ""

	// first do html login
	postHdr := http.Header{}
	postHdr.Set("Content-Type", "application/x-www-form-urlencoded")
	setCookieHeader(postHdr, cookies)
	hdr, loginResp, err := r.RawRequest(ctx, httputils.POST, "/data/login", postHdr, []byte(loginData))
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
	forwardUrlPattern := regexp.MustCompile(`<forwardUrl>(.*)</forwardUrl>`)
	matched := forwardUrlPattern.FindAllStringSubmatch(string(loginResp), -1)
	indexUrlStr := ""
	if len(matched) > 0 && len(matched[0]) > 1 {
		indexUrlStr = matched[0][1]
	}
	if len(indexUrlStr) == 0 {
		return "", errors.Wrapf(httperrors.ErrBadRequest, "no valid forwardUrl")
	}

	tokenPattern := regexp.MustCompile(`ST1=(\w+),ST2=`)
	matched = tokenPattern.FindAllStringSubmatch(indexUrlStr, -1)
	log.Debugf("%s", matched)
	token := ""
	if len(matched) > 0 && len(matched[0]) > 1 {
		token = matched[0][1]
	}
	cookies["tokenvalue"] = token

	getHdr := http.Header{}
	setCookieHeader(getHdr, cookies)

	sysStr := url.QueryEscape(fmt.Sprintf("idrac-%s, %s, User: %s", sku, model, r.username))
	path := fmt.Sprintf("viewer.jnlp(%s@0@%s@%d@ST1=%s)", r.host, sysStr, time.Now().UnixNano()/1000000, token)

	_, rspBody, err := r.RawRequest(ctx, httputils.GET, path, getHdr, nil)
	if err != nil {
		return "", errors.Wrapf(err, "r.RawGet %s", path)
	}
	return string(rspBody), nil
}
