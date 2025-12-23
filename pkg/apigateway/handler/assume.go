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

package handler

import (
	"context"
	"net/http"
	"strings"

	"yunion.io/x/onecloud/pkg/apigateway/clientman"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/pkg/errors"
)

const (
	ASSUME_TOKEN_HEADER = "X-Assume-Token"
)

type SAssumeToken struct {
	Token     string `json:"token"`
	UserId    string `json:"user_id"`
	ProjectId string `json:"project_id"`
}

func getAssumeToken(r *http.Request) (*SAssumeToken, error) {
	auth := r.Header.Get(ASSUME_TOKEN_HEADER)
	if auth == "" {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "Assume token header is empty")
	}
	parts := strings.Split(strings.TrimSpace(auth), ":")
	if len(parts) != 3 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "Assume token header is invalid")
	}
	return &SAssumeToken{
		Token:     parts[0],
		UserId:    parts[1],
		ProjectId: parts[2],
	}, nil
}

func doAssumeLogin(assumeToken *SAssumeToken, w http.ResponseWriter, req *http.Request) (mcclient.TokenCredential, error) {
	cliIp := netutils2.GetHttpRequestIp(req)
	token, err := auth.Client().AuthenticateAssume(assumeToken.Token, assumeToken.UserId, assumeToken.ProjectId, cliIp)
	if err != nil {
		return nil, errors.Wrapf(httperrors.ErrInvalidCredential, "AuthenticateAssume fail %s", err)
	}
	authToken := clientman.NewAuthToken(token.GetTokenString(), false, false, false)
	saveLoginCookies(w, authToken, token, nil)
	return token, nil
}

func (h *AuthHandlers) handleAssumeLogin(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// no auth cookie, try to get get assume user token from header
	assumeToken, err := getAssumeToken(req)
	if err != nil {
		appsrv.Send(w, err.Error())
		return
	}
	token, err := doAssumeLogin(assumeToken, w, req)
	if err != nil {
		appsrv.Send(w, err.Error())
		return
	}
	_, query, _ := appsrv.FetchEnv(ctx, w, req)
	redirect := ""
	if query != nil {
		queryMap, err := query.GetMap()
		if err != nil {
			appsrv.Send(w, err.Error())
			return
		}
		for k, v := range queryMap {
			if k == "p" {
				redirect, _ = v.GetString()
			} else {
				value, _ := v.GetString()
				saveCookie(w, k, value, "", token.GetExpires(), false)
			}
		}
	}
	if !strings.HasPrefix(redirect, "/") {
		redirect = "/" + redirect
	}
	appsrv.SendRedirect(w, redirect)
}
