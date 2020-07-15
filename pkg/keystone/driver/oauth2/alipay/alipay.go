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

package alipay

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
	"yunion.io/x/onecloud/pkg/util/alipayclient"
)

type SAlipayOAuth2Driver struct {
	oauth2.SOAuth2BaseDriver
}

func NewAlipayOAuth2Driver(appId string, secret string) oauth2.IOAuth2Driver {
	drv := &SAlipayOAuth2Driver{
		SOAuth2BaseDriver: oauth2.SOAuth2BaseDriver{
			AppId:  appId,
			Secret: secret,
		},
	}
	return drv
}

const (
	AuthUrl = "https://openauth.alipay.com/oauth2/publicAppAuthorize.htm"
)

func (drv *SAlipayOAuth2Driver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	req := map[string]string{
		"app_id":        drv.AppId,
		"redirect_uri":  callbackUrl,
		"scope":         "auth_user",
		"response_type": "code",
		"state":         state,
	}
	urlStr := fmt.Sprintf("%s?%s", AuthUrl, jsonutils.Marshal(req).QueryString())
	return urlStr, nil
}

func (drv *SAlipayOAuth2Driver) Authenticate(ctx context.Context, code string) (map[string][]string, error) {
	alipayCli, err := alipayclient.NewDefaultAlipayClient(drv.AppId, drv.Secret, "", true)
	if err != nil {
		return nil, errors.Wrap(err, "alipayclient.NewDefaultAlipayClient")
	}
	resp, err := alipayCli.GetOAuthToken(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "alipayCli.GetOAuthToken")
	}
	userInfo, err := alipayCli.GetUserInfo(ctx, resp.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, "alipayCli.GetUserInfo")
	}
	attrs := make(map[string][]string)
	for k, v := range userInfo {
		attrs[k] = []string{v}
	}
	attrs["user_name"] = []string{fmt.Sprintf("alipay%s", userInfo["user_id"])}
	return attrs, nil
}
