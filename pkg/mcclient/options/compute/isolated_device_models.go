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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type IsolatedDeviceModelListOptions struct {
	options.BaseListOptions

	DevType  string `help:"filter by dev_type"`
	Model    string `help:"filter by device model"`
	VendorId string `help:"filter by vendor id"`
	DeviceId string `help:"filter by device id"`
}

func (o *IsolatedDeviceModelListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type IsolatedDeviceModelCreateOptions struct {
	MODEL     string   `help:"device model name"`
	DEV_TYPE  string   `help:"custom device type"`
	VENDOR_ID string   `help:"pci vendor id"`
	DEVICE_ID string   `help:"pci device id"`
	Hosts     []string `help:"hosts id or name rescan isolated device"`
}

func (o *IsolatedDeviceModelCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type IsolatedDeviceModelUpdateOptions struct {
	options.BaseIdOptions

	MODEL     string `help:"device model name"`
	VENDOR_ID string `help:"pci vendor id"`
	DEVICE_ID string `help:"pci device id"`
}

func (o *IsolatedDeviceModelUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type IsolatedDeviceIdsOptions struct {
	options.BaseIdOptions
}

type IsolatedDeviceModelSetHardwareInfoOptions struct {
	options.BaseIdOptions

	api.IsolatedDeviceModelHardwareInfo
}

func (o *IsolatedDeviceModelSetHardwareInfoOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
