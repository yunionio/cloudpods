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

package session

import (
	"context"
	"net/url"

	"yunion.io/x/pkg/errors"

	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	identity_api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/webconsole/options"
)

type SDisplayInfo struct {
	WaterMark    string `json:"water_mark"`
	InstanceName string `json:"instance_name"`
	Ips          string `json:"ips"`

	Hypervisor     string `json:"hypervisor"`
	OsType         string `json:"os_type"`
	OsName         string `json:"os_name"`
	OsArch         string `json:"os_arch"`
	OsDistribution string `json:"os_distribution"`
	SecretLevel    string `json:"secret_level"`
}

func (dispInfo *SDisplayInfo) fetchGuestInfo(guestDetails *compute_api.ServerDetails) {
	dispInfo.Hypervisor = guestDetails.Hypervisor
	dispInfo.OsName = guestDetails.OsName
	dispInfo.OsType = guestDetails.OsType
	dispInfo.OsArch = guestDetails.OsArch
	dispInfo.OsDistribution = guestDetails.Metadata[compute_api.VM_METADATA_OS_DISTRO]
	dispInfo.InstanceName = guestDetails.Name
	dispInfo.Ips = guestDetails.IPs
	dispInfo.SecretLevel = guestDetails.Metadata["cls:secret_level"]
}

func (dispInfo *SDisplayInfo) fetchHostInfo(hostDetails *compute_api.HostDetails) {
	dispInfo.Hypervisor = hostDetails.HostType
	dispInfo.OsArch = hostDetails.CpuArchitecture
	dispInfo.InstanceName = hostDetails.Name
	dispInfo.Ips = hostDetails.AccessIp
}

func (dispInfo *SDisplayInfo) populateParams(params url.Values) url.Values {
	if options.Options.EnableWatermark && len(dispInfo.WaterMark) > 0 {
		params["water_mark"] = []string{dispInfo.WaterMark}
	}
	if len(dispInfo.InstanceName) > 0 {
		params["instance_name"] = []string{dispInfo.InstanceName}
	}
	if len(dispInfo.Ips) > 0 {
		params["ips"] = []string{dispInfo.Ips}
	}
	if len(dispInfo.Hypervisor) > 0 {
		params["hypervisor"] = []string{dispInfo.Hypervisor}
	}
	if len(dispInfo.OsType) > 0 {
		params["os_type"] = []string{dispInfo.OsType}
	}
	if len(dispInfo.OsName) > 0 {
		params["os_name"] = []string{dispInfo.OsName}
	}
	if len(dispInfo.OsArch) > 0 {
		params["os_arch"] = []string{dispInfo.OsArch}
	}
	if len(dispInfo.OsDistribution) > 0 {
		params["os_distribution"] = []string{dispInfo.OsDistribution}
	}
	if len(dispInfo.SecretLevel) > 0 {
		params["secret_level"] = []string{dispInfo.SecretLevel}
	}

	return params
}

func fetchWaterMark(userInfo *identity_api.UserDetails) string {
	info := userInfo.Name
	if len(userInfo.Displayname) > 0 {
		info += " (" + userInfo.Displayname + ")"
	}
	info += "<br/>"
	if len(userInfo.Mobile) > 0 {
		info += userInfo.Mobile
	} else if len(userInfo.Email) > 0 {
		info += userInfo.Email
	} else {
		info += userInfo.Id
	}
	return info
}

func fetchUserInfo(ctx context.Context, s *mcclient.ClientSession) (*identity_api.UserDetails, error) {
	usrObj, err := identity.UsersV3.GetById(auth.GetAdminSession(ctx, s.GetRegion()), s.GetUserId(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetById")
	}

	usr := identity_api.UserDetails{}
	err = usrObj.Unmarshal(&usr)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}

	return &usr, nil
}

func FetchServerInfo(ctx context.Context, s *mcclient.ClientSession, sid string) (*compute_api.ServerDetails, error) {
	guestInfo, err := compute.Servers.Get(s, sid, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetById %s", sid)
	}
	guestDetails := compute_api.ServerDetails{}
	err = guestInfo.Unmarshal(&guestDetails)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal guest info")
	}
	return &guestDetails, nil
}

func FetchHostInfo(ctx context.Context, s *mcclient.ClientSession, id string) (*compute_api.HostDetails, error) {
	hostInfo, err := compute.Hosts.Get(s, id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetById %s", id)
	}
	hostDetails := compute_api.HostDetails{}
	err = hostInfo.Unmarshal(&hostDetails)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal guest info")
	}
	return &hostDetails, nil
}
