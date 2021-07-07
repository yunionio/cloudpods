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

// OpenID Connect Config Options
type SOIDCIdpConfigOptions struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	Scopes []string `json:"scopes"`

	Endpoint    string `json:"endpoint"`
	AuthUrl     string `json:"auth_url"`
	TokenUrl    string `json:"token_url"`
	UserinfoUrl string `json:"userinfo_url"`

	TimeoutSecs int `json:"timeout_secs"`

	SIdpAttributeOptions
}

type SOIDCDexConfigOptions struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Endpoint     string `json:"endpoint"`

	SIdpAttributeOptions
}

type SOIDCGithubConfigOptions struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	SIdpAttributeOptions
}

type SOIDCGoogleConfigOptions struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	SIdpAttributeOptions
}

const (
	AZURE_CLOUD_ENV_CHINA  = "china"
	AZURE_CLOUD_ENV_GLOBAL = "global"
)

type SOIDCAzureConfigOptions struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TenantId     string `json:"tenant_id"`
	CloudEnv     string `json:"cloud_env"`

	SIdpAttributeOptions
}
