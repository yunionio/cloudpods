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
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

type SNetworkAdditionalWireManager struct {
	db.SModelBaseManager
}

var NetworkAdditionalWireManager *SNetworkAdditionalWireManager

func init() {
	NetworkAdditionalWireManager = &SNetworkAdditionalWireManager{
		SModelBaseManager: db.NewModelBaseManager(
			SNetworkAdditionalWire{},
			"network_additional_wire_tbl",
			"network_additional_wire",
			"network_additional_wires",
		),
	}
	NetworkAdditionalWireManager.SetVirtualObject(NetworkAdditionalWireManager)
}

type SNetworkAdditionalWire struct {
	db.SModelBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" primary:"true"`
	WireId    string `width:"36" charset:"ascii" nullable:"false" primary:"true"`
	Synced    *bool
	Marked    *bool
}

func (manager *SNetworkAdditionalWireManager) newRecord(ctx context.Context, netId, wireId string, synced *bool, marked *bool) error {
	rec := &SNetworkAdditionalWire{
		NetworkId: netId,
		WireId:    wireId,
		Synced:    synced,
		Marked:    marked,
	}
	err := manager.TableSpec().InsertOrUpdate(ctx, rec)
	return errors.Wrap(err, "InsertOrUpdate")
}

func (manager *SNetworkAdditionalWireManager) Query(fields ...string) *sqlchemy.SQuery {
	q := manager.SModelBaseManager.Query(fields...)
	q = q.Filter(sqlchemy.OR(
		sqlchemy.IsTrue(q.Field("synced")),
		sqlchemy.IsTrue(q.Field("marked")),
	))
	return q
}

func (manager *SNetworkAdditionalWireManager) networkIdQuery(wireId string) *sqlchemy.SQuery {
	q := manager.Query("network_id").Equals("wire_id", wireId)
	return q
}

func (manager *SNetworkAdditionalWireManager) fetchNetworkAdditionalWireIdsQuery(netId string) *sqlchemy.SQuery {
	q := manager.Query("wire_id").Equals("network_id", netId)
	return q
}

func (manager *SNetworkAdditionalWireManager) DeleteNetwork(ctx context.Context, netId string) error {
	False := false
	wireIds, err := manager.FetchNetworkAdditionalWireIds(netId)
	if err != nil {
		return errors.Wrap(err, "FetchNetworkAdditionalWireIds")
	}
	for _, wireId := range wireIds {
		err := manager.newRecord(ctx, netId, wireId, &False, &False)
		if err != nil {
			return errors.Wrap(err, "newRecord")
		}
	}
	return nil
}

func (manager *SNetworkAdditionalWireManager) DeleteWire(ctx context.Context, wireId string) error {
	False := false
	netIds, err := manager.FetchWireAdditionalNetworkIds(wireId)
	if err != nil {
		return errors.Wrap(err, "FetchWireAdditionalNetworkIds")
	}
	for _, netId := range netIds {
		err := manager.newRecord(ctx, netId, wireId, &False, &False)
		if err != nil {
			return errors.Wrap(err, "newRecord")
		}
	}
	return nil
}

func (manager *SNetworkAdditionalWireManager) FetchNetworkAdditionalWireIds(netId string) ([]string, error) {
	q := manager.fetchNetworkAdditionalWireIdsQuery(netId)
	ret := make([]SNetworkAdditionalWire, 0)
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	wireIds := make([]string, len(ret))
	for i := range ret {
		wireIds[i] = ret[i].WireId
	}
	return wireIds, nil
}

func (manager *SNetworkAdditionalWireManager) FetchWireAdditionalNetworkIds(wireId string) ([]string, error) {
	q := manager.networkIdQuery(wireId)
	ret := make([]SNetworkAdditionalWire, 0)
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	netIds := make([]string, len(ret))
	for i := range ret {
		netIds[i] = ret[i].NetworkId
	}
	return netIds, nil
}

func (manager *SNetworkAdditionalWireManager) FetchNetworkAdditionalWires(netId string) ([]api.SSimpleWire, error) {
	subq := manager.fetchNetworkAdditionalWireIdsQuery(netId).SubQuery()
	q := WireManager.Query("id", "name")
	q = q.Join(subq, sqlchemy.Equals(q.Field("id"), subq.Field("wire_id")))

	ret := make([]SWire, 0)
	err := db.FetchModelObjects(WireManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	wires := make([]api.SSimpleWire, len(ret))
	for i := range ret {
		wires[i].WireId = ret[i].Id
		wires[i].Wire = ret[i].Name
	}
	return wires, nil
}

func (net *SNetwork) syncAdditionalWires(ctx context.Context, wireIds []string) error {
	// find out all vmware wires in the same zone
	wires, err := WireManager.FetchWires(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		managersQ := CloudproviderManager.Query().Equals("provider", api.CLOUD_PROVIDER_VMWARE).SubQuery()
		q = q.Join(managersQ, sqlchemy.Equals(q.Field("manager_id"), managersQ.Field("id")))
		return q
	})
	if err != nil {
		return errors.Wrap(err, "FetchWires")
	}
	for i := range wires {
		w := &wires[i]
		if net.WireId == w.Id {
			continue
		}
		connected := net.checkNetWireConnectivity(ctx, w)
		var markedPtr *bool
		if wireIds != nil {
			marked := false
			if utils.IsInArray(w.Id, wireIds) {
				marked = true
			}
			markedPtr = &marked
		}
		err := NetworkAdditionalWireManager.newRecord(ctx, net.Id, w.Id, &connected, markedPtr)
		if err != nil {
			return errors.Wrap(err, "NetworkAdditionalWireManager.newRecord")
		}
	}
	return nil
}

func (net *SNetwork) checkNetWireConnectivity(ctx context.Context, wire *SWire) bool {
	vmIps := wire.GetMetadata(ctx, "vm_ips", nil)
	if len(vmIps) > 0 {
		ips, err := netutils2.ExpandCompactIps(vmIps)
		if err != nil {
			log.Errorf("ExpandCompactIps net %s wire %s vm_ips %s fail %s", net.Name, wire.Name, vmIps, err)
		} else {
			for _, ip := range ips {
				if net.Contains(ip) {
					return true
				}
			}
		}
	}
	vmMacs := wire.GetMetadata(ctx, "vm_macs", nil)
	if len(vmMacs) > 0 {
		macs := strings.Split(vmMacs, ",")
		for _, mac := range macs {
			gns, err := GuestnetworkManager.fetchGuestNetworks(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				q = q.Equals("network_id", net.Id).Equals("mac_addr", mac)
				return q
			})
			if err != nil {
				log.Errorf("fetchGuestNetworks net %s wire %s mac %s fail %s", net.Name, wire.Name, mac, err)
			} else if len(gns) > 0 {
				return true
			}
		}
	}
	return false
}
