package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

type QuotaBaseOptions struct {
	Cpu            int64 `help:"CPU count"`
	Memory         int64 `help:"Memory size in MB"`
	Storage        int64 `help:"Storage size in MB"`
	Port           int64 `help:"Internal NIC count"`
	Eport          int64 `help:"External NIC count"`
	Bw             int64 `help:"Internal bandwidth in Mbps"`
	Ebw            int64 `help:"External bandwidth in Mbps"`
	Image          int64 `help:"Template count"`
	IsolatedDevice int64 `help:"Isolated device count"`
}

func quotaArgs2Params(args *QuotaBaseOptions) *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if args.Cpu > 0 {
		params.Add(jsonutils.NewInt(args.Cpu), "cpu")
	}
	if args.Memory > 0 {
		params.Add(jsonutils.NewInt(args.Memory), "memory")
	}
	if args.Storage > 0 {
		params.Add(jsonutils.NewInt(args.Storage), "storage")
	}
	if args.Image > 0 {
		params.Add(jsonutils.NewInt(args.Image), "image")
	}
	if args.Port > 0 {
		params.Add(jsonutils.NewInt(args.Port), "port")
	}
	if args.Eport > 0 {
		params.Add(jsonutils.NewInt(args.Eport), "eport")
	}
	if args.Bw > 0 {
		params.Add(jsonutils.NewInt(args.Bw), "bw")
	}
	if args.Ebw > 0 {
		params.Add(jsonutils.NewInt(args.Ebw), "ebw")
	}
	if args.IsolatedDevice > 0 {
		params.Add(jsonutils.NewInt(args.IsolatedDevice), "isolated_device")
	}
	return params
}

func init() {
	type QuotaOptions struct {
		Tenant string `help:"Tenant name of ID"`
		User   string `help:"User name of ID"`
	}
	R(&QuotaOptions{}, "quota", "Show quota for current user or tenant", func(s *mcclient.ClientSession, args *QuotaOptions) error {
		params := jsonutils.NewDict()
		if len(args.Tenant) > 0 {
			params.Add(jsonutils.NewString(args.Tenant), "tenant")
		}
		if len(args.User) > 0 {
			params.Add(jsonutils.NewString(args.User), "user")
		}
		result, err := modules.Quotas.GetQuota(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type QuotaSetOptions struct {
		Tenant string `help:"Tenant name or ID to set quota"`
		User   string `help:"User name of ID"`
		QuotaBaseOptions
	}
	R(&QuotaSetOptions{}, "quota-set", "Set quota for tenant", func(s *mcclient.ClientSession, args *QuotaSetOptions) error {
		params := quotaArgs2Params(&args.QuotaBaseOptions)
		if len(args.Tenant) > 0 {
			params.Add(jsonutils.NewString(args.Tenant), "tenant")
		}
		if len(args.User) > 0 {
			params.Add(jsonutils.NewString(args.User), "user")
		}
		result, e := modules.Quotas.DoQuotaSet(s, params)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	type QuotaCheckOptions struct {
		TENANT string `help:"Tenant name or ID to check quota"`
		QuotaBaseOptions
	}
	R(&QuotaCheckOptions{}, "quota-check", "Check quota for tenant", func(s *mcclient.ClientSession, args *QuotaCheckOptions) error {
		params := quotaArgs2Params(&args.QuotaBaseOptions)
		params.Add(jsonutils.NewString(args.TENANT), "tenant")
		result, e := modules.Quotas.DoQuotaCheck(s, params)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

}
