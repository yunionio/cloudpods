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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type InstanceGroupListOptions struct {
	options.BaseListOptions

	ServiceType       string `help:"Service Type"`
	ParentId          string `help:"Parent ID"`
	ZoneId            string `help:"Zone ID"`
	Server            string `help:"Guest ID or Name"`
	OrderByVips       string
	OrderByGuestCount string
}

func (opts *InstanceGroupListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type InstanceGroupCreateOptions struct {
	NAME string `help:"name of instance group"`

	ZoneId          string `help:"zone id" json:"zone_id"`
	ServiceType     string `help:"service type"`
	ParentId        string `help:"parent id"`
	SchedStrategy   string `help:"scheduler strategy"`
	Granularity     string `help:"the upper limit number of guests with this group in a host"`
	ForceDispersion bool   `help:"force to make guest dispersion"`
}

func (opts *InstanceGroupCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, errors.Wrap(err, "StructToParams")
	}
	return params, nil
}

type InstanceGroupUpdateOptions struct {
	options.BaseIdOptions

	Name            string `help:"New name to change"`
	Granularity     string `help:"the upper limit number of guests with this group in a host"`
	ForceDispersion string `help:"force to make guest dispersion" choices:"yes|no" json:"-"`
}

func (opts *InstanceGroupUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, errors.Wrap(err, "StructToParams")
	}
	if opts.ForceDispersion == "yes" {
		params.Set("force_dispersion", jsonutils.JSONTrue)
	} else {
		params.Set("force_dispersion", jsonutils.JSONFalse)
	}
	return params, nil
}

type InstanceGroupBindGuestsOptions struct {
	options.BaseIdOptions
	Guest []string `help:"ID or Name of Guest"`
}

func (opts *InstanceGroupBindGuestsOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type InstanceGroupAttachnetworkOptions struct {
	options.BaseIdOptions

	api.GroupAttachNetworkInput
}

func (opts *InstanceGroupAttachnetworkOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type InstanceGroupDetachnetworkOptions struct {
	options.BaseIdOptions

	api.GroupDetachNetworkInput
}

func (opts *InstanceGroupDetachnetworkOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type InstanceGroupCreateEipOptions struct {
	options.BaseIdOptions

	api.ServerCreateEipInput
}

func (opts *InstanceGroupCreateEipOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type InstanceGroupAssociateEipOptions struct {
	options.BaseIdOptions

	api.ServerAssociateEipInput
}

func (opts *InstanceGroupAssociateEipOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type InstanceGroupDissociateEipOptions struct {
	options.BaseIdOptions

	api.ServerDissociateEipInput
}

func (opts *InstanceGroupDissociateEipOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
