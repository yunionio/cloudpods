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

package mcclient

const (
	AuthSourceWeb      = "web"
	AuthSourceAPI      = "api"
	AuthSourceCli      = "cli"
	AuthSourceSrv      = "srv"
	AuthSourceOperator = "operator"
)

type SAuthContext struct {
	// 认证来源类型, 可能的值有：
	//
	// | source   | 说明                      |
	// |----------|---------------------------|
	// | web      | 通过web控制台认证         |
	// | api      | api调用认证               |
	// | cli      | climc客户端认证           |
	// | srv      | 作为服务认证              |
	// | operator | 作为onecloud-operator认证 |
	//
	Source string `json:"source,omitempty"`
	// 认证来源IP
	Ip string `json:"ip,omitempty"`
}

type SAuthenticationInputV2 struct {
	// keystone v2 认证接口认证信息
	// required:true
	Auth struct {
		// 如果使用用户名/密码认证，则需要设置passwordCredentials
		PasswordCredentials struct {
			// 用户名
			Username string `json:"username,omitempty"`
			// 用户密码
			Password string `json:"password,omitempty"`
		} `json:"passwordCredentials,omitempty"`
		// 指定认证用户的所属项目名称，该字段和tenantId二选一，或者不设置。
		// 如果不提供tenantName和tenantId，则用户认证成功后，获得一个unscoped token
		// 此时，如果用户需要访问具体项目的资源，还是需要用unscoped token进行认证，获得指定项目的token
		// required:false
		TenantName string `json:"tenantName,omitempty"`
		// 指定认证用户的所属项目ID，该字段和tenantName二选一，或者不设置。
		// required:false
		TenantId string `json:"tenantId,omitempty"`
		// 如果使用token认证，则需要设置token.Id
		Token struct {
			// token的字符串
			Id string `json:"id,omitempty"`
		} `json:"token,omitempty"`
		// 认证上下文
		// required:false
		Context SAuthContext `json:"context,omitempty"`
	} `json:"auth,omitempty"`
}

type SAuthenticationIdentity struct {
	// ID of identity provider, optional
	// required:false
	Id string `json:"id,omitempty"`
	// 认证方式列表，支持认证方式如下：
	//
	// | method   | 说明                                                                |
	// |----------|--------------------------------------------------------------------|
	// | password | 用户名密码认证                                                       |
	// | token    | token认证，已经通过其他方式获得token之后，可以用旧的token认证获得新的token   |
	// | aksk     | Access Key/Secret key认证                                           |
	// | cas      | 通过SSO统一认证平台CAS认证                                             |
	// | saml     | 作为SAML 2.0 SP通过IDP认证                                            |
	// | oidc     | 作为OpenID Connect/OAuth2 Client认证                                 |
	// | oauth2   | OAuth2认证                                                          |
	//
	Methods []string `json:"methods,omitempty"`
	// 当认证方式为password时，通过该字段提供密码认证信息
	Password struct {
		User struct {
			// 用户ID
			Id string `json:"id,omitempty"`
			// 用户名称
			Name string `json:"name,omitempty"`
			// 密码
			Password string `json:"password,omitempty"`
			// 用户所属域的信息
			Domain struct {
				// 域ID
				Id string `json:"id,omitempty"`
				// 域名称
				Name string `json:"name,omitempty"`
			}
		} `json:"user,omitempty"`
	} `json:"password,omitempty"`
	// 当认证方式为token时，通过该字段提供token认证信息
	Token struct {
		// token
		Id string `json:"id,omitempty"`
	} `json:"token,omitempty"`
	// 当认证方式为aksk时，通过该字段提供客户端AK/SK信息
	// 为了兼容不同版本的AK/SK认证方式，使用编码后的字符串传递该信息
	AccessKeyRequest string `json:"access_key_secret,omitempty"`
	// 当认证方式为cas时，通过该字段提供CAS认证的ID
	// required:false
	CASTicket struct {
		Id      string `json:"id,omitempty"`
		Service string `json:"service,omitempty"`
	} `json:"cas_ticket,omitempty"`
	// 当认证方式为saml时，通过该字段提供SAML认证的Response信息
	SAMLAuth struct {
		Response string `json:"response,omitempty"`
	} `json:"saml_auth,omitempty"`
	OIDCAuth struct {
		Code        string `json:"code,omitempty"`
		RedirectUri string `json:"redirect_uri,omitempty"`
	} `json:"oidc_auth,omitempty"`
	OAuth2 struct {
		Code string `json:"code,omitempty"`
	}
}

type SAuthenticationInputV3 struct {
	// keystone v3 认证接口认证信息
	// required:true
	Auth struct {
		// 认证信息
		// required:true
		Identity SAuthenticationIdentity `json:"identity,omitempty"`
		// 指定认证范围, 该字段可选。如果未指定scope，则用户认证成功后获得一个unscoped token，
		// 当用户需要访问指定项目的资源时，需要通过该unscope token进行认证，获得该项目scope的token
		// 目前只支持Project scope的token
		// required:false
		Scope struct {
			// 指定token的scope为指定的项目
			// required:false
			Project struct {
				// 指定项目的ID，由于ID全局唯一，因此指定ID后不需要指定项目所在的域（Domain）,ID和Name只需要指定其中一个
				// required:false
				Id string `json:"id,omitempty"`
				// 指定项目的Name，指定Name时，需要指定项目所在的域（domain）
				// required:false
				Name string `json:"name,omitempty"`
				// 指定项目所在的域（domain）
				// required:false
				Domain struct {
					// 指定项目所在域的ID，ID和Name只需要指定其中一个
					// required:false
					Id string `json:"id,omitempty"`
					// 指定项目所在域的Name
					// required:false
					Name string `json:"name,omitempty"`
				} `json:"domain,omitempty"`
			} `json:"project,omitempty"`
			// 指定token的scope为指定的域
			// required:false
			Domain struct {
				// 指定domain的ID，ID和Name只需要指定其中一个
				// required:false
				Id string `json:"id,omitempty"`
				// 指定Domain的Name
				// required:false
				Name string `json:"name,omitempty"`
			} `json:"domain,omitempty"`
		} `json:"scope,omitempty"`
		// 认证上下文
		// required:false
		Context SAuthContext `json:"context,omitempty"`
	} `json:"auth,omitempty"`
}
