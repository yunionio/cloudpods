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

package models

import (
	compute_models "yunion.io/x/onecloud/pkg/compute/models"
)

type Vpc struct {
	compute_models.SVpc

	Networks Networks `json:"-"`
}

func (el *Vpc) Copy() *Vpc {
	return &Vpc{
		SVpc: el.SVpc,
	}
}

type Network struct {
	compute_models.SNetwork
	// returned as extra column
	VpcId string

	Vpc           *Vpc          `json:"-"`
	Guestnetworks Guestnetworks `json:"-"`
}

func (el *Network) Copy() *Network {
	return &Network{
		SNetwork: el.SNetwork,
		VpcId:    el.VpcId,
	}
}

type Guestnetwork struct {
	compute_models.SGuestnetwork

	Network *Network `json:"-"`
}

func (el *Guestnetwork) Copy() *Guestnetwork {
	return &Guestnetwork{
		SGuestnetwork: el.SGuestnetwork,
	}
}
