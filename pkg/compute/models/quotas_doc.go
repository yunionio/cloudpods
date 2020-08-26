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

package models

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
)

// 主机配额详情
type SQuotaDetail struct {
	SQuota

	quotas.SZonalCloudResourceDetailKeys
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/quotas/{scope}/{scopeId}
// +onecloud:swagger-gen-route-tag=quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项目的配额和域的配额
// +onecloud:swagger-gen-param-path=scopeId
// +onecloud:swagger-gen-param-path=指定项目或者域的ID
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=quotas
// +onecloud:swagger-gen-resp-body-list

// 获取指定项目或者域的主机配额
func GetQuota(query quotas.SBaseQuotaQueryInput) *SQuotaDetail {
	return nil
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/quotas/{scope}
// +onecloud:swagger-gen-route-tag=quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项 目的配额和域的配额
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=quotas
// +onecloud:swagger-gen-resp-body-list

// 获取所有项目或者域的主机配额
func ListQuotas(query quotas.SBaseQuotaQueryInput) *SQuotaDetail {
	return nil
}

// 设置主机配额输入参数
type SetQuotaInput struct {
	quotas.SBaseQuotaSetInput

	SQuota
}

// +onecloud:swagger-gen-route-method=POST
// +onecloud:swagger-gen-route-path=/quotas/{scope}/{scopeId}
// +onecloud:swagger-gen-route-tag=quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项目的配额和域的配额
// +onecloud:swagger-gen-param-path=scopeId
// +onecloud:swagger-gen-param-path=指定项目或者域的ID
// +onecloud:swagger-gen-param-body-index=0
// +onecloud:swagger-gen-param-body-key=quotas
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=quotas
// +onecloud:swagger-gen-resp-body-list

// 设置指定项目或者域的主机配额
func SetQuotas(input SetQuotaInput) *SQuotaDetail {
	return nil
}

// 项目配额详情
type SProjectQuotaDetail struct {
	SProjectQuota

	quotas.SBaseProjectQuotaDetailKeys
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/project_quotas/{scope}/{scopeId}
// +onecloud:swagger-gen-route-tag=project_quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项目的配额和域的配额
// +onecloud:swagger-gen-param-path=scopeId
// +onecloud:swagger-gen-param-path=指定项目或者域的ID
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=project_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取指定项目或者域的项目配额
func GetProjectQuota(query quotas.SBaseQuotaQueryInput) *SProjectQuotaDetail {
	return nil
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/project_quotas/{scope}
// +onecloud:swagger-gen-route-tag=project_quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项 目的配额和域的配额
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=project_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取所有项目或者域的项目配额
func ListProjectQuotas(query quotas.SBaseQuotaQueryInput) *SProjectQuotaDetail {
	return nil
}

// 设置项目配额输入参数
type SetProjectQuotaInput struct {
	quotas.SBaseQuotaSetInput

	SProjectQuota
}

// +onecloud:swagger-gen-route-method=POST
// +onecloud:swagger-gen-route-path=/project_quotas/{scope}/{scopeId}
// +onecloud:swagger-gen-route-tag=project_quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项目的配额和域的配额
// +onecloud:swagger-gen-param-path=scopeId
// +onecloud:swagger-gen-param-path=指定项目或者域的ID
// +onecloud:swagger-gen-param-body-index=0
// +onecloud:swagger-gen-param-body-key=project_quotas
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=project_quotas
// +onecloud:swagger-gen-resp-body-list

// 设置指定项目或者域的项目配额
func SetProjectQuotas(input SetProjectQuotaInput) *SProjectQuotaDetail {
	return nil
}

// 可用区配额详情
type SZoneQuotaDetail struct {
	SZoneQuota

	quotas.SZonalCloudResourceDetailKeys
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/zone_quotas/{scope}/{scopeId}
// +onecloud:swagger-gen-route-tag=zone_quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项目的配额和域的配额
// +onecloud:swagger-gen-param-path=scopeId
// +onecloud:swagger-gen-param-path=指定项目或者域的ID
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=zone_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取指定项目或者域的可用区配额
func GetZoneQuota(query quotas.SBaseQuotaQueryInput) *SZoneQuotaDetail {
	return nil
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/zone_quotas/{scope}
// +onecloud:swagger-gen-route-tag=zone_quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项 目的配额和域的配额
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=zone_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取所有项目或者域的可用区配额
func ListZoneQuotas(query quotas.SBaseQuotaQueryInput) *SZoneQuotaDetail {
	return nil
}

// 设置可用区配额输入参数
type SetZoneQuotaInput struct {
	quotas.SBaseQuotaSetInput

	SZoneQuota
}

// +onecloud:swagger-gen-route-method=POST
// +onecloud:swagger-gen-route-path=/zone_quotas/{scope}/{scopeId}
// +onecloud:swagger-gen-route-tag=zone_quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项目的配额和域的配额
// +onecloud:swagger-gen-param-path=scopeId
// +onecloud:swagger-gen-param-path=指定项目或者域的ID
// +onecloud:swagger-gen-param-body-index=0
// +onecloud:swagger-gen-param-body-key=zone_quotas
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=zone_quotas
// +onecloud:swagger-gen-resp-body-list

// 设置指定项目或者域的可用区配额
func SetZoneQuotas(input SetZoneQuotaInput) *SZoneQuotaDetail {
	return nil
}

// 区域配额详情
type SRegionQuotaDetail struct {
	SRegionQuota

	quotas.SRegionalCloudResourceDetailKeys
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/region_quotas/{scope}/{scopeId}
// +onecloud:swagger-gen-route-tag=region_quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项目的配额和域的配额
// +onecloud:swagger-gen-param-path=scopeId
// +onecloud:swagger-gen-param-path=指定项目或者域的ID
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=region_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取指定项目或者域的区域配额
func GetRegionQuota(query quotas.SBaseQuotaQueryInput) *SRegionQuotaDetail {
	return nil
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/region_quotas/{scope}
// +onecloud:swagger-gen-route-tag=region_quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项 目的配额和域的配额
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=region_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取所有项目或者域的区域配额
func ListRegionQuotas(query quotas.SBaseQuotaQueryInput) *SRegionQuotaDetail {
	return nil
}

// 设置区域配额输入参数
type SetRegionQuotaInput struct {
	quotas.SBaseQuotaSetInput

	SRegionQuota
}

// +onecloud:swagger-gen-route-method=POST
// +onecloud:swagger-gen-route-path=/region_quotas/{scope}/{scopeId}
// +onecloud:swagger-gen-route-tag=region_quota
// +onecloud:swagger-gen-param-path=scope
// +onecloud:swagger-gen-param-path=配额所属范围，可能值为projects和domains，分别代表项目的配额和域的配额
// +onecloud:swagger-gen-param-path=scopeId
// +onecloud:swagger-gen-param-path=指定项目或者域的ID
// +onecloud:swagger-gen-param-body-index=0
// +onecloud:swagger-gen-param-body-key=region_quotas
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=region_quotas
// +onecloud:swagger-gen-resp-body-list

// 设置指定项目或者域的区域配额
func SetRegionQuotas(input SetRegionQuotaInput) *SRegionQuotaDetail {
	return nil
}

// 域配额详情
type SDomainQuotaDetail struct {
	SDomainQuota

	quotas.SBaseDomainQuotaDetailKeys
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/domain_quotas/domains/{domainId}
// +onecloud:swagger-gen-route-tag=domain_quota
// +onecloud:swagger-gen-param-path=domainId
// +onecloud:swagger-gen-param-path=指定域的ID
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=domain_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取指定域的配额
func GetDomainQuota(query quotas.SBaseQuotaQueryInput) *SDomainQuotaDetail {
	return nil
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/domain_quotas/domains
// +onecloud:swagger-gen-route-tag=domain_quota
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=domain_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取所有域的域配额
func ListDomainQuotas(query quotas.SBaseQuotaQueryInput) *SDomainQuotaDetail {
	return nil
}

// 设置域配额输入参数
type SetDomainQuotaInput struct {
	quotas.SBaseQuotaSetInput

	SDomainQuota
}

// +onecloud:swagger-gen-route-method=POST
// +onecloud:swagger-gen-route-path=/domain_quotas/domains/{domainId}
// +onecloud:swagger-gen-route-tag=domain_quota
// +onecloud:swagger-gen-param-path=domainId
// +onecloud:swagger-gen-param-path=指定域的ID
// +onecloud:swagger-gen-param-body-index=0
// +onecloud:swagger-gen-param-body-key=domain_quotas
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=domain_quotas
// +onecloud:swagger-gen-resp-body-list

// 设置域的域配额
func SetDomainQuotas(input SetDomainQuotaInput) *SDomainQuotaDetail {
	return nil
}

// 基础设施配额详情
type SInfrasQuotaDetail struct {
	SInfrasQuota

	quotas.SDomainRegionalCloudResourceDetailKeys
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/infras_quotas/domains/{domainId}
// +onecloud:swagger-gen-route-tag=infras_quota
// +onecloud:swagger-gen-param-path=domainId
// +onecloud:swagger-gen-param-path=指定域的ID
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=infras_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取指定域的基础设施配额
func GetInfrasQuota(query quotas.SBaseQuotaQueryInput) *SInfrasQuotaDetail {
	return nil
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/infras_quotas/domains
// +onecloud:swagger-gen-route-tag=infras_quota
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=infras_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取所有域的基础设施配额
func ListInfrasQuotas(query quotas.SBaseQuotaQueryInput) *SInfrasQuotaDetail {
	return nil
}

// 设置基础设施配额输入参数
type SetInfrasQuotaInput struct {
	quotas.SBaseQuotaSetInput

	SInfrasQuota
}

// +onecloud:swagger-gen-route-method=POST
// +onecloud:swagger-gen-route-path=/infras_quotas/domains/{domainId}
// +onecloud:swagger-gen-route-tag=infras_quota
// +onecloud:swagger-gen-param-path=domainId
// +onecloud:swagger-gen-param-path=指定域的ID
// +onecloud:swagger-gen-param-body-index=0
// +onecloud:swagger-gen-param-body-key=infras_quotas
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=infras_quotas
// +onecloud:swagger-gen-resp-body-list

// 设置指定域的基础设施配额
func SetInfrasQuotas(input SetInfrasQuotaInput) *SInfrasQuotaDetail {
	return nil
}
