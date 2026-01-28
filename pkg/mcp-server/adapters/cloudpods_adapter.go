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

package adapters

import (
	"context"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcp-server/options"
)

// CloudpodsAdapter 是与 Cloudpods API 交互的适配器，负责认证和资源管理
type CloudpodsAdapter struct {
	client  *mcclient.Client
	session *mcclient.ClientSession
}

type CloudRegion struct {
	RegionId string `json:"region_id"`
}

// NewCloudpodsAdapter 创建一个新的 Cloudpods 适配器实例
func NewCloudpodsAdapter() *CloudpodsAdapter {

	client := mcclient.NewClient(
		options.Options.AuthURL,
		options.Options.Timeout,
		false,
		true,
		"",
		"",
	)

	return &CloudpodsAdapter{
		client: client,
	}
}

// authenticate 实现 Cloudpods 的认证逻辑，例如获取访问令牌
func (a *CloudpodsAdapter) authenticate(ak string, sk string) (mcclient.TokenCredential, error) {
	if a.session != nil {
		return a.session.GetToken(), nil
	}

	token, err := a.client.AuthenticateByAccessKey(ak, sk, "")
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (a *CloudpodsAdapter) getSession(ctx context.Context, ak string, sk string) (*mcclient.ClientSession, error) {
	var userCred mcclient.TokenCredential
	if auth.IsAuthed() {
		userCred = policy.FetchUserCredential(ctx)
		if userCred != nil {
			log.Infof("getSessionWithUserCred: %v", userCred)
		} else {
			log.Infof("No userCred in context, will use ak/sk for authentication")
			token, err := a.authenticate(ak, sk)
			if err != nil {
				return nil, err
			}
			userCred = token
		}
		a.session = auth.GetSession(ctx, userCred, "")
	} else {
		token, err := a.authenticate(ak, sk)
		if err != nil {
			return nil, err
		}
		a.session = a.client.NewSession(
			context.Background(),
			"",
			"",
			api.EndpointInterfaceApigateway,
			token,
		)
	}
	return a.session, nil
}
