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
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	cmd := shell.NewResourceCmd(&modules.Cloudaccounts).WithKeyword("cloud-account")
	cmd.List(&options.CloudaccountListOptions{})
	cmd.Show(&options.SCloudAccountIdOptions{})
	cmd.Delete(&options.SCloudAccountIdOptions{})
	cmd.Update(&options.SCloudAccountUpdateBaseOptions{})
	cmd.PerformClassWithKeyword("preparenets-vmware", "prepare-nets", &options.SVMwareCloudAccountPrepareNetsOptions{})

	cmd.CreateWithKeyword("create-vmware", &options.SVMwareCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-aliyun", &options.SAliyunCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-azure", &options.SAzureCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-qcloud", &options.SQcloudCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-google", &options.SGoogleCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-aws", &options.SAWSCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-openstack", &options.SOpenStackCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-huawei", &options.SHuaweiCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-hcso", &options.SHCSOAccountCreateOptions{})
	cmd.CreateWithKeyword("create-hcs", &options.SHCSAccountCreateOptions{})
	cmd.CreateWithKeyword("create-ucloud", &options.SUcloudCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-zstack", &options.SZStackCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-s3", &options.SS3CloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-ceph", &options.SCephCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-xsky", &options.SXskyCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-ctyun", &options.SCtyunCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-apsara", &options.SApsaraCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-ecloud", &options.SEcloudCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-jdcloud", &options.SJDcloudCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-cloudpods", &options.SCloudpodsCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-nutanix", &options.SNutanixCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-bingocloud", &options.SBingoCloudAccountCreateOptions{})
	cmd.CreateWithKeyword("create-incloudsphere", &options.SInCloudSphereAccountCreateOptions{})
	cmd.CreateWithKeyword("create-proxmox", &options.SProxmoxAccountCreateOptions{})
	cmd.CreateWithKeyword("create-remotefile", &options.SRemoteFileAccountCreateOptions{})

	cmd.UpdateWithKeyword("update-vmware", &options.SVMwareCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-aliyun", &options.SAliyunCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-azure", &options.SAzureCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-qcloud", &options.SQcloudCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-google", &options.SGoogleCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-aws", &options.SAWSCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-openstack", &options.SOpenStackCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-huawei", &options.SHuaweiCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-hcso", &options.SHCSOAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-hcs", &options.SHCSOAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-ucloud", &options.SUcloudCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-zstack", &options.SZStackCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-s3", &options.SS3CloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-ctyun", &options.SCtyunCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-jdcloud", &options.SJDcloudCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-cloudpods", &options.SCloudpodsCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-nutanix", &options.SNutanixCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-bingocloud", &options.SBingoCloudAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-incloudsphere", &options.SInCloudSphereAccountUpdateOptions{})
	cmd.UpdateWithKeyword("update-proxmox", &options.SProxmoxAccountUpdateOptions{})

	cmd.Perform("update-credential", &options.CloudaccountUpdateCredentialOptions{})

	cmd.PerformWithKeyword("update-credential-google", "update-credential", &options.SGoogleCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-vmware", "update-credential", &options.SVMwareCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-aliyun", "update-credential", &options.SAliyunCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-azure", "update-credential", &options.SAzureCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-qcloud", "update-credential", &options.SQcloudCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-aws", "update-credential", &options.SAWSCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-openstack", "update-credential", &options.SOpenStackCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-huawei", "update-credential", &options.SHuaweiCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-apsara", "update-credential", &options.SApsaraCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-hcso", "update-credential", &options.SHCSOAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-hcs", "update-credential", &options.SHCSOAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-ucloud", "update-credential", &options.SUcloudCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-zstack", "update-credential", &options.SZStackCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-s3", "update-credential", &options.SS3CloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-ctyun", "update-credential", &options.SCtyunCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-jdcloud", "update-credential", &options.SJDcloudCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-cloudpods", "update-credential", &options.SCloudpodsCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-nutanix", "update-credential", &options.SNutanixCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-bingocloud", "update-credential", &options.SBingoCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-incloudsphere", "update-credential", &options.SInCloudSphereAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("update-credential-proxmox", "update-credential", &options.SProxmoxAccountUpdateCredentialOptions{})

	cmd.PerformWithKeyword("test-connectivity-google", "test-connectivity", &options.SGoogleCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-vmware", "test-connectivity", &options.SVMwareCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-aliyun", "test-connectivity", &options.SAliyunCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-azure", "test-connectivity", &options.SAzureCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-qcloud", "test-connectivity", &options.SQcloudCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-aws", "test-connectivity", &options.SAWSCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-openstack", "test-connectivity", &options.SOpenStackCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-huawei", "test-connectivity", &options.SHuaweiCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-ucloud", "test-connectivity", &options.SUcloudCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-zstack", "test-connectivity", &options.SZStackCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-s3", "test-connectivity", &options.SS3CloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-ctyun", "test-connectivity", &options.SCtyunCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-jdcloud", "test-connectivity", &options.SJDcloudCloudAccountUpdateCredentialOptions{})
	cmd.PerformWithKeyword("test-connectivity-cloudpods", "test-connectivity", &options.SCloudpodsCloudAccountUpdateCredentialOptions{})

	cmd.Perform("enable", &options.SCloudAccountIdOptions{})
	cmd.Perform("disable", &options.SCloudAccountIdOptions{})
	cmd.Perform("sync", &options.CloudaccountSyncOptions{})
	cmd.Perform("enable-auto-sync", &options.CloudaccountEnableAutoSyncOptions{})
	cmd.Perform("disable-auto-sync", &options.SCloudAccountIdOptions{})
	cmd.Perform("public", &options.CloudaccountPublicOptions{})
	cmd.Perform("private", &options.SCloudAccountIdOptions{})
	cmd.Perform("share-mode", &options.CloudaccountShareModeOptions{})
	cmd.Perform("sync-skus", &options.CloudaccountSyncSkusOptions{})
	cmd.Perform("change-owner", &options.ClouaccountChangeOwnerOptions{})
	cmd.Perform("change-project", &options.ClouaccountChangeProjectOptions{})
	cmd.Perform("create-subscription", &options.SubscriptionCreateOptions{})
	cmd.Perform("project-mapping", &options.ClouaccountProjectMappingOptions{})

	cmd.Get("change-owner-candidate-domains", &options.SCloudAccountIdOptions{})
	cmd.Get("enrollment-accounts", &options.SCloudAccountIdOptions{})
	cmd.Get("balance", &options.SCloudAccountIdOptions{})
	cmd.Get("saml", &options.SCloudAccountIdOptions{})
}
