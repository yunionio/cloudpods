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
// PageNumberations under the License.

package shell

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DnsProductListOptions struct {
		PageNumber int `help:"page size"`
		PageSize   int `help:"page PageSize"`
	}
	shellutils.R(&DnsProductListOptions{}, "dnsproduct-list", "List dnsproduct", func(cli *aliyun.SRegion, args *DnsProductListOptions) error {
		//products, e := cli.GetClient().DescribeDnsProductInstances(args.PageNumber, args.PageSize)
		products, e := cli.GetClient().GetAllDnsProductInstances()
		if e != nil {
			return e
		}
		//printList(products.DNSProducts.DNSProduct, products.TotalCount, args.PageSize, args.PageNumber, []string{})
		printList(products, len(products), args.PageNumber, args.PageSize, []string{})
		return nil
	})

	type DomianListOptions struct {
		PageNumber int `help:"page size"`
		PageSize   int `help:"page PageSize"`
	}
	shellutils.R(&DomianListOptions{}, "domain-list", "List Domain", func(cli *aliyun.SRegion, args *DomianListOptions) error {
		sdomains, e := cli.GetClient().DescribeDomains(args.PageNumber, args.PageSize)
		if e != nil {
			return e
		}
		printList(sdomains.Domains.Domain, sdomains.TotalCount, args.PageNumber, args.PageSize, []string{})
		return nil
	})

	type SDomainCreateOptions struct {
		DOMAINNAME string `help:"Domain name"`
	}
	shellutils.R(&SDomainCreateOptions{}, "domain-create", "Create domain", func(cli *aliyun.SRegion, args *SDomainCreateOptions) error {
		hostzones, err := cli.GetClient().AddDomain(args.DOMAINNAME)
		if err != nil {
			return err
		}
		printObject(hostzones)
		return nil
	})

	type SDomainDeleteOptions struct {
		DOMAINNAME string
	}
	shellutils.R(&SDomainDeleteOptions{}, "domain-delete", "delete domain", func(cli *aliyun.SRegion, args *SDomainDeleteOptions) error {
		err := cli.GetClient().DeleteDomain(args.DOMAINNAME)
		if err != nil {
			return err
		}
		return nil
	})

	type SDomainShowOptions struct {
		DOMAINNAME string
	}
	shellutils.R(&SDomainShowOptions{}, "domain-show", "Show domain", func(cli *aliyun.SRegion, args *SDomainShowOptions) error {
		szone, e := cli.GetClient().DescribeDomainInfo(args.DOMAINNAME)
		if e != nil {
			return e
		}
		printObject(szone)
		return nil
	})

	type SDomainRecordListOptions struct {
		DOMAINNAME string
		PageNumber int `help:"page size"`
		PageSize   int `help:"page PageSize"`
	}
	shellutils.R(&SDomainRecordListOptions{}, "domainrecord-list", "List domainrecord", func(cli *aliyun.SRegion, args *SDomainRecordListOptions) error {
		srecords, e := cli.GetClient().DescribeDomainRecords(args.DOMAINNAME, args.PageNumber, args.PageSize)
		if e != nil {
			return e
		}
		printList(srecords.DomainRecords.Record, srecords.TotalCount, args.PageNumber, args.PageSize, []string{})
		return nil
	})

	type DomainRecordCreateOptions struct {
		DOMAINNAME  string
		NAME        string
		VALUE       string `help:"dns record value"`
		TTL         int64  `help:"ttl"`
		TYPE        string `help:"dns type"`
		PolicyType  string `help:"PolicyType"`
		PolicyValue string
	}
	shellutils.R(&DomainRecordCreateOptions{}, "domainrecord-create", "create domainrecord", func(cli *aliyun.SRegion, args *DomainRecordCreateOptions) error {
		opts := cloudprovider.DnsRecordSet{}
		opts.DnsName = args.NAME
		opts.DnsType = cloudprovider.TDnsType(args.TYPE)
		opts.DnsValue = args.VALUE
		opts.Ttl = args.TTL
		opts.PolicyType = cloudprovider.TDnsPolicyType(args.PolicyType)

		_, err := cli.GetClient().AddDomainRecord(args.DOMAINNAME, opts)
		if err != nil {
			return err
		}
		return nil
	})

	type DomainRecordupdateOptions struct {
		DOMAINRECORDID string `help:"DOMAINRECORDID"`
		NAME           string `help:"Domain name"`
		VALUE          string `help:"dns record value"`
		TTL            int64  `help:"ttl"`
		TYPE           string `help:"dns type"`
		PolicyType     string `help:"PolicyType"`
	}
	shellutils.R(&DomainRecordupdateOptions{}, "domainrecord-update", "update domainrecord", func(cli *aliyun.SRegion, args *DomainRecordupdateOptions) error {
		opts := cloudprovider.DnsRecordSet{}
		opts.DnsName = args.NAME
		opts.DnsType = cloudprovider.TDnsType(args.TYPE)
		opts.DnsValue = args.VALUE
		opts.Ttl = args.TTL
		opts.ExternalId = args.DOMAINRECORDID
		opts.PolicyType = cloudprovider.TDnsPolicyType(args.PolicyType)
		err := cli.GetClient().UpdateDomainRecord(opts)
		if err != nil {
			return err
		}
		return nil
	})

	type DomainRecordDeleteOptions struct {
		DOMAINRECORDID string `help:"DOMAINRECORDID"`
	}
	shellutils.R(&DomainRecordDeleteOptions{}, "domainrecord-delete", "delete domainrecord", func(cli *aliyun.SRegion, args *DomainRecordDeleteOptions) error {
		err := cli.GetClient().DeleteDomainRecord(args.DOMAINRECORDID)
		if err != nil {
			return err
		}
		return nil
	})

	type DomainRecordSetStatusOptions struct {
		DOMAINRECORDID string `help:"PRIVATEZONEID"`
		STATUS         string `choices:"Enable|Disable"`
	}
	shellutils.R(&DomainRecordSetStatusOptions{}, "domainrecord-setstatus", "set domainrecord status", func(cli *aliyun.SRegion, args *DomainRecordSetStatusOptions) error {

		err := cli.GetClient().SetDomainRecordStatus(args.DOMAINRECORDID, args.STATUS)
		if err != nil {
			return err
		}
		return nil
	})
}
