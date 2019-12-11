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
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type GeneralUsageOptions struct {
	HostType []string `help:"Host types" choices:"hypervisor|baremetal|esxi|xen|kubelet|hyperv|aliyun|azure|aws|huawei|qcloud|openstack|ucloud|zstack|ctyun"`
	Provider []string `help:"Provider" choices:"OneCloud|VMware|Aliyun|Azure|Aws|Qcloud|Huawei|OpenStack|Ucloud|ZStack"`
	Brand    []string `help:"Brands" choices:"OneCloud|VMware|Aliyun|Azure|Aws|Qcloud|Huawei|OpenStack|Ucloud|ZStack|DStack"`
	Project  string   `help:"show usage of specified project"`

	ProjectDomain string `help:"show usage of specified domain"`

	CloudEnv string `help:"show usage of specified cloudenv" choices:"public|private|onpremise"`
	Scope    string `help:"show usage of specified privilege scope" choices:"system|domain|project"`
}

func fetchHostTypeOptions(args *GeneralUsageOptions) *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if len(args.HostType) > 0 {
		params.Add(jsonutils.NewStringArray(args.HostType), "host_type")
	}
	if len(args.Provider) > 0 {
		params.Add(jsonutils.NewStringArray(args.Provider), "provider")
	}
	if len(args.Brand) > 0 {
		params.Add(jsonutils.NewStringArray(args.Brand), "brand")
	}
	if len(args.CloudEnv) > 0 {
		params.Add(jsonutils.NewString(args.CloudEnv), "cloud_env")
	}
	return params
}

func init() {
	R(&GeneralUsageOptions{}, "usage", "Show general usage", func(s *mcclient.ClientSession, args *GeneralUsageOptions) error {
		params := fetchHostTypeOptions(args)
		if args.Project != "" {
			params.Add(jsonutils.NewString(args.Project), "project")
		} else if args.ProjectDomain != "" {
			params.Add(jsonutils.NewString(args.ProjectDomain), "project_domain")
		}
		if len(args.Scope) > 0 {
			params.Add(jsonutils.NewString(args.Scope), "scope")
		}
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ResourceUsageOptions struct {
		GeneralUsageOptions
		ID string `help:"ID or name of resource"`
	}
	R(&ResourceUsageOptions{}, "zone-usage", "Show general usage of zone", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("zones"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "wire-usage", "Show general usage of wire", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("wires"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "cloud-provider-usage", "Show general usage of vcenter", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("cloudproviders"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "cloud-account-usage", "Show general usage of vcenter", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("cloudaccounts"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "schedtag-usage", "Show general usage of a scheduler tag", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("schedtags"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "cloud-region-usage", "Show general usage of a cloud region", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("cloudregions"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ImageUsageOptions struct {
		Project string `help:"check image usage of a project"`
		Domain  string `help:"check image usage of a domain"`
		Scope   string `help:"query scope" choices:"project|domain|system"`
	}
	R(&ImageUsageOptions{}, "image-usage", "Show general usage of images", func(s *mcclient.ClientSession, args *ImageUsageOptions) error {
		params := jsonutils.NewDict()
		if args.Project != "" {
			params.Add(jsonutils.NewString(args.Project), "project")
		} else if args.Domain != "" {
			params.Add(jsonutils.NewString(args.Domain), "domain")
		}
		if args.Scope != "" {
			params.Add(jsonutils.NewString(args.Scope), "scope")
		}
		result, err := modules.Images.GetUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type IdentityUsageOptions struct {
	}
	R(&IdentityUsageOptions{}, "identity-usage", "Show general usage of identity", func(s *mcclient.ClientSession, args *IdentityUsageOptions) error {
		result, err := modules.IdentityUsages.GetUsage(s, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
