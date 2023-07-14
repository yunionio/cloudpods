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

import (
	"fmt"
	"io/ioutil"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type CloudaccountListOptions struct {
	BaseListOptions
	Capability []string `help:"capability filter" choices:"project|compute|network|loadbalancer|objectstore|rds|cache|event|tablestore"`

	//DistinctField string `help:"distinct field"`
	ProxySetting string `help:"Proxy setting id or name"`
	// 按宿主机数量排序
	OrderByHostCount string
	// 按虚拟机数量排序
	OrderByGuestCount string
}

func (opts *CloudaccountListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type SUserPasswordCredential struct {
	Username string `help:"Username" positional:"true"`
	Password string `help:"Password" positional:"true"`
}

type SVMwareCredentialWithEnvironment struct {
	SUserPasswordCredential

	Host string `help:"VMware VCenter/ESXi host" positional:"true"`
	Port string `help:"VMware VCenter/ESXi host port" default:"443"`

	Zone string `help:"zone for this account"`
}

type SNutanixCredentialWithEnvironment struct {
	SUserPasswordCredential

	Host string `help:"Nutanix host" positional:"true"`
	Port string `help:"Nutanix host port" default:"9440"`
}

type SProxmoxCredentialWithEnvironment struct {
	SUserPasswordCredential
	Host string `help:"Proxmox host" positional:"true"`
	Port string `help:"Proxmox host port" default:"8006"`
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

	Project       string `help:"project for this account"`
	ProjectDomain string `help:"domain for this account"`

	Disabled *bool `help:"create cloud account with disabled status"`

	SkipDuplicateAccountCheck bool `help:"skip check duplicate account"`

	SamlAuth string `help:"Enable or disable saml auth" choices:"true|false"`

	ProxySetting    string `help:"proxy setting id or name" json:"proxy_setting"`
	DryRun          bool   `help:"test create cloudaccount params"`
	ShowSubAccounts bool   `help:"test and show subaccount info"`
	ReadOnly        bool   `help:"Read only account"`

	ProjectMappingId   string
	EnableProjectSync  bool
	EnableResourceSync bool

	SkipSyncResources []string `help:"Skip sync resource, etc snapshot"`
}

type SVMwareCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SVMwareCredentialWithEnvironment
}

type SAliyunAccessKeyCredentialWithEnvironment struct {
	SAccessKeyCredential

	Environment string `help:"Cloud environment" choices:"InternationalCloud|FinanceCloud" default:"InternationalCloud"`
}

func (opts *SVMwareCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("VMware"), "provider")
	return params, nil
}

type SAliyunCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAliyunAccessKeyCredentialWithEnvironment

	OptionsBillingReportBucket  string `help:"bucket that stores billing report" json:"-"`
	OptionsBillingBucketAccount string `help:"id of account that can access bucket, blank if this account can access" json:"-"`
	OptionsBillingFilePrefix    string `help:"prefix of billing file name" json:"-"`
}

func (opts *SAliyunCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Aliyun"), "provider")
	options := jsonutils.NewDict()
	if len(opts.OptionsBillingReportBucket) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingReportBucket), "billing_report_bucket")
	}
	if len(opts.OptionsBillingBucketAccount) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingBucketAccount), "billing_bucket_account")
	}
	if len(opts.OptionsBillingFilePrefix) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingFilePrefix), "billing_file_prefix")
	}
	if options.Size() > 0 {
		params.(*jsonutils.JSONDict).Add(options, "options")
	}
	return params, nil
}

type SAzureCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAzureCredentialWithEnvironment
}

func (opts *SAzureCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Azure"), "provider")
	return params, nil
}

type SQcloudCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SQcloudCredential
}

func (opts *SQcloudCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Qcloud"), "provider")
	return params, nil
}

type SGoogleCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	GoogleJsonFile string `help:"Google auth json file" positional:"true"`
}

func parseGcpCredential(filename string) (jsonutils.JSONObject, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	authParams, err := jsonutils.Parse(data)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	for _, k := range []string{
		"client_email",
		"project_id",
		"private_key_id",
		"private_key",
	} {
		v, _ := authParams.Get(k)
		ret.Add(v, fmt.Sprintf("gcp_%s", k))
	}
	return ret, nil
}

func (opts *SGoogleCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Google"), "provider")
	authParams, err := parseGcpCredential(opts.GoogleJsonFile)
	if err != nil {
		return nil, err
	}
	err = jsonutils.Update(params, authParams)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type SAWSCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredentialWithEnvironment

	OptionsBillingReportBucket  string `help:"bucket that stores billing report" json:"-"`
	OptionsBillingBucketAccount string `help:"id of account that can access bucket, blank if this account can access" json:"-"`
	OptionsBillingFilePrefix    string `help:"prefix of billing file name" json:"-"`
	OptionsAssumeRoleName       string `help:"assume role name" json:"-"`
}

func (opts *SAWSCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	options := jsonutils.NewDict()
	if len(opts.OptionsBillingReportBucket) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingReportBucket), "billing_report_bucket")
	}
	if len(opts.OptionsBillingBucketAccount) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingBucketAccount), "billing_bucket_account")
	}
	if len(opts.OptionsBillingFilePrefix) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingFilePrefix), "billing_file_prefix")
	}
	if len(opts.OptionsAssumeRoleName) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsAssumeRoleName), "aws_assume_role_name")
	}
	if options.Size() > 0 {
		params.Add(options, "options")
	}
	params.Add(jsonutils.NewString("Aws"), "provider")
	return params, nil
}

type SOpenStackCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SOpenStackCredentialWithAuthURL
}

func (opts *SOpenStackCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("OpenStack"), "provider")
	return params, nil
}

type SHuaweiCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredentialWithEnvironment
}

func (opts *SHuaweiCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Huawei"), "provider")
	return params, nil
}

type SHCSOAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	cloudprovider.SHCSOEndpoints
	SAccessKeyCredential

	DefaultRegion string `json:"default_region"`
}

func (opts *SHCSOAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("HCSO"), "provider")
	return params, nil
}

type SHCSAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	AuthURL string `help:"Hcs auth_url" positional:"true" json:"auth_url"`
	SAccessKeyCredential
}

func (opts *SHCSAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("HCS"), "provider")
	return params, nil
}

type SUcloudCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredential
}

func (opts *SUcloudCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Ucloud"), "provider")
	return params, nil
}

type SZStackCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SUserPasswordCredential
	AuthURL string `help:"ZStack auth_url" positional:"true" json:"auth_url"`
}

func (opts *SZStackCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("ZStack"), "provider")
	return params, nil
}

type SHcsOpCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SUserPasswordCredential
	AuthURL string `help:"HcsOp auth_url" positional:"true" json:"auth_url"`
}

func (opts *SHcsOpCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("HCSOP"), "provider")
	return params, nil
}

type SS3CloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredential
	Endpoint string `help:"S3 endpoint" required:"true" positional:"true" json:"endpoint"`
}

func (opts *SS3CloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("S3"), "provider")
	return params, nil
}

type SCephCloudAccountCreateOptions struct {
	SS3CloudAccountCreateOptions
}

func (opts *SCephCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Ceph"), "provider")
	return params, nil
}

type SXskyCloudAccountCreateOptions struct {
	SS3CloudAccountCreateOptions
}

func (opts *SXskyCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Xsky"), "provider")
	return params, nil
}

type SCtyunCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredentialWithEnvironment

	cloudprovider.SCtyunExtraOptions
}

func (opts *SCtyunCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Ctyun"), "provider")
	return params, nil
}

type SEcloudCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredentialWithEnvironment
}

func (opts *SEcloudCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Ecloud"), "provider")
	return params, nil
}

type SJDcloudCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredential
}

func (opts *SJDcloudCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("JDcloud"), "provider")
	return params, nil
}

type SCloudpodsCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SAccessKeyCredential
	AuthURL string `help:"Cloudpods auth_url" positional:"true" json:"auth_url"`
}

func (opts *SCloudpodsCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Cloudpods"), "provider")
	return params, nil
}

// update credential options

type SCloudAccountIdOptions struct {
	ID string `help:"ID or Name of cloud account" json:"-"`
}

func (opts *SCloudAccountIdOptions) GetId() string {
	return opts.ID
}

func (opts *SCloudAccountIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type SVMwareCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SUserPasswordCredential
}

func (opts *SVMwareCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SAliyunCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SAliyunCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SApsaraCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
	OrganizationId int
}

func (opts *SApsaraCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SAzureCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAzureCredential
}

func (opts *SAzureCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SQcloudCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SQcloudCredential
}

func (opts *SQcloudCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SAWSCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SAWSCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SOpenStackCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SOpenStackCredential
}

func (opts *SOpenStackCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SHuaweiCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SHuaweiCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SHCSOAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	cloudprovider.SHCSOEndpoints
	SAccessKeyCredential
}

func (opts *SHCSOAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SHCSAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SHCSAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SUcloudCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SUcloudCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SZStackCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SUserPasswordCredential
}

func (opts *SZStackCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SS3CloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SS3CloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SCtyunCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SCtyunCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SJDcloudCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SJDcloudCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SCloudpodsCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SCloudpodsCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.SAccessKeyCredential), nil
}

type SGoogleCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	GoogleJsonFile string `help:"Google auth json file" positional:"true"`
}

func (opts *SGoogleCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return parseGcpCredential(opts.GoogleJsonFile)
}

// update

type SCloudAccountUpdateBaseOptions struct {
	SCloudAccountIdOptions
	Name string `help:"New name to update"`

	SyncIntervalSeconds *int   `help:"auto synchornize interval in seconds"`
	AutoCreateProject   *bool  `help:"automatically create local project for new remote project" negative:"no_auto_create_project"`
	ProxySetting        string `help:"proxy setting name or id" json:"proxy_setting"`
	SamlAuth            string `help:"Enable or disable saml auth" choices:"true|false"`

	ReadOnly *bool `help:"is account read only" negative:"no_read_only"`

	CleanLakeOfPermissions bool `help:"clean lake of permissions"`

	SkipSyncResources       []string
	AddSkipSyncResources    []string
	RemoveSkipSyncResources []string

	Desc string `help:"Description" json:"description" token:"desc"`
}

func (opts *SCloudAccountUpdateBaseOptions) Params() (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("obsolete, please try cloud-account-update-xxx, where xxx is vmware, aliyun, azure, qcloud, aws, openstack, huawei etc.")
}

type SVMwareCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SVMwareCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SAliyunCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions

	OptionsBillingReportBucket        string `help:"update Aliyun S3 bucket that stores account billing report" json:"-"`
	RemoveOptionsBillingReportBucket  bool   `help:"remove Aliyun S3 bucket that stores account billing report" json:"-"`
	OptionsBillingBucketAccount       string `help:"update id of account that can access bucket, blank if this account can access" json:"-"`
	RemoveOptionsBillingBucketAccount bool   `help:"remove id of account that can access bucket, blank if this account can access" json:"-"`
	OptionsBillingFilePrefix          string `help:"update prefix of billing file name" json:"-"`
	RemoveOptionsBillingFilePrefix    bool   `help:"remove prefix of billing file name" json:"-"`
}

func (opts *SAliyunCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)

	options := jsonutils.NewDict()
	if len(opts.OptionsBillingReportBucket) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingReportBucket), "billing_report_bucket")
	}
	if len(opts.OptionsBillingBucketAccount) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingBucketAccount), "billing_bucket_account")
	}
	if len(opts.OptionsBillingFilePrefix) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingFilePrefix), "billing_file_prefix")
	}
	if options.Size() > 0 {
		params.Add(options, "options")
	}
	removeOptions := make([]string, 0)
	if opts.RemoveOptionsBillingReportBucket {
		removeOptions = append(removeOptions, "billing_report_bucket")
	}
	if opts.RemoveOptionsBillingBucketAccount {
		removeOptions = append(removeOptions, "billing_bucket_account")
	}
	if opts.RemoveOptionsBillingFilePrefix {
		removeOptions = append(removeOptions, "billing_file_prefix")
	}
	if len(removeOptions) > 0 {
		params.Add(jsonutils.NewStringArray(removeOptions), "remove_options")
	}
	return params, nil
}

type SAzureCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions

	OptionsBalanceKey       string `help:"update cloud balance account key, such as Azure EA key" json:"-"`
	RemoveOptionsBalanceKey bool   `help:"remove cloud blance account key" json:"-"`
}

func (opts *SAzureCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)

	options := jsonutils.NewDict()
	if len(opts.OptionsBalanceKey) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBalanceKey), "balance_key")
	}
	if options.Size() > 0 {
		params.Add(options, "options")
	}
	removeOptions := make([]string, 0)
	if opts.RemoveOptionsBalanceKey {
		removeOptions = append(removeOptions, "balance_key")
	}
	if len(removeOptions) > 0 {
		params.Add(jsonutils.NewStringArray(removeOptions), "remove_options")
	}
	return params, nil
}

type SQcloudCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SQcloudCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SGoogleCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions

	OptionsBillingReportBigqueryTable   string `help:"update Google big query table that stores account billing report" json:"-"`
	OptionsBillingReportBigqueryAccount string `help:"update Google account for big query table" json:"-"`
	OptionsBillingReportBucket          string `help:"update Google S3 bucket that stores account billing report" json:"-"`
	RemoveOptionsBillingReportBucket    bool   `help:"remove Google S3 bucket that stores account billing report" json:"-"`
	OptionsBillingBucketAccount         string `help:"update id of account that can access bucket, blank if this account can access" json:"-"`
	RemoveOptionsBillingBucketAccount   bool   `help:"remove id of account that can access bucket, blank if this account can access" json:"-"`
	OptionsBillingFilePrefix            string `help:"update prefix of billing file name" json:"-"`
	RemoveOptionsBillingFilePrefix      bool   `help:"remove prefix of billing file name" json:"-"`

	OptionsUsageReportBucket       string `help:"update Google S3 bucket that stores account usage report" json:"-"`
	RemoveOptionsUsageReportBucket bool   `help:"remove Google S3 bucket that stores account usage report" json:"-"`
	OptionsUsageFilePrefix         string `help:"update prefix of usage file name" json:"-"`
	RemoveOptionsUsageFilePrefix   bool   `help:"remove prefix of usage file name" json:"-"`
}

func (opts *SGoogleCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)

	options := jsonutils.NewDict()
	if len(opts.OptionsBillingReportBigqueryTable) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingReportBigqueryTable), "billing_bigquery_table")
	}
	if len(opts.OptionsBillingReportBigqueryAccount) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingReportBigqueryAccount), "billing_bigquery_account")
	}
	if len(opts.OptionsBillingReportBucket) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingReportBucket), "billing_report_bucket")
	}
	if len(opts.OptionsBillingBucketAccount) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingBucketAccount), "billing_bucket_account")
	}
	if len(opts.OptionsBillingFilePrefix) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingFilePrefix), "billing_file_prefix")
	}
	if len(opts.OptionsUsageReportBucket) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsUsageReportBucket), "usage_report_bucket")
	}
	if len(opts.OptionsUsageFilePrefix) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsUsageFilePrefix), "usage_file_prefix")
	}
	if options.Size() > 0 {
		params.Add(options, "options")
	}
	removeOptions := make([]string, 0)
	if opts.RemoveOptionsBillingReportBucket {
		removeOptions = append(removeOptions, "billing_report_bucket")
	}
	if opts.RemoveOptionsBillingBucketAccount {
		removeOptions = append(removeOptions, "billing_bucket_account")
	}
	if opts.RemoveOptionsBillingFilePrefix {
		removeOptions = append(removeOptions, "billing_file_prefix")
	}
	if opts.RemoveOptionsUsageReportBucket {
		removeOptions = append(removeOptions, "usage_report_bucket")
	}
	if opts.RemoveOptionsUsageFilePrefix {
		removeOptions = append(removeOptions, "usage_file_prefix")
	}
	if len(removeOptions) > 0 {
		params.Add(jsonutils.NewStringArray(removeOptions), "remove_options")
	}
	return params, nil
}

type SAWSCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions

	OptionsBillingReportBucket        string `help:"update AWS S3 bucket that stores account billing report" json:"-"`
	RemoveOptionsBillingReportBucket  bool   `help:"remove AWS S3 bucket that stores account billing report" json:"-"`
	OptionsBillingBucketAccount       string `help:"update id of account that can access bucket, blank if this account can access" json:"-"`
	RemoveOptionsBillingBucketAccount bool   `help:"remove id of account that can access bucket, blank if this account can access" json:"-"`
	OptionsBillingFilePrefix          string `help:"update prefix of billing file name" json:"-"`
	RemoveOptionsBillingFilePrefix    bool   `help:"remove prefix of billing file name" json:"-"`
	OptionsAssumeRoleName             string `help:"name of assume role" json:"-"`
	RemoveOptionsAssumeRoleName       bool   `help:"remove option of aws_assume_role_name"`
}

func (opts *SAWSCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)

	options := jsonutils.NewDict()
	if len(opts.OptionsBillingReportBucket) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingReportBucket), "billing_report_bucket")
	}
	if len(opts.OptionsBillingBucketAccount) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingBucketAccount), "billing_bucket_account")
	}
	if len(opts.OptionsBillingFilePrefix) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingFilePrefix), "billing_file_prefix")
	}
	if len(opts.OptionsAssumeRoleName) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsAssumeRoleName), "aws_assume_role_name")
	}
	if options.Size() > 0 {
		params.Add(options, "options")
	}
	removeOptions := make([]string, 0)
	if opts.RemoveOptionsBillingReportBucket {
		removeOptions = append(removeOptions, "billing_report_bucket")
	}
	if opts.RemoveOptionsBillingBucketAccount {
		removeOptions = append(removeOptions, "billing_bucket_account")
	}
	if opts.RemoveOptionsBillingFilePrefix {
		removeOptions = append(removeOptions, "billing_file_prefix")
	}
	if opts.RemoveOptionsAssumeRoleName {
		removeOptions = append(removeOptions, "aws_assume_role_name")
	}
	if len(removeOptions) > 0 {
		params.Add(jsonutils.NewStringArray(removeOptions), "remove_options")
	}
	return params, nil
}

type SOpenStackCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SOpenStackCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SHuaweiCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions

	OptionsBillingReportBucket        string `help:"update Huawei S3 bucket that stores account billing report" json:"-"`
	RemoveOptionsBillingReportBucket  bool   `help:"remove Huawei S3 bucket that stores account billing report" json:"-"`
	OptionsBillingBucketAccount       string `help:"update id of account that can access bucket, blank if this account can access" json:"-"`
	RemoveOptionsBillingBucketAccount bool   `help:"remove id of account that can access bucket, blank if this account can access" json:"-"`
	OptionsBillingFilePrefix          string `help:"update prefix of billing file name" json:"-"`
	RemoveOptionsBillingFilePrefix    bool   `help:"remove prefix of billing file name" json:"-"`
}

func (opts *SHuaweiCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)

	options := jsonutils.NewDict()
	if len(opts.OptionsBillingReportBucket) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingReportBucket), "billing_report_bucket")
	}
	if len(opts.OptionsBillingBucketAccount) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingBucketAccount), "billing_bucket_account")
	}
	if len(opts.OptionsBillingFilePrefix) > 0 {
		options.Add(jsonutils.NewString(opts.OptionsBillingFilePrefix), "billing_file_prefix")
	}
	if options.Size() > 0 {
		params.Add(options, "options")
	}
	removeOptions := make([]string, 0)
	if opts.RemoveOptionsBillingReportBucket {
		removeOptions = append(removeOptions, "billing_report_bucket")
	}
	if opts.RemoveOptionsBillingBucketAccount {
		removeOptions = append(removeOptions, "billing_bucket_account")
	}
	if opts.RemoveOptionsBillingFilePrefix {
		removeOptions = append(removeOptions, "billing_file_prefix")
	}
	if len(removeOptions) > 0 {
		params.Add(jsonutils.NewStringArray(removeOptions), "remove_options")
	}
	return params, nil
}

type SHCSOAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SHCSOAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SHCSAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
	Account  string
	Password string
}

func (opts *SHCSAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts.SCloudAccountUpdateBaseOptions).(*jsonutils.JSONDict)
	options := jsonutils.NewDict()
	if len(opts.Account) > 0 {
		options.Set("account", jsonutils.NewString(opts.Account))
	}
	if len(opts.Password) > 0 {
		options.Set("password", jsonutils.NewString(opts.Password))
	}
	if options.Length() > 0 {
		params.Set("options", options)
	}
	return params, nil
}

type SUcloudCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SUcloudCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SZStackCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SZStackCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SS3CloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SS3CloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SCtyunCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions

	cloudprovider.SCtyunExtraOptions
}

func (opts *SCtyunCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	options := jsonutils.NewDict()
	if len(opts.SCtyunExtraOptions.CrmBizId) > 0 {
		options.Add(jsonutils.NewString(opts.SCtyunExtraOptions.CrmBizId), "crm_biz_id")
	}
	if options.Size() > 0 {
		params.Add(options, "options")
	}

	return params, nil
}

type SJDcloudCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SJDcloudCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SCloudpodsCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SCloudpodsCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

/*type SVMwareCloudAccountPrepareNetsOptions struct {
	SVMwareCredentialWithEnvironment

	Zone          string `help:"zone for this account"`
	Project       string `help:"project for this account"`
	ProjectDomain string `help:"domain for this account"`
	WireLevel     string `help:"wire level for this account" choices:"vcenter|datacenter|cluster" json:"wire_level_for_vmware"`
	Dvs           bool   `help:"whether to enable dvs corresponding wire"`
	NAME          string `help:"name for this account"`
}

func (opts *SVMwareCloudAccountPrepareNetsOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("VMware"), "provider")
	return params, nil
}*/

type SApsaraCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	Endpoint string
	SAccessKeyCredential
}

func (opts *SApsaraCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Apsara"), "provider")
	return params, nil
}

type CloudaccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	AccessKeyID     string `help:"Aiyun|HuaWei|Aws access_key_id"`
	AccessKeySecret string `help:"Aiyun|HuaWei|Aws access_key_secret"`
	AppID           string `help:"Qcloud appid"`
	SecretID        string `help:"Qcloud secret_id"`
	SecretKey       string `help:"Qcloud secret_key"`
	ProjectName     string `help:"OpenStack project_name"`
	Username        string `help:"OpenStack|VMware username"`
	Password        string `help:"OpenStack|VMware password"`
	EndpointType    string `help:"OpenStack endpointType"`
	ClientID        string `help:"Azure tenant_id"`
	ClientSecret    string `help:"Azure clinet_secret"`
}

func (opts *CloudaccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("obsolete command, please try cloud-account-update-credential-xxx, where xxx is vmware, aliyun, azure, qcloud, aws, openstack, huawei, etc.")
}

type CloudaccountSyncOptions struct {
	SCloudAccountIdOptions

	api.SyncRangeInput
}

func (opts *CloudaccountSyncOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type CloudaccountEnableAutoSyncOptions struct {
	SCloudAccountIdOptions
	SyncIntervalSeconds int `help:"new sync interval in seconds"`
}

func (opts *CloudaccountEnableAutoSyncOptions) Params() (jsonutils.JSONObject, error) {
	return StructToParams(opts)
}

type CloudaccountPublicOptions struct {
	SCloudAccountIdOptions
	Scope         string   `help:"public_sccope" choices:"domain|system" json:"scope"`
	SharedDomains []string `help:"shared domains" json:"shared_domains"`
	ShareMode     string   `help:"share_mode" choices:"account_domain|provider_domain|system"`
}

func (opts *CloudaccountPublicOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type CloudaccountShareModeOptions struct {
	SCloudAccountIdOptions
	MODE string `help:"cloud account share mode" choices:"account_domain|system|provider_domain"`
}

func (opts *CloudaccountShareModeOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"share_mode": opts.MODE}), nil
}

type CloudaccountSyncSkusOptions struct {
	SCloudAccountIdOptions
	RESOURCE      string `help:"Resource of skus" choices:"serversku|elasticcachesku|dbinstance_sku|nat_sku|nas_sku"`
	Force         bool   `help:"Force sync no matter what"`
	Cloudprovider string `help:"provider to sync"`
	Region        string `help:"region to sync"`
}

func (opts *CloudaccountSyncSkusOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("resource", jsonutils.NewString(opts.RESOURCE))
	if opts.Force {
		params.Add(jsonutils.JSONTrue, "force")
	}

	if len(opts.Cloudprovider) > 0 {
		params.Add(jsonutils.NewString(opts.Cloudprovider), "cloudprovider")
	}

	if len(opts.Region) > 0 {
		params.Add(jsonutils.NewString(opts.Region), "cloudregion")
	}
	return params, nil
}

type ClouaccountChangeOwnerOptions struct {
	SCloudAccountIdOptions
	ProjectDomain string `json:"project_domain" help:"target domain"`
}

func (opts *ClouaccountChangeOwnerOptions) Params() (jsonutils.JSONObject, error) {
	if len(opts.ProjectDomain) == 0 {
		return nil, fmt.Errorf("empty project_domain")
	}
	return jsonutils.Marshal(opts), nil
}

type ClouaccountChangeProjectOptions struct {
	SCloudAccountIdOptions
	PROJECT string `json:"project" help:"target domain"`
}

func (opts *ClouaccountChangeProjectOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SubscriptionCreateOptions struct {
	SCloudAccountIdOptions
	NAME              string
	ENROLLMENTACCOUNT string
	OfferType         string `choices:"MS-AZR-0148P|MS-AZR-0017P" default:"MS-AZR-0017P"`
}

func (opts *SubscriptionCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{
		"name":                  opts.NAME,
		"offer_type":            opts.OfferType,
		"enrollment_account_id": opts.ENROLLMENTACCOUNT,
	}), nil
}

type ClouaccountProjectMappingOptions struct {
	SCloudAccountIdOptions
	ProjectId          string `json:"project_id" help:"default project id"`
	AutoCreateProject  bool   `help:"auto create project"`
	ProjectMappingId   string `json:"project_mapping_id" help:"project mapping id"`
	EnableProjectSync  bool
	EnableResourceSync bool
}

func (opts *ClouaccountProjectMappingOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SNutanixCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SNutanixCredentialWithEnvironment
}

func (opts *SNutanixCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Nutanix"), "provider")
	return params, nil
}

type SNutanixCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SUserPasswordCredential
}

func (opts *SNutanixCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.SUserPasswordCredential), nil
}

type SNutanixCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SNutanixCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SBingoCloudAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	Endpoint string
	SAccessKeyCredential
}

func (opts *SBingoCloudAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString("BingoCloud"), "provider")
	return params, nil
}

type SBingoCloudAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SBingoCloudAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SBingoCloudAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SBingoCloudAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.SAccessKeyCredential), nil
}

type SInCloudSphereAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	HOST string
	SAccessKeyCredential
}

func (opts *SInCloudSphereAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString(api.CLOUD_PROVIDER_INCLOUD_SPHERE), "provider")
	return params, nil
}

type SInCloudSphereAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SInCloudSphereAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SInCloudSphereAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SAccessKeyCredential
}

func (opts *SInCloudSphereAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.SAccessKeyCredential), nil
}

type SProxmoxAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	SProxmoxCredentialWithEnvironment
}

func (opts *SProxmoxAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString(api.CLOUD_PROVIDER_PROXMOX), "provider")
	return params, nil
}

type SProxmoxAccountUpdateOptions struct {
	SCloudAccountUpdateBaseOptions
}

func (opts *SProxmoxAccountUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SProxmoxAccountUpdateCredentialOptions struct {
	SCloudAccountIdOptions
	SUserPasswordCredential
}

func (opts *SProxmoxAccountUpdateCredentialOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.SUserPasswordCredential), nil
}

type SRemoteFileAccountCreateOptions struct {
	SCloudAccountCreateBaseOptions
	AuthURL string
}

func (opts *SRemoteFileAccountCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	params.(*jsonutils.JSONDict).Add(jsonutils.NewString(api.CLOUD_PROVIDER_REMOTEFILE), "provider")
	return params, nil
}
