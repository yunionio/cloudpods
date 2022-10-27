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

package openstack

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type QuotaDetail struct {
	Reserved int
	Limit    int
	InUse    int
}

type QuotaSet struct {
	InjectedFileContentBytes QuotaDetail
	MetadataItems            QuotaDetail
	ServerGroupMembers       QuotaDetail
	ServerGroups             QuotaDetail
	Ram                      QuotaDetail
	FloatingIps              QuotaDetail
	KeyPairs                 QuotaDetail
	Id                       string
	Instances                QuotaDetail
	SecurityGroupRules       QuotaDetail
	InjectedFiles            QuotaDetail
	Cores                    QuotaDetail
	FixedIps                 QuotaDetail
	InjectedFilePathBytes    QuotaDetail
	SecurityGroups           QuotaDetail
}

type SQuota struct {
	Cores                    int
	Instances                int
	KeyPairs                 int
	FixedIps                 int
	MetadataItems            int
	Ram                      int
	ServerGroups             int
	ServerGroupMembers       int
	InjectedFileContentBytes int
	InjectedFilePathBytes    int
	InjectedFiles            int
	Floatingips              int
	Networks                 int
	Port                     int
	RbacPolicy               int
	Router                   int
	SecurityGroups           int
	SecurityGroupRules       int
}

func (region *SRegion) GetQuota() (*QuotaSet, error) {
	resource := fmt.Sprintf("/os-quota-sets/%s/detail", region.client.tokenCredential.GetTenantId())
	resp, err := region.ecsGet(resource)
	if err != nil {
		return nil, errors.Wrap(err, "ecsGet")
	}
	quota := &QuotaSet{}
	return quota, resp.Unmarshal(quota, "quota_set")
}

func (region *SRegion) SetQuota(quota *SQuota) error {
	params := map[string]map[string]interface{}{
		"quota_set": {
			"force": "True",
		},
	}

	if quota.Floatingips > 0 {
		params["quota_set"]["floating_ips"] = quota.Floatingips
	}

	if quota.SecurityGroups > 0 {
		params["quota_set"]["security_group"] = quota.SecurityGroups
	}

	if quota.SecurityGroupRules > 0 {
		params["quota_set"]["security_group_rules"] = quota.SecurityGroupRules
	}

	if quota.FixedIps > 0 {
		params["quota_set"]["fixed_ips"] = quota.FixedIps
	}

	if quota.Networks > 0 {
		params["quota_set"]["networks"] = quota.Networks
	}

	resource := "/os-quota-sets/" + region.client.tokenCredential.GetTenantId()
	_, err := region.ecsUpdate(resource, params)
	return err
}

type IQuota struct {
	Name  string
	Limit int
	InUse int
}

func (iq *IQuota) GetGlobalId() string {
	return iq.Name
}

func (iq *IQuota) GetName() string {
	return iq.Name
}

func (iq *IQuota) GetDesc() string {
	return ""
}

func (iq *IQuota) GetQuotaType() string {
	return iq.Name
}

func (iq *IQuota) GetMaxQuotaCount() int {
	return iq.Limit
}

func (iq *IQuota) GetCurrentQuotaUsedCount() int {
	return iq.InUse
}

func (region *SRegion) GetICloudQuotas() ([]cloudprovider.ICloudQuota, error) {
	quota, err := region.GetQuota()
	if err != nil {
		return nil, errors.Wrap(err, "GetQuota")
	}

	ret := []cloudprovider.ICloudQuota{}

	ret = append(ret, &IQuota{Name: "injected_file_content_bytes", Limit: quota.InjectedFileContentBytes.Limit, InUse: quota.InjectedFileContentBytes.InUse})
	ret = append(ret, &IQuota{Name: "metadata_items", Limit: quota.MetadataItems.Limit, InUse: quota.MetadataItems.InUse})
	ret = append(ret, &IQuota{Name: "server_group_members", Limit: quota.ServerGroupMembers.Limit, InUse: quota.ServerGroupMembers.InUse})
	ret = append(ret, &IQuota{Name: "server_groups", Limit: quota.ServerGroups.Limit, InUse: quota.ServerGroups.InUse})
	ret = append(ret, &IQuota{Name: "ram", Limit: quota.Ram.Limit, InUse: quota.Ram.InUse})
	ret = append(ret, &IQuota{Name: "floating_ips", Limit: quota.FloatingIps.Limit, InUse: quota.FloatingIps.InUse})
	ret = append(ret, &IQuota{Name: "key_pairs", Limit: quota.KeyPairs.Limit, InUse: quota.KeyPairs.InUse})
	ret = append(ret, &IQuota{Name: "instances", Limit: quota.Instances.Limit, InUse: quota.Instances.InUse})
	ret = append(ret, &IQuota{Name: "security_group_rules", Limit: quota.SecurityGroupRules.Limit, InUse: quota.SecurityGroupRules.InUse})
	ret = append(ret, &IQuota{Name: "injected_files", Limit: quota.InjectedFiles.Limit, InUse: quota.InjectedFiles.InUse})
	ret = append(ret, &IQuota{Name: "cores", Limit: quota.Cores.Limit, InUse: quota.Cores.InUse})
	ret = append(ret, &IQuota{Name: "fixed_ips", Limit: quota.FixedIps.Limit, InUse: quota.FixedIps.InUse})
	ret = append(ret, &IQuota{Name: "injected_file_path_bytes", Limit: quota.InjectedFilePathBytes.Limit, InUse: quota.InjectedFilePathBytes.InUse})
	ret = append(ret, &IQuota{Name: "security_groups", Limit: quota.SecurityGroups.Limit, InUse: quota.SecurityGroups.InUse})
	return ret, nil
}
