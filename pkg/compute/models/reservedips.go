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
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
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
	Notes     string `width:"512" charset:"utf8" nullable:"true" list:"admin"`  // ]Column(VARCHAR(512, charset='utf8'), nullable=True)
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

func (manager *SReservedipManager) GetReservedIPs(network *SNetwork) ([]SReservedip, error) {
	rips := make([]SReservedip, 0)
	q := manager.Query().Equals("network_id", network.Id)
	err := db.FetchModelObjects(manager, q, &rips)
	if err != nil {
		return nil, errors.Wrapf(err, "GetReservedIPs.FetchModelObjects")
	}
	return rips, nil
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

func (manager *SReservedipManager) SyncReservedIps(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork, _reservedIps []cloudprovider.SReservedIp) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	syncResult := compare.SyncResult{}
	_dbReservedIps, err := manager.GetReservedIPs(network)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	dbReservedIps := map[string]*SReservedip{}

	for i := 0; i < len(_dbReservedIps); i++ {
		dbReservedIps[_dbReservedIps[i].IpAddr] = &_dbReservedIps[i]
	}

	reservedIps := map[string]string{}
	for _, reservedIp := range _reservedIps {
		reservedIps[reservedIp.IP] = reservedIp.Notes
	}

	for ip, notes := range reservedIps {
		if _, exist := dbReservedIps[ip]; !exist {
			err = manager.newFromCloudNetwork(ctx, userCred, network, ip, notes)
			if err != nil {
				syncResult.AddError(err)
			} else {
				syncResult.Add()
			}
		} else {
			syncResult.Update()
		}
	}

	for ip, dbReservedIp := range dbReservedIps {
		if _, exist := reservedIps[ip]; !exist {
			err = dbReservedIp.Delete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}
	}

	return syncResult
}

func (manager *SReservedipManager) newFromCloudNetwork(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork, ip, notes string) error {
	ipAddr, err := netutils.NewIPV4Addr(ip)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudNetwork.NewIPV4Addr")
	}

	if !network.isAddressInRange(ipAddr) {
		return fmt.Errorf("ip %s not in network %s(%s) ip range", ip, network.Name, network.Id)
	}
	return manager.ReserveIP(userCred, network, ip, notes)
}
