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

package oidc

import api "yunion.io/x/onecloud/pkg/apis/identity"

var (
	// map[at_hash:KgtZpGvTuIaud0SVcmmkKQ aud:example-app email:kilgore@kilgore.trout email_verified:true exp:1593434672 groups:["authors"] iat:1593348272 iss:http://127.0.0.1:5556/dex name:Kilgore Trout sub:Cg0wLTM4NS0yODA4OS0wEgRtb2Nr]
	DexOIDCTemplate = api.SOIDCIdpConfigOptions{
		Scopes: []string{
			"openid",
			"email",
			"groups",
			"profile",
		},
		SIdpAttributeOptions: api.SIdpAttributeOptions{
			UserNameAttribute:        "name",
			UserIdAttribute:          "sub",
			UserEmailAttribute:       "email",
			UserDisplaynameAttribtue: "name",
		},
	}
	// https://developer.github.com/apps/building-oauth-apps/authorizing-oauth-apps/
	// map[avatar_url:https://avatars1.githubusercontent.com/u/1121362?v=4 bio: blog:https://yunion.io collaborators:0 company:Yunion.io created_at:2011-10-12T04:18:27Z disk_usage:925302 email: events_url:https://api.github.com/users/swordqiu/events{/privacy} followers:13 followers_url:https://api.github.com/users/swordqiu/followers following:1 following_url:https://api.github.com/users/swordqiu/following{/other_user} gists_url:https://api.github.com/users/swordqiu/gists{/gist_id} gravatar_id: hireable: html_url:https://github.com/swordqiu
	// id:1121362 location:Beijing, China
	// login:swordqiu name:Jian Qiu
	// node_id:MDQ6VXNlcjExMjEzNjI= organizations_url:https://api.github.com/users/swordqiu/orgs owned_private_repos:0 plan:{"name":"free","space":976562499,"collaborators":0,"private_repos":10000} private_gists:0 public_gists:0 public_repos:37 received_events_url:https://api.github.com/users/swordqiu/received_events repos_url:https://api.github.com/users/swordqiu/repos site_admin:false starred_url:https://api.github.com/users/swordqiu/starred{/owner}{/repo} subscriptions_url:https://api.github.com/users/swordqiu/subscriptions total_private_repos:0 twitter_username: two_factor_authentication:false type:User updated_at:2020-06-29T01:39:42Z url:https://api.github.com/users/swordqiu]
	GithubOIDCTemplate = api.SOIDCIdpConfigOptions{
		Scopes: []string{
			"user",
		},
		AuthUrl:     "https://github.com/login/oauth/authorize",
		TokenUrl:    "https://github.com/login/oauth/access_token",
		UserinfoUrl: "https://api.github.com/user",
		TimeoutSecs: 60,
		SIdpAttributeOptions: api.SIdpAttributeOptions{
			UserIdAttribute:          "id",
			UserNameAttribute:        "login",
			UserEmailAttribute:       "email",
			UserDisplaynameAttribtue: "name",
		},
	}

	// {
	//  "sub": "112176790568447731603",
	//  "name": "Jian Qiu",
	//  "given_name": "Jian",
	//  "family_name": "Qiu",
	//  "picture": "https://lh3.googleusercontent.com/a/AATXAJyj32UmKhmwI38ljm8xI53LX4Lw3w5wYxKsj4JS\u003ds96-c",
	//  "email": "swordqiu@gmail.com",
	//  "email_verified": true,
	//  "locale": "zh-CN"
	// }
	GoogleOIDCTemplate = api.SOIDCIdpConfigOptions{
		Endpoint: "https://accounts.google.com",
		SIdpAttributeOptions: api.SIdpAttributeOptions{
			UserIdAttribute:          "sub",
			UserNameAttribute:        "email",
			UserEmailAttribute:       "email",
			UserDisplaynameAttribtue: "name",
		},
	}

	AzureADTemplate = api.SOIDCIdpConfigOptions{
		Scopes: []string{
			"openid",
			"profile",
			"email",
		},
		TimeoutSecs: 60,
		SIdpAttributeOptions: api.SIdpAttributeOptions{
			UserIdAttribute:          "sub",
			UserNameAttribute:        "name",
			UserEmailAttribute:       "email",
			UserDisplaynameAttribtue: "name",
		},
	}
)
