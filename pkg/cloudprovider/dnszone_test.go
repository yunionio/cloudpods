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
	"testing"

	"yunion.io/x/jsonutils"
)

func TestDiff(t *testing.T) {
	cases := []struct {
		Name        string
		Remote      []DnsRecordSet
		Local       []DnsRecordSet
		CommonCount int
		AddCount    int
		DelCount    int
		UpdateCount int
	}{
		{
			Name:        "Test delete",
			CommonCount: 4,
			AddCount:    0,
			DelCount:    1,
			UpdateCount: 0,
			Remote: []DnsRecordSet{
				DnsRecordSet{ExternalId: "650124294", Enabled: true, DnsName: "@", DnsType: DnsTypeNS, DnsValue: "f1g1ns1.dnspod.net.", Ttl: 86400, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "650124301", Enabled: true, DnsName: "@", DnsType: DnsTypeNS, DnsValue: "f1g1ns2.dnspod.net.", Ttl: 86400, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "650124650", Enabled: true, DnsName: "@", DnsType: DnsTypeMX, DnsValue: "qiye163mx01.mxmail.netease.com.", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "650124659", Enabled: true, DnsName: "@", DnsType: DnsTypeMX, DnsValue: "qiye163mx02.mxmail.netease.com.", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "650124661", Enabled: true, DnsName: "mail", DnsType: DnsTypeCNAME, DnsValue: "qiye.163.com.", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
			},
			Local: []DnsRecordSet{
				DnsRecordSet{Id: "d599c0e0-0653-40ed-85e1-86502a8d23d4", Enabled: true, DnsName: "mail", DnsType: DnsTypeCNAME, DnsValue: "qiye.163.com.", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Id: "5728b06e-f8cb-41eb-86e9-0e5836195ad1", Enabled: true, DnsName: "@", DnsType: DnsTypeMX, DnsValue: "qiye163mx01.mxmail.netease.com.", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Id: "427b38d2-77e2-4705-8880-0852da9cfb6b", Enabled: true, DnsName: "@", DnsType: DnsTypeNS, DnsValue: "f1g1ns1.dnspod.net.", Ttl: 86400, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Id: "0390724d-cb49-43f8-8ccd-117fef3f5034", Enabled: true, DnsName: "@", DnsType: DnsTypeNS, DnsValue: "f1g1ns2.dnspod.net.", Ttl: 86400, PolicyType: DnsPolicyTypeSimple},
			},
		},
		{
			Name:        "Test update",
			CommonCount: 14,
			AddCount:    0,
			DelCount:    0,
			UpdateCount: 1,
			Remote: []DnsRecordSet{
				DnsRecordSet{ExternalId: "647776715", Enabled: true, DnsName: "@", Ttl: 86400, DnsType: DnsTypeNS, DnsValue: "f1g1ns1.dnspod.net.", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "647776716", Enabled: true, DnsName: "@", Ttl: 86400, DnsType: DnsTypeNS, DnsValue: "f1g1ns2.dnspod.net.", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "647850198", Enabled: true, DnsName: "abc", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.12.12.12", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "647850256", Enabled: true, DnsName: "abc", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.21.21.21", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651667846", Enabled: false, DnsName: "ert", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.23.23.23", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651667854", Enabled: false, DnsName: "ert2", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.23.23.23", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651690475", Enabled: false, DnsName: "ert7", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.23.23.23", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651667834", Enabled: true, DnsName: "example3.com", Ttl: 600, DnsType: DnsTypeA, DnsValue: "12.12.12.12", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651694910", Enabled: true, DnsName: "example3.com", Ttl: 600, DnsType: DnsTypeA, DnsValue: "12.12.21.122", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651694952", Enabled: true, DnsName: "sd.sd", Ttl: 600, DnsType: DnsTypeA, DnsValue: "13.34.34.34", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651694923", Enabled: true, DnsName: "stest", Ttl: 600, DnsType: DnsTypeA, DnsValue: "234.90.8.8", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651694918", Enabled: true, DnsName: "teset34", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.56.56.56", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651694931", Enabled: true, DnsName: "teset66", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.56.56.56", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651694942", Enabled: true, DnsName: "teset67", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.56.56.56", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{ExternalId: "651694960", Enabled: true, DnsName: "test", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.34.34.34", PolicyType: DnsPolicyTypeSimple},
			},
			Local: []DnsRecordSet{
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.23.23.23", Enabled: false, Id: "1a38903e-dac8-4f75-877e-05f88f515a1f", DnsName: "ert7", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.23.23.23", Enabled: false, Id: "6e7d2c83-770c-4ecd-8c6b-4caf87fe8c23", DnsName: "ert2", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.23.23.23", Enabled: false, Id: "7fd3796a-fada-4af5-88c4-afbf36c697cd", DnsName: "ert", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeNS, DnsValue: "f1g1ns2.dnspod.net.", Enabled: true, Id: "3de80b12-851d-4418-8ffc-bd6ef90fb1f0", DnsName: "@", Ttl: 86400, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeNS, DnsValue: "f1g1ns1.dnspod.net.", Enabled: true, Id: "fc7dde12-1c96-479f-82dd-e97200b8737f", DnsName: "@", Ttl: 86400, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.56.56.56", Enabled: true, Id: "6d6ede14-01e8-49e4-8316-13b77d481b6c", DnsName: "teset34", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "13.34.34.34", Enabled: true, Id: "0e001e25-4567-4d76-8e45-ddb0012dcedf", DnsName: "sd.sd", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.56.56.56", Enabled: true, Id: "6479e99f-8031-43c7-855a-8abc1f82028c", DnsName: "teset67", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.56.56.56", Enabled: false, Id: "3b0e373f-ba22-4137-8900-d93fb2e55f12", DnsName: "teset66", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.34.34.34", Enabled: true, Id: "c9b607e5-e5ac-485f-8189-966f74914203", DnsName: "test", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.21.21.21", Enabled: true, Id: "a08cd8a6-fc7c-4de7-89b1-f962f2d9d5e5", DnsName: "abc", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "12.12.21.122", Enabled: true, Id: "251224a5-c6a1-447c-87f3-b64be34f4dd6", DnsName: "example3.com", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "12.12.12.12", Enabled: true, Id: "0ad65043-a4ca-4866-8032-56a51b018b46", DnsName: "example3.com", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "234.90.8.8", Enabled: true, Id: "f0ee44c9-12d4-40a3-84c4-ce9df25d0831", DnsName: "stest", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.12.12.12", Enabled: true, Id: "f82d1a9b-f3db-4617-8e59-4c264cecc1b6", DnsName: "abc", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
			},
		},
		{
			Name:        "Test update",
			CommonCount: 13,
			AddCount:    2,
			DelCount:    0,
			UpdateCount: 3,
			Remote: []DnsRecordSet{
				DnsRecordSet{Enabled: true, ExternalId: "647776715", DnsName: "@", Ttl: 86400, DnsType: DnsTypeNS, DnsValue: "f1g1ns1.dnspod.net.", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "647776716", DnsName: "@", Ttl: 86400, DnsType: DnsTypeNS, DnsValue: "f1g1ns2.dnspod.net.", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "647850198", DnsName: "abc", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.12.12.12", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "647850256", DnsName: "abc", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.21.21.21", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: false, ExternalId: "651667846", DnsName: "ert", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.23.23.23", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: false, ExternalId: "651667854", DnsName: "ert2", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.23.23.23", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: false, ExternalId: "651690475", DnsName: "ert7", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.23.23.23", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "652048718", DnsName: "ert8", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.23.23.23", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "652048736", DnsName: "example3.com", Ttl: 600, DnsType: DnsTypeA, DnsValue: "12.12.12.12", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "652048745", DnsName: "example3.com", Ttl: 600, DnsType: DnsTypeA, DnsValue: "12.12.21.122", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "652048753", DnsName: "sd.sd", Ttl: 600, DnsType: DnsTypeA, DnsValue: "13.34.34.34", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "652048760", DnsName: "stest", Ttl: 600, DnsType: DnsTypeA, DnsValue: "234.90.8.8", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "652048770", DnsName: "teset34", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.56.56.56", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "652048778", DnsName: "teset66", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.56.56.56", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "652048785", DnsName: "teset67", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.56.56.56", PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{Enabled: true, ExternalId: "652048789", DnsName: "test", Ttl: 600, DnsType: DnsTypeA, DnsValue: "123.34.34.34", PolicyType: DnsPolicyTypeSimple},
			},
			Local: []DnsRecordSet{
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.23.23.23", Enabled: false, Id: "9402de3c-43e4-499d-8257-3b546abff684", DnsName: "ert10", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.23.23.23", Enabled: false, Id: "89a3d7f9-7d68-4f2f-834f-2caf787d7ff1", DnsName: "ert9", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.23.23.23", Enabled: false, Id: "5edb132b-7c67-4cc7-8b85-f32f6e22d88a", DnsName: "ert8", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.23.23.23", Enabled: false, Id: "1a38903e-dac8-4f75-877e-05f88f515a1f", DnsName: "ert7", Status: "init", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.23.23.23", Enabled: false, Id: "6e7d2c83-770c-4ecd-8c6b-4caf87fe8c23", DnsName: "ert2", Status: "init", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.23.23.23", Enabled: false, Id: "7fd3796a-fada-4af5-88c4-afbf36c697cd", DnsName: "ert", Status: "init", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeNS, DnsValue: "f1g1ns2.dnspod.net.", Enabled: true, Id: "3de80b12-851d-4418-8ffc-bd6ef90fb1f0", DnsName: "@", Status: "available", Ttl: 86400, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeNS, DnsValue: "f1g1ns1.dnspod.net.", Enabled: true, Id: "fc7dde12-1c96-479f-82dd-e97200b8737f", DnsName: "@", Status: "available", Ttl: 86400, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.56.56.56", Enabled: false, Id: "6d6ede14-01e8-49e4-8316-13b77d481b6c", DnsName: "teset34", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "13.34.34.34", Enabled: true, Id: "0e001e25-4567-4d76-8e45-ddb0012dcedf", DnsName: "sd.sd", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.56.56.56", Enabled: true, Id: "6479e99f-8031-43c7-855a-8abc1f82028c", DnsName: "teset67", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.56.56.56", Enabled: false, Id: "3b0e373f-ba22-4137-8900-d93fb2e55f12", DnsName: "teset66", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.34.34.34", Enabled: true, Id: "c9b607e5-e5ac-485f-8189-966f74914203", DnsName: "test", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.21.21.21", Enabled: true, Id: "a08cd8a6-fc7c-4de7-89b1-f962f2d9d5e5", DnsName: "abc", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "12.12.21.122", Enabled: true, Id: "251224a5-c6a1-447c-87f3-b64be34f4dd6", DnsName: "example3.com", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "12.12.12.12", Enabled: true, Id: "0ad65043-a4ca-4866-8032-56a51b018b46", DnsName: "example3.com", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "234.90.8.8", Enabled: true, Id: "f0ee44c9-12d4-40a3-84c4-ce9df25d0831", DnsName: "stest", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
				DnsRecordSet{DnsType: DnsTypeA, DnsValue: "123.12.12.12", Enabled: true, Id: "f82d1a9b-f3db-4617-8e59-4c264cecc1b6", DnsName: "abc", Status: "available", Ttl: 600, PolicyType: DnsPolicyTypeSimple},
			},
		},
	}
	for _, c := range cases {
		iRecords := []ICloudDnsRecordSet{}
		for i := range c.Remote {
			iRecords = append(iRecords, &c.Remote[i])
		}
		common, added, removed, updated := CompareDnsRecordSet(iRecords, c.Local, true)
		if len(common) != c.CommonCount {
			t.Fatalf("[%s] common should be %d current is %d", c.Name, c.CommonCount, len(common))
		}
		if len(added) != c.AddCount {
			t.Fatalf("[%s] added should be %d current is %d", c.Name, c.AddCount, len(added))
		}
		if len(removed) != c.DelCount {
			t.Fatalf("[%s] removed should be %d current is %d", c.Name, c.DelCount, len(removed))
		}
		if len(updated) != c.UpdateCount {
			t.Fatalf("[%s] updated should be %d current is %d", c.Name, c.UpdateCount, len(updated))
		}
		t.Logf("%s update:", c.Name)
		for i, update := range updated {
			t.Logf("%d  %s", i, jsonutils.Marshal(update))
		}
	}
}
