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

// CloudpodsAdapter 负责 Cloudpods 认证并创建 mcclient.ClientSession，供 climc 工具执行使用。
type CloudpodsAdapter struct {
	client *mcclient.Client
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
	return &CloudpodsAdapter{client: client}
}

func (a *CloudpodsAdapter) authenticate(ak string, sk string) (mcclient.TokenCredential, error) {
	return a.client.AuthenticateByAccessKey(ak, sk, "")
}

// GetSession 获取 Cloudpods API 会话
func (a *CloudpodsAdapter) GetSession(ctx context.Context, ak string, sk string) (*mcclient.ClientSession, error) {
	if ak == "" && sk == "" {
		ak, sk = GetAKSKFromContext(ctx)
	}
	var userCred mcclient.TokenCredential
	if auth.IsAuthed() {
		userCred = policy.FetchUserCredential(ctx)
		if userCred != nil {
			log.Debugf("GetSession with userCred from context")
		} else {
			token, err := a.authenticate(ak, sk)
			if err != nil {
				return nil, err
			}
			userCred = token
		}
		return auth.GetSession(ctx, userCred, ""), nil
	}
	token, err := a.authenticate(ak, sk)
	if err != nil {
		return nil, err
	}
	return a.client.NewSession(
		context.Background(),
		"",
		"",
		api.EndpointInterfaceApigateway,
		token,
	), nil
}
