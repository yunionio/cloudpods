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
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SReservedipManager struct {
	db.SResourceBaseManager
}

var ReservedipManager *SReservedipManager

func init() {
	ReservedipManager = &SReservedipManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SReservedip{},
			"reservedips_tbl",
			"reservedip",
			"reservedips",
		),
	}
	ReservedipManager.SetVirtualObject(ReservedipManager)
}

type SReservedip struct {
	db.SResourceBase

	Id        int64  `primary:"true" auto_increment:"true" list:"admin"`        // = Column(BigInteger, primary_key=True)
	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"admin"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	IpAddr    string `width:"16" charset:"ascii" list:"admin"`                  // Column(VARCHAR(16, charset='ascii'))

	Notes string `width:"512" charset:"utf8" nullable:"true" list:"admin" update:"admin"` // ]Column(VARCHAR(512, charset='utf8'), nullable=True)
}

func (manager *SReservedipManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SReservedipManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SReservedip) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return false
}

func (self *SReservedip) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SReservedip) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SReservedipManager) ReserveIP(userCred mcclient.TokenCredential, network *SNetwork, ip string, notes string) error {
	rip := SReservedip{NetworkId: network.Id, IpAddr: ip, Notes: notes}
	err := manager.TableSpec().Insert(&rip)
	if err != nil {
		log.Errorf("ReserveIP fail: %s", err)
		return err
	}
	db.OpsLog.LogEvent(network, db.ACT_RESERVE_IP, ip, userCred)
	return nil
}

func (manager *SReservedipManager) GetReservedIP(network *SNetwork, ip string) *SReservedip {
	rip := SReservedip{}
	rip.SetModelManager(manager, &rip)

	err := manager.Query().Equals("network_id", network.Id).Equals("ip_addr", ip).First(&rip)
	if err != nil {
		log.Errorf("GetReservedIP fail: %s", err)
		return nil
	}
	return &rip
}

func (manager *SReservedipManager) GetReservedIPs(network *SNetwork) []SReservedip {
	rips := make([]SReservedip, 0)
	q := manager.Query().Equals("network_id", network.Id)
	err := db.FetchModelObjects(manager, q, &rips)
	if err != nil {
		log.Errorf("GetReservedIPs fail: %s", err)
		return nil
	}
	return rips
}

func (self *SReservedip) GetNetwork() *SNetwork {
	net, _ := NetworkManager.FetchById(self.NetworkId)
	if net != nil {
		return net.(*SNetwork)
	}
	return nil
}

func (self *SReservedip) Release(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork) error {
	// db.DeleteModel(self.ResourceModelManager(), self)
	if network == nil {
		network = self.GetNetwork()
	}
	err := db.DeleteModel(ctx, userCred, self)
	if err == nil && network != nil {
		db.OpsLog.LogEvent(network, db.ACT_RELEASE_IP, self.IpAddr, userCred)
	}
	return err
}

func (self *SReservedip) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SResourceBase.GetCustomizeColumns(ctx, userCred, query)
	net := self.GetNetwork()
	if net != nil {
		extra.Add(jsonutils.NewString(net.Name), "network")
	}
	return extra
}

func (manager *SReservedipManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		log.Errorf("ListItemFilter %s", err)
		return nil, err
	}
	network, _ := query.GetString("network")
	if len(network) > 0 {
		netObj, _ := NetworkManager.FetchByIdOrName(userCred, network)
		if netObj == nil {
			return nil, httperrors.NewResourceNotFoundError("network %s not found", network)
		}
		q = q.Equals("network_id", netObj.GetId())
	}
	return q, nil
}

func (rip *SReservedip) GetId() string {
	return strconv.FormatInt(rip.Id, 10)
}

func (rip *SReservedip) GetName() string {
	return rip.GetId()
}
