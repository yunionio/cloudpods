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

package profiles

import (
	"context"

	"yunion.io/x/pkg/errors"

	baremetalapi "yunion.io/x/onecloud/pkg/apis/compute/baremetal"
	"yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	baremetalmodules "yunion.io/x/onecloud/pkg/mcclient/modules/compute/baremetal"
)

func GetProfile(ctx context.Context, sysinfo *types.SSystemInfo) (*baremetalapi.BaremetalProfileSpec, error) {
	adminSession := auth.GetAdminSession(ctx, options.Options.Region)
	specs, err := baremetalmodules.BaremetalProfiles.GetMatchProfiles(adminSession, sysinfo.OemName, sysinfo.Model)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMatchProfile %s %s", sysinfo.OemName, sysinfo.Model)
	}
	if len(specs) > 0 {
		return &specs[len(specs)-1], nil
	}
	return nil, errors.ErrNotFound
}

func GetLanChannels(ctx context.Context, sysinfo *types.SSystemInfo) ([]uint8, error) {
	profile, err := GetProfile(ctx, sysinfo)
	if err != nil {
		return nil, errors.Wrap(err, "GetProfile")
	}
	return profile.LanChannels, nil
}

func GetDefaultLanChannel(ctx context.Context, sysinfo *types.SSystemInfo) (uint8, error) {
	channels, err := GetLanChannels(ctx, sysinfo)
	if err != nil {
		return 0, errors.Wrap(err, "GetLanChannels")
	}
	if len(channels) > 0 {
		return channels[0], nil
	}
	return 0, errors.ErrNotFound
}

func GetRootId(ctx context.Context, sysinfo *types.SSystemInfo) (int, error) {
	profile, err := GetProfile(ctx, sysinfo)
	if err != nil {
		return 0, errors.Wrap(err, "GetProfile")
	}
	return profile.RootId, nil
}

func GetRootName(ctx context.Context, sysinfo *types.SSystemInfo) (string, error) {
	profile, err := GetProfile(ctx, sysinfo)
	if err != nil {
		return "", errors.Wrap(err, "GetProfile")
	}
	return profile.RootName, nil
}

func IsStrongPass(ctx context.Context, sysinfo *types.SSystemInfo) (bool, error) {
	profile, err := GetProfile(ctx, sysinfo)
	if err != nil {
		return false, errors.Wrap(err, "GetProfile")
	}
	return profile.StrongPass, nil
}
