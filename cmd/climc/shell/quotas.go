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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type QuotaBaseOptions struct {
	Cpu            int64 `help:"CPU count" json:"cpu,omitzero"`
	Memory         int64 `help:"Memory size in MB" json:"memory,omitzero"`
	Storage        int64 `help:"Storage size in MB" json:"storage,omitzero"`
	Port           int64 `help:"Internal NIC count" json:"port,omitzero"`
	Eport          int64 `help:"External NIC count" json:"eport,omitzero"`
	Eip            int64 `help:"Elastic IP count" json:"eip,omitzero"`
	Bw             int64 `help:"Internal bandwidth in Mbps" json:"bw,omitzero"`
	Ebw            int64 `help:"External bandwidth in Mbps" json:"ebw,omitzero"`
	IsolatedDevice int64 `help:"Isolated device count" json:"isolated_device,omitzero"`
	Snapshot       int64 `help:"Snapshot count" json:"snapshot,omitzero"`
	Image          int64 `help:"Template count" json:"image,omitzero"`
	Secgroup       int64 `help:"Secgroup count" json:"secgroup,omitzero"`
	Bucket         int64 `help:"bucket count" json:"bucket,omitzero"`
	ObjectGB       int64 `help:"object size in GB" json:"object_gb,omitzero"`
	ObjectCnt      int64 `help:"object count" json:"object_cnt,omitzero"`
}

func init() {
	type QuotaOptions struct {
		Tenant        string `help:"Tenant name or ID"`
		ProjectDomain string `help:"Domain name or ID"`
	}
	R(&QuotaOptions{}, "quota", "Show quota for current user or tenant", func(s *mcclient.ClientSession, args *QuotaOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Quotas.GetQuota(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&QuotaOptions{}, "image-quota", "Show image quota for current user or tenant", func(s *mcclient.ClientSession, args *QuotaOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.ImageQuotas.GetQuota(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type QuotaSetOptions struct {
		Tenant        string `help:"Tenant name or ID to set quota" json:"tenant,omitempty"`
		ProjectDomain string `help:"Domain name or ID to set quota" json:",omitempty"`
		Action        string `help:"quota set action" choices:"add|reset|replace"`
		Cascade       bool   `help:"cascade set quota so that auto increment domain quota if total project quota exceeds parent domain quota"`
		QuotaBaseOptions
	}
	R(&QuotaSetOptions{}, "quota-set", "Set quota for tenant", func(s *mcclient.ClientSession, args *QuotaSetOptions) error {
		params := jsonutils.Marshal(args)
		result, e := modules.Quotas.DoQuotaSet(s, params)
		if e != nil {
			return e
		}
		printObject(result)
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
		printList(modulebase.JSON2ListResult(result), nil)
		return nil
	})

	/*type QuotaCheckOptions struct {
		TENANT string `help:"Tenant name or ID to check quota" json:"tenant,omitempty"`
		QuotaBaseOptions
	}
	R(&QuotaCheckOptions{}, "quota-check", "Check quota for tenant", func(s *mcclient.ClientSession, args *QuotaCheckOptions) error {
		params := jsonutils.Marshal(args)
		result, e := modules.Quotas.DoQuotaCheck(s, params)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})*/

}
