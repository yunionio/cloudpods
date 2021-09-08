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

package cloudprovider

import (
	"reflect"
	"strings"

	"yunion.io/x/pkg/utils"
)

type SApsaraEndpoints struct {
	EcsEndpoint             string `default:"$APSARA_ECS_ENDPOINT" metavar:"APSARA_ECS_ENDPOINT"`
	RdsEndpoint             string `default:"$APSARA_RDS_ENDPOINT"`
	VpcEndpoint             string `default:"$APSARA_VPC_ENDPOINT"`
	KvsEndpoint             string `default:"$APSARA_KVS_ENDPOINT"`
	SlbEndpoint             string `default:"$APSARA_SLB_ENDPOINT"`
	OssEndpoint             string `default:"$APSARA_OSS_ENDPOINT"`
	StsEndpoint             string `default:"$APSARA_STS_ENDPOINT"`
	ActionTrailEndpoint     string `default:"$APSARA_ACTION_TRAIL_ENDPOINT"`
	RamEndpoint             string `default:"$APSARA_RAM_ENDPOINT"`
	MetricsEndpoint         string `default:"$APSRRA_METRICS_ENDPOINT"`
	ResourcemanagerEndpoint string `default:"$APSARA_RESOURCEMANAGER_ENDPOINT"`
	DefaultRegion           string `default:"$APSARA_DEFAULT_REGION"`
}

// SHCSOEndpoints 华为私有云endpoints配置
/*
endpoint获取方式优先级：
通过参数明确指定使用指定endpoint。否则，程序根据华为云endpoint命名规则自动拼接endpoint
*/
type SHCSOEndpoints struct {
	caches map[string]string

	// 华为私有云Endpoint域名
	// example: hcso.com.cn
	// required:true
	EndpointDomain string `default:"$HUAWEI_ENDPOINT_DOMAIN" metavar:"HUAWEI_ENDPOINT_DOMAIN"`

	// 可用区ID
	// example: cn-north-2
	// required: true
	DefaultRegion string `default:"$HUAWEI_DEFAULT_REGION" metavar:"$HUAWEI_DEFAULT_REGION"`

	// 默认DNS
	// example: 10.125.0.26,10.125.0.27
	// required: false
	DefaultSubnetDns string `default:"$HUAWEI_DEFAULT_SUBNET_DNS" metavar:"$HUAWEI_DEFAULT_SUBNET_DNS"`

	// 弹性云服务
	Ecs string `default:"$HUAWEI_ECS_ENDPOINT"`
	// 云容器服务
	Cce string `default:"$HUAWEI_CCE_ENDPOINT"`
	// 弹性伸缩服务
	As string `default:"$HUAWEI_AS_ENDPOINT"`
	// 统一身份认证服务
	Iam string `default:"$HUAWEI_IAM_ENDPOINT"`
	// 镜像服务
	Ims string `default:"$HUAWEI_IMS_ENDPOINT"`
	// 云服务器备份服务
	Csbs string `default:"$HUAWEI_CSBS_ENDPOINT"`
	// 云容器实例 CCI
	Cci string `default:"$HUAWEI_CCI_ENDPOINT"`
	// 裸金属服务器
	Bms string `default:"$HUAWEI_BMS_ENDPOINT"`
	// 云硬盘 EVS
	Evs string `default:"$HUAWEI_EVS_ENDPOINT"`
	// 云硬盘备份 VBS
	Vbs string `default:"$HUAWEI_VBS_ENDPOINT"`
	// 对象存储服务 OBS
	Obs string `default:"$HUAWEI_OBS_ENDPOINT"`
	// 虚拟私有云 VPC
	Vpc string `default:"$HUAWEI_VPC_ENDPOINT"`
	// 弹性负载均衡 ELB
	Elb string `default:"$HUAWEI_ELB_ENDPOINT"`
	// 合作伙伴运营能力
	Bss string `default:"$HUAWEI_BSS_ENDPOINT"`
	// Nat网关 NAT
	Nat string `default:"$HUAWEI_NAT_ENDPOINT"`
	// 分布式缓存服务
	Dcs string `default:"$HUAWEI_DCS_ENDPOINT"`
	// 关系型数据库 RDS
	Rds string `default:"$HUAWEI_RDS_ENDPOINT"`
	// 云审计服务
	Cts string `default:"$HUAWEI_CTS_ENDPOINT"`
	// 监控服务 CloudEye
	Ces string `default:"$HUAWEI_CES_ENDPOINT"`
	// 企业项目
	Eps string `default:"$HUAWEI_EPS_ENDPOINT"`
	// 文件系统
	SfsTurbo string `default:"$HUAWEI_SFS_TURBO_ENDPOINT"`
}

func (self *SHCSOEndpoints) GetEndpoint(serviceName string, region string) string {
	sn := utils.Kebab2Camel(serviceName, "-")
	if self.caches == nil {
		self.caches = make(map[string]string, 0)
	}

	key := self.DefaultRegion + "." + sn
	if len(region) > 0 {
		key = region + "." + sn
	}

	if endpoint, ok := self.caches[key]; ok && len(endpoint) > 0 {
		return endpoint
	}

	var endpoint string
	fileds := reflect.Indirect(reflect.ValueOf(self))
	f := fileds.FieldByNameFunc(func(c string) bool {
		return c == sn
	})

	if f.Kind() == reflect.String {
		endpoint = f.String()
	}

	if len(endpoint) == 0 {
		endpoint = strings.Join([]string{serviceName, self.DefaultRegion, self.EndpointDomain}, ".")
	}

	if len(region) > 0 {
		endpoint = strings.Replace(endpoint, self.DefaultRegion, region, 1)
	}

	self.caches[key] = endpoint
	return endpoint
}
