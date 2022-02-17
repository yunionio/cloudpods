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

package apigateway

type SIdp struct {
	Id        string `json:"id,allowempty"`
	Name      string `json:"name,allowempty"`
	Driver    string `json:"driver,allowempty"`
	Template  string `json:"template,allowempty"`
	IconUrl   string `json:"icon_url,allowempty"`
	IsDefault bool   `json:"is_default"`
}

type SCommonConfig struct {
	ApiServer         string `json:"api_server,allowempty"`
	IsForgetLoginUser bool   `json:"is_forget_login_user"`
}

type SRegionsReponse struct {
	Regions []string `json:"regions,allowempty"`
	Domains []string `json:"domains,allowempty"`

	Captcha bool `json:"captcha,allowempty"`

	Idps              []SIdp `json:"idps,allowempty"`
	ReturnFullDomains bool   `json:"return_full_domains"`

	SCommonConfig
}
