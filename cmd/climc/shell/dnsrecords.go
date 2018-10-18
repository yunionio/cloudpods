package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DNSRecordOptions struct {
	A     []string `help:"DNS A record" metavar:"A_RECORD" positional:"false"`
	AAAA  []string `help:"DNS AAAA record" metavar:"AAAA_RECORD" positional:"false"`
	CNAME string   `help:"DNS CNAME record" metavar:"CNAME_RECORD" positional:"false"`
	PTR   string   `help:"DNS PTR record" metavar:"PTR_RECORD" positional:"false"`

	SRVHost string   `help:"(deprecated) DNS SRV record, server of service" metavar:"SRV_RECORD_HOST" positional:"false"`
	SRVPort int64    `help:"(deprecated) DNS SRV record, port of service" metavar:"SRV_RECORD_PORT" positional:"false"`
	SRV     []string `help:"DNS SRV record, in the format of host:port:weight:priority" metavar:"SRV_RECORD" positional:"false"`
}

func parseDNSRecords(args *DNSRecordOptions, params *jsonutils.JSONDict) {
	if len(args.A) > 0 || len(args.AAAA) > 0 {
		for i, a := range args.A {
			params.Add(jsonutils.NewString(a), fmt.Sprintf("A.%d", i))
		}
		for i, a := range args.AAAA {
			params.Add(jsonutils.NewString(a), fmt.Sprintf("AAAA.%d", i))
		}
	} else if len(args.CNAME) > 0 {
		params.Add(jsonutils.NewString(args.CNAME), "CNAME")
	} else if len(args.SRV) > 0 || (len(args.SRVHost) > 0 && args.SRVPort > 0) {
		for i, s := range args.SRV {
			params.Set(fmt.Sprintf("SRV.%d", i), jsonutils.NewString(s))
		}
		// Keep using the original argument passing method in case a
		// newer climc is used against old service
		if len(args.SRVHost) > 0 && args.SRVPort > 0 {
			params.Set("SRV_host", jsonutils.NewString(args.SRVHost))
			params.Set("SRV_port", jsonutils.NewInt(args.SRVPort))
		}
	} else if len(args.PTR) > 0 {
		params.Add(jsonutils.NewString(args.PTR), "PTR")
	}
}

func init() {
	type DNSListOptions struct {
		options.BaseListOptions
	}
	R(&DNSListOptions{}, "dns-list", "List dns records", func(s *mcclient.ClientSession, suboptions *DNSListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = suboptions.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.DNSRecords.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DNSRecords.GetColumns(s))
		return nil
	})

	type DNSCreateOptions struct {
		NAME string `help:"DNS name to create"`
		TTL  int64  `help:"TTL in seconds" positional:"false"`
		Desc string `help:"Description" metavar:"DESCRIPTION"`
		DNSRecordOptions
	}
	R(&DNSCreateOptions{}, "dns-create", "Create dns record", func(s *mcclient.ClientSession, args *DNSCreateOptions) error {
		params := jsonutils.NewDict()
		parseDNSRecords(&args.DNSRecordOptions, params)
		if params.Size() == 0 {
			return fmt.Errorf("No records to create")
		}
		params.Add(jsonutils.NewString(args.NAME), "name")
		if args.TTL > 0 {
			params.Add(jsonutils.NewInt(args.TTL), "ttl")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		rec, e := modules.DNSRecords.Create(s, params)
		if e != nil {
			return e
		}
		printObject(rec)
		return nil
	})

	type DNSShowOptions struct {
		ID string `help:"ID of DNS record to show"`
	}
	R(&DNSShowOptions{}, "dns-show", "Show details of a dns records", func(s *mcclient.ClientSession, args *DNSShowOptions) error {
		dns, e := modules.DNSRecords.Get(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	type DNSUpdateOptions struct {
		ID   string `help:"ID of DNS record to update"`
		Name string `help:"Domain name"`
		TTL  int64  `help:"TTL in seconds" positional:"false"`
		Desc string `help:"Description"`
		DNSRecordOptions
	}
	R(&DNSUpdateOptions{}, "dns-update", "Update details of a dns records", func(s *mcclient.ClientSession, args *DNSUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.TTL > 0 {
			params.Add(jsonutils.NewInt(args.TTL), "ttl")
		}
		parseDNSRecords(&args.DNSRecordOptions, params)
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		dns, e := modules.DNSRecords.Update(s, args.ID, params)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&DNSShowOptions{}, "dns-delete", "Delete a dns record", func(s *mcclient.ClientSession, args *DNSShowOptions) error {
		dns, e := modules.DNSRecords.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&DNSShowOptions{}, "dns-public", "Make a dns record publicly available", func(s *mcclient.ClientSession, args *DNSShowOptions) error {
		dns, e := modules.DNSRecords.PerformAction(s, args.ID, "public", nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&DNSShowOptions{}, "dns-private", "Make a dns record private", func(s *mcclient.ClientSession, args *DNSShowOptions) error {
		dns, e := modules.DNSRecords.PerformAction(s, args.ID, "private", nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&DNSShowOptions{}, "dns-enable", "Enable dns record", func(s *mcclient.ClientSession, args *DNSShowOptions) error {
		dns, e := modules.DNSRecords.PerformAction(s, args.ID, "enable", nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&DNSShowOptions{}, "dns-disable", "Disable dns record", func(s *mcclient.ClientSession, args *DNSShowOptions) error {
		dns, e := modules.DNSRecords.PerformAction(s, args.ID, "disable", nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	type DNSUpdateRecordsOptions struct {
		ID string `help:"ID of dns record to modify"`
		DNSRecordOptions
	}
	R(&DNSUpdateRecordsOptions{}, "dns-add-records", "Add DNS records to a name", func(s *mcclient.ClientSession, args *DNSUpdateRecordsOptions) error {
		params := jsonutils.NewDict()
		parseDNSRecords(&args.DNSRecordOptions, params)
		if params.Size() == 0 {
			return fmt.Errorf("Nothing to add")
		}
		dns, e := modules.DNSRecords.PerformAction(s, args.ID, "add-records", params)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&DNSUpdateRecordsOptions{}, "dns-remove-records", "Remove DNS records from a name", func(s *mcclient.ClientSession, args *DNSUpdateRecordsOptions) error {
		params := jsonutils.NewDict()
		parseDNSRecords(&args.DNSRecordOptions, params)
		if params.Size() == 0 {
			return fmt.Errorf("Nothing to remove")
		}
		dns, e := modules.DNSRecords.PerformAction(s, args.ID, "remove-records", params)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

}
