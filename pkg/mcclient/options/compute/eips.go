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

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ElasticipListOptions struct {
	Region string `help:"List eips in cloudregion"`

	Usable                    *bool  `help:"List all zones that is usable"`
	UsableEipForAssociateType string `help:"With associate id filter which eip can associate" choices:"server|natgateway|loadbalancer"`
	UsableEipForAssociateId   string `help:"With associate type filter which eip can associate"`
	OrderByIp                 string
	AssociateId               []string
	AssociateType             []string
	AssociateName             []string
	IsAssociated              *bool

	IpAddr []string
	options.BaseListOptions
}

func (opts *ElasticipListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type EipCreateOptions struct {
	options.BaseCreateOptions
	Manager    *string `help:"cloud provider"`
	Region     *string `help:"cloud region in which EIP is allocated"`
	Bandwidth  *int    `help:"Bandwidth in Mbps"`
	IpAddr     *string `help:"IP address of the EIP" json:"ip_addr"`
	Network    *string `help:"Network of the EIP"`
	BgpType    *string `help:"BgpType of the EIP" positional:"false" choices:"BGP|BGP_PRO"`
	ChargeType *string `help:"bandwidth charge type" choices:"traffic|bandwidth"`
}

func (opts *EipCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type EipUpdateOptions struct {
	options.BaseUpdateOptions

	AutoDellocate *string `help:"enable or disable automatically dellocate when dissociate from instance" choices:"true|false"`
	IpAddr        string
	AssociateId   string
	AssociateType string
}

func (opts *EipUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type EipAssociateOptions struct {
	options.BaseIdOptions
	INSTANCE_ID  string `help:"ID of instance the eip associated with"`
	InstanceType string `default:"server" help:"Instance type that the eip associated with, default is server" choices:"server|natgateway|loadbalancer"`
}

func (opts *EipAssociateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"instance_id": opts.INSTANCE_ID, "instance_type": opts.InstanceType}), nil
}

type EipDissociateOptions struct {
	options.BaseIdOptions
	AutoDelete bool `help:"automatically delete the dissociate EIP" json:"auto_delete,omitfalse"`
}

func (opts *EipDissociateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]bool{"auto_delete": opts.AutoDelete}), nil
}

type EipChangeBandwidthOptions struct {
	options.BaseIdOptions
	BANDWIDTH int `help:"new bandwidth of EIP"`
}

func (opts *EipChangeBandwidthOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]int{"bandwidth": opts.BANDWIDTH}), nil
}

type EipChangeOwnerOptions struct {
	options.BaseIdOptions
	PROJECT string `help:"Project ID or change"`
	//RawId   bool   `help:"User raw ID, instead of name"`
}

func (opts *EipChangeOwnerOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"tenant": opts.PROJECT}), nil
	/*
		params := jsonutils.NewDict()
		if opts.RawId {
			projid, err := modules.Projects.GetId(s, opts.PROJECT, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(projid), "tenant")
			params.Add(jsonutils.JSONTrue, "raw_id")
		} else {
			params.Add(jsonutils.NewString(opts.PROJECT), "tenant")
		}
	*/
}
