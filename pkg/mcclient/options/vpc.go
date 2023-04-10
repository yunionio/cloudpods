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

package options

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type VpcListOptions struct {
	BaseListOptions

	Usable                     *bool  `help:"Filter usable vpcs"`
	Region                     string `help:"ID or Name of region" json:"-"`
	Globalvpc                  string `help:"Filter by globalvpc"`
	DnsZoneId                  string `help:"Filter by DnsZone"`
	InterVpcNetworkId          string `help:"Filter by InterVpcNetwork"`
	ExternalAccessMode         string `help:"Filter by external access mode" choices:"distgw|eip|eip-distgw"`
	ZoneId                     string `help:"Filter by zone which has networks"`
	UsableForInterVpcNetworkId string `help:"Filter usable vpcs for inter vpc network"`
	OrderByWireCount           string
}

func (opts *VpcListOptions) GetContextId() string {
	return opts.Region
}

func (opts *VpcListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type VpcCreateOptions struct {
	REGION             string `help:"ID or name of the region where the VPC is created" json:"cloudregion_id"`
	Id                 string `help:"ID of the new VPC"`
	NAME               string `help:"Name of the VPC" json:"name"`
	CIDR               string `help:"CIDR block"`
	Default            bool   `help:"default VPC for the region" default:"false"`
	Desc               string `help:"Description of the VPC"`
	Manager            string `help:"ID or Name of Cloud provider" json:"manager_id"`
	ExternalAccessMode string `help:"Filter by external access mode" choices:"distgw|eip|eip-distgw" default:""`
	GlobalvpcId        string `help:"Global vpc id, Only for Google Cloud"`
}

func (opts *VpcCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(opts.REGION), "cloudregion_id")
	params.Add(jsonutils.NewString(opts.NAME), "name")
	params.Add(jsonutils.NewString(opts.CIDR), "cidr_block")
	if len(opts.Id) > 0 {
		params.Add(jsonutils.NewString(opts.Id), "id")
	}
	if len(opts.ExternalAccessMode) > 0 {
		params.Add(jsonutils.NewString(opts.ExternalAccessMode), "external_access_mode")
	}
	if len(opts.Desc) > 0 {
		params.Add(jsonutils.NewString(opts.Desc), "description")
	}
	if opts.Default {
		params.Add(jsonutils.JSONTrue, "is_default")
	}
	if len(opts.Manager) > 0 {
		params.Add(jsonutils.NewString(opts.Manager), "manager_id")
	}
	if len(opts.GlobalvpcId) > 0 {
		params.Add(jsonutils.NewString(opts.GlobalvpcId), "globalvpc_id")
	}
	return params, nil
}

type VpcIdOptions struct {
	ID string `help:"ID or name of the vpc"`
}

func (opts *VpcIdOptions) GetId() string {
	return opts.ID
}

func (opts *VpcIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type VpcUpdateOptions struct {
	BaseUpdateOptions
	ExternalAccessMode string `help:"Filter by external access mode" choices:"distgw|eip|eip-distgw"`
	Direct             bool   `help:"Can it be connected directly"`
}

func (opts *VpcUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("id")
	return params, nil
}

type VpcStatusOptions struct {
	VpcIdOptions
	STATUS string `help:"Set Vpc status" choices:"available|pending"`
}

func (opts *VpcStatusOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"status": opts.STATUS}), nil
}

type VpcChangeOwnerOptions struct {
	VpcIdOptions
	ProjectDomain string `json:"project_domain" help:"target domain"`
}

func (opts *VpcChangeOwnerOptions) Params() (jsonutils.JSONObject, error) {
	if len(opts.ProjectDomain) == 0 {
		return nil, fmt.Errorf("empty project_domain")
	}
	return jsonutils.Marshal(map[string]string{"project_domain": opts.ProjectDomain}), nil

}
