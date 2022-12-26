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
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
)

func (r *SBMCConsole) GetSupermicroConsoleJNLP(ctx context.Context) (string, error) {
	loginData := strings.Join([]string{
		"name=" + url.QueryEscape(r.username),
		"pwd=" + url.QueryEscape(r.password),
	}, "&")

	// cookie:
	// -http-session-=::http.session::0103fd02ceac2d642361b6fdcd4a5994;
	// sysidledicon=ledIcon%20grayLed;
	// tokenvalue=478be97abdaeb4d454c0418fcca9094d

	cookies := make(map[string]string)

	// first do html login
	postHdr := http.Header{}
	postHdr.Set("Content-Type", "application/x-www-form-urlencoded")
	postHdr.Set("Referer", fmt.Sprintf("http://%s/", r.host))
	setCookieHeader(postHdr, cookies)
	hdr, _, err := r.RawRequest(ctx, httputils.POST, "/cgi/login.cgi", postHdr, []byte(loginData))
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
	getHdr.Set("Referer", fmt.Sprintf("https://%s/cgi/url_redirect.cgi?url_name=man_ikvm", r.host))
	setCookieHeader(getHdr, cookies)

	// now := time.Now()
	// fwtype=255&time_stamp=Thu%20Apr%2023%202020%2002%3A25%3A13%20GMT%2B0800%20(%E4%B8%AD%E5%9B%BD%E6%A0%87%E5%87%86%E6%97%B6%E9%97%B4)&_=
	/*loginData = strings.Join([]string{
		"fwtype=255",
		"time_stamp=" + url.QueryEscape(timeutils.RFC2882Time(now)),
		"_=",
	}, "&")

	_, rspBody, err := r.RawRequest(ctx, httputils.POST, "/cgi/upgrade_process.cgi", getHdr, []byte(loginData))
	if err != nil {
		return "", errors.Wrapf(err, "r.RawPost %s", loginData)
	}

	if r.isDebug {
		log.Debugf("upgrade_process.cgi %s", rspBody)
	}
	*/

	_, rspBody, err := r.RawRequest(ctx, httputils.GET, "/cgi/url_redirect.cgi?url_name=ikvm&url_type=jwsk", getHdr, nil)
	if err != nil {
		return "", errors.Wrap(err, "r.RawGet")
	}

	return string(rspBody), nil
}
