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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type HostedZoneListOptions struct{}
	shellutils.R(&HostedZoneListOptions{}, "hostedzone-list", "List hostedzone", func(cli *aws.SRegion, args *HostedZoneListOptions) error {
		hostzones, err := cli.GetClient().GetHostedZones()
		if err != nil {
			return err
		}
		printList(hostzones, len(hostzones), 0, 20, []string{})
		return nil
	})

	type HostedZoneCreateOptions struct {
		NAME   string `help:"Domain name"`
		Type   string `choices:"PublicZone|PrivateZone"`
		Vpc    string `help:"vpc id"`
		Region string `help:"region id"`
	}
	shellutils.R(&HostedZoneCreateOptions{}, "hostedzone-create", "Create hostedzone", func(cli *aws.SRegion, args *HostedZoneCreateOptions) error {
		opts := cloudprovider.SDnsZoneCreateOptions{}
		opts.Name = args.NAME
		opts.ZoneType = cloudprovider.TDnsZoneType(args.Type)
		if len(args.Vpc) > 0 && len(args.Region) > 0 {
			vpc := cloudprovider.SPrivateZoneVpc{}
			vpc.Id = args.Vpc
			vpc.RegionId = args.Region
			opts.Vpcs = []cloudprovider.SPrivateZoneVpc{vpc}
		}
		hostzones, err := cli.GetClient().CreateHostedZone(&opts)
		if err != nil {
			return err
		}
		printObject(hostzones)
		return nil
	})

	type HostedZoneAddVpcOptions struct {
		HOSTEDZONEID string
		VPC          string
		REGION       string
	}
	shellutils.R(&HostedZoneAddVpcOptions{}, "hostedzone-add-vpc", "associate vpc with hostedzone", func(cli *aws.SRegion, args *HostedZoneAddVpcOptions) error {

		err := cli.GetClient().AssociateVPCWithHostedZone(args.VPC, args.REGION, args.HOSTEDZONEID)
		if err != nil {
			return err
		}
		return nil
	})

	type HostedZoneRemoveVpcOptions struct {
		HOSTEDZONEID string
		VPC          string
		REGION       string
	}
	shellutils.R(&HostedZoneRemoveVpcOptions{}, "hostedzone-rmvpc", "disassociate vpc with hostedzone", func(cli *aws.SRegion, args *HostedZoneRemoveVpcOptions) error {

		err := cli.GetClient().DisassociateVPCFromHostedZone(args.VPC, args.REGION, args.HOSTEDZONEID)
		if err != nil {
			return err
		}
		return nil
	})

	type HostedZoneDeleteOptions struct {
		HOSTEDZONEID string
	}
	shellutils.R(&HostedZoneDeleteOptions{}, "hostedzone-delete", "delete hostedzone", func(cli *aws.SRegion, args *HostedZoneDeleteOptions) error {
		err := cli.GetClient().DeleteHostedZone(args.HOSTEDZONEID)
		if err != nil {
			return err
		}
		return nil
	})

	type DnsRecordSetListOptions struct {
		HOSTEDZONEID string
	}
	shellutils.R(&DnsRecordSetListOptions{}, "dnsrecordset-list", "List dnsrecordset", func(cli *aws.SRegion, args *DnsRecordSetListOptions) error {
		dnsrecordsets, err := cli.GetClient().GetSdnsRecordSets(args.HOSTEDZONEID)
		if err != nil {
			return err
		}
		printList(dnsrecordsets, len(dnsrecordsets), 0, 20, []string{})
		return nil
	})

	type DnsRecordSetCreateOptions struct {
		HOSTEDZONEID string `help:"HostedzoneId"`
		NAME         string `help:"Domain name"`
		VALUE        string `help:"dns record value"`
		TTL          int64  `help:"ttl"`
		TYPE         string `help:"dns type"`
		PolicyType   string `help:"PolicyType"`
		Identify     string `help:"Identify"`
	}
	shellutils.R(&DnsRecordSetCreateOptions{}, "dnsrecordset-create", "create dnsrecordset", func(cli *aws.SRegion, args *DnsRecordSetCreateOptions) error {
		opts := cloudprovider.DnsRecordSet{}
		opts.DnsName = args.NAME
		opts.DnsType = cloudprovider.TDnsType(args.TYPE)
		opts.DnsValue = args.VALUE
		opts.Ttl = args.TTL
		opts.ExternalId = args.Identify
		err := cli.GetClient().AddDnsRecordSet(args.HOSTEDZONEID, &opts)
		if err != nil {
			return err
		}
		return nil
	})

	type DnsRecordSetupdateOptions struct {
		HOSTEDZONEID string `help:"HostedzoneId"`
		NAME         string `help:"Domain name"`
		VALUE        string `help:"dns record value"`
		TTL          int64  `help:"ttl"`
		TYPE         string `help:"dns type"`
		Identify     string `help:"Identify"`
	}
	shellutils.R(&DnsRecordSetupdateOptions{}, "dnsrecordset-update", "update dnsrecordset", func(cli *aws.SRegion, args *DnsRecordSetupdateOptions) error {
		opts := cloudprovider.DnsRecordSet{}
		opts.DnsName = args.NAME
		opts.DnsType = cloudprovider.TDnsType(args.TYPE)
		opts.DnsValue = args.VALUE
		opts.Ttl = args.TTL
		opts.ExternalId = args.Identify
		err := cli.GetClient().UpdateDnsRecordSet(args.HOSTEDZONEID, &opts)
		if err != nil {
			return err
		}
		return nil
	})

	type DnsRecordSetDeleteOptions struct {
		HOSTEDZONEID string `help:"HostedzoneId"`
		NAME         string `help:"Domain name"`
		VALUE        string `help:"dns record value"`
		TTL          int64  `help:"ttl"`
		TYPE         string `help:"dns type"`
		Identify     string `help:"Identify"`
	}
	shellutils.R(&DnsRecordSetDeleteOptions{}, "dnsrecordset-delete", "delete dnsrecordset", func(cli *aws.SRegion, args *DnsRecordSetDeleteOptions) error {
		opts := cloudprovider.DnsRecordSet{}
		opts.DnsName = args.NAME
		opts.DnsType = cloudprovider.TDnsType(args.TYPE)
		opts.DnsValue = args.VALUE
		opts.Ttl = args.TTL
		opts.ExternalId = args.Identify
		err := cli.GetClient().RemoveDnsRecordSet(args.HOSTEDZONEID, &opts)
		if err != nil {
			return err
		}
		return nil
	})
}
