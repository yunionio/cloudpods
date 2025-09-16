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

package google

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SGlobalNetwork struct {
	GoogleTags
	SResourceBase

	client *SGoogleClient

	CreationTimestamp     time.Time
	Description           string
	AutoCreateSubnetworks bool
	Subnetworks           []string
	RoutingConfig         map[string]string
	Kind                  string
}

func (self *SGlobalNetwork) GetStatus() string {
	return api.GLOBAL_VPC_STATUS_AVAILABLE
}

func (self *SGlobalNetwork) IsEmulated() bool {
	return false
}

func (self *SGlobalNetwork) Delete() error {
	return self.client.ecsDelete(self.SelfLink, nil)
}

func (self *SGlobalNetwork) GetCreatedAt() time.Time {
	return self.CreationTimestamp
}

func (self *SGlobalNetwork) Refresh() error {
	gvpc, err := self.client.GetGlobalNetwork(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, gvpc)
}

func (self *SGlobalNetwork) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	firewalls, err := self.client.GetFirewalls(self.SelfLink, 0, "")
	if err != nil {
		return nil, err
	}
	groups := map[string]*SSecurityGroup{}
	for i := range firewalls {
		if len(firewalls[i].TargetTags) == 0 {
			continue
		}
		_, ok := groups[firewalls[i].TargetTags[0]]
		if !ok {
			groups[firewalls[i].TargetTags[0]] = &SSecurityGroup{Rules: []SFirewall{}}
		}
		groups[firewalls[i].TargetTags[0]].gvpc = self
		groups[firewalls[i].TargetTags[0]].Tag = firewalls[i].TargetTags[0]
		groups[firewalls[i].TargetTags[0]].Rules = append(groups[firewalls[i].TargetTags[0]].Rules, firewalls[i])
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for _, group := range groups {
		ret = append(ret, group)
	}
	return ret, nil
}

func (cli *SGoogleClient) GetGlobalNetwork(id string) (*SGlobalNetwork, error) {
	net := &SGlobalNetwork{client: cli}
	return net, cli.ecsGet("global/networks", id, net)
}

func (self *SGoogleClient) GetICloudGlobalVpcById(id string) (cloudprovider.ICloudGlobalVpc, error) {
	return self.GetGlobalNetwork(id)
}

func (cli *SGoogleClient) GetGlobalNetworks(maxResults int, pageToken string) ([]SGlobalNetwork, error) {
	networks := []SGlobalNetwork{}
	params := map[string]string{}
	resource := "global/networks"
	if maxResults == 0 && len(pageToken) == 0 {
		err := cli.ecsListAll(resource, params, &networks)
		if err != nil {
			return nil, errors.Wrap(err, "ecsListAll")
		}
		return networks, nil
	}
	resp, err := cli.ecsList(resource, params)
	if err != nil {
		return nil, errors.Wrap(err, "ecsList")
	}
	if resp.Contains("items") {
		err = resp.Unmarshal(&networks, "items")
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
	}
	return networks, nil
}

func (self *SGoogleClient) CreateGlobalNetwork(name string, desc string) (*SGlobalNetwork, error) {
	body := map[string]interface{}{
		"name":                  name,
		"description":           desc,
		"autoCreateSubnetworks": false,
		"mtu":                   1460,
		"routingConfig": map[string]string{
			"routingMode": "REGIONAL",
		},
	}
	globalnetwork := &SGlobalNetwork{client: self}
	err := self.Insert("global/networks", jsonutils.Marshal(body), globalnetwork)
	if err != nil {
		return nil, errors.Wrap(err, "self.Insert")
	}
	return globalnetwork, nil
}

func (self *SGoogleClient) CreateICloudGlobalVpc(opts *cloudprovider.GlobalVpcCreateOptions) (cloudprovider.ICloudGlobalVpc, error) {
	gvpc, err := self.CreateGlobalNetwork(opts.NAME, opts.Desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateICloudGlobalVpc")
	}
	return gvpc, nil
}

func (self *SGoogleClient) GetICloudGlobalVpcs() ([]cloudprovider.ICloudGlobalVpc, error) {
	gvpcs, err := self.GetGlobalNetworks(0, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetGlobalNetworks")
	}
	ret := []cloudprovider.ICloudGlobalVpc{}
	for i := range gvpcs {
		gvpcs[i].client = self
		ret = append(ret, &gvpcs[i])
	}
	return ret, nil
}
