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

package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis"
)

type CloudaccountCreateInput struct {
	apis.Meta

	Provider            string
	Brand               string
	IsPublicCloud       bool
	IsOnPremise         bool
	Account             string
	Secret              string
	AccessUrl           string
	TenantId            string
	Name                string
	Description         string
	Enabled             bool
	EnableAutoSync      bool
	SyncIntervalSeconds int
	AutoCreateProject   bool
	Options             *jsonutils.JSONObject

	ProjectName string //OpenStack
	DomainName  string //OpenStack
	Username    string //OpenStack Esxi ZStack
	Password    string //OpenStack Esxi ZStack
	AuthUrl     string //OpenStack ZStack

	AccessKeyId     string //Huawei Aliyun Ucloud Aws
	AccessKeySecret string //Huawei Aliyun Ucloud Aws
	Environment     string //Huawei Azure Aws

	DirectoryId  string //Azure
	ClientId     string //Azure
	ClientSecret string //Azure

	Host string //Esxi
	Port int    //Esxi

	Endpoint string

	AppId     string //Qcloud
	SecretId  string //Qcloud
	SecretKey string //Qcloud
}
