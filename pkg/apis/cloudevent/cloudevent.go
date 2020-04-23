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

package cloudevent

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type CloudeventListInput struct {
	apis.ModelBaseListInput

	// 列出指定云平台的资源，支持的云平台如下
	//
	// | Provider  | 开始支持版本 | 平台                                |
	// |-----------|------------|-------------------------------------|
	// | OneCloud  | 0.0        | OneCloud内置私有云，包括KVM和裸金属管理 |
	// | VMware    | 1.2        | VMware vCenter                      |
	// | OpenStack | 2.6        | OpenStack M版本以上私有云             |
	// | ZStack    | 2.10       | ZStack私有云                         |
	// | Aliyun    | 2.0        | 阿里云                               |
	// | Aws       | 2.3        | Amazon AWS                          |
	// | Azure     | 2.2        | Microsoft Azure                     |
	// | Google    | 2.13       | Google Cloud Platform               |
	// | Qcloud    | 2.3        | 腾讯云                               |
	// | Huawei    | 2.5        | 华为公有云                           |
	// | Ucloud    | 2.7        | UCLOUD                               |
	// | Ctyun     | 2.13       | 天翼云                               |
	// | S3        | 2.11       | 通用s3对象存储                        |
	// | Ceph      | 2.11       | Ceph对象存储                         |
	// | Xsky      | 2.11       | XSKY启明星辰Ceph对象存储              |
	//
	// enum: OneCloud,VMware,Aliyun,Qcloud,Azure,Aws,Huawei,OpenStack,Ucloud,ZStack,Google,Ctyun,S3,Ceph,Xsky"
	Providers []string `json:"providers"`
	// swagger:ignore
	// Deprecated
	Provider []string `json:"provider" "yunion:deprecated-by":"providers"`

	// 列出指定云平台品牌的资源，一般来说brand和provider相同，除了以上支持的provider之外，还支持以下band
	//
	// |   Brand  | Provider | 说明        |
	// |----------|----------|------------|
	// | DStack   | ZStack   | 滴滴云私有云 |
	//
	Brands []string `json:"brands"`
	// swagger:ignore
	// Deprecated
	Brand []string `json:"brand" "yunion:deprecated-by":"brands"`

	// 服务类型
	Service []string `json:"service"`

	// 订阅
	Manager []string `json:"manager"`

	// 账号
	Account []string `json:"account"`

	// 操作类型
	Action []string `json:"action"`

	// 操作日志起始时间
	Since time.Time `json:"since"`
	// 操作日志截止时间
	Until time.Time `json:"until"`
}
