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

func (r *SBMCConsole) GetIdrac6ConsoleJNLP(ctx context.Context, sku, model string) (string, error) {
	cookies := make(map[string]string)

	hdr, _, err := r.RawRequest(ctx, httputils.GET, "/start.html", nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "r.Get start.html")
	}
	log.Debugf("start.html hdr %s", hdr)
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

	loginData := strings.Join([]string{
		"user=" + url.QueryEscape(r.username),
		"password=" + url.QueryEscape(strings.ReplaceAll(r.password, "@", "@040")),
	}, "&")

	postHdr := http.Header{}
	postHdr.Set("Content-Type", "application/x-www-form-urlencoded")
	setCookieHeader(postHdr, cookies)
	_, loginResp, err := r.RawRequest(ctx, httputils.POST, "/data/login", postHdr, []byte(loginData))
	if err != nil {
		return "", errors.Wrap(err, "r.FormPost Login")
	}
	log.Debugf("LoginResp: %s", loginResp)
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

	cookies["batteriesIcon"] = "status_normal"
	cookies["fansIcon"] = "status_normal"
	cookies["intrusionIcon"] = "status_normal"
	cookies["powerSuppliesIcon"] = "status_normal"
	cookies["removableFlashMediaIcon"] = "status_normal"
	cookies["temperaturesIcon"] = "status_normal"
	cookies["voltagesIcon"] = "status_normal"

	getHdr := http.Header{}
	setCookieHeader(getHdr, cookies)
	getHdr.Set("Referer", fmt.Sprintf("https://%s/sysSummaryData.html", r.host))
	getHdr.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	getHdr.Set("Accept-Encoding", "gzip, deflate, br")
	getHdr.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	sysStr := url.QueryEscape(fmt.Sprintf("idrac-%s, %s, User:%s", sku, model, r.username))
	// sysStr := url.QueryEscape(fmt.Sprintf("idrac-%s, %s,  slot , &#29992;&#25143;&#65306; %s", sku, model, r.username))
	path := fmt.Sprintf("viewer.jnlp(%s@0@%s@%d)", r.host, sysStr, time.Now().UnixNano()/1000000)

	_, rspBody, err := r.RawRequest(ctx, httputils.GET, path, getHdr, nil)
	if err != nil {
		return "", errors.Wrapf(err, "r.RawGet %s", path)
	}
	return string(rspBody), nil
}
