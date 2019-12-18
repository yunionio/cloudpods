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

	"yunion.io/x/pkg/errors"
)

type SGlobalNetwork struct {
	SResourceBase

	Id                    string
	CreationTimestamp     time.Time
	Description           string
	AutoCreateSubnetworks bool
	Subnetworks           []string
	RoutingConfig         map[string]string
	Kind                  string
}

func (cli *SGoogleClient) GetGlobalNetwork(id string) (*SGlobalNetwork, error) {
	net := &SGlobalNetwork{}
	return net, cli.ecsGet(id, net)
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
