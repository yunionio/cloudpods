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

package redfish

import (
	"context"

	"yunion.io/x/pkg/errors"
)

func MountVirtualCdrom(ctx context.Context, api IRedfishDriver, cdromUrl string, boot bool) error {
	path, cdInfo, err := api.GetVirtualCdromInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "api.GetVirtualCdromInfo")
	}
	if cdInfo.Image == cdromUrl {
		return nil
	}
	if !cdInfo.SupportAction {
		return errors.Error("action not supported")
	}
	if cdInfo.Image != "" {
		err = api.UmountVirtualCdrom(ctx, path)
		if err != nil {
			return errors.Wrap(err, "api.UmountVirtualCdrom")
		}
	}
	return api.MountVirtualCdrom(ctx, path, cdromUrl, boot)
}

func UmountVirtualCdrom(ctx context.Context, api IRedfishDriver) error {
	path, cdInfo, err := api.GetVirtualCdromInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "api.GetVirtualCdromInfo")
	}
	if cdInfo.Image == "" {
		return nil
	}
	if !cdInfo.SupportAction {
		return errors.Error("action not supported")
	}
	return api.UmountVirtualCdrom(ctx, path)
}
