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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/httputils"
)

func (r *SBMCConsole) GetIloConsoleJNLP(ctx context.Context) (string, error) {
	loginData := jsonutils.NewDict()
	loginData.Add(jsonutils.NewString("login"), "method")
	loginData.Add(jsonutils.NewString(r.username), "user_login")
	loginData.Add(jsonutils.NewString(r.password), "password")

	postHdr := http.Header{}
	postHdr.Set("Content-Type", "application/json")
	_, loginRespBytes, err := r.RawRequest(ctx, httputils.POST, "/json/login_session", postHdr, []byte(loginData.String()))
	if err != nil {
		return "", errors.Wrap(err, "r.FormPost Login")
	}

	loginRespJson, err := jsonutils.Parse(loginRespBytes)
	if err != nil {
		return "", errors.Wrap(err, "jsonutils.Parse loginRespBytes")
	}

	sessionKey, err := loginRespJson.GetString("session_key")
	if err != nil {
		return "", errors.Wrap(err, "Get session_key")
	}

	endpoint := fmt.Sprintf("https://%s/", r.host)

	cookies := make(map[string]string)
	cookies["sessionKey"] = sessionKey
	cookies["sessionLang"] = "en"
	cookies["sessionUrl"] = endpoint

	getHdr := http.Header{}
	setCookieHeader(getHdr, cookies)
	_, tempBytes, err := r.RawRequest(ctx, httputils.GET, "/html/jnlp_template.html", getHdr, nil)
	if err != nil {
		return "", errors.Wrap(err, "request template")
	}

	startToken := []byte("<![CDATA[\n")
	endToken := []byte("]]>")
	pos := bytes.Index(tempBytes, startToken)
	if pos < 0 {
		return "", errors.Wrapf(err, "invalid template content %s: no start token", tempBytes)
	}
	tempBytes = tempBytes[pos+len(startToken):]
	pos = bytes.Index(tempBytes, endToken)
	if pos < 0 {
		return "", errors.Wrapf(err, "invalid template content %s: no end token", tempBytes)
	}
	template := string(tempBytes[:pos])

	// replace variables
	template = strings.ReplaceAll(template, "<%= this.baseUrl %>", endpoint)
	template = strings.ReplaceAll(template, "<%= this.sessionKey %>", sessionKey)
	template = strings.ReplaceAll(template, "<%= this.langId %>", "en")

	return template, nil
}
