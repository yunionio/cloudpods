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

package bingoiam

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
	"yunion.io/x/onecloud/pkg/keystone/models"
)

type SBingoIAMOAuth2Driver struct {
	oauth2.SOAuth2BaseDriver

	idp         *models.SIdentityProvider
	accessToken string
	domains     map[string]*models.SDomain
	projects    map[string]*models.SProject
}

func NewBingoIAMOAuth2Driver(appId string, secret string) oauth2.IOAuth2Driver {
	drv := &SBingoIAMOAuth2Driver{
		domains:  map[string]*models.SDomain{},
		projects: map[string]*models.SProject{},
		SOAuth2BaseDriver: oauth2.SOAuth2BaseDriver{
			AppId:  appId,
			Secret: secret,
		},
	}
	return drv
}

func (drv *SBingoIAMOAuth2Driver) getIAMEndpoint(ctx context.Context) string {
	endpoint := ""
	if config, isOk := ctx.Value("config").(identity.TConfigs); isOk {
		endpoint = strings.Trim(config["oauth2"]["iam_endpoint"].String(), `"`)
	}
	return endpoint
}

func (drv *SBingoIAMOAuth2Driver) getIAMApiEndpoint(ctx context.Context) string {
	endpoint := ""
	if config, isOk := ctx.Value("config").(identity.TConfigs); isOk {
		endpoint = strings.Trim(config["oauth2"]["iam_api_endpoint"].String(), `"`)
	}
	return endpoint
}

func (drv *SBingoIAMOAuth2Driver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	req := map[string]string{
		"client_id":     drv.AppId,
		"redirect_uri":  callbackUrl,
		"response_type": "code id_token",
		"state":         state,
	}
	authUrl := drv.getIAMEndpoint(ctx)
	urlStr := fmt.Sprintf("%s/oauth2/authorize?%s", authUrl, jsonutils.Marshal(req).QueryString())
	return urlStr, nil
}

func (drv *SBingoIAMOAuth2Driver) Authenticate(ctx context.Context, code string) (map[string][]string, error) {
	accessToken, err := drv.fetchAccessToken(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "fetch bingo iam accessToken")
	}
	userInfo, err := drv.fetchUserInfo(ctx, accessToken)
	if err != nil {
		return nil, errors.Wrap(err, "fetch bingo iam userInfo")
	}
	attrs := make(map[string][]string)
	attrs["name"] = []string{fmt.Sprintf("%s", userInfo["username"])}
	attrs["display_name"] = []string{fmt.Sprintf("%s", userInfo["name"])}
	attrs["user_id"] = []string{fmt.Sprintf("%s", userInfo["sub"])}
	attrs["name_en"] = []string{fmt.Sprintf("%s", userInfo["username"])}
	attrs["email"] = []string{fmt.Sprintf("%s", userInfo["email"])}
	attrs["org_id"] = []string{fmt.Sprintf("%s", userInfo["org_id"])}
	attrs["mobile"] = []string{fmt.Sprintf("%s", userInfo["phone_number"])}
	attrs["tenant_id"] = []string{fmt.Sprintf("%s", userInfo["tenant_id"])}
	attrs["tenant_name"] = []string{fmt.Sprintf("%s", userInfo["tenant_name"])}
	return attrs, nil
}

func (drv *SBingoIAMOAuth2Driver) Sync(ctx context.Context, idpId string) error {
	var err error
	drv.idp, err = models.IdentityProviderManager.FetchIdentityProviderById(idpId)
	if err != nil {
		return err
	}

	err = drv.syncTenants(ctx, drv.idp)
	if err != nil {
		return err
	}
	err = drv.syncProjects(ctx, drv.idp)
	if err != nil {
		return err
	}
	err = drv.syncUsers(ctx, drv.idp)
	if err != nil {
		return err
	}

	return nil
}

func (drv *SBingoIAMOAuth2Driver) getAccessToken(ctx context.Context) (string, error) {
	authUrl := drv.getIAMEndpoint(ctx)
	authUrl = fmt.Sprintf("%s/oauth2/token?grant_type=client_credentials", authUrl)
	headers := http.Header{}
	headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", drv.AppId, drv.Secret))))

	httpclient := httputils.GetDefaultClient()
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.POST, authUrl, headers, nil, true)
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	err = resp.Unmarshal(&data)
	if err != nil {
		return "", err
	}
	drv.accessToken = data["access_token"].(string)

	return drv.accessToken, nil
}

func (drv *SBingoIAMOAuth2Driver) fetchAccessToken(ctx context.Context, code string) (string, error) {
	authUrl := drv.getIAMEndpoint(ctx)
	authUrl = fmt.Sprintf("%s/oauth2/token?grant_type=authorization_code&code=%s", authUrl, code)
	headers := http.Header{}
	headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", drv.AppId, drv.Secret))))

	httpclient := httputils.GetDefaultClient()
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.POST, authUrl, headers, nil, true)
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	err = resp.Unmarshal(&data)
	if err != nil {
		return "", err
	}
	return data["access_token"].(string), nil
}

func (drv *SBingoIAMOAuth2Driver) fetchUserInfo(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	authUrl := drv.getIAMEndpoint(ctx)
	authUrl = fmt.Sprintf("%s/oauth2/userinfo?access_token=%s", authUrl, accessToken)
	headers := http.Header{}
	headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", drv.AppId, drv.Secret))))

	httpclient := httputils.GetDefaultClient()
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.POST, authUrl, headers, nil, true)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	err = resp.Unmarshal(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
