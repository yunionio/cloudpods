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

package options

type SUserPasswordCredential struct {
	Username string `help:"Username" positional:"true"`
	Password string `help:"Password" positional:"true"`
}

type SVMwareCredentialWithEnvironment struct {
	SUserPasswordCredential

	Host string `help:"VMware VCenter/ESXi host" positional:"true"`
	Port string `help:"VMware VCenter/ESXi host port" default:"443"`
}

type SAzureCredential struct {
	ClientID     string `help:"Azure client_id" positional:"true"`
	ClientSecret string `help:"Azure clinet_secret" positional:"true"`
}

type SAzureCredentialWithEnvironment struct {
	DirectoryID string `help:"Azure directory_id" positional:"true"`

	SAzureCredential

	Environment string `help:"Cloud environment" choices:"AzureGermanCloud|AzureChinaCloud|AzurePublicCloud" default:"AzureChinaCloud"`
}

type SQcloudCredential struct {
	AppID     string `help:"Qcloud appid" positional:"true"`
	SecretID  string `help:"Qcloud secret_id" positional:"true"`
	SecretKey string `help:"Qcloud secret_key" positional:"true"`
}

type SOpenStackCredential struct {
	ProjectName string `help:"OpenStack project_name" positional:"true"`

	SUserPasswordCredential

	DomainName string `help:"OpenStack domain name"`
}

type SOpenStackCredentialWithAuthURL struct {
	SOpenStackCredential

	AuthURL string `help:"OpenStack auth_url" positional:"true" json:"auth_url"`
}

type SAccessKeyCredential struct {
	AccessKeyID     string `help:"Access_key_id" positional:"true"`
	AccessKeySecret string `help:"Access_key_secret" positional:"true"`
}

type SAccessKeyCredentialWithEnvironment struct {
	SAccessKeyCredential
	Environment string `help:"Cloud environment" choices:"InternationalCloud|ChinaCloud" default:"ChinaCloud"`
}

/// create options

type SCloudAccountCreateBaseOptions struct {
	Name string `help:"Name of cloud account" positional:"true"`
	// PROVIDER string `help:"Driver for cloud account" choices:"VMware|Aliyun|Azure|Qcloud|OpenStack|Huawei|Aws"`
	Desc  string `help:"Description" token:"desc" json:"description"`
	Brand string `help:"Brand of cloud account" choices:"DStack"`

	AutoCreateProject bool `help:"Enable the account with same name project"`
	EnableAutoSync    bool `help:"Enable automatically synchronize resources of this account"`

	SyncIntervalSeconds int `help:"Interval to synchronize if auto sync is enable" metavar:"SECONDS"`

	ProjectDomain string `help:"domain for this account"`
}

type SVMwareCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SVMwareCredentialWithEnvironment
}

type SAliyunCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredential
}

type SAzureCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAzureCredentialWithEnvironment
}

type SQcloudCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SQcloudCredential
}

type SAWSCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredentialWithEnvironment

	OptionsBillingReportBucket string `help:"bucket that stores billing report" json:"-"`
}

type SOpenStackCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SOpenStackCredentialWithAuthURL
}

type SHuaweiCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredentialWithEnvironment
}

type SUcloudCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredential
}

type SZStackCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SUserPasswordCredential
	AuthURL string `help:"ZStack auth_url" positional:"true" json:"auth_url"`
}

type SS3CloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredential
	Endpoint string `help:"S3 endpoint" required:"true" positional:"true" json:"endpoint"`
}

// update credential options

type SCloudAccountUpdateCredentialBaseOptions struct {
	ID string `help:"ID or Name of cloud account" json:"-"`
}

type SVMwareCloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SUserPasswordCredential
}

type SAliyunCloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SAccessKeyCredential
}

type SAzureCloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SAzureCredential
}

type SQcloudCloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SQcloudCredential
}

type SAWSCloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SAccessKeyCredential
}

type SOpenStackCloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SOpenStackCredential
}

type SHuaweiCloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SAccessKeyCredential
}

type SUcloudCloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SAccessKeyCredential
}

type SZStackCloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SUserPasswordCredential
}

type SS3CloudAccountUpdateCredentialOptions struct {
	SCloudAccountUpdateCredentialBaseOptions
	SAccessKeyCredential
}

// update

type SCloudAccountUpdateBaseOptions struct {
	ID   string `help:"ID or Name of cloud account" json:"-"`
	Name string `help:"New name to update"`

	SyncIntervalSeconds int   `help:"auto synchornize interval in seconds"`
	AutoCreateProject   *bool `help:"automatically create local project for new remote project"`

	Desc string `help:"Description" json:"description" token:"desc"`
}

type SVMwareCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

type SAliyunCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

type SAzureCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions

	OptionsBalanceKey       string `help:"update cloud balance account key, such as Azure EA key" json:"-"`
	RemoveOptionsBalanceKey bool   `help:"remove cloud blance account key" json:"-"`
}

type SQcloudCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

type SAWSCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions

	OptionsBillingReportBucket       string `help:"update AWS S3 bucket that stores account billing report" json:"-"`
	RemoveOptionsBillingReportBucket bool   `help:"remote AWS S3 bucket that stores account billing report" json:"-"`
}

type SOpenStackCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

type SHuaweiCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

type SUcloudCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

type SZStackCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

type SS3CloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}
