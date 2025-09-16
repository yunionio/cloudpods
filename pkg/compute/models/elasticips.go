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
	"fmt"
	"sort"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/pinyinutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=eip
// +onecloud:swagger-gen-model-plural=eips
type SElasticipManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var ElasticipManager *SElasticipManager

func init() {
	ElasticipManager = &SElasticipManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SElasticip{},
			"elasticips_tbl",
			"eip",
			"eips",
		),
	}
	ElasticipManager.SetVirtualObject(ElasticipManager)
	ElasticipManager.TableSpec().AddIndex(true, "associate_id", "associate_type")
}

type SElasticip struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`

	SBillingResourceBase

	// IP子网Id, 仅私有云不为空
	NetworkId string `width:"36" charset:"ascii" nullable:"true" get:"user" list:"user" create:"optional"`
	// 标识弹性或非弹性
	// | Mode       | 说明       |
	// |------------|------------|
	// | public_ip  | 公网IP     |
	// | elastic_ip | 弹性公网IP |
	//
	// example: elastic_ip
	Mode string `width:"32" charset:"ascii" get:"user" list:"user" create:"optional"`

	// IP地址
	IpAddr string `width:"17" charset:"ascii" list:"user" update:"admin"`

	// 绑定资源类型
	AssociateType string `width:"32" charset:"ascii" list:"user" update:"admin"`
	// 绑定资源Id
	AssociateId string `width:"256" charset:"ascii" list:"user" update:"admin"`

	// 带宽大小
	Bandwidth int `list:"user" create:"optional" default:"0"`

	// 计费类型: 流量、带宽
	// example: bandwidth
	ChargeType string `width:"64" name:"charge_type" list:"user" create:"required"`
	// 线路类型
	BgpType string `width:"64" charset:"utf8" nullable:"true" get:"user" list:"user" create:"optional"`

	// 是否跟随主机删除而自动释放
	AutoDellocate tristate.TriState `default:"false" get:"user" create:"optional" update:"user"`

	// 区域Id
	// CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

// 弹性公网IP列表
func (manager *SElasticipManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticipListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	associateType := query.UsableEipForAssociateType
	associateId := query.UsableEipForAssociateId
	if len(associateType) > 0 && len(associateId) > 0 {
		q = q.Equals("status", api.EIP_STATUS_READY)
		switch associateType {
		case api.EIP_ASSOCIATE_TYPE_SERVER:
			serverObj, err := GuestManager.FetchByIdOrName(ctx, userCred, associateId)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("server %s not found", associateId)
				}
				return nil, httperrors.NewGeneralError(err)
			}
			guest := serverObj.(*SGuest)
			region, err := guest.GetRegion()
			if err != nil {
				return nil, errors.Wrapf(err, "GetRegion")
			}
			if guest.Hypervisor == api.HYPERVISOR_KVM || (utils.IsInStringArray(region.Provider, api.PRIVATE_CLOUD_PROVIDERS) &&
				guest.Hypervisor != api.HYPERVISOR_HCSO && guest.Hypervisor != api.HYPERVISOR_HCS) {
				zone, _ := guest.getZone()
				networks := NetworkManager.Query().SubQuery()
				wires := WireManager.Query().SubQuery()

				sq := networks.Query(networks.Field("id")).Join(wires, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id"))).
					Filter(sqlchemy.Equals(wires.Field("zone_id"), zone.Id)).SubQuery()
				q = q.Filter(sqlchemy.In(q.Field("network_id"), sq))
				gns := GuestnetworkManager.Query("network_id").Equals("guest_id", guest.Id).SubQuery()
				q = q.Filter(sqlchemy.NotIn(q.Field("network_id"), gns))
			} else {
				region, _ := guest.getRegion()
				q = q.Equals("cloudregion_id", region.Id)
			}
			host, _ := guest.GetHost()
			managerId := host.ManagerId
			if managerId != "" {
				q = q.Equals("manager_id", managerId)
			} else {
				q = q.IsNullOrEmpty("manager_id")
			}
		case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
			groupObj, err := GroupManager.FetchByIdOrName(ctx, userCred, associateId)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2(GroupManager.Keyword(), associateId)
				}
				return nil, httperrors.NewGeneralError(err)
			}
			group := groupObj.(*SGroup)
			net, err := group.getAttachedNetwork()
			if err != nil {
				return nil, errors.Wrap(err, "group.getAttachedNetwork")
			}
			if net == nil {
				return nil, errors.Wrap(httperrors.ErrInvalidStatus, "group is not attached to network")
			}

			zone, _ := net.GetZone()
			networks := NetworkManager.Query().SubQuery()
			wires := WireManager.Query().SubQuery()

			sq := networks.Query(networks.Field("id")).Join(wires, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id"))).
				Filter(sqlchemy.Equals(wires.Field("zone_id"), zone.Id)).SubQuery()
			q = q.Filter(sqlchemy.In(q.Field("network_id"), sq))
			q = q.Filter(sqlchemy.NotEquals(q.Field("network_id"), net.Id))
			q = q.IsNullOrEmpty("manager_id")
		case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
			_nat, err := validators.ValidateModel(ctx, userCred, NatGatewayManager, &query.UsableEipForAssociateId)
			if err != nil {
				return nil, err
			}
			nat := _nat.(*SNatGateway)
			vpc, err := nat.GetVpc()
			if err != nil {
				return nil, httperrors.NewGeneralError(errors.Wrapf(err, "nat.GetVpc"))
			}
			q = q.Equals("cloudregion_id", vpc.CloudregionId)
			if len(vpc.ManagerId) > 0 {
				q = q.Equals("manager_id", vpc.ManagerId)
			}
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.Equals(q.Field("associate_id"), nat.Id),
					sqlchemy.IsNullOrEmpty(q.Field("associate_id")),
				),
			)
		case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
			_lb, err := validators.ValidateModel(ctx, userCred, LoadbalancerManager, &query.UsableEipForAssociateId)
			if err != nil {
				return nil, err
			}
			lb := _lb.(*SLoadbalancer)
			q = q.Equals("cloudregion_id", lb.CloudregionId)
			if len(lb.ManagerId) > 0 {
				q = q.Equals("manager_id", lb.ManagerId)
			} else {
				zone, _ := lb.GetZone()
				networks := NetworkManager.Query().SubQuery()
				wires := WireManager.Query().SubQuery()

				sq := networks.Query(networks.Field("id")).Join(wires, sqlchemy.Equals(wires.Field("id"), networks.Field("wire_id"))).
					Filter(sqlchemy.Equals(wires.Field("zone_id"), zone.Id)).SubQuery()
				q = q.Filter(sqlchemy.In(q.Field("network_id"), sq))
				gns := LoadbalancernetworkManager.Query("network_id").Equals("loadbalancer_id", lb.Id).SubQuery()
				q = q.Filter(sqlchemy.NotIn(q.Field("network_id"), gns))
			}
			q = q.IsNullOrEmpty("associate_type")
		default:
			return nil, httperrors.NewInputParameterError("Not support associate type %s, only support %s", associateType, api.EIP_ASSOCIATE_VALID_TYPES)
		}
	}

	if query.Usable != nil && *query.Usable {
		q = q.Equals("status", api.EIP_STATUS_READY)
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("associate_id")), sqlchemy.IsEmpty(q.Field("associate_id"))))
	}

	if query.IsAssociated != nil {
		if *query.IsAssociated {
			q = q.IsNotEmpty("associate_type")
		} else {
			q = q.IsNullOrEmpty("associate_type")
		}
	}

	if len(query.Mode) > 0 {
		q = q.In("mode", query.Mode)
	}
	if len(query.IpAddr) > 0 {
		q = q.In("ip_addr", query.IpAddr)
	}
	if len(query.AssociateType) > 0 {
		q = q.In("associate_type", query.AssociateType)
	}
	if len(query.AssociateId) > 0 {
		q = q.In("associate_id", query.AssociateId)
	}
	if len(query.ChargeType) > 0 {
		q = q.In("charge_type", query.ChargeType)
	}
	if len(query.BgpType) > 0 {
		q = q.In("bgp_type", query.BgpType)
	}
	if query.AutoDellocate != nil {
		if *query.AutoDellocate {
			q = q.IsTrue("auto_dellocate")
		} else {
			q = q.IsFalse("auto_dellocate")
		}
	}
	if len(query.AssociateName) > 0 {
		filters := []sqlchemy.ICondition{}
		likeQuery := func(sq *sqlchemy.SQuery) *sqlchemy.SQuery {
			conditions := []sqlchemy.ICondition{}
			for _, name := range query.AssociateName {
				conditions = append(conditions, sqlchemy.Contains(sq.Field("name"), name))
			}
			return sq.Filter(sqlchemy.OR(conditions...))
		}
		for _, m := range []db.IModelManager{
			GuestManager,
			GroupManager,
			LoadbalancerManager,
			NatGatewayManager,
		} {
			sq := m.Query("id")
			sq = likeQuery(sq)
			filters = append(filters, sqlchemy.In(q.Field("associate_id"), sq))
		}
		q = q.Filter(sqlchemy.OR(filters...))
	}

	return q, nil
}

func (manager *SElasticipManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticipListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	if db.NeedOrderQuery([]string{query.OrderByIp}) {
		db.OrderByFields(q, []string{query.OrderByIp}, []sqlchemy.IQueryField{sqlchemy.INET_ATON(q.Field("ip_addr"))})
	}
	return q, nil
}

func (manager *SElasticipManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SElasticip) GetNetwork() (*SNetwork, error) {
	network, err := NetworkManager.FetchById(self.NetworkId)
	if err != nil {
		return nil, err
	}
	return network.(*SNetwork), nil
}

func (self *SElasticip) GetZone() (*SZone, error) {
	network, err := self.GetNetwork()
	if err != nil {
		return nil, err
	}
	return network.GetZone()
}

func (self *SElasticip) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SVirtualResourceBase.GetShortDesc(ctx)

	// desc.Add(jsonutils.NewString(self.ChargeType), "charge_type")

	desc.Add(jsonutils.NewInt(int64(self.Bandwidth)), "bandwidth")
	desc.Add(jsonutils.NewString(self.Mode), "mode")
	desc.Add(jsonutils.NewString(self.IpAddr), "ip_addr")
	desc.Add(jsonutils.NewString(self.BgpType), "bgp_type")

	// region := self.GetRegion()
	// if len(region.ExternalId) > 0 {
	// regionInfo := strings.Split(region.ExternalId, "/")
	// if len(regionInfo) == 2 {
	// desc.Add(jsonutils.NewString(strings.ToLower(regionInfo[0])), "hypervisor")
	// desc.Add(jsonutils.NewString(regionInfo[1]), "region")
	// }
	//}

	billingInfo := SCloudBillingInfo{}

	billingInfo.SCloudProviderInfo = self.getCloudProviderInfo()

	billingInfo.SBillingBaseInfo = self.getBillingBaseInfo()

	billingInfo.InternetChargeType = self.ChargeType

	if priceKey := self.GetMetadata(ctx, "ext:price_key", nil); len(priceKey) > 0 {
		billingInfo.PriceKey = priceKey
	}

	desc.Update(jsonutils.Marshal(billingInfo))

	return desc
}

func (manager *SElasticipManager) SyncEips(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	region *SCloudregion,
	eips []cloudprovider.ICloudEIP,
	syncOwnerId mcclient.IIdentityProvider,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, manager.KeywordPlural(), region.Id)
	defer lockman.ReleaseRawObject(ctx, manager.KeywordPlural(), region.Id)

	syncResult := compare.SyncResult{}

	dbEips, err := region.GetElasticIps(provider.Id, api.EIP_MODE_STANDALONE_EIP)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := range dbEips {
		if taskman.TaskManager.IsInTask(&dbEips[i]) {
			syncResult.Error(fmt.Errorf("object in task"))
			return syncResult
		}
	}

	removed := make([]SElasticip, 0)
	commondb := make([]SElasticip, 0)
	commonext := make([]cloudprovider.ICloudEIP, 0)
	added := make([]cloudprovider.ICloudEIP, 0)

	err = compare.CompareSets(dbEips, eips, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudEip(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithCloudEip(ctx, userCred, provider, commonext[i], syncOwnerId)
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		eip := &SElasticip{}
		eip.SetModelManager(ElasticipManager, eip)
		err := ElasticipManager.Query().
			Equals("ip_addr", added[i].GetIpAddr()).
			Equals("manager_id", provider.Id).
			Equals("cloudregion_id", region.Id).
			Equals("mode", api.EIP_MODE_INSTANCE_PUBLICIP).First(eip)

		// 公网IP转弹性IP
		if err == nil {
			err = eip.SyncWithCloudEip(ctx, userCred, provider, added[i], syncOwnerId)
			if err != nil {
				syncResult.UpdateError(err)
				continue
			}
			syncResult.Update()
			continue
		}

		_, err = manager.newFromCloudEip(ctx, userCred, added[i], provider, region, syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}

	return syncResult
}

func (self *SElasticip) syncRemoveCloudEip(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.RealDelete(ctx, userCred)
	if err != nil {
		return err
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionSyncDelete,
	})
	return nil
}

func (self *SElasticip) SyncInstanceWithCloudEip(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudEIP) error {
	resource := self.GetAssociateResource()
	vmExtId := ext.GetAssociationExternalId()

	if resource == nil && len(vmExtId) == 0 {
		return nil
	}
	if resource != nil && resource.(db.IExternalizedModel).GetExternalId() == vmExtId {
		return nil
	}

	if resource != nil { // dissociate
		err := self.Dissociate(ctx, userCred)
		if err != nil {
			log.Errorf("fail to dissociate vm: %s", err)
			return err
		}
	}

	if len(vmExtId) > 0 {
		var manager db.IModelManager
		switch ext.GetAssociationType() {
		case api.EIP_ASSOCIATE_TYPE_SERVER:
			manager = GuestManager
		case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
			manager = NatGatewayManager
		case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
			manager = LoadbalancerManager
		// case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
		// not supported
		// 	manager = GroupManager
		default:
			return errors.Error("unsupported association type")
		}

		extRes, err := db.FetchByExternalIdAndManagerId(manager, vmExtId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			switch ext.GetAssociationType() {
			case api.EIP_ASSOCIATE_TYPE_SERVER:
				sq := HostManager.Query().SubQuery()
				return q.Join(sq, sqlchemy.Equals(sq.Field("id"), q.Field("host_id"))).Filter(sqlchemy.Equals(sq.Field("manager_id"), self.ManagerId))
			case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
				return q.Equals("manager_id", self.ManagerId)
			case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
				sq := VpcManager.Query("id").Equals("manager_id", self.ManagerId)
				return q.In("vpc_id", sq.SubQuery())
			}
			return q
		})
		if err != nil {
			return errors.Wrapf(err, "db.FetchByExternalIdAndManagerId %s %s", ext.GetAssociationType(), vmExtId)
		}
		err = self.AssociateInstance(ctx, userCred, ext.GetAssociationType(), extRes.(db.IStatusStandaloneModel))
		if err != nil {
			return errors.Wrapf(err, "AssociateInstance")
		}
	}

	return nil
}

func (self *SElasticip) SyncWithCloudEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudEIP, syncOwnerId mcclient.IIdentityProvider) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, ext.GetName())
			if len(newName) > 0 {
				self.Name = newName
			}
		}
		if bandwidth := ext.GetBandwidth(); bandwidth != 0 {
			self.Bandwidth = bandwidth
		}
		self.IpAddr = ext.GetIpAddr()
		self.Mode = ext.GetMode()
		self.Status = ext.GetStatus()
		self.ExternalId = ext.GetGlobalId()
		self.IsEmulated = ext.IsEmulated()
		self.AssociateType = ext.GetAssociationType()

		if chargeType := ext.GetInternetChargeType(); len(chargeType) > 0 {
			self.ChargeType = chargeType
		}

		factory, _ := provider.GetProviderFactory()
		if factory != nil && factory.IsSupportPrepaidResources() {
			self.BillingType = ext.GetBillingType()
			if expired := ext.GetExpiredAt(); !expired.IsZero() {
				self.ExpiredAt = expired
			}
			self.AutoRenew = ext.IsAutoRenew()
		}

		if createAt := ext.GetCreatedAt(); !createAt.IsZero() {
			self.CreatedAt = createAt
		}

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.UpdateWithLock")
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    self,
			Action: notifyclient.ActionSyncUpdate,
		})
	}

	//err = self.SyncInstanceWithCloudEip(ctx, userCred, ext)
	//if err != nil {
	//	return errors.Wrap(err, "fail to sync associated instance of EIP")
	//}
	if account := self.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}

	// eip有绑定资源，并且绑定资源是项目资源,eip项目信息跟随绑定资源
	if res := self.GetAssociateResource(); res != nil && len(res.GetOwnerId().GetProjectId()) > 0 {
		self.SyncCloudProjectId(userCred, res.GetOwnerId())
	} else {
		SyncCloudProject(ctx, userCred, self, syncOwnerId, ext, provider)
	}

	return nil
}

func (manager *SElasticipManager) newFromCloudEip(ctx context.Context, userCred mcclient.TokenCredential, extEip cloudprovider.ICloudEIP, provider *SCloudprovider, region *SCloudregion, syncOwnerId mcclient.IIdentityProvider) (*SElasticip, error) {
	eip := SElasticip{}
	eip.SetModelManager(manager, &eip)

	eip.Status = extEip.GetStatus()
	eip.ExternalId = extEip.GetGlobalId()
	eip.IpAddr = extEip.GetIpAddr()
	eip.Mode = extEip.GetMode()
	eip.IsEmulated = extEip.IsEmulated()
	eip.ManagerId = provider.Id
	eip.CloudregionId = region.Id
	eip.ChargeType = extEip.GetInternetChargeType()
	eip.AssociateType = extEip.GetAssociationType()
	if !extEip.GetCreatedAt().IsZero() {
		eip.CreatedAt = extEip.GetCreatedAt()
	}
	if len(eip.ChargeType) == 0 {
		eip.ChargeType = api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
	eip.Bandwidth = extEip.GetBandwidth()
	if networkId := extEip.GetINetworkId(); len(networkId) > 0 {
		network, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			wire := WireManager.Query().SubQuery()
			vpc := VpcManager.Query().SubQuery()
			return q.Join(wire, sqlchemy.Equals(wire.Field("id"), q.Field("wire_id"))).
				Join(vpc, sqlchemy.Equals(vpc.Field("id"), wire.Field("vpc_id"))).
				Filter(sqlchemy.Equals(vpc.Field("manager_id"), provider.Id))
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to found network by externalId %s", networkId)
		}
		eip.NetworkId = network.GetId()
	}

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, syncOwnerId, extEip.GetName())
		if err != nil {
			return err
		}
		eip.Name = newName

		return manager.TableSpec().Insert(ctx, &eip)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudEip")
	}

	//err = eip.SyncInstanceWithCloudEip(ctx, userCred, extEip)
	//if err != nil {
	//	return nil, errors.Wrap(err, "fail to sync associated instance of EIP")
	//}

	syncVirtualResourceMetadata(ctx, userCred, &eip, extEip, false)

	if res := eip.GetAssociateResource(); res != nil && len(res.GetOwnerId().GetProjectId()) > 0 {
		eip.SyncCloudProjectId(userCred, res.GetOwnerId())
	} else {
		SyncCloudProject(ctx, userCred, &eip, syncOwnerId, extEip, provider)
	}

	db.OpsLog.LogEvent(&eip, db.ACT_CREATE, eip.GetShortDesc(ctx), userCred)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &eip,
		Action: notifyclient.ActionSyncCreate,
	})

	return &eip, nil
}

func (manager *SElasticipManager) getEipForInstance(instanceType string, instanceId string) (*SElasticip, error) {
	return manager.getEip(instanceType, instanceId, api.EIP_MODE_STANDALONE_EIP)
}

func (manager *SElasticipManager) getEip(instanceType string, instanceId string, eipMode string) (*SElasticip, error) {
	eip := SElasticip{}

	q := manager.Query()
	q = q.Equals("associate_type", instanceType)
	q = q.Equals("associate_id", instanceId)
	if len(eipMode) > 0 {
		q = q.Equals("mode", eipMode)
	}

	err := q.First(&eip)

	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("getEipForInstance query fail %s", err)
			return nil, err
		} else {
			return nil, nil
		}
	}

	eip.SetModelManager(manager, &eip)

	return &eip, nil
}

func (self *SElasticip) IsAssociated() bool {
	if len(self.AssociateId) == 0 {
		return false
	}
	if self.GetAssociateVM() != nil {
		return true
	}
	if self.GetAssociateLoadbalancer() != nil {
		return true
	}
	if self.GetAssociateNatGateway() != nil {
		return true
	}
	if self.GetAssociateInstanceGroup() != nil {
		return true
	}
	return false
}

func (self *SElasticip) GetAssociateVM() *SGuest {
	if self.AssociateType == api.EIP_ASSOCIATE_TYPE_SERVER && len(self.AssociateId) > 0 {
		return GuestManager.FetchGuestById(self.AssociateId)
	}
	return nil
}

func (self *SElasticip) GetAssociateInstanceGroup() *SGroup {
	if self.AssociateType == api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP && len(self.AssociateId) > 0 {
		_grp, err := GroupManager.FetchById(self.AssociateId)
		if err != nil {
			return nil
		}
		return _grp.(*SGroup)
	}
	return nil
}

func (self *SElasticip) GetAssociateLoadbalancer() *SLoadbalancer {
	if self.AssociateType == api.EIP_ASSOCIATE_TYPE_LOADBALANCER && len(self.AssociateId) > 0 {
		_lb, err := LoadbalancerManager.FetchById(self.AssociateId)
		if err != nil {
			return nil
		}
		lb := _lb.(*SLoadbalancer)
		if lb.PendingDeleted {
			return nil
		}
		return lb
	}
	return nil
}

func (self *SElasticip) GetAssociateNatGateway() *SNatGateway {
	if self.AssociateType == api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY && len(self.AssociateId) > 0 {
		natGateway, err := NatGatewayManager.FetchById(self.AssociateId)
		if err != nil {
			return nil
		}
		return natGateway.(*SNatGateway)
	}
	return nil
}

func (self *SElasticip) GetAssociateResource() db.IModel {
	if vm := self.GetAssociateVM(); vm != nil {
		return vm
	}
	if lb := self.GetAssociateLoadbalancer(); lb != nil {
		return lb
	}
	if nat := self.GetAssociateNatGateway(); nat != nil {
		return nat
	}
	if grp := self.GetAssociateInstanceGroup(); grp != nil {
		return grp
	}
	return nil
}

func (self *SElasticip) Dissociate(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(self.AssociateType) == 0 {
		return nil
	}
	var vm *SGuest
	var nat *SNatGateway
	var lb *SLoadbalancer
	var grp *SGroup
	switch self.AssociateType {
	case api.EIP_ASSOCIATE_TYPE_SERVER:
		vm = self.GetAssociateVM()
		if vm == nil {
			log.Errorf("dissociate VM not exists???")
		}
	case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
		nat = self.GetAssociateNatGateway()
		if nat == nil {
			log.Errorf("dissociate Nat gateway not exists???")
		}
	case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
		lb = self.GetAssociateLoadbalancer()
		if lb == nil {
			log.Errorf("dissociate loadbalancer not exists???")
		}
	case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
		grp = self.GetAssociateInstanceGroup()
		if grp == nil {
			log.Errorf("dissociate instance_group not exists???")
		}
	}

	_, err := db.Update(self, func() error {
		self.AssociateId = ""
		self.AssociateType = ""
		return nil
	})
	if err != nil {
		return err
	}
	if vm != nil {
		db.OpsLog.LogDetachEvent(ctx, vm, self, userCred, self.GetShortDesc(ctx))
		db.OpsLog.LogEvent(self, db.ACT_EIP_DETACH, vm.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(vm, db.ACT_EIP_DETACH, self.GetShortDesc(ctx), userCred)
	}

	if nat != nil {
		db.OpsLog.LogDetachEvent(ctx, nat, self, userCred, self.GetShortDesc(ctx))
		db.OpsLog.LogEvent(self, db.ACT_EIP_DETACH, nat.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(nat, db.ACT_EIP_DETACH, self.GetShortDesc(ctx), userCred)
	}

	if lb != nil {
		db.OpsLog.LogDetachEvent(ctx, lb, self, userCred, self.GetShortDesc(ctx))
		db.OpsLog.LogEvent(self, db.ACT_EIP_DETACH, lb.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(lb, db.ACT_EIP_DETACH, self.GetShortDesc(ctx), userCred)
	}

	if grp != nil {
		db.OpsLog.LogDetachEvent(ctx, grp, self, userCred, self.GetShortDesc(ctx))
		db.OpsLog.LogEvent(self, db.ACT_EIP_DETACH, grp.GetShortDesc(ctx), userCred)
		db.OpsLog.LogEvent(grp, db.ACT_EIP_DETACH, self.GetShortDesc(ctx), userCred)
	}

	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		self.RealDelete(ctx, userCred)
	}
	return nil
}

func (self *SElasticip) AssociateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer) error {
	if lb.PendingDeleted {
		return fmt.Errorf("loadbalancer is deleted")
	}
	if len(self.AssociateType) > 0 && len(self.AssociateId) > 0 {
		if self.AssociateType == api.EIP_ASSOCIATE_TYPE_LOADBALANCER && self.AssociateId == lb.Id {
			return nil
		} else {
			return fmt.Errorf("EIP has been associated!!")
		}
	}
	_, err := db.Update(self, func() error {
		self.AssociateType = api.EIP_ASSOCIATE_TYPE_LOADBALANCER
		self.AssociateId = lb.Id
		return nil
	})
	if err != nil {
		return err
	}

	db.OpsLog.LogAttachEvent(ctx, lb, self, userCred, self.GetShortDesc(ctx))
	db.OpsLog.LogEvent(self, db.ACT_EIP_ATTACH, lb.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(lb, db.ACT_EIP_ATTACH, self.GetShortDesc(ctx), userCred)

	return nil
}

func (self *SElasticip) AssociateInstance(ctx context.Context, userCred mcclient.TokenCredential, insType string, ins db.IStatusStandaloneModel) error {
	switch insType {
	case api.EIP_ASSOCIATE_TYPE_SERVER:
		vm := ins.(*SGuest)
		if vm.PendingDeleted || vm.Deleted {
			return fmt.Errorf("vm is deleted")
		}
	}
	if len(self.AssociateType) > 0 && len(self.AssociateId) > 0 {
		if self.AssociateType == insType && self.AssociateId == ins.GetId() {
			return nil
		}
		return fmt.Errorf("EIP has been associated!!")
	}
	_, err := db.Update(self, func() error {
		self.AssociateType = insType
		self.AssociateId = ins.GetId()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}

	db.OpsLog.LogAttachEvent(ctx, ins, self, userCred, self.GetShortDesc(ctx))
	db.OpsLog.LogEvent(self, db.ACT_EIP_ATTACH, ins.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(ins, db.ACT_EIP_ATTACH, self.GetShortDesc(ctx), userCred)

	return nil
}

func (self *SElasticip) AssociateInstanceGroup(ctx context.Context, userCred mcclient.TokenCredential, insType string, ins db.IStatusStandaloneModel) error {
	switch insType {
	case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
		vm := ins.(*SGroup)
		if vm.PendingDeleted || vm.Deleted {
			return fmt.Errorf("group is deleted")
		}
	}
	if len(self.AssociateType) > 0 && len(self.AssociateId) > 0 {
		if self.AssociateType == insType && self.AssociateId == ins.GetId() {
			return nil
		}
		return fmt.Errorf("EIP has been associated!!")
	}
	_, err := db.Update(self, func() error {
		self.AssociateType = insType
		self.AssociateId = ins.GetId()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}

	db.OpsLog.LogAttachEvent(ctx, ins, self, userCred, self.GetShortDesc(ctx))
	db.OpsLog.LogEvent(self, db.ACT_EIP_ATTACH, ins.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(ins, db.ACT_EIP_ATTACH, self.GetShortDesc(ctx), userCred)

	return nil
}

func (self *SElasticip) AssociateNatGateway(ctx context.Context, userCred mcclient.TokenCredential, nat *SNatGateway) error {
	if nat.Deleted {
		return fmt.Errorf("nat gateway is deleted")
	}
	if len(self.AssociateId) > 0 && self.AssociateType == api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY && self.AssociateId == nat.Id {
		return nil
	}
	_, err := db.Update(self, func() error {
		self.AssociateType = api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
		self.AssociateId = nat.Id
		return nil
	})
	if err != nil {
		return err
	}

	db.OpsLog.LogAttachEvent(ctx, nat, self, userCred, self.GetShortDesc(ctx))
	db.OpsLog.LogEvent(self, db.ACT_EIP_ATTACH, nat.GetShortDesc(ctx), userCred)
	db.OpsLog.LogEvent(nat, db.ACT_EIP_ATTACH, self.GetShortDesc(ctx), userCred)

	return nil
}

func (manager *SElasticipManager) getEipByExtEip(ctx context.Context, userCred mcclient.TokenCredential, extEip cloudprovider.ICloudEIP, provider *SCloudprovider, region *SCloudregion, syncOwnerId mcclient.IIdentityProvider) (*SElasticip, error) {
	eipId := extEip.GetGlobalId()
	eipObj, err := db.FetchByExternalIdAndManagerId(manager, eipId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("manager_id", provider.Id)
	})
	if err == nil {
		return eipObj.(*SElasticip), nil
	}
	if errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrapf(err, "FetchByExternalIdAndManagerId %s", eipId)
	}

	return manager.newFromCloudEip(ctx, userCred, extEip, provider, region, syncOwnerId)
}

func (manager *SElasticipManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SElasticipCreateInput) (api.SElasticipCreateInput, error) {
	if input.CloudregionId == "" {
		input.CloudregionId = api.DEFAULT_REGION_ID
	}
	obj, err := CloudregionManager.FetchByIdOrName(ctx, nil, input.CloudregionId)
	if err != nil {
		if err != sql.ErrNoRows {
			return input, httperrors.NewGeneralError(err)
		}
		return input, httperrors.NewResourceNotFoundError2("cloudregion", input.CloudregionId)
	}
	var (
		region       = obj.(*SCloudregion)
		regionDriver = region.GetDriver()
	)
	input.CloudregionId = region.GetId()

	// publicIp cannot be created standalone
	input.Mode = api.EIP_MODE_STANDALONE_EIP

	if input.ChargeType == "" {
		input.ChargeType = regionDriver.GetEipDefaultChargeType()
	}

	if !utils.IsInStringArray(input.ChargeType, []string{api.EIP_CHARGE_TYPE_BY_BANDWIDTH, api.EIP_CHARGE_TYPE_BY_TRAFFIC}) {
		return input, httperrors.NewInputParameterError("charge type %s not supported", input.ChargeType)
	}

	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, err
	}

	err = regionDriver.ValidateCreateEipData(ctx, userCred, &input)
	if err != nil {
		return input, err
	}

	var provider *SCloudprovider = nil
	if input.ManagerId != "" {
		providerObj, err := CloudproviderManager.FetchByIdOrName(ctx, nil, input.ManagerId)
		if err != nil {
			if err != sql.ErrNoRows {
				return input, httperrors.NewGeneralError(err)
			}
			return input, httperrors.NewResourceNotFoundError2("cloudprovider", input.ManagerId)
		}
		provider = providerObj.(*SCloudprovider)
		input.ManagerId = provider.Id
	}

	//避免参数重名后还有pending.eip残留
	eipPendingUsage := &SRegionQuota{Eip: 1}
	quotaKeys := fetchRegionalQuotaKeys(rbacscope.ScopeProject, ownerId, region, provider)
	eipPendingUsage.SetKeys(quotaKeys)
	if err = quotas.CheckSetPendingQuota(ctx, userCred, eipPendingUsage); err != nil {
		return input, err
	}

	return input, nil
}

func (eip *SElasticip) GetQuotaKeys() (quotas.IQuotaKeys, error) {
	region, err := eip.GetRegion()
	if region == nil {
		return nil, errors.Wrapf(err, "eip.GetRegion")
	}
	return fetchRegionalQuotaKeys(
		rbacscope.ScopeProject,
		eip.GetOwnerId(),
		region,
		eip.GetCloudprovider(),
	), nil
}

func (self *SElasticip) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	eipPendingUsage := &SRegionQuota{Eip: 1}
	keys, err := self.GetQuotaKeys()
	if err != nil {
		log.Errorf("GetQuotaKeys fail %s", err)
	} else {
		eipPendingUsage.SetKeys(keys)
		err := quotas.CancelPendingUsage(ctx, userCred, eipPendingUsage, eipPendingUsage, true)
		if err != nil {
			log.Errorf("SElasticip CancelPendingUsage error: %s", err)
		}
	}

	self.startEipAllocateTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (self *SElasticip) startEipAllocateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "EipAllocateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.EIP_STATUS_ALLOCATE, "start allocate")
	return task.ScheduleRun(nil)
}

func (self *SElasticip) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// Elasticip delete do nothing
	return nil
}

func (self *SElasticip) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SElasticip) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartEipDeallocateTask(ctx, userCred, "")
}

func (self *SElasticip) ValidateDeleteCondition(ctx context.Context, info *api.ElasticipDetails) error {
	if gotypes.IsNil(info) {
		if self.IsAssociated() {
			return fmt.Errorf("eip is associated with resources")
		}
	} else if len(info.AssociateName) > 0 {
		return fmt.Errorf("eip is associated with resources")
	}
	return self.SVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SElasticip) StartEipDeallocateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "EipDeallocateTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("newTask EipDeallocateTask fail %s", err)
		return err
	}
	self.SetStatus(ctx, userCred, api.EIP_STATUS_DEALLOCATE, "start to delete")
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticip) PerformAssociate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ElasticipAssociateInput) (api.ElasticipAssociateInput, error) {
	if self.IsAssociated() {
		return input, httperrors.NewConflictError("eip has been associated with instance")
	}

	if self.Status != api.EIP_STATUS_READY {
		return input, httperrors.NewInvalidStatusError("eip cannot associate in status %s", self.Status)
	}

	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return input, httperrors.NewUnsupportOperationError("fixed eip cannot be associated")
	}

	if len(input.InstanceId) == 0 {
		return input, httperrors.NewMissingParameterError("instance_id")
	}
	if len(input.InstanceType) == 0 {
		input.InstanceType = api.EIP_ASSOCIATE_TYPE_SERVER
	}

	if !utils.IsInStringArray(input.InstanceType, api.EIP_ASSOCIATE_VALID_TYPES) {
		return input, httperrors.NewUnsupportOperationError("Unsupported instance type %s", input.InstanceType)
	}

	switch input.InstanceType {
	case api.EIP_ASSOCIATE_TYPE_SERVER:
		vmObj, err := GuestManager.FetchByIdOrName(ctx, userCred, input.InstanceId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError("server %s not found", input.InstanceId)
			}
			return input, httperrors.NewGeneralError(err)
		}
		server := vmObj.(*SGuest)

		lockman.LockObject(ctx, server)
		defer lockman.ReleaseObject(ctx, server)

		if server.PendingDeleted {
			return input, httperrors.NewInvalidStatusError("cannot associate pending delete server")
		}
		// IMPORTANT: this serves as a guard against a guest to have multiple
		// associated elastic_ips
		seip, _ := server.GetEipOrPublicIp()
		if seip != nil && (seip.Mode == api.EIP_MODE_STANDALONE_EIP || seip.GetProviderName() != api.CLOUD_PROVIDER_AWS) {
			return input, httperrors.NewInvalidStatusError("instance is already associated with eip")
		}

		if ok, _ := utils.InStringArray(server.Status, []string{api.VM_READY, api.VM_RUNNING}); !ok {
			return input, httperrors.NewInvalidStatusError("cannot associate server in status %s", server.Status)
		}

		err = ValidateAssociateEip(server)
		if err != nil {
			return input, err
		}

		if len(self.NetworkId) > 0 {
			gns, err := server.GetNetworks("")
			if err != nil {
				return input, httperrors.NewGeneralError(errors.Wrap(err, "GetNetworks"))
			}
			for _, gn := range gns {
				if gn.NetworkId == self.NetworkId {
					return input, httperrors.NewInputParameterError("cannot associate eip with same network")
				}
			}
		}
		serverRegion, _ := server.getRegion()
		if serverRegion == nil {
			return input, httperrors.NewInputParameterError("server region is not found???")
		}

		eipRegion, err := self.GetRegion()
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetRegion"))
		}

		if serverRegion.Id != eipRegion.Id {
			return input, httperrors.NewInputParameterError("eip and server are not in the same region")
		}
		eipZone, _ := self.GetZone()
		if eipZone != nil {
			serverZone, _ := server.getZone()
			if serverZone.Id != eipZone.Id {
				return input, httperrors.NewInputParameterError("eip and server are not in the same zone")
			}
		}

		srvHost, _ := server.GetHost()
		if srvHost == nil {
			return input, httperrors.NewInputParameterError("server host is not found???")
		}

		if srvHost.ManagerId != self.ManagerId {
			return input, httperrors.NewInputParameterError("server and eip are not managed by the same provider")
		}
		input.InstanceExternalId = server.ExternalId
	case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
		grpObj, err := GroupManager.FetchByIdOrName(ctx, userCred, input.InstanceId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError("instance group %s not found", input.InstanceId)
			}
			return input, httperrors.NewGeneralError(err)
		}
		group := grpObj.(*SGroup)

		lockman.LockObject(ctx, group)
		defer lockman.ReleaseObject(ctx, group)

		net, err := group.isEipAssociable()
		if err != nil {
			return input, errors.Wrap(err, "grp.isEipAssociable")
		}
		if net.Id == self.NetworkId {
			return input, httperrors.NewInputParameterError("cannot associate eip with same network")
		}

		eipZone, _ := self.GetZone()
		if eipZone != nil {
			insZone, _ := net.GetZone()
			if eipZone.Id != insZone.Id {
				return input, httperrors.NewInputParameterError("cannot associate eip and instance in different zone")
			}
		}

	case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
		natgwObj, err := NatGatewayManager.FetchByIdOrName(ctx, userCred, input.InstanceId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError("nat gateway %s not found", input.InstanceId)
			}
			return input, httperrors.NewGeneralError(err)
		}
		natgw := natgwObj.(*SNatGateway)

		lockman.LockObject(ctx, natgw)
		defer lockman.ReleaseObject(ctx, natgw)
	case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
		obj, err := LoadbalancerManager.FetchByIdOrName(ctx, userCred, input.InstanceId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError("loadbalancer %s not found", input.InstanceId)
			}
			return input, httperrors.NewGeneralError(err)
		}
		lb := obj.(*SLoadbalancer)

		if len(self.NetworkId) > 0 {
			nets, err := lb.GetNetworks()
			if err != nil {
				return input, httperrors.NewGeneralError(errors.Wrap(err, "GetNetworks"))
			}
			for _, net := range nets {
				if net.Id == self.NetworkId {
					return input, httperrors.NewInputParameterError("cannot associate eip with same network")
				}
			}
		}

		lockman.LockObject(ctx, lb)
		defer lockman.ReleaseObject(ctx, lb)

		if lb.PendingDeleted {
			return input, httperrors.NewInvalidStatusError("cannot associate with pending deleted loadbalancer")
		}
		seip, _ := lb.GetEip()
		if seip != nil {
			return input, httperrors.NewInvalidStatusError("loadbalancer is already associated with eip")
		}
	}

	return input, self.StartEipAssociateInstanceTask(ctx, userCred, input, "")
}

func (self *SElasticip) StartEipAssociateInstanceTask(ctx context.Context, userCred mcclient.TokenCredential, input api.ElasticipAssociateInput, parentTaskId string) error {
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	return self.StartEipAssociateTask(ctx, userCred, params, parentTaskId)
}

func (self *SElasticip) StartEipAssociateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "EipAssociateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.EIP_STATUS_ASSOCIATE, "start to associate")
	return task.ScheduleRun(nil)
}

func (self *SElasticip) PerformDissociate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ElasticDissociateInput) (jsonutils.JSONObject, error) {
	if len(self.AssociateId) == 0 {
		return nil, nil // success
	}

	// associate with an invalid vm
	res := self.GetAssociateResource()
	if res == nil {
		return nil, self.Dissociate(ctx, userCred)
	}

	err := db.IsObjectRbacAllowed(ctx, res, userCred, policy.PolicyActionGet)
	if err != nil {
		return nil, errors.Wrap(err, "associated resource is not accessible")
	}

	if self.Status != api.EIP_STATUS_READY {
		return nil, httperrors.NewInvalidStatusError("eip cannot dissociate in status %s", self.Status)
	}

	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed public eip cannot be dissociated")
	}

	err = self.StartEipDissociateTask(ctx, userCred, input.AutoDelete, "")
	return nil, err
}

func (self *SElasticip) StartEipDissociateTask(ctx context.Context, userCred mcclient.TokenCredential, autoDelete bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	if autoDelete {
		params.Add(jsonutils.JSONTrue, "auto_delete")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "EipDissociateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("create EipDissociateTask fail %s", err)
		return nil
	}
	self.SetStatus(ctx, userCred, api.EIP_STATUS_DISSOCIATE, "start to dissociate")
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticip) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetDriver")
	}

	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetRegion")
	}

	return provider.GetIRegionById(region.GetExternalId())
}

func (self *SElasticip) GetIEip(ctx context.Context) (cloudprovider.ICloudEIP, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iregion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, err
	}
	return iregion.GetIEipById(self.GetExternalId())
}

// 同步弹性公网IP状态
func (self *SElasticip) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ElasticipSyncstatusInput) (jsonutils.JSONObject, error) {
	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed eip cannot sync status")
	}
	if self.IsManaged() {
		return nil, StartResourceSyncStatusTask(ctx, userCred, self, "EipSyncstatusTask", "")
	} else {
		return nil, self.SetStatus(ctx, userCred, api.EIP_STATUS_READY, "eip sync status")
	}
}

func (self *SElasticip) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed eip cannot sync status")
	}
	if self.IsManaged() {
		return nil, StartResourceSyncStatusTask(ctx, userCred, self, "EipSyncstatusTask", "")
	}
	return nil, nil
}

func (manager *SElasticipManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticipDetails {
	rows := make([]api.ElasticipDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	lbIds := []string{}
	guestIds := []string{}
	natIds := []string{}
	groupIds := []string{}
	for i := range rows {
		rows[i] = api.ElasticipDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     managerRows[i],
			CloudregionResourceInfo: regionRows[i],
		}
		eip := objs[i].(*SElasticip)
		if len(eip.AssociateId) > 0 {
			switch eip.AssociateType {
			case api.EIP_ASSOCIATE_TYPE_SERVER:
				guestIds = append(guestIds, eip.AssociateId)
			case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
				lbIds = append(lbIds, eip.AssociateId)
			case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
				natIds = append(natIds, eip.AssociateId)
			case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
				groupIds = append(groupIds, eip.AssociateId)
			}
		}
	}
	guests, err := db.FetchIdNameMap2(GuestManager, guestIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 guests")
		return rows
	}
	lbs, err := db.FetchIdNameMap2(LoadbalancerManager, lbIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 loadbalancer")
		return rows
	}
	nats, err := db.FetchIdNameMap2(NatGatewayManager, natIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 natgateway")
		return rows
	}
	groups, err := db.FetchIdNameMap2(GroupManager, groupIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 group")
		return rows
	}
	for i := range rows {
		eip := objs[i].(*SElasticip)
		if len(eip.AssociateId) > 0 {
			switch eip.AssociateType {
			case api.EIP_ASSOCIATE_TYPE_SERVER:
				rows[i].AssociateName, _ = guests[eip.AssociateId]
			case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
				rows[i].AssociateName, _ = lbs[eip.AssociateId]
			case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
				rows[i].AssociateName, _ = nats[eip.AssociateId]
			case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
				rows[i].AssociateName, _ = groups[eip.AssociateId]
			}
		}
	}
	return rows
}

type SEipNetwork struct {
	owner   mcclient.IIdentityProvider
	user    mcclient.TokenCredential
	freeCnt int
	net     *SNetwork
}

func NewEipNetwork(net *SNetwork, owner mcclient.IIdentityProvider, user mcclient.TokenCredential, freecnt int) SEipNetwork {
	return SEipNetwork{
		owner:   owner,
		user:    user,
		freeCnt: freecnt,
		net:     net,
	}
}

func (en SEipNetwork) isOwnerProject() bool {
	return en.owner.GetProjectId() == en.net.ProjectId
}

func (en SEipNetwork) isOwnerProjectDomain() bool {
	return en.owner.GetProjectDomainId() == en.net.DomainId
}

func (en SEipNetwork) isUserProject() bool {
	return en.user.GetProjectId() == en.net.ProjectId
}

func (en SEipNetwork) isUserProjectDomain() bool {
	return en.user.GetProjectDomainId() == en.net.DomainId
}

func (en SEipNetwork) GetNetwork() *SNetwork {
	return en.net
}

type SEipNetworks []SEipNetwork

func (a SEipNetworks) Len() int {
	return len(a)
}

func (a SEipNetworks) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a SEipNetworks) Less(i, j int) bool {
	if a[i].isOwnerProject() && !a[j].isOwnerProject() {
		return true
	} else if !a[i].isOwnerProject() && a[j].isOwnerProject() {
		return false
	}
	if a[i].isOwnerProjectDomain() && !a[j].isOwnerProjectDomain() {
		return true
	} else if !a[i].isOwnerProjectDomain() && a[j].isOwnerProjectDomain() {
		return false
	}
	if a[i].isUserProject() && !a[j].isUserProject() {
		return true
	} else if !a[i].isUserProject() && a[j].isUserProject() {
		return false
	}
	if a[i].isUserProjectDomain() && !a[j].isUserProjectDomain() {
		return true
	} else if !a[i].isUserProjectDomain() && a[j].isUserProjectDomain() {
		return false
	}
	if a[i].freeCnt > a[j].freeCnt {
		return true
	} else if a[i].freeCnt < a[j].freeCnt {
		return false
	}
	if a[i].net.Id < a[j].net.Id {
		return true
	}
	return false
}

type NewEipForVMOnHostArgs struct {
	Bandwidth     int
	BgpType       string
	ChargeType    string
	AutoDellocate bool

	Group        *SGroup
	Guest        *SGuest
	Host         *SHost
	Natgateway   *SNatGateway
	Loadbalancer *SLoadbalancer

	PendingUsage quotas.IQuota
}

func (manager *SElasticipManager) NewEipForVMOnHost(ctx context.Context, userCred mcclient.TokenCredential, args *NewEipForVMOnHostArgs) (*SElasticip, error) {
	var (
		bw            = args.Bandwidth
		bgpType       = args.BgpType
		chargeType    = args.ChargeType
		autoDellocate = args.AutoDellocate
		grp           = args.Group
		vm            = args.Guest
		host          = args.Host
		nat           = args.Natgateway
		lb            = args.Loadbalancer
		pendingUsage  = args.PendingUsage

		region *SCloudregion = nil
	)

	if host != nil {
		region, _ = host.GetRegion()
	} else if lb != nil {
		region, _ = lb.GetRegion()
	} else if nat != nil {
		region, _ = nat.GetRegion()
	} else if grp != nil {
		net, _ := grp.getAttachedNetwork()
		region, _ = net.GetRegion()
	} else {
		return nil, fmt.Errorf("invalid host or nat")
	}

	regionDriver := region.GetDriver()

	if chargeType == "" {
		chargeType = regionDriver.GetEipDefaultChargeType()
	}
	if err := regionDriver.ValidateEipChargeType(chargeType); err != nil {
		return nil, err
	}

	eip := &SElasticip{}
	eip.SetModelManager(manager, eip)

	eip.Mode = api.EIP_MODE_STANDALONE_EIP
	// do not implicitly auto dellocate EIP, should be set by user explicitly
	// eip.AutoDellocate = tristate.True
	eip.Bandwidth = bw
	eip.ChargeType = chargeType
	eip.BgpType = args.BgpType
	eip.AutoDellocate = tristate.NewFromBool(autoDellocate)
	ownerCred := userCred.(mcclient.IIdentityProvider)
	if vm != nil {
		ownerCred = vm.GetOwnerId()
	} else if nat != nil {
		ownerCred = nat.GetOwnerId()
	} else if lb != nil {
		ownerCred = lb.GetOwnerId()
	} else if grp != nil {
		ownerCred = grp.GetOwnerId()
	} else {
		panic("unsupported associate type")
	}
	eip.DomainId = ownerCred.GetProjectDomainId()
	eip.ProjectId = ownerCred.GetProjectId()
	eip.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
	if host != nil {
		eip.ManagerId = host.ManagerId
	} else if nat != nil {
		vpc, err := nat.GetVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "nat.GetVpc")
		}
		eip.ManagerId = vpc.ManagerId
	} else if lb != nil {
		vpc, err := lb.GetVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "nat.GetVpc")
		}
		eip.ManagerId = vpc.ManagerId
	} else if grp != nil {
	} else {
		panic("unsupported associate type")
	}
	eip.CloudregionId = region.Id
	if vm != nil {
		eip.Name = fmt.Sprintf("eip-for-srv-%s", pinyinutils.Text2Pinyin(vm.GetName()))
	} else if nat != nil {
		eip.Name = fmt.Sprintf("eip-for-nat-%s", pinyinutils.Text2Pinyin(nat.GetName()))
	} else if grp != nil {
		eip.Name = fmt.Sprintf("eip-for-grp-%s", pinyinutils.Text2Pinyin(grp.GetName()))
	} else if lb != nil {
		eip.Name = fmt.Sprintf("eip-for-lb-%s", pinyinutils.Text2Pinyin(lb.GetName()))
	} else {
		panic("unsupported associate type")
	}

	if (host != nil && host.ManagerId == "") || grp != nil || lb != nil { // kvm
		q := NetworkManager.Query()

		var zoneId string
		if host != nil {
			zoneId = host.ZoneId
		} else if grp != nil {
			net, _ := grp.getAttachedNetwork()
			zone, _ := net.GetZone()
			zoneId = zone.Id
		} else if lb != nil {
			zone, _ := lb.GetZone()
			zoneId = zone.Id
		}

		wireq := WireManager.Query().SubQuery()
		scope, _ := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), NetworkManager.KeywordPlural(), policy.PolicyActionList)
		q = NetworkManager.FilterByOwner(ctx, q, NetworkManager, userCred, userCred, scope)
		q = q.Join(wireq, sqlchemy.Equals(wireq.Field("id"), q.Field("wire_id"))).
			Filter(sqlchemy.Equals(wireq.Field("zone_id"), zoneId))

		q = q.Equals("server_type", api.NETWORK_TYPE_EIP)
		q = q.Equals("bgp_type", bgpType)

		var nets []SNetwork
		if err := db.FetchModelObjects(NetworkManager, q, &nets); err != nil {
			return nil, errors.Wrapf(err, "fetch eip networks usable in host %s(%s)",
				host.Name, host.Id)
		}
		eipNets := make([]SEipNetwork, 0)
		for i := range nets {
			net := &nets[i]
			cnt, _ := net.GetFreeAddressCount()
			if cnt > 0 {
				eipNets = append(eipNets, NewEipNetwork(net, ownerCred, userCred, cnt))
			}
		}
		if len(eipNets) == 0 {
			return nil, httperrors.NewNotFoundError("no usable eip network")
		}
		// prefer networks with identical project, domain, more free address, Id
		sort.Sort(SEipNetworks(eipNets))
		log.Debugf("eipnets: %s", jsonutils.Marshal(eipNets))
		eip.NetworkId = eipNets[0].GetNetwork().Id
	}

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		var err error
		eip.Name, err = db.GenerateName(ctx, manager, userCred, eip.Name)
		if err != nil {
			return errors.Wrap(err, "db.GenerateName")
		}

		return manager.TableSpec().Insert(ctx, eip)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "TableSpec().Insert")
	}
	db.OpsLog.LogEvent(eip, db.ACT_CREATE, eip.GetShortDesc(ctx), userCred)

	var ownerId mcclient.IIdentityProvider = nil
	if vm != nil {
		ownerId = vm.GetOwnerId()
	} else if nat != nil {
		ownerId = nat.GetOwnerId()
	} else if grp != nil {
		ownerId = grp.GetOwnerId()
	} else if lb != nil {
		ownerId = lb.GetOwnerId()
	} else {
		panic("unsupported associate type")
	}

	var provider *SCloudprovider = nil
	if host != nil {
		provider = host.GetCloudprovider()
	} else if nat != nil {
		provider = nat.GetCloudprovider()
	} else if lb != nil {
		provider = lb.GetCloudprovider()
	} else if grp != nil {
	} else {
		panic("unsupported associate type")
	}

	eipPendingUsage := &SRegionQuota{Eip: 1}
	keys := fetchRegionalQuotaKeys(
		rbacscope.ScopeProject,
		ownerId,
		region,
		provider,
	)
	eipPendingUsage.SetKeys(keys)
	quotas.CancelPendingUsage(ctx, userCred, pendingUsage, eipPendingUsage, true)

	return eip, nil
}

func (eip *SElasticip) AllocateAndAssociateInstance(ctx context.Context, userCred mcclient.TokenCredential, ins IEipAssociateInstance, input api.ElasticipAssociateInput, parentTaskId string) error {
	err := ValidateAssociateEip(ins)
	if err != nil {
		return err
	}

	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)

	db.StatusBaseSetStatus(ctx, ins, userCred, api.INSTANCE_ASSOCIATE_EIP, "allocate and associate EIP")
	return eip.startEipAllocateTask(ctx, userCred, params, parentTaskId)
}

func (self *SElasticip) PerformChangeBandwidth(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != api.EIP_STATUS_READY {
		return nil, httperrors.NewInvalidStatusError("cannot change bandwidth in status %s", self.Status)
	}

	bandwidth, err := data.Int("bandwidth")
	if err != nil || bandwidth <= 0 {
		return nil, httperrors.NewInputParameterError("Invalid bandwidth")
	}

	if self.IsManaged() {
		factory, err := self.GetProviderFactory()
		if err != nil {
			return nil, err
		}

		if err := factory.ValidateChangeBandwidth(self.AssociateId, bandwidth); err != nil {
			return nil, httperrors.NewInputParameterError("%v", err)
		}
	}

	err = self.StartEipChangeBandwidthTask(ctx, userCred, bandwidth)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, nil
}

func (self *SElasticip) StartEipChangeBandwidthTask(ctx context.Context, userCred mcclient.TokenCredential, bandwidth int64) error {

	self.SetStatus(ctx, userCred, api.EIP_STATUS_CHANGE_BANDWIDTH, "change bandwidth")

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(bandwidth), "bandwidth")

	task, err := taskman.TaskManager.NewTask(ctx, "EipChangeBandwidthTask", self, userCred, params, "", "", nil)
	if err != nil {
		log.Errorf("create EipChangeBandwidthTask fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticip) DoChangeBandwidth(ctx context.Context, userCred mcclient.TokenCredential, bandwidth int) error {
	changes := jsonutils.NewDict()
	changes.Add(jsonutils.NewInt(int64(self.Bandwidth)), "obw")

	_, err := db.Update(self, func() error {
		self.Bandwidth = bandwidth
		return nil
	})

	self.SetStatus(ctx, userCred, api.EIP_STATUS_READY, "finish change bandwidth")

	if err != nil {
		log.Errorf("DoChangeBandwidth update fail %s", err)
		return err
	}

	changes.Add(jsonutils.NewInt(int64(bandwidth)), "nbw")
	db.OpsLog.LogEvent(self, db.ACT_CHANGE_BANDWIDTH, changes, userCred)

	return nil
}

type EipUsage struct {
	PublicIPCount     int
	PublicIpBandwidth int
	EIPCount          int
	EipBandwidth      int
	EIPUsedCount      int
}

func (u EipUsage) Total() int {
	return u.PublicIPCount + u.EIPCount
}

func (manager *SElasticipManager) usageQByCloudEnv(q *sqlchemy.SQuery, providers []string, brands []string, cloudEnv string) *sqlchemy.SQuery {
	return CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
}

func (manager *SElasticipManager) usageQByRanges(q *sqlchemy.SQuery, rangeObjs []db.IStandaloneModel) *sqlchemy.SQuery {
	return RangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"), nil, nil)
}

func (manager *SElasticipManager) usageQ(
	ctx context.Context,
	scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, q *sqlchemy.SQuery, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) *sqlchemy.SQuery {
	q = manager.usageQByRanges(q, rangeObjs)
	q = manager.usageQByCloudEnv(q, providers, brands, cloudEnv)
	switch scope {
	case rbacscope.ScopeSystem:
		// do nothing
	case rbacscope.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacscope.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}
	q = db.ObjectIdQueryWithPolicyResult(ctx, q, manager, policyResult)
	return q
}

func (manager *SElasticipManager) TotalCount(
	ctx context.Context,
	scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) EipUsage {
	usage := EipUsage{}
	q1sq := manager.Query().SubQuery()
	q1 := q1sq.Query(
		sqlchemy.COUNT("public_ip_count", q1sq.Field("id")),
		sqlchemy.SUM("public_ip_bandwidth", q1sq.Field("bandwidth")),
	).Equals("mode", api.EIP_MODE_INSTANCE_PUBLICIP)
	q1 = manager.usageQ(ctx, scope, ownerId, q1, rangeObjs, providers, brands, cloudEnv, policyResult)
	q2sq := manager.Query().SubQuery()
	q2 := q2sq.Query(
		sqlchemy.COUNT("eip_count", q2sq.Field("id")),
		sqlchemy.SUM("eip_bandwidth", q2sq.Field("bandwidth")),
	).Equals("mode", api.EIP_MODE_STANDALONE_EIP)
	q2 = manager.usageQ(ctx, scope, ownerId, q2, rangeObjs, providers, brands, cloudEnv, policyResult)
	q3sq := manager.Query().SubQuery()
	q3 := q3sq.Query(
		sqlchemy.COUNT("eip_used_count", q3sq.Field("id")),
	).Equals("mode", api.EIP_MODE_STANDALONE_EIP).IsNotEmpty("associate_type")
	q3 = manager.usageQ(ctx, scope, ownerId, q3, rangeObjs, providers, brands, cloudEnv, policyResult)

	err := q1.First(&usage)
	if err != nil {
		log.Errorf("q.First error: %v", err)
	}
	err = q2.First(&usage)
	if err != nil {
		log.Errorf("q.First error: %v", err)
	}
	err = q3.First(&usage)
	if err != nil {
		log.Errorf("q.First error: %v", err)
	}
	return usage
}

func (self *SElasticip) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return nil, err
	}
	provider := self.GetCloudprovider()
	if provider != nil {
		if provider.GetEnabled() {
			return nil, httperrors.NewInvalidStatusError("Cannot purge elastic_ip on enabled cloud provider")
		}
	}
	err = self.RealDelete(ctx, userCred)
	return nil, err
}

func (self *SElasticip) DoPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	if self.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		self.SVirtualResourceBase.DoPendingDelete(ctx, userCred)
		return
	}
	self.Dissociate(ctx, userCred)
}

func (self *SElasticip) getCloudProviderInfo() SCloudProviderInfo {
	region, _ := self.GetRegion()
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(region, nil, provider)
}

func (eip *SElasticip) GetUsages() []db.IUsage {
	if eip.PendingDeleted || eip.Deleted {
		return nil
	}
	usage := SRegionQuota{Eip: 1}
	keys, err := eip.GetQuotaKeys()
	if err != nil {
		log.Errorf("disk.GetQuotaKeys fail %s", err)
		return nil
	}
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (self *SElasticip) PerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ElasticipRemoteUpdateInput) (jsonutils.JSONObject, error) {
	err := self.StartRemoteUpdateTask(ctx, userCred, (input.ReplaceTags != nil && *input.ReplaceTags), "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRemoteUpdateTask")
	}
	return nil, nil
}

func (self *SElasticip) StartRemoteUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, replaceTags bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewBool(replaceTags), "replace_tags")
	task, err := taskman.TaskManager.NewTask(ctx, "EipRemoteUpdateTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_UPDATE_TAGS, "StartRemoteUpdateTask")
	return task.ScheduleRun(nil)
}

func (self *SElasticip) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(self.ExternalId) == 0 || options.Options.KeepTagLocalization {
		return
	}
	if account := self.GetCloudaccount(); account != nil && account.ReadOnly {
		return
	}
	err := self.StartRemoteUpdateTask(ctx, userCred, true, "")
	if err != nil {
		log.Errorf("StartRemoteUpdateTask fail: %s", err)
	}
}

func (manager *SElasticipManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
