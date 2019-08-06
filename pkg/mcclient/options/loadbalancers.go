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

type LoadbalancerCreateOptions struct {
	NAME             string
	Network          string
	Address          string
	AddressType      string `choices:"intranet|internet"`
	LoadbalancerSpec string `choices:"slb.s1.small|slb.s2.small|slb.s2.medium|slb.s3.small|slb.s3.medium|slb.s3.large"`
	ChargeType       string `choices:"traffic|bandwidth"`
	Bandwidth        int
	Zone             string
	Cluster          string
	Manager          string
}

type LoadbalancerGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	BackendGroup string
}

type LoadbalancerDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerPurgeOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListOptions struct {
	BaseListOptions
	Address      string
	AddressType  string `choices:"intranet|internet"`
	NetworkType  string `choices:"classic|vpc"`
	Network      string
	BackendGroup string
	Cloudregion  string
	Zone         string
	Cluster      string
}

type LoadbalancerActionStatusOptions struct {
	ID     string `json:"-"`
	Status string `choices:"enabled|disabled"`
}

type LoadbalancerActionSyncStatusOptions struct {
	ID string `json:"-"`
}

type LoadbalancerIdOptions struct {
	ID string `json:"-"`
}
