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
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	KUBE_CLUSTER_STATUS_RUNNING  = "running"
	KUBE_CLUSTER_STATUS_CREATING = "creating"
	KUBE_CLUSTER_STATUS_ABNORMAL = "abnormal"
	// 升级中
	KUBE_CLUSTER_STATUS_UPDATING = "updating"
	// 升级失败
	KUBE_CLUSTER_STATUS_UPDATING_FAILED = "updating_failed"
	// 伸缩中
	KUBE_CLUSTER_STATUS_SCALING = "scaling"
	// 停止
	KUBE_CLUSTER_STATUS_STOPPED = "stopped"
)

type KubeClusterListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput

	RegionalFilterListInput
	ManagedResourceListInput
	VpcFilterListInput
}

type KubeClusterCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	Version string `json:"version"`
	// required: true
	NetworkIds SKubeNetworkIds `json:"network_ids"`
	// swagger:ignore
	ManagerId string `json:"manager_id"`
	// swagger:ignore
	CloudregionId string `json:"cloudregion_id"`
	// required: true
	VpcResourceInput

	PrivateAccess bool `json:"private_access"`
	PublicAccess  bool `json:"public_access"`

	RoleName string `json:"role_name"`
}

type KubeClusterDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails

	SKubeCluster
	VpcResourceInfo
}

func (self KubeClusterDetails) GetMetricTags() map[string]string {
	ret := map[string]string{
		"id":             self.ExternalClusterId,
		"res_type":       "kube_cluster",
		"cluster_id":     self.ExternalClusterId,
		"cluster_name":   self.Name,
		"status":         self.Status,
		"cloudregion":    self.Cloudregion,
		"cloudregion_id": self.CloudregionId,
		"region_ext_id":  self.RegionExtId,
		"domain_id":      self.DomainId,
		"project_domain": self.ProjectDomain,
		"account":        self.Account,
		"account_id":     self.AccountId,
		"external_id":    self.ExternalId,
	}

	return AppendMetricTags(ret, self.MetadataResourceInfo)
}

func (self KubeClusterDetails) GetMetricPairs() map[string]string {
	ret := map[string]string{}
	return ret
}

type KubeClusterUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput
}

type GetKubeConfigInput struct {
	// 是否获取内网kubeconfig, 默认false即获取外网kubeconfig
	Private bool `json:"private"`
	// kubeconfig 到期时间
	// 阿里云: 15（15分钟）～4320（3天）
	// 腾讯云不传此参数默认时效是20年
	ExpireMinutes int `json:"expire_minutes"`
}

type KubeClusterDeleteInput struct {
	// 是否保留集群关联的实例及slb
	// default: false
	Retain bool `json:"retain"`
}

type SInstanceTypes []string

func (kn SInstanceTypes) String() string {
	return jsonutils.Marshal(kn).String()
}

func (kn SInstanceTypes) IsZero() bool {
	return len(kn) == 0
}

type SKubeNetworkIds []string

func (kn SKubeNetworkIds) String() string {
	return jsonutils.Marshal(kn).String()
}

func (kn SKubeNetworkIds) IsZero() bool {
	return len(kn) == 0
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SKubeNetworkIds{}), func() gotypes.ISerializable {
		return &SKubeNetworkIds{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&SInstanceTypes{}), func() gotypes.ISerializable {
		return &SInstanceTypes{}
	})
}
