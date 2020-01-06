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

package shell

import (
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type CloudaccountListOptions struct {
		options.BaseListOptions
	}
	R(&CloudaccountListOptions{}, "cloud-account-list", "List cloud accounts", func(s *mcclient.ClientSession, args *CloudaccountListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
		}
		result, err := modules.Cloudaccounts.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cloudaccounts.GetColumns(s))
		return nil
	})

	type CloudaccountCreateOptions struct {
		AccessKeyID     string `help:"Aiyun|HuaWei|Aws access_key_id"`
		AccessKeySecret string `help:"Aiyun|HuaWei|Aws access_key_secret"`
		AppID           string `help:"Qcloud appid"`
		SecretID        string `help:"Qcloud secret_id"`
		SecretKey       string `help:"Qcloud secret_key"`
		ProjectName     string `help:"OpenStack project_name"`
		Username        string `help:"OpenStack|VMware username"`
		Password        string `help:"OpenStack|VMware password"`
		AuthURL         string `help:"OpenStack auth_url"`
		Host            string `help:"VMware host"`
		Port            string `help:"VMware host port" default:"443"`
		DirectoryID     string `help:"Azure directory_id"`
		ClientID        string `help:"Azure client_id"`
		ClientSecret    string `help:"Azure clinet_secret"`
		Environment     string `help:"Azure|Huawei|Aws environment" choices:"AzureGermanCloud|AzureChinaCloud|AzureUSGovernmentCloud|AzurePublicCloud|InternationalCloud|ChinaCloud|"`

		Enabled bool `help:"Enabled the account automatically"`

		Import   bool `help:"Import all sub account automatically"`
		AutoSync bool `help:"Enabled the account automatically"`
	}
	R(&CloudaccountCreateOptions{}, "cloud-account-create", "Create a cloud account", func(s *mcclient.ClientSession, args *CloudaccountCreateOptions) error {
		return fmt.Errorf("obsolete, please try cloud-account-create-xxx, where xxx is vmware, aliyun, azure, qcloud, aws, openstack, huawei etc.")
	})

	R(&options.SVMwareCloudAccountCreateOptions{}, "cloud-account-create-vmware", "Create a VMware cloud account", func(s *mcclient.ClientSession, args *options.SVMwareCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("VMware"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SAliyunCloudAccountCreateOptions{}, "cloud-account-create-aliyun", "Create an Aliyun cloud account", func(s *mcclient.ClientSession, args *options.SAliyunCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Aliyun"), "provider")
		options := jsonutils.NewDict()
		if len(args.OptionsBillingReportBucket) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingReportBucket), "billing_report_bucket")
		}
		if len(args.OptionsBillingBucketAccount) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingBucketAccount), "billing_bucket_account")
		}
		if options.Size() > 0 {
			params.(*jsonutils.JSONDict).Add(options, "options")
		}
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SAzureCloudAccountCreateOptions{}, "cloud-account-create-azure", "Create an Azure cloud account", func(s *mcclient.ClientSession, args *options.SAzureCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Azure"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SQcloudCloudAccountCreateOptions{}, "cloud-account-create-qcloud", "Create a Qcloud cloud account", func(s *mcclient.ClientSession, args *options.SQcloudCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Qcloud"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SGoogleCloudAccountCreateOptions{}, "cloud-account-create-google", "Create a Google cloud account", func(s *mcclient.ClientSession, args *options.SGoogleCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Google"), "provider")
		data, err := ioutil.ReadFile(args.GoogleJsonFile)
		if err != nil {
			return err
		}
		authParams, err := jsonutils.Parse(data)
		if err != nil {
			return err
		}
		err = jsonutils.Update(params, authParams)
		if err != nil {
			return err
		}
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SAWSCloudAccountCreateOptions{}, "cloud-account-create-aws", "Create an AWS cloud account", func(s *mcclient.ClientSession, args *options.SAWSCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		options := jsonutils.NewDict()
		if len(args.OptionsBillingReportBucket) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingReportBucket), "billing_report_bucket")
		}
		if len(args.OptionsBillingBucketAccount) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingBucketAccount), "billing_bucket_account")
		}
		if len(args.OptionsBillingFileAccount) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingFileAccount), "billing_file_account")
		}
		if options.Size() > 0 {
			params.Add(options, "options")
		}
		params.Add(jsonutils.NewString("Aws"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SOpenStackCloudAccountCreateOptions{}, "cloud-account-create-openstack", "Create an OpenStack cloud account", func(s *mcclient.ClientSession, args *options.SOpenStackCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("OpenStack"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SHuaweiCloudAccountCreateOptions{}, "cloud-account-create-huawei", "Create a Huawei cloud account", func(s *mcclient.ClientSession, args *options.SHuaweiCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Huawei"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SUcloudCloudAccountCreateOptions{}, "cloud-account-create-ucloud", "Create a Ucloud cloud account", func(s *mcclient.ClientSession, args *options.SUcloudCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Ucloud"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SZStackCloudAccountCreateOptions{}, "cloud-account-create-zstack", "Create a ZStack cloud account", func(s *mcclient.ClientSession, args *options.SZStackCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("ZStack"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SS3CloudAccountCreateOptions{}, "cloud-account-create-s3", "Create a generaic S3 object storage account", func(s *mcclient.ClientSession, args *options.SS3CloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("S3"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SS3CloudAccountCreateOptions{}, "cloud-account-create-ceph", "Create a ceph object storage account", func(s *mcclient.ClientSession, args *options.SS3CloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Ceph"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SS3CloudAccountCreateOptions{}, "cloud-account-create-xsky", "Create a xsky object storage account", func(s *mcclient.ClientSession, args *options.SS3CloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Xsky"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SCtyunCloudAccountCreateOptions{}, "cloud-account-create-ctyun", "Create a Ctyun cloud account", func(s *mcclient.ClientSession, args *options.SCtyunCloudAccountCreateOptions) error {
		params := jsonutils.Marshal(args)
		params.(*jsonutils.JSONDict).Add(jsonutils.NewString("Ctyun"), "provider")
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountUpdateOptions struct {
		ID        string `help:"ID or Name of cloud account"`
		Name      string `help:"New name to update"`
		AccessUrl string `help:"New access url"`

		SyncIntervalSeconds int    `help:"auto synchornize interval in seconds"`
		BalanceKey          string `help:"update cloud balance account key, such as Azure EA key"`
		RemoveBalanceKey    bool   `help:"remove cloud blance account key"`

		Desc string `help:"Description"`
	}
	R(&CloudaccountUpdateOptions{}, "cloud-account-update", "Update a cloud account", func(s *mcclient.ClientSession, args *CloudaccountUpdateOptions) error {
		return fmt.Errorf("obsolete, please try cloud-account-update-xxx, where xxx is vmware, aliyun, azure, qcloud, aws, openstack, huawei etc.")
	})

	R(&options.SVMwareCloudAccountUpdateOptions{}, "cloud-account-update-vmware", "update a vmware cloud account", func(s *mcclient.ClientSession, args *options.SVMwareCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SAliyunCloudAccountUpdateOptions{}, "cloud-account-update-aliyun", "update an Aliyun cloud account", func(s *mcclient.ClientSession, args *options.SAliyunCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}

		options := jsonutils.NewDict()
		if len(args.OptionsBillingReportBucket) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingReportBucket), "billing_report_bucket")
		}
		if len(args.OptionsBillingBucketAccount) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingBucketAccount), "billing_bucket_account")
		}
		if options.Size() > 0 {
			params.Add(options, "options")
		}
		removeOptions := make([]string, 0)
		if args.RemoveOptionsBillingReportBucket {
			removeOptions = append(removeOptions, "billing_report_bucket")
		}
		if args.RemoveOptionsBillingBucketAccount {
			removeOptions = append(removeOptions, "billing_bucket_account")
		}
		if len(removeOptions) > 0 {
			params.Add(jsonutils.NewStringArray(removeOptions), "remove_options")
		}

		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SAzureCloudAccountUpdateOptions{}, "cloud-account-update-azure", "update an Azure cloud account", func(s *mcclient.ClientSession, args *options.SAzureCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)

		options := jsonutils.NewDict()
		if len(args.OptionsBalanceKey) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBalanceKey), "balance_key")
		}
		if options.Size() > 0 {
			params.Add(options, "options")
		}
		removeOptions := make([]string, 0)
		if args.RemoveOptionsBalanceKey {
			removeOptions = append(removeOptions, "balance_key")
		}
		if len(removeOptions) > 0 {
			params.Add(jsonutils.NewStringArray(removeOptions), "remove_options")
		}

		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SQcloudCloudAccountUpdateOptions{}, "cloud-account-update-qcloud", "update a Tencent cloud account", func(s *mcclient.ClientSession, args *options.SQcloudCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SAWSCloudAccountUpdateOptions{}, "cloud-account-update-aws", "update an AWS cloud account", func(s *mcclient.ClientSession, args *options.SAWSCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)

		options := jsonutils.NewDict()
		if len(args.OptionsBillingReportBucket) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingReportBucket), "billing_report_bucket")
		}
		if len(args.OptionsBillingBucketAccount) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingBucketAccount), "billing_bucket_account")
		}
		if len(args.OptionsBillingFileAccount) > 0 {
			options.Add(jsonutils.NewString(args.OptionsBillingFileAccount), "billing_file_account")
		}
		if options.Size() > 0 {
			params.Add(options, "options")
		}
		removeOptions := make([]string, 0)
		if args.RemoveOptionsBillingReportBucket {
			removeOptions = append(removeOptions, "billing_report_bucket")
		}
		if args.RemoveOptionsBillingBucketAccount {
			removeOptions = append(removeOptions, "billing_bucket_account")
		}
		if args.RemoveOptionsBillingFileAccount {
			removeOptions = append(removeOptions, "billing_file_account")
		}
		if len(removeOptions) > 0 {
			params.Add(jsonutils.NewStringArray(removeOptions), "remove_options")
		}

		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SOpenStackCloudAccountUpdateOptions{}, "cloud-account-update-openstack", "update an AWS cloud account", func(s *mcclient.ClientSession, args *options.SOpenStackCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SHuaweiCloudAccountUpdateOptions{}, "cloud-account-update-huawei", "update a Huawei cloud account", func(s *mcclient.ClientSession, args *options.SHuaweiCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SUcloudCloudAccountUpdateOptions{}, "cloud-account-update-ucloud", "update a Ucloud cloud account", func(s *mcclient.ClientSession, args *options.SUcloudCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SZStackCloudAccountUpdateOptions{}, "cloud-account-update-zstack", "update a ZStack cloud account", func(s *mcclient.ClientSession, args *options.SZStackCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SS3CloudAccountUpdateOptions{}, "cloud-account-update-s3", "update a generic S3 cloud account", func(s *mcclient.ClientSession, args *options.SS3CloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SCtyunCloudAccountUpdateOptions{}, "cloud-account-update-ctyun", "update a Ctyun cloud account", func(s *mcclient.ClientSession, args *options.SCtyunCloudAccountUpdateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountShowOptions struct {
		ID string `help:"ID or Name of cloud account"`
	}
	R(&CloudaccountShowOptions{}, "cloud-account-show", "Get details of a cloud account", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudaccountShowOptions{}, "cloud-account-delete", "Delete a cloud account", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudaccountShowOptions{}, "cloud-account-enable", "Enable cloud account", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudaccountShowOptions{}, "cloud-account-disable", "Disable cloud account", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudaccountShowOptions{}, "cloud-account-balance", "Get balance", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.GetSpecific(s, args.ID, "balance", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountImportOptions struct {
		ID                string `help:"ID or Name of cloud account" json:"-"`
		AutoSync          bool   `help:"Import sub accounts with enabled status"`
		AutoCreateProject bool   `help:"Import sub account with project"`
	}
	R(&CloudaccountImportOptions{}, "cloud-account-import", "Import sub cloud account", func(s *mcclient.ClientSession, args *CloudaccountImportOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "import", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountUpdateCredentialOptions struct {
		ID              string `help:"ID or Name of cloud account"`
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
	R(&CloudaccountUpdateCredentialOptions{}, "cloud-account-update-credential", "Update credential of a cloud account", func(s *mcclient.ClientSession, args *CloudaccountUpdateCredentialOptions) error {
		return fmt.Errorf("obsolete command, please try cloud-account-update-credential-xxx, where xxx is vmware, aliyun, azure, qcloud, aws, openstack, huawei, etc.")
	})

	R(&options.SVMwareCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-vmware", "Update credential of a VMware cloud account", func(s *mcclient.ClientSession, args *options.SVMwareCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SAliyunCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-aliyun", "Update credential of an Aliyun cloud account", func(s *mcclient.ClientSession, args *options.SAliyunCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SAzureCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-azure", "Update credential of an Azure cloud account", func(s *mcclient.ClientSession, args *options.SAzureCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SQcloudCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-qcloud", "Update credential of a Qcloud cloud account", func(s *mcclient.ClientSession, args *options.SQcloudCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SAWSCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-aws", "Update credential of an AWS cloud account", func(s *mcclient.ClientSession, args *options.SAWSCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SOpenStackCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-openstack", "Update credential of an OpenStack cloud account", func(s *mcclient.ClientSession, args *options.SOpenStackCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SHuaweiCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-huawei", "Update credential of an Huawei cloud account", func(s *mcclient.ClientSession, args *options.SHuaweiCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SUcloudCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-ucloud", "Update credential of a Ucloud cloud account", func(s *mcclient.ClientSession, args *options.SUcloudCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SZStackCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-zstack", "Update credential of a ZStack cloud account", func(s *mcclient.ClientSession, args *options.SZStackCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SS3CloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-s3", "Update credential of a generic S3 cloud account", func(s *mcclient.ClientSession, args *options.SS3CloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.SCtyunCloudAccountUpdateCredentialOptions{}, "cloud-account-update-credential-ctyun", "Update credential of an Ctyun cloud account", func(s *mcclient.ClientSession, args *options.SCtyunCloudAccountUpdateCredentialOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountSyncOptions struct {
		ID       string   `help:"ID or Name of cloud account"`
		Force    bool     `help:"Force sync no matter what"`
		FullSync bool     `help:"Synchronize everything"`
		Region   []string `help:"region to sync"`
		Zone     []string `help:"region to sync"`
		Host     []string `help:"region to sync"`
	}
	R(&CloudaccountSyncOptions{}, "cloud-account-sync", "Sync of a cloud account account", func(s *mcclient.ClientSession, args *CloudaccountSyncOptions) error {
		params := jsonutils.NewDict()
		if args.Force {
			params.Add(jsonutils.JSONTrue, "force")
		}
		if args.FullSync {
			params.Add(jsonutils.JSONTrue, "full_sync")
		}
		if len(args.Region) > 0 {
			params.Add(jsonutils.NewStringArray(args.Region), "region")
		}
		if len(args.Zone) > 0 {
			params.Add(jsonutils.NewStringArray(args.Zone), "zone")
		}
		if len(args.Host) > 0 {
			params.Add(jsonutils.NewStringArray(args.Host), "host")
		}
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "sync", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountEnableAutoSyncOptions struct {
		ID                  string `help:"ID or name of cloud account" json:"-"`
		SyncIntervalSeconds int    `help:"new sync interval in seconds"`
	}
	R(&CloudaccountEnableAutoSyncOptions{}, "cloud-account-enable-auto-sync", "Enable automatic sync for this account", func(s *mcclient.ClientSession, args *CloudaccountEnableAutoSyncOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "enable-auto-sync", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountDisableAutoSyncOptions struct {
		ID string `help:"ID or name of cloud account" json:"-"`
	}
	R(&CloudaccountDisableAutoSyncOptions{}, "cloud-account-disable-auto-sync", "Disable automatic sync for this account", func(s *mcclient.ClientSession, args *CloudaccountDisableAutoSyncOptions) error {
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "disable-auto-sync", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountPublicOptions struct {
		ID string `help:"ID or name of cloud account"`
	}
	R(&CloudaccountPublicOptions{}, "cloud-account-public", "Mark this cloud account public ", func(s *mcclient.ClientSession, args *CloudaccountPublicOptions) error {
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "public", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&CloudaccountPublicOptions{}, "cloud-account-private", "Mark this cloud account private", func(s *mcclient.ClientSession, args *CloudaccountPublicOptions) error {
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "private", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountShareModeOptions struct {
		ID   string `help:"ID or name of cloud account"`
		MODE string `help:"cloud account share mode" choices:"account_domain|system|provider_domain"`
	}
	R(&CloudaccountShareModeOptions{}, "cloud-account-share-mode", "Set share_mode of a cloud account", func(s *mcclient.ClientSession, args *CloudaccountShareModeOptions) error {
		input := api.CloudaccountShareModeInput{}
		input.ShareMode = args.MODE
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "share-mode", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountSyncSkusOptions struct {
		ID       string `help:"ID or Name of cloud account"`
		RESOURCE string `help:"Resource of skus" choices:"serversku|elasticcachesku|dbinstance_sku"`
		Force    bool   `help:"Force sync no matter what"`
		Provider string `help:"provider to sync"`
		Region   string `help:"region to sync"`
	}
	R(&CloudaccountSyncSkusOptions{}, "cloud-account-sync-skus", "Sync skus of a cloud account", func(s *mcclient.ClientSession, args *CloudaccountSyncSkusOptions) error {
		params := jsonutils.NewDict()
		params.Set("resource", jsonutils.NewString(args.RESOURCE))
		if args.Force {
			params.Add(jsonutils.JSONTrue, "force")
		}

		if len(args.Provider) > 0 {
			params.Add(jsonutils.NewString(args.Provider), "cloudprovider")
		}

		if len(args.Region) > 0 {
			params.Add(jsonutils.NewString(args.Region), "cloudregion")
		}

		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "sync-skus", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
