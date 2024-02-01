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

package identity

import (
	"time"
)

const (
	PasswordResetHintAdminReset = "admin_reset"
	PasswordResetHintExpire     = "expire"
)

type UserDetails struct {
	EnabledIdentityBaseResourceDetails
	// IdpResourceInfo

	SUser

	UserUsage

	// 归属该用户的外部资源统计信息
	ExternalResourceInfo

	// 用户归属的的项目信息
	Projects []SFetchDomainObjectWithMetadata `json:"projects"`
}

type UserUsage struct {
	// 用户归属用户组的数量
	GroupCount int `json:"group_count"`
	// 用户归属项目的数量
	ProjectCount int `json:"project_count"`
	// 归属该用户的密钥凭证（含AKSK，TOTP，Secret等）的数量
	CredentialCount int `json:"credential_count"`
	// 连续登录失败的次数
	FailedAuthCount int `json:"failed_auth_count"`
	// 上传登录失败的时间
	FailedAuthAt time.Time `json:"failed_auth_at"`

	// 登录后是否需要重置密码
	NeedResetPassword bool `json:"need_reset_password"`
	// 重置密码原因: admin_reset|expire
	PasswordResetHint string `json:"password_reset_hint"`

	// 密码过期时间（如果开启了密码过期）
	PasswordExpiresAt time.Time `json:"password_expires_at"`

	// 该用户是否为本地用户（SQL维护的用户）
	IsLocal bool `json:"is_local"`

	// 该用户关联的外部认证源的认证信息
	Idps []IdpResourceInfo `json:"idps"`
}

type ResetCredentialInput struct {
	// 密钥的类型
	Type string `json:"type"`
}
