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

package compute

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type GeneralUsageOptions struct {
	HostType []string `help:"Host types" choices:"hypervisor|baremetal|esxi|xen|kubelet|hyperv|aliyun|azure|aws|huawei|qcloud|openstack|ucloud|zstack|google|ctyun"`
	Provider []string `help:"Provider" choices:"OneCloud|VMware|Aliyun|Azure|Aws|Qcloud|Huawei|OpenStack|Ucloud|VolcEngine|ZStack|Google|Ctyun"`
	Brand    []string `help:"Brands" choices:"OneCloud|VMware|Aliyun|Azure|Aws|Qcloud|Huawei|OpenStack|Ucloud|VolcEngine|ZStack|Google|Ctyun"`
	Project  string   `help:"show usage of specified project"`

	ProjectDomain string `help:"show usage of specified domain"`

	CloudEnv string `help:"show usage of specified cloudenv" choices:"public|private|onpremise"`
	Scope    string `help:"show usage of specified privilege scope" choices:"system|domain|project"`

	Refresh bool `help:"force refresh usage statistics"`
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
	if args.Refresh {
		params.Add(jsonutils.JSONTrue, "refresh")
	}
	return params
}

type HistoryUsageListOptions struct {
	options.ExtraListOptions
	StartDate time.Time
	EndDate   time.Time
	Interval  string `choices:"hour|day|month|year" default:"day"`
}

func (opts *HistoryUsageListOptions) Params() (jsonutils.JSONObject, error) {
	if opts.StartDate.IsZero() || opts.EndDate.IsZero() {
		opts.EndDate = time.Now()
		switch opts.Interval {
		case "hour":
			opts.StartDate = time.Now().Add(time.Hour * -24)
		case "day":
			opts.StartDate = time.Now().AddDate(0, 0, -30)
		case "month":
			opts.StartDate = time.Now().AddDate(0, -12, 0)
		case "year":
			opts.StartDate = time.Now().AddDate(-3, 0, 0)
		}
	}
	return jsonutils.Marshal(opts), nil
}

func (o *HistoryUsageListOptions) GetExportKeys() string {
	return ""
}

func (o *HistoryUsageListOptions) GetId() string {
	return ""
}

func init() {

	cmd := shell.NewResourceCmd(modules.HistoryUsages)
	cmd.GetWithCustomShow("list", func(data jsonutils.JSONObject) {
		ret := map[string][]jsonutils.JSONObject{}
		data.Unmarshal(&ret)
		for k, d := range ret {
			fmt.Println(k)
			printutils.PrintJSONList(&printutils.ListResult{Data: d}, nil)
		}
	}, &HistoryUsageListOptions{})

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
		result, err := image.Images.GetUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type IdentityUsageOptions struct {
	}
	R(&IdentityUsageOptions{}, "identity-usage", "Show general usage of identity", func(s *mcclient.ClientSession, args *IdentityUsageOptions) error {
		result, err := identity.IdentityUsages.GetUsage(s, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type K8sUsageOptions struct{}
	R(&K8sUsageOptions{}, "k8s-usage", "Show general usage of k8s", func(s *mcclient.ClientSession, args *K8sUsageOptions) error {
		result, err := k8s.Usages.GetUsage(s, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
