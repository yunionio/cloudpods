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
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ComputeQuotaKeys struct {
	RegionQuotaKeys

	Hypervisor string `help:"hypervisor" choices:"kvm|baremetal"`
}

type ComputeQuotaOptions struct {
	ComputeQuotaKeys

	Count          int64 `help:"server count" json:"count,omitzero"`
	Cpu            int64 `help:"CPU count" json:"cpu,omitzero"`
	Memory         int64 `help:"Memory size in MB" json:"memory,omitzero"`
	Storage        int64 `help:"Storage size in MB" json:"storage,omitzero"`
	IsolatedDevice int64 `help:"Isolated device count" json:"isolated_device,omitzero"`
}

type RegionQuotaKeys struct {
	Provider  string `help:"cloud provider" json:"provider,omitempty"`
	Brand     string `help:"cloud brand" json:"brand,omitempty"`
	CloudEnv  string `help:"cloud environment" json:"cloud_env,omitempty" choices:"onpremise|private|public"`
	AccountId string `help:"cloud account id" json:"account_id,omitempty"`
	ManagerId string `help:"cloud provider id" json:"manager_id,omitempty"`
	RegionId  string `help:"region id" json:"region_id,omitempty"`
}

type RegionQuotaOptions struct {
	RegionQuotaKeys

	Port  int64 `help:"Internal NIC count" json:"port,omitzero"`
	Eport int64 `help:"External NIC count" json:"eport,omitzero"`
	Bw    int64 `help:"Internal bandwidth in Mbps" json:"bw,omitzero"`
	Ebw   int64 `help:"External bandwidth in Mbps" json:"ebw,omitzero"`

	Eip int64 `help:"Elastic IP count" json:"eip,omitzero"`

	Snapshot int64 `help:"Snapshot count" json:"snapshot,omitzero"`

	Bucket    int64 `help:"bucket count" json:"bucket,omitzero"`
	ObjectGB  int64 `help:"object size in GB" json:"object_gb,omitzero"`
	ObjectCnt int64 `help:"object count" json:"object_cnt,omitzero"`

	Rds   int64 `help:"Rds count" json:"rds,omitzero"`
	Cache int64 `help:"redis count" json:"cache,omitzero"`
}

type ZoneQuotaKeys struct {
	RegionQuotaKeys

	ZoneId string `help:"zone id" json:"zone_id,omitempty"`
}

type ZoneQuotaOptions struct {
	ZoneQuotaKeys

	Loadbalancer int64 `help:"loadbalancer instance count" json:"loadbalancer,omitzero"`
}

type ProjectQuotaOptions struct {
	Secgroup int64 `help:"Secgroup count" json:"secgroup,omitzero"`
}

type ImageQuotaKeys struct {
	Type string `help:"image type, either iso or image" choices:"iso|image" json:"type,omitempty"`
}

type ImageQuotaOptions struct {
	ImageQuotaKeys

	Image int64 `help:"Template count" json:"image,omitzero"`
}

type QuotaSetBaseOptions struct {
	Project string `help:"Tenant name or ID to set quota" json:"tenant,omitempty"`
	Domain  string `help:"Domain name or ID to set quota" json:"domain,omitempty"`
	Action  string `help:"quota set action" choices:"add|sub|reset|replace|delete|update"`
}

func printQuotaList(result jsonutils.JSONObject) {
	printList(modulebase.JSON2ListResult(result), nil)
}

func init() {
	type QuotaOptions struct {
		Scope   string `help:"scope" choices:"domain|project"`
		Project string `help:"Tenant name or ID" json:"tenant"`
		Domain  string `help:"Domain name or ID" json:"domain"`
		Refresh bool   `help:"refresh" json:"refresh,omitfalse"`
	}
	R(&QuotaOptions{}, "quota", "Show quota for current user or tenant", func(s *mcclient.ClientSession, args *QuotaOptions) error {
		params := jsonutils.Marshal(args)
		log.Debugf("%s", params)
		quotas, err := modules.Quotas.GetQuota(s, params)
		if err != nil {
			return err
		}
		printQuotaList(quotas)
		return nil
	})
	R(&QuotaOptions{}, "project-quota", "Show project-quota for current user or tenant", func(s *mcclient.ClientSession, args *QuotaOptions) error {
		params := jsonutils.Marshal(args)
		quotas, err := modules.ProjectQuotas.GetQuota(s, params)
		if err != nil {
			return err
		}
		printQuotaList(quotas)
		return nil
	})
	R(&QuotaOptions{}, "region-quota", "Show region-quota for current user or tenant", func(s *mcclient.ClientSession, args *QuotaOptions) error {
		params := jsonutils.Marshal(args)
		quotas, err := modules.RegionQuotas.GetQuota(s, params)
		if err != nil {
			return err
		}
		printQuotaList(quotas)
		return nil
	})
	R(&QuotaOptions{}, "zone-quota", "Show zone-quota for current user or tenant", func(s *mcclient.ClientSession, args *QuotaOptions) error {
		params := jsonutils.Marshal(args)
		quotas, err := modules.ZoneQuotas.GetQuota(s, params)
		if err != nil {
			return err
		}
		printQuotaList(quotas)
		return nil
	})
	R(&QuotaOptions{}, "project-quota", "Show project-quota for current user or tenant", func(s *mcclient.ClientSession, args *QuotaOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.ProjectQuotas.GetQuota(s, params)
		if err != nil {
			return err
		}
		printQuotaList(result)
		return nil
	})
	R(&QuotaOptions{}, "image-quota", "Show image quota for current user or tenant", func(s *mcclient.ClientSession, args *QuotaOptions) error {
		params := jsonutils.Marshal(args)
		quotas, err := modules.ImageQuotas.GetQuota(s, params)
		if err != nil {
			return err
		}
		printQuotaList(quotas)
		return nil
	})

	type ComputeQuotaSetOptions struct {
		QuotaSetBaseOptions
		ComputeQuotaOptions
	}
	R(&ComputeQuotaSetOptions{}, "quota-set", "Set quota for tenant", func(s *mcclient.ClientSession, args *ComputeQuotaSetOptions) error {
		params := jsonutils.Marshal(args)
		quotas, e := modules.Quotas.DoQuotaSet(s, params)
		if e != nil {
			return e
		}
		printQuotaList(quotas)
		return nil
	})

	type RegionQuotaSetOptions struct {
		QuotaSetBaseOptions
		RegionQuotaOptions
	}
	R(&RegionQuotaSetOptions{}, "region-quota-set", "Set regional quota for tenant", func(s *mcclient.ClientSession, args *RegionQuotaSetOptions) error {
		params := jsonutils.Marshal(args)
		quotas, e := modules.RegionQuotas.DoQuotaSet(s, params)
		if e != nil {
			return e
		}
		printQuotaList(quotas)
		return nil
	})

	type ZoneQuotaSetOptions struct {
		QuotaSetBaseOptions
		ZoneQuotaOptions
	}
	R(&ZoneQuotaSetOptions{}, "zone-quota-set", "Set zonal quota for tenant", func(s *mcclient.ClientSession, args *ZoneQuotaSetOptions) error {
		params := jsonutils.Marshal(args)
		quotas, e := modules.ZoneQuotas.DoQuotaSet(s, params)
		if e != nil {
			return e
		}
		printQuotaList(quotas)
		return nil
	})

	type ProjectQuotaSetOptions struct {
		QuotaSetBaseOptions
		ProjectQuotaOptions
	}
	R(&ProjectQuotaSetOptions{}, "project-quota-set", "Set project quota for tenant", func(s *mcclient.ClientSession, args *ProjectQuotaSetOptions) error {
		params := jsonutils.Marshal(args)
		quotas, e := modules.ProjectQuotas.DoQuotaSet(s, params)
		if e != nil {
			return e
		}
		printQuotaList(quotas)
		return nil
	})

	type ImageQuotaSetOptions struct {
		QuotaSetBaseOptions
		ImageQuotaOptions
	}
	R(&ImageQuotaSetOptions{}, "image-quota-set", "Set image quota for tenant", func(s *mcclient.ClientSession, args *ImageQuotaSetOptions) error {
		params := jsonutils.Marshal(args)
		quotas, e := modules.ImageQuotas.DoQuotaSet(s, params)
		if e != nil {
			return e
		}
		printQuotaList(quotas)
		return nil
	})

	type QuotaListOptions struct {
		ProjectDomain string `help:"domain name or ID to query project quotas"`
	}
	R(&QuotaListOptions{}, "quota-list", "List quota of domains or projects of a domain", func(s *mcclient.ClientSession, args *QuotaListOptions) error {
		params := jsonutils.Marshal(args)
		result, e := modules.Quotas.GetQuotaList(s, params)
		if e != nil {
			return e
		}
		printQuotaList(result)
		return nil
	})

	R(&QuotaListOptions{}, "region-quota-list", "List region quota of domains or projects of a domain", func(s *mcclient.ClientSession, args *QuotaListOptions) error {
		params := jsonutils.Marshal(args)
		result, e := modules.RegionQuotas.GetQuotaList(s, params)
		if e != nil {
			return e
		}
		printQuotaList(result)
		return nil
	})

	R(&QuotaListOptions{}, "zone-quota-list", "List zone quota of domains or projects of a domain", func(s *mcclient.ClientSession, args *QuotaListOptions) error {
		params := jsonutils.Marshal(args)
		result, e := modules.ZoneQuotas.GetQuotaList(s, params)
		if e != nil {
			return e
		}
		printQuotaList(result)
		return nil
	})

	R(&QuotaListOptions{}, "project-quota-list", "List project quota of domains or projects of a domain", func(s *mcclient.ClientSession, args *QuotaListOptions) error {
		params := jsonutils.Marshal(args)
		result, e := modules.ProjectQuotas.GetQuotaList(s, params)
		if e != nil {
			return e
		}
		printQuotaList(result)
		return nil
	})

	R(&QuotaListOptions{}, "image-quota-list", "List image quota of domains or projects of a domain", func(s *mcclient.ClientSession, args *QuotaListOptions) error {
		params := jsonutils.Marshal(args)
		result, e := modules.ImageQuotas.GetQuotaList(s, params)
		if e != nil {
			return e
		}
		printQuotaList(result)
		return nil
	})

	type CleanPendingUsageOptions struct {
		Scope   string `help:"scope" choices:"domain|project"`
		Project string `help:"Tenant name or ID" json:"tenant"`
		Domain  string `help:"Domain name or ID" json:"domain"`
	}
	R(&CleanPendingUsageOptions{}, "clean-pending-usage", "Clean pending usage for project or domain", func(s *mcclient.ClientSession, args *CleanPendingUsageOptions) error {
		params := jsonutils.Marshal(args)
		_, err := modules.Quotas.DoCleanPendingUsage(s, params)
		if err != nil {
			return err
		}
		return nil
	})

	R(&CleanPendingUsageOptions{}, "clean-region-pending-usage", "Clean pending usage for project or domain", func(s *mcclient.ClientSession, args *CleanPendingUsageOptions) error {
		params := jsonutils.Marshal(args)
		_, err := modules.RegionQuotas.DoCleanPendingUsage(s, params)
		if err != nil {
			return err
		}
		return nil
	})

	R(&CleanPendingUsageOptions{}, "clean-zone-pending-usage", "Clean pending usage for project or domain", func(s *mcclient.ClientSession, args *CleanPendingUsageOptions) error {
		params := jsonutils.Marshal(args)
		_, err := modules.ZoneQuotas.DoCleanPendingUsage(s, params)
		if err != nil {
			return err
		}
		return nil
	})

	R(&CleanPendingUsageOptions{}, "clean-project-pending-usage", "Clean pending usage for project or domain", func(s *mcclient.ClientSession, args *CleanPendingUsageOptions) error {
		params := jsonutils.Marshal(args)
		_, err := modules.ProjectQuotas.DoCleanPendingUsage(s, params)
		if err != nil {
			return err
		}
		return nil
	})

	R(&CleanPendingUsageOptions{}, "clean-image-pending-usage", "Clean pending usage for project or domain", func(s *mcclient.ClientSession, args *CleanPendingUsageOptions) error {
		params := jsonutils.Marshal(args)
		_, err := modules.ImageQuotas.DoCleanPendingUsage(s, params)
		if err != nil {
			return err
		}
		return nil
	})

}
