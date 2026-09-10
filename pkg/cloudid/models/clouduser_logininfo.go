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

package models

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func (self *SClouduser) GetDetailsLoginInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*api.ClouduserLoginInfo, error) {
	if len(self.Secret) == 0 {
		return nil, httperrors.NewNotFoundError("No login secret found")
	}
	password, err := self.GetPassword()
	if err != nil {
		return nil, errors.Wrap(err, "GetPassword")
	}
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudaccount")
	}
	return formatClouduserLoginInfo(self.Name, account.Provider, account.IamLoginUrl, password), nil
}

func formatClouduserLoginInfo(name, provider, iamLoginUrl, password string) *api.ClouduserLoginInfo {
	username := name
	account := ""
	switch provider {
	case computeapi.CLOUD_PROVIDER_ALIYUN:
		suffix := strings.TrimPrefix(iamLoginUrl, "https://signin.aliyun.com/")
		suffix = strings.TrimSuffix(suffix, "/login.htm")
		if len(suffix) > 0 {
			username = fmt.Sprintf("%s@%s", name, suffix)
			account = suffix
		}
	case computeapi.CLOUD_PROVIDER_QCLOUD, computeapi.CLOUD_PROVIDER_HUAWEI:
		u, _ := url.Parse(iamLoginUrl)
		if u != nil {
			account = u.Query().Get("account")
		}
	case computeapi.CLOUD_PROVIDER_AWS:
		account = strings.TrimPrefix(iamLoginUrl, "https://")
		if info := strings.Split(account, "."); len(info) > 0 {
			account = info[0]
		}
	}
	return &api.ClouduserLoginInfo{
		Account:  account,
		Username: username,
		Password: password,
		Url:      iamLoginUrl,
	}
}
