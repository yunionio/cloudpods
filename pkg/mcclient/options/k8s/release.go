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

package k8s

import (
	"io/ioutil"

	"yunion.io/x/jsonutils"
)

type ReleaseListOptions struct {
	NamespaceResourceListOptions
	Filter     string `help:"Filter, split by space"`
	Admin      bool   `help:"Admin to show all namespace releases"`
	Deployed   bool   `help:"Show deployed status releases"`
	Deleted    bool   `help:"Show deleted status releases"`
	Deleting   bool   `help:"Show deleting status releases"`
	Failed     bool   `help:"Show failed status releases"`
	Superseded bool   `help:"Show superseded status releases"`
	Pending    bool   `help:"Show pending status releases"`
}

func (o ReleaseListOptions) Params() *jsonutils.JSONDict {
	params := o.NamespaceResourceListOptions.Params()
	if o.Filter != "" {
		params.Add(jsonutils.NewString(o.Filter), "filter")
	}
	if o.Namespace != "" {
		params.Add(jsonutils.NewString(o.Namespace), "namespace")
	}
	if o.Name != "" {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	params.Add(jsonutils.JSONTrue, "all")
	if o.Admin {
		params.Add(jsonutils.JSONTrue, "admin")
	}
	if o.Deployed {
		params.Add(jsonutils.JSONTrue, "deployed")
	}
	if o.Deleted {
		params.Add(jsonutils.JSONTrue, "deleted")
	}
	if o.Deleting {
		params.Add(jsonutils.JSONTrue, "deleting")
	}
	if o.Failed {
		params.Add(jsonutils.JSONTrue, "failed")
	}
	if o.Superseded {
		params.Add(jsonutils.JSONTrue, "superseded")
	}
	if o.Pending {
		params.Add(jsonutils.JSONTrue, "pending")
	}
	return params
}

type ReleaseCreateUpdateOptions struct {
	Values  string `help:"Specify values in a YAML file (can specify multiple)" short-token:"f"`
	Version string `help:"Specify the exact chart version to install. If not specified, latest version installed"`
	//Set     []string `help:"set STRING values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)"`
	DryRun  bool  `help:"Simulate an install"`
	Details bool  `help:"Show release deploy details, include kubernetes created resources"`
	Timeout int64 `help:"Time in seconds to wait for any individual kubernetes operation (like Jobs for hooks)" default:"600"`
}

func (o ReleaseCreateUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	if o.Version != "" {
		params.Add(jsonutils.NewString(o.Version), "version")
	}
	if o.DryRun {
		params.Add(jsonutils.JSONTrue, "dry_run")
	}
	params.Add(jsonutils.NewInt(o.Timeout), "timeout")
	if o.Values != "" {
		//vals, err := helm.MergeValuesF(args.Values, args.Set, []string{})
		vals, err := ioutil.ReadFile(o.Values)
		if err != nil {
			return nil, err
		}
		params.Add(jsonutils.NewString(string(vals)), "values")
	}
	return params, nil
}

type ReleaseCreateOptions struct {
	AppBaseCreateOptions
	CHARTNAME string `help:"Helm chart name, e.g stable/etcd"`
}

func (o ReleaseCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.AppBaseCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(o.CHARTNAME), "chart_name")
	return params, nil
}

type ReleaseUpgradeOptions struct {
	NamespaceWithClusterOptions
	ReleaseCreateUpdateOptions
	NAME        string `help:"Release instance name"`
	CHARTNAME   string `help:"Helm chart name, e.g stable/etcd"`
	ReuseValues bool   `help:"When upgrading, reuse the last release's values, and merge in any new values. If '--reset-values' is specified, this is ignored"`
	ResetValues bool   `help:"When upgrading, reset the values to the ones built into the chart"`
}

func (o ReleaseUpgradeOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.ReleaseCreateUpdateOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Update(o.NamespaceWithClusterOptions.Params())
	params.Add(jsonutils.NewString(o.CHARTNAME), "chart_name")
	params.Add(jsonutils.NewString(o.NAME), "release_name")
	if o.ReuseValues {
		params.Add(jsonutils.JSONTrue, "reuse_values")
	}
	if o.ResetValues {
		params.Add(jsonutils.JSONTrue, "reset_values")
	}
	return params, nil
}

type ReleaseDeleteOptions struct {
	NamespaceWithClusterOptions
	NAME string `help:"Release instance name"`
}

type ReleaseHistoryOptions struct {
	NamespaceWithClusterOptions
	NAME string `help:"Release instance name"`
	Max  int64  `help:"History limit size"`
}

func (o ReleaseHistoryOptions) Params() *jsonutils.JSONDict {
	params := o.NamespaceWithClusterOptions.Params()
	if o.Max >= 1 {
		params.Add(jsonutils.NewInt(o.Max), "max")
	}
	return params
}

type ReleaseRollbackOptions struct {
	NamespaceWithClusterOptions
	NAME        string `help:"Release instance name"`
	REVISION    int64  `help:"Release history revision number"`
	Description string `help:"Release rollback description string"`
}

func (o ReleaseRollbackOptions) Params() *jsonutils.JSONDict {
	params := o.NamespaceWithClusterOptions.Params()
	params.Add(jsonutils.NewInt(o.REVISION), "revision")
	if o.Description != "" {
		params.Add(jsonutils.NewString(o.Description), "description")
	}
	return params
}
