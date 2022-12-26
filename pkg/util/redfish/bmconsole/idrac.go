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
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/httperrors"
)

func (r *SBMCConsole) GetIdracConsoleJNLP(ctx context.Context, sku, model string) (string, error) {
	if len(model) > 0 {
		parts := strings.Split(model, " ")
		if len(parts) > 1 && parts[1][0] == 'R' && len(parts[1]) >= 4 {
			// PownerEdge R730
			switch parts[1][2] {
			case '1':
				return r.GetIdrac6ConsoleJNLP(ctx, sku, model)
			case '2', '3':
				return r.GetIdrac7ConsoleJNLP(ctx, sku, model)
			default:
				return r.GetIdrac9ConsoleJNLP(ctx)
			}
		}
	}
	jnlp, err := r.GetIdrac7ConsoleJNLP(ctx, sku, model)
	if err == nil {
		return jnlp, nil
	}
	jnlp, err = r.GetIdrac9ConsoleJNLP(ctx)
	if err == nil {
		return jnlp, nil
	}
	return r.GetIdrac6ConsoleJNLP(ctx, sku, model)
}

func (r *SBMCConsole) GetIdrac7ConsoleJNLP(ctx context.Context, sku, model string) (string, error) {
	loginData := strings.Join([]string{
		"user=" + url.QueryEscape(r.username),
		"password=" + url.QueryEscape(r.password),
	}, "&")

	// cookie:
	// -http-session-=::http.session::0103fd02ceac2d642361b6fdcd4a5994;
	// sysidledicon=ledIcon%20grayLed;
	// tokenvalue=478be97abdaeb4d454c0418fcca9094d

	cookies := make(map[string]string)
	// cookies["-http-session-"] = ""

	// first do html login
	postHdr := http.Header{}
	postHdr.Set("Content-Type", "application/x-www-form-urlencoded")
	setCookieHeader(postHdr, cookies)
	hdr, loginResp, err := r.RawRequest(ctx, httputils.POST, "/data/login", postHdr, []byte(loginData))
	if err != nil {
		return "", errors.Wrap(err, "r.FormPost Login")
	}
	log.Debugf("Header: %s %s", hdr, loginResp)
	if setCookies, ok := hdr["Set-Cookie"]; ok {
		for _, cookieHdr := range setCookies {
			parts := strings.Split(cookieHdr, ";")
			if len(parts) > 0 {
				pparts := strings.Split(parts[0], "=")
				if len(pparts) > 1 {
					cookies[pparts[0]] = pparts[1]
				}
			}
		}
	} else {
		// find no cookie
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
	log.Debugf("token: %s", token)
	cookies["tokenvalue"] = token
	cookies["batteriesIcon"] = "status_ok"
	cookies["fansIcon"] = "status_ok"
	cookies["intrusionIcon"] = "status_ok"
	cookies["removableFlashMediaIcon"] = "status_ok"
	cookies["temperaturesIcon"] = "status_ok"
	cookies["voltagesIcon"] = "status_ok"
	cookies["powerSuppliesIcon"] = "status_ok"
	cookies["sysidledicon"] = "ledIcon grayLed"

	getHdr := http.Header{}
	setCookieHeader(getHdr, cookies)
	getHdr.Set("Referer", fmt.Sprintf("https://%s/sysSummaryData.html", r.host))
	getHdr.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	getHdr.Set("Accept-Encoding", "gzip, deflate, br")
	getHdr.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	sysStr := url.QueryEscape(fmt.Sprintf("idrac-%s, %s,  slot , User: %s", sku, model, r.username))
	// sysStr := url.QueryEscape(fmt.Sprintf("idrac-%s, %s,  slot , &#29992;&#25143;&#65306; %s", sku, model, r.username))
	var path string
	if len(token) > 0 {
		path = fmt.Sprintf("viewer.jnlp(%s@0@%s@%d@ST1=%s)", r.host, sysStr, time.Now().UnixNano()/1000000, token)
	} else {
		path = fmt.Sprintf("viewer.jnlp(%s@0@%s@%d)", r.host, sysStr, time.Now().UnixNano()/1000000)
	}

	_, rspBody, err := r.RawRequest(ctx, httputils.GET, path, getHdr, nil)
	if err != nil {
		return "", errors.Wrapf(err, "r.RawGet %s", path)
	}
	return string(rspBody), nil
}
