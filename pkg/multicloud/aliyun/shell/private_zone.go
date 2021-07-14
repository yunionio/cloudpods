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
// PageSizeations under the License.

package shell

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type PrivateZoneListOptions struct {
		PageSize   int `help:"page size"`
		PageNumber int `help:"page PageNumber"`
	}
	shellutils.R(&PrivateZoneListOptions{}, "privatezone-list", "List privatezone", func(cli *aliyun.SRegion, args *PrivateZoneListOptions) error {
		szones, e := cli.GetClient().DescribeZones(args.PageNumber, args.PageSize)
		if e != nil {
			return e
		}
		printList(szones.Zones.Zone, szones.TotalItems, args.PageNumber, args.PageSize, []string{})
		return nil
	})

	type PrivateZoneCreateOptions struct {
		NAME   string `help:"Domain name"`
		Type   string `choices:"PublicZone|PrivateZone"`
		Vpc    string `help:"vpc id"`
		Region string `help:"region id"`
	}
	shellutils.R(&PrivateZoneCreateOptions{}, "privatezone-create", "Create privatezone", func(cli *aliyun.SRegion, args *PrivateZoneCreateOptions) error {
		opts := cloudprovider.SDnsZoneCreateOptions{}
		opts.Name = args.NAME
		opts.ZoneType = cloudprovider.TDnsZoneType(args.Type)
		if len(args.Vpc) > 0 && len(args.Region) > 0 {
			vpc := cloudprovider.SPrivateZoneVpc{}
			vpc.Id = args.Vpc
			vpc.RegionId = args.Region
			opts.Vpcs = []cloudprovider.SPrivateZoneVpc{vpc}
		}
		hostzones, err := cli.GetClient().CreateZone(&opts)
		if err != nil {
			return err
		}
		printObject(hostzones)
		return nil
	})

	type PrivateZoneDeleteOptions struct {
		PRIVATEZONEID string
	}
	shellutils.R(&PrivateZoneDeleteOptions{}, "privatezone-delete", "delete privatezone", func(cli *aliyun.SRegion, args *PrivateZoneDeleteOptions) error {
		err := cli.GetClient().DeleteZone(args.PRIVATEZONEID)
		if err != nil {
			return err
		}
		return nil
	})

	type PrivateZoneShowOptions struct {
		ID string `help:"ID or name of privatezone"`
	}
	shellutils.R(&PrivateZoneShowOptions{}, "privatezone-show", "Show privatezone", func(cli *aliyun.SRegion, args *PrivateZoneShowOptions) error {
		szone, e := cli.GetClient().DescribeZoneInfo(args.ID)
		if e != nil {
			return e
		}
		printObject(szone)
		return nil
	})

	type PrivateZoneAddVpcOptions struct {
		PRIVATEZONEID string
		VPC           string
		REGION        string
	}
	shellutils.R(&PrivateZoneAddVpcOptions{}, "privatezone-add-vpc", "associate vpc with privatezone", func(cli *aliyun.SRegion, args *PrivateZoneAddVpcOptions) error {
		vpc := cloudprovider.SPrivateZoneVpc{}
		vpc.Id = args.VPC
		vpc.RegionId = args.REGION
		err := cli.GetClient().BindZoneVpc(args.PRIVATEZONEID, &vpc)
		if err != nil {
			return err
		}
		return nil
	})

	type PrivateZoneDeleteVpcsOptions struct {
		PRIVATEZONEID string
	}
	shellutils.R(&PrivateZoneDeleteVpcsOptions{}, "privatezone-delete-vpc", "delete vpc with privatezone", func(cli *aliyun.SRegion, args *PrivateZoneDeleteVpcsOptions) error {
		err := cli.GetClient().UnBindZoneVpcs(args.PRIVATEZONEID)
		if err != nil {
			return err
		}
		return nil
	})

	type PrivateZoneRecordListOptions struct {
		ID         string `help:"ID or name of privatezone"`
		PageSize   int    `help:"page size"`
		PageNumber int    `help:"page PageNumber"`
	}
	shellutils.R(&PrivateZoneRecordListOptions{}, "privatezonerecord-list", "List privatezonerecord", func(cli *aliyun.SRegion, args *PrivateZoneRecordListOptions) error {
		srecords, e := cli.GetClient().DescribeZoneRecords(args.ID, args.PageNumber, args.PageSize)
		if e != nil {
			return e
		}
		printList(srecords.Records.Record, srecords.TotalItems, args.PageNumber, args.PageSize, []string{})
		return nil
	})

	type PvtzRecordCreateOptions struct {
		PRIVATEZONEID string `help:"PRIVATEZONEID"`
		NAME          string `help:"Domain name"`
		VALUE         string `help:"dns record value"`
		TTL           int64  `help:"ttl"`
		TYPE          string `help:"dns type"`
		PolicyType    string `help:"PolicyType"`
		Identify      string `help:"Identify"`
	}
	shellutils.R(&PvtzRecordCreateOptions{}, "privatezonerecord-create", "create privatezonerecord", func(cli *aliyun.SRegion, args *PvtzRecordCreateOptions) error {
		opts := cloudprovider.DnsRecordSet{}
		opts.DnsName = args.NAME
		opts.DnsType = cloudprovider.TDnsType(args.TYPE)
		opts.DnsValue = args.VALUE
		opts.Ttl = args.TTL
		opts.ExternalId = args.Identify
		_, err := cli.GetClient().AddZoneRecord(args.PRIVATEZONEID, opts)
		if err != nil {
			return err
		}
		return nil
	})

	type PvtzRecordupdateOptions struct {
		PRIVATEZONEID string `help:"PRIVATEZONEID"`
		NAME          string `help:"Domain name"`
		VALUE         string `help:"dns record value"`
		TTL           int64  `help:"ttl"`
		TYPE          string `help:"dns type"`
		Identify      string `help:"Identify"`
	}
	shellutils.R(&PvtzRecordupdateOptions{}, "privatezonerecord-update", "update privatezonerecord", func(cli *aliyun.SRegion, args *PvtzRecordupdateOptions) error {
		opts := cloudprovider.DnsRecordSet{}
		opts.DnsName = args.NAME
		opts.DnsType = cloudprovider.TDnsType(args.TYPE)
		opts.DnsValue = args.VALUE
		opts.Ttl = args.TTL
		opts.ExternalId = args.Identify
		err := cli.GetClient().UpdateZoneRecord(opts)
		if err != nil {
			return err
		}
		return nil
	})

	type PvtzRecordDeleteOptions struct {
		PRIVATEZONERECORDID string `help:"PRIVATEZONEID"`
	}
	shellutils.R(&PvtzRecordDeleteOptions{}, "privatezonerecord-delete", "delete privatezonerecord", func(cli *aliyun.SRegion, args *PvtzRecordDeleteOptions) error {
		err := cli.GetClient().DeleteZoneRecord(args.PRIVATEZONERECORDID)
		if err != nil {
			return err
		}
		return nil
	})

	type PvtzRecordSetStatusOptions struct {
		PRIVATEZONERECORDID string `help:"PRIVATEZONEID"`
		STATUS              string `choices:"ENABLE|DISABLE"`
	}
	shellutils.R(&PvtzRecordSetStatusOptions{}, "privatezonerecord-setstatus", "set privatezonerecord status", func(cli *aliyun.SRegion, args *PvtzRecordSetStatusOptions) error {

		err := cli.GetClient().SetZoneRecordStatus(args.PRIVATEZONERECORDID, args.STATUS)
		if err != nil {
			return err
		}
		return nil
	})
}
