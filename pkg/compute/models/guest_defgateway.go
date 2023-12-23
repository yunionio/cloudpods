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
	"context"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func (guest *SGuest) setDefaultGateway(ctx context.Context, userCred mcclient.TokenCredential, macAddr string) error {
	lockman.LockObject(ctx, guest)
	defer lockman.ReleaseObject(ctx, guest)

	gns, err := guest.GetNetworks("")
	if err != nil {
		return errors.Wrap(err, "GetNetworks")
	}
	var defaultGw *SGuestnetwork
	for i := range gns {
		gn := gns[i]
		if (gn.IsDefault && gn.MacAddr != macAddr) || (!gn.IsDefault && gn.MacAddr == macAddr) {
			_, err := db.Update(&gn, func() error {
				gn.IsDefault = (gn.MacAddr == macAddr)
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "set default gateway")
			}
			if gn.MacAddr == macAddr {
				defaultGw = &gn
			}
		}
	}
	if defaultGw != nil {
		notes := defaultGw.GetShortDesc(ctx)
		db.OpsLog.LogEvent(guest, db.ACT_UPDATE, notes, userCred)
		logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_UPDATE, notes, userCred, true)
	}
	return nil
}
