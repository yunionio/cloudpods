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
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

// Context key 类型，用于从 HTTP Header 传入的 AK/SK 存入 context（供 Cursor/Claude 等客户端使用）
type headerCredKey string

const (
	ContextKeyAK headerCredKey = "mcp_header_ak"
	ContextKeySK headerCredKey = "mcp_header_sk"
)

// GetAKSKFromContext 从 context 中读取连接时通过 Header 传入的 AK/SK（未设置时返回空字符串）
func GetAKSKFromContext(ctx context.Context) (ak, sk string) {
	if v := ctx.Value(ContextKeyAK); v != nil {
		if s, ok := v.(string); ok {
			ak = s
		}
	}
	if v := ctx.Value(ContextKeySK); v != nil {
		if s, ok := v.(string); ok {
			sk = s
		}
	}
	return ak, sk
}

// HasRequestCredentials 判断请求上下文是否已带用户凭据（Token 或 AK/SK）。
func HasRequestCredentials(ctx context.Context) bool {
	if auth.IsAuthed() && policy.FetchUserCredential(ctx) != nil {
		return true
	}
	ak, sk := GetAKSKFromContext(ctx)
	return ak != "" && sk != ""
}

// ErrAuthenticationRequired tools/call 无凭据时的明确错误。
var ErrAuthenticationRequired = fmt.Errorf("authentication required: provide X-Auth-Token, AK/SK headers, or X-API-Key on tools/call")
