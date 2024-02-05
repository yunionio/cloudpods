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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

/*
 * Migrate a server from one network to another network, without change IP address
 * Scenerio: the server used to be in a VPC, migrate it to a underlay network without network interruption
 */
func (guest *SGuest) PerformMigrateNetwork(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ServerMigrateNetworkInput,
) (jsonutils.JSONObject, error) {
	if guest.Hypervisor != api.HYPERVISOR_KVM {
		return nil, errors.Wrap(httperrors.ErrNotSupported, "operation not supported for this hypervisor")
	}

	// first validate it against the source network, ensure the following:
	// 1. the network is attach to this guest
	srcModel, err := NetworkManager.FetchByIdOrName(ctx, userCred, input.Src)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "source network %s not found", input.Src)
		} else {
			return nil, errors.Wrap(err, "NetworkManager.FetchByIdOrName")
		}
	}
	srcNics, err := guest.GetNetworks(srcModel.GetId())
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrBadRequest, "server not attach to network %s", input.Src)
		} else {
			return nil, errors.Wrap(err, "GetNetworks")
		}
	}
	if len(srcNics) == 0 {
		return nil, errors.Wrapf(httperrors.ErrBadRequest, "server not attach to network %s", input.Src)
	} else if len(srcNics) > 1 {
		return nil, errors.Wrapf(httperrors.ErrNotSupported, "not support to migrate multiple interfaces")
	}
	srcNic := srcNics[0]
	ipAddr, err := netutils.NewIPV4Addr(srcNic.IpAddr)
	if err != nil {
		return nil, errors.Wrapf(err, "NewIPV4Addr")
	}
	// next validate against the destination network, ensure the following:
	// 1. the network is reachable to this server
	// 1. the IP address is availalble in the new network
	destModel, err := NetworkManager.FetchByIdOrName(ctx, userCred, input.Dest)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "destination network %s not found", input.Src)
		} else {
			return nil, errors.Wrap(err, "NetworkManager.FetchByIdOrName")
		}
	}
	destNet := destModel.(*SNetwork)

	host, _ := guest.GetHost()
	if host == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "guest is not allocated!")
	}
	if destNet.isOneCloudVpcNetwork() {
		// vpc network should be in the same Zone
		destZone, _ := destNet.GetZone()
		if destZone == nil || destZone.Id != host.ZoneId {
			return nil, errors.Wrap(httperrors.ErrBadRequest, "destination overlay network not in same zone as server")
		}
	} else {
		// underlay network should be reachable in wire
		var destWire *SWire
		wires := host.getAttachedWires()
		for i := range wires {
			if wires[i].Id == destNet.WireId {
				// reachable
				destWire = &wires[i]
				break
			}
		}
		if destWire == nil {
			return nil, errors.Wrap(httperrors.ErrBadRequest, "destination underlay network not reachable")
		}
	}

	lockman.LockObject(ctx, destNet)
	defer lockman.ReleaseObject(ctx, destNet)

	if !destNet.IsAddressInRange(ipAddr) {
		return nil, errors.Wrapf(httperrors.ErrBadRequest, "ip %s not in range of destination network", ipAddr.String())
	}
	used, err := destNet.isAddressUsed(ctx, ipAddr.String())
	if err != nil {
		return nil, errors.Wrap(err, "isAddressUsed")
	}
	if used {
		return nil, errors.Wrapf(httperrors.ErrBadRequest, "ip %s has been allocated in destination network", ipAddr.String())
	}

	// perform the database change
	_, err = db.Update(&srcNic, func() error {
		srcNic.NetworkId = destNet.Id
		srcNic.MappedIpAddr = "" // reset MappedIpAddr anyway
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "fail to update nic network_id")
	}

	// synchronize the change to host, and wait it to be effective
	err = guest.StartSyncTask(ctx, userCred, false, "")
	if err != nil {
		return nil, errors.Wrap(err, "fail to SyncTask")
	}

	return nil, nil
}
