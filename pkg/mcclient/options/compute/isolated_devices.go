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

type DeviceListOptions struct {
	options.BaseListOptions
	Unused bool   `help:"Only show unused devices"`
	Gpu    bool   `help:"Only show gpu devices"`
	Host   string `help:"Host ID or Name"`
	Region string `help:"Cloudregion ID or Name"`
	Zone   string `help:"Zone ID or Name"`
	Server string `help:"Server ID or Name"`

	DevType        []string `help:"filter by dev_type"`
	Model          []string `help:"filter by model"`
	Addr           []string `help:"filter by addr"`
	DevicePath     []string `help:"filter by device path"`
	VendorDeviceId []string `help:"filter by vendor device id(PCIID)"`
	NumaNode       []uint8  `help:"fitler by numa node index"`
}

func (o *DeviceListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type DeviceShowOptions struct {
	options.BaseIdOptions
}

type DeviceDeleteOptions struct {
	options.BaseIdsOptions
}

type DeviceUpdateOptions struct {
	options.BaseIdOptions

	ReservedCpu     *int   `help:"reserved cpu for isolated device"`
	ReservedMem     *int   `help:"reserved mem for isolated device"`
	ReservedStorage *int   `help:"reserved storage for isolated device"`
	DevType         string `help:"Device type"`
}

func (o *DeviceUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
