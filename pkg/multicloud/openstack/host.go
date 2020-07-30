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

package openstack

import (
	"time"

	"github.com/pkg/errors"
)

type SHost struct {
	HostName string
	Zone     string
	Service  string
}

type SAggregate struct {
	AvailabilityZone string
	CreatedAt        time.Time
	Deleted          bool
	Hosts            []string
	Id               string
	Metadata         map[string]string
	Name             string
	Uuid             string
}

func (region *SRegion) GetAggregates() ([]SAggregate, error) {
	resp, err := region.ecsList("/os-aggregates", nil)
	if err != nil {
		return nil, errors.Wrap(err, "ecsList")
	}
	aggregates := []SAggregate{}
	err = resp.Unmarshal(&aggregates, "aggregates")
	if err != nil {
		return nil, errors.Wrap(err, `resp.Unmarshal(&aggregates, "aggregates")`)
	}
	return aggregates, nil
}

func (region *SRegion) GetHosts() ([]SHost, error) {
	hosts := []SHost{}
	resp, err := region.ecsList("/os-hosts", nil)
	if err != nil {
		return nil, errors.Wrap(err, "ecsList(os-hosts)")
	}
	err = resp.Unmarshal(&hosts, "hosts")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return hosts, nil
}
