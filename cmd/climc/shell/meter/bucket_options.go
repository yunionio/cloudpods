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

package meter

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type BucketOptionsVerifyOptions struct {
		options.BaseListOptions
		CloudaccountId       string `help:"cloudaccount Id" required:"true"`
		BillingReportBucket  string `help:"S3 bucket that stores account billing report"`
		BillingFilePrefix    string `help:"prefix of billing file name"`
		BillingBucketAccount string `help:"Id of cloudaccount that can access billing bucket"`
		UsageReportBucket    string `help:"S3 bucket that stores account usage report"`
		UsageFilePrefix      string `help:"prefix of usage file name"`
		UsageBucketAccount   string `help:"Id of cloudaccount that can access usage bucket"`
	}
	R(&BucketOptionsVerifyOptions{}, "bucket-options-verify", "billing bucket connection test",
		func(s *mcclient.ClientSession, args *BucketOptionsVerifyOptions) error {
			params := jsonutils.Marshal(args)
			result, err := modules.BucketOptions.PerformAction(s, args.CloudaccountId, "verify", params)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})
}
