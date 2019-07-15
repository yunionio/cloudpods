package models

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
)

type SNetworkinterfacenetworkManager struct {
	db.SJointResourceBaseManager
}

var NetworkinterfacenetworkManager *SNetworkinterfacenetworkManager

func init() {
	db.InitManager(func() {
		NetworkinterfacenetworkManager = &SNetworkinterfacenetworkManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SNetworkinterfacenetwork{},
				"networkinterfacenetworks_tbl",
				"networkinterfacenetwork",
				"networkinterfacenetworks",
				NetworkInterfaceManager,
				NetworkManager,
			),
		}
		GuestnetworkManager.SetVirtualObject(GuestnetworkManager)
	})
}

type SNetworkinterfacenetwork struct {
	db.SJointResourceBase

	Primary            bool   `nullable:"false" list:"user"`
	IpAddr             string `width:"16" charset:"ascii" nullable:"false" list:"user"`
	NetworkinterfaceId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
	NetworkId          string `width:"36" charset:"ascii" nullable:"false" list:"admin"`
}

func (manager *SNetworkinterfacenetworkManager) GetMasterFieldName() string {
	return "networkinterface_id"
}

func (manager *SNetworkinterfacenetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (joint *SNetworkinterfacenetwork) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SNetworkinterfacenetwork) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (manager *SNetworkinterfacenetworkManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SNetworkinterfacenetworkManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (manager *SNetworkinterfacenetworkManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model db.IStandaloneModel, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SNetworkinterfacenetworkManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master db.IStandaloneModel, slave db.IStandaloneModel) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SNetworkinterfacenetwork) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SNetworkinterfacenetwork) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SNetworkinterfacenetwork) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SNetworkinterfacenetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SNetworkinterfacenetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SNetworkinterfacenetworkManager) SyncInterfaceAddresses(ctx context.Context, userCred mcclient.TokenCredential, networkinterface *SNetworkInterface, exts []cloudprovider.ICloudInterfaceAddress) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, networkinterface.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, networkinterface.GetOwnerId()))

	syncResult := compare.SyncResult{}

	dbResources, err := networkinterface.GetNetworks()
	if err != nil {
		return syncResult
	}

	removed := make([]SNetworkinterfacenetwork, 0)
	commondb := make([]SNetworkinterfacenetwork, 0)
	commonext := make([]cloudprovider.ICloudInterfaceAddress, 0)
	added := make([]cloudprovider.ICloudInterfaceAddress, 0)
	if err := compare.CompareSets(dbResources, exts, &removed, &commondb, &commonext, &added); err != nil {
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].Delete(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].SyncWithCloudkInterfaceAddress(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncResult.Update()
	}

	for i := 0; i < len(added); i += 1 {
		err := manager.newFromCloudInterfaceAddress(ctx, userCred, networkinterface, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncResult.Add()
	}
	return syncResult
}

func (self *SNetworkinterfacenetwork) SyncWithCloudkInterfaceAddress(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudInterfaceAddress) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Primary = ext.IsPrimary()

		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SNetworkinterfacenetworkManager) newFromCloudInterfaceAddress(ctx context.Context, userCred mcclient.TokenCredential, networkinterface *SNetworkInterface, ext cloudprovider.ICloudInterfaceAddress) error {
	address := SNetworkinterfacenetwork{
		IpAddr:             ext.GetIP(),
		NetworkinterfaceId: networkinterface.Id,
		Primary:            ext.IsPrimary(),
	}
	address.SetModelManager(manager, &address)

	networkId := ext.GetINetworkId()
	_network, err := db.FetchByExternalId(NetworkManager, networkId)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudInterfaceAddress.FetchByExternalId(%s)", networkId)
	}

	ipAddr, err := netutils.NewIPV4Addr(address.IpAddr)
	if err != nil {
		return errors.Wrap(err, "netutils.NewIPV4Addr")
	}

	network := _network.(*SNetwork)
	if !network.isAddressInRange(ipAddr) {
		return fmt.Errorf("ip %s not in network %s(%s) range", address.IpAddr, network.Name, network.Id)
	}

	address.NetworkId = network.Id

	err = manager.TableSpec().Insert(&address)
	if err != nil {
		return errors.Wrap(err, "TableSpec().Insert(&address)")
	}

	db.OpsLog.LogEvent(&address, db.ACT_CREATE, address.GetShortDesc(ctx), userCred)
	return nil
}
