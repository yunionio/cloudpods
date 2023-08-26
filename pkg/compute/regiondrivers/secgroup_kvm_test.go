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

package regiondrivers

import (
	"testing"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

func TestKvmRuleSync(t *testing.T) {
	data := []TestData{
		{
			Name:      "Test kvm deny rules",
			SrcRules:  cloudprovider.SecurityRuleSet{},
			DestRules: []cloudprovider.SecurityRule{},
			Common:    []cloudprovider.SecurityRule{},
			InAdds:    []cloudprovider.SecurityRule{},
			OutAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "out:deny any", 1),
			},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
	}

	for _, d := range data {
		d.Test(t, &SAzureRegionDriver{}, &SKVMRegionDriver{})
	}

	aliyun := []TestData{

		{
			Name: "Test aliyun rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithName("allow tcp 443", "in:allow tcp 443", 1),
				ruleWithName("allow tcp 6379", "in:allow tcp 6379", 1),
				ruleWithName("allow tcp 3389", "in:allow tcp 3389", 1),
				ruleWithName("allow tcp 1521", "in:allow tcp 1521", 1),
				ruleWithName("allow tcp 80", "in:allow tcp 80", 1),
				ruleWithName("deny tcp 1521", "in:deny tcp 1521", 12),
			},
			DestRules: []cloudprovider.SecurityRule{
				ruleWithName("allow tcp", "in:allow tcp", 51),
				ruleWithName("allow tcp 1521", "in:allow tcp 1521", 50),
				ruleWithName("allow tcp 3389", "in:allow tcp 3389", 50),
				ruleWithName("allow tcp 443", "in:allow tcp 443", 50),
				ruleWithName("allow tcp 6379", "in:allow tcp 6379", 50),
				ruleWithName("allow tcp 80", "in:allow tcp 80", 50),
				ruleWithName("allow tcp", "in:allow tcp", 1),
			},
			Common: []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{
				ruleWithName("", "in:allow tcp 80,443,1521,3389,6379", 1),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels: []cloudprovider.SecurityRule{
				ruleWithName("allow tcp", "in:allow tcp", 51),
				ruleWithName("allow tcp", "in:allow tcp", 1),
				ruleWithName("allow tcp 1521", "in:allow tcp 1521", 50),
				ruleWithName("allow tcp 3389", "in:allow tcp 3389", 50),
				ruleWithName("allow tcp 443", "in:allow tcp 443", 50),
				ruleWithName("allow tcp 6379", "in:allow tcp 6379", 50),
				ruleWithName("allow tcp 80", "in:allow tcp 80", 50),
			},
			OutDels: []cloudprovider.SecurityRule{},
		},
		{
			Name: "Test aliyun peer rules",
			SrcRules: cloudprovider.SecurityRuleSet{
				ruleWithPeerSecgroup("allow tcp 443", "in:allow tcp 443", 1, "peer1"),
				ruleWithName("deny tcp 1521", "in:deny tcp 1521", 12),
			},
			DestRules: []cloudprovider.SecurityRule{},
			Common:    []cloudprovider.SecurityRule{},
			InAdds: []cloudprovider.SecurityRule{
				ruleWithPeerSecgroup("allow tcp 443", "in:allow tcp 443", 3, "peer1"),
				ruleWithName("deny tcp 1521", "in:deny tcp 1521", 2),
			},
			OutAdds: []cloudprovider.SecurityRule{},
			InDels:  []cloudprovider.SecurityRule{},
			OutDels: []cloudprovider.SecurityRule{},
		},
	}

	for _, d := range aliyun {
		d.Test(t, &SAliyunRegionDriver{}, &SKVMRegionDriver{})
	}

}
