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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/sshkeys"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=cloud_kube_node_pool
// +onecloud:swagger-gen-model-plural=cloud_kube_node_pools
type SKubeNodePoolManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
}

var KubeNodePoolManager *SKubeNodePoolManager

func init() {
	KubeNodePoolManager = &SKubeNodePoolManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SKubeNodePool{},
			"cloud_kube_node_pools_tbl",
			"cloud_kube_node_pool",
			"cloud_kube_node_pools",
		),
	}
	KubeNodePoolManager.SetVirtualObject(KubeNodePoolManager)
}

type SKubeNodePool struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	NetworkIds    *api.SKubeNetworkIds `list:"user" update:"user" create:"required"`
	InstanceTypes *api.SInstanceTypes  `list:"user" update:"user" create:"required"`

	MinInstanceCount     int `nullable:"false" list:"user" create:"optional" default:"0"`
	MaxInstanceCount     int `nullable:"false" list:"user" create:"optional" default:"2"`
	DesiredInstanceCount int `nullable:"false" list:"user" create:"optional" default:"0"`

	RootDiskSizeGb int `nullable:"false" list:"user" create:"optional" default:"100"`

	CloudKubeClusterId string `width:"36" charset:"ascii" name:"cloud_kube_cluster_id" nullable:"false" list:"user" create:"required" index:"true"`
}

func (manager *SKubeNodePoolManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{KubeClusterManager},
	}
}

func (manager *SKubeNodePoolManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SKubeNodePool) GetCloudproviderId() string {
	cluster, err := self.GetKubeCluster()
	if err != nil {
		return ""
	}
	return cluster.ManagerId
}

func (self *SKubeNodePool) GetKubeCluster() (*SKubeCluster, error) {
	cluster, err := KubeClusterManager.FetchById(self.CloudKubeClusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "KubeClusterManager.FetchById")
	}
	return cluster.(*SKubeCluster), nil
}

func (self *SKubeNodePool) GetRegion() (*SCloudregion, error) {
	cluster, err := self.GetKubeCluster()
	if err != nil {
		return nil, errors.Wrapf(err, "GetKubeCluster")
	}
	return cluster.GetRegion()
}

func (self *SKubeNodePool) GetOwnerId() mcclient.IIdentityProvider {
	cluster, err := self.GetKubeCluster()
	if err != nil {
		log.Errorf("failed to get cluster for node pool %s(%s)", self.Name, self.Id)
		return nil
	}
	return cluster.GetOwnerId()
}

func (self *SKubeNodePool) GetNetworks() ([]SNetwork, error) {
	ret := []SNetwork{}
	q := NetworkManager.Query().In("id", self.NetworkIds)
	return ret, db.FetchModelObjects(NetworkManager, q, &ret)
}

func (self *SKubeNodePool) GetIKubeCluster(ctx context.Context) (cloudprovider.ICloudKubeCluster, error) {
	cluster, err := self.GetKubeCluster()
	if err != nil {
		return nil, err
	}
	return cluster.GetIKubeCluster(ctx)
}

func (self *SKubeNodePool) GetIKubeNodePool(ctx context.Context) (cloudprovider.ICloudKubeNodePool, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	cluster, err := self.GetIKubeCluster(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIKubeCluster")
	}
	pools, err := cluster.GetIKubeNodePools()
	if err != nil {
		return nil, err
	}
	for i := range pools {
		if pools[i].GetGlobalId() == self.ExternalId {
			return pools[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.ExternalId)
}

func (manager *SKubeNodePoolManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	info := struct{ KubeClusterId string }{}
	data.Unmarshal(&info)
	if len(info.KubeClusterId) > 0 {
		cluster, err := db.FetchById(KubeClusterManager, info.KubeClusterId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(%s)", info.KubeClusterId)
		}
		return cluster.(*SKubeCluster).GetOwnerId(), nil
	}
	return db.FetchProjectInfo(ctx, data)
}

func (manager *SKubeNodePoolManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if ownerId != nil {
		sq := KubeClusterManager.Query("id")
		switch scope {
		case rbacscope.ScopeDomain, rbacscope.ScopeProject:
			sq = sq.Equals("domain_id", ownerId.GetProjectDomainId())
			return q.In("cloud_kube_cluster_id", sq.SubQuery())
		}
	}
	return q
}

// 同步Kube Node Pool 状态
func (self *SKubeNodePool) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.SyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "KubeNodePoolSyncstatusTask", "")
}

func (self *SKubeNodePool) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.KubeNodePoolUpdateInput) (api.KubeNodePoolUpdateInput, error) {
	var err error
	input.StatusStandaloneResourceBaseUpdateInput, err = self.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrapf(err, "SStatusStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SKubeNodePoolManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.KubeNodePoolDetails {
	rows := make([]api.KubeNodePoolDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	clusterIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.KubeNodePoolDetails{
			StatusStandaloneResourceDetails: stdRows[i],
		}
		pool := objs[i].(*SKubeNodePool)
		clusterIds[i] = pool.CloudKubeClusterId
	}

	clusters := make(map[string]SKubeCluster)
	err := db.FetchStandaloneObjectsByIds(KubeClusterManager, clusterIds, &clusters)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail: %v", err)
		return rows
	}

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if cluster, ok := clusters[clusterIds[i]]; ok {
			virObjs[i] = &cluster
			rows[i].DomainId = cluster.DomainId
		}
	}

	domainRows := KubeClusterManager.SInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, virObjs, stringutils2.SSortedStrings{}, isList)
	for i := range rows {
		rows[i].InfrasResourceBaseDetails = domainRows[i]
	}

	return rows
}

// Kube Node Pool列表
func (manager *SKubeNodePoolManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KubeNodePoolListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	if query.CloudKubeClusterId != "" {
		q = q.Equals("cloud_kube_cluster_id", query.CloudKubeClusterId)
	}
	return q, nil
}

func (manager *SKubeNodePoolManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KubeNodePoolListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SKubeNodePoolManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

type sKubeNodePool struct {
	Name               string
	CloudKubeClusterId string `json:"cloud_kube_cluster_id"`
}

func (self *SKubeNodePool) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(sKubeNodePool{Name: self.Name, CloudKubeClusterId: self.CloudKubeClusterId})
}

func (manager *SKubeNodePoolManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	info := sKubeNodePool{}
	data.Unmarshal(&info)
	return jsonutils.Marshal(info)
}

func (manager *SKubeNodePoolManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	info := sKubeNodePool{}
	values.Unmarshal(&info)
	if len(info.CloudKubeClusterId) > 0 {
		q = q.Equals("cloud_kube_cluster_id", info.CloudKubeClusterId)
	}
	if len(info.Name) > 0 {
		q = q.Equals("name", info.Name)
	}
	return q
}

func (manager *SKubeNodePoolManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.KubeNodePoolCreateInput) (*api.KubeNodePoolCreateInput, error) {
	var err error
	input.StatusStandaloneResourceCreateInput, err = manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}
	clusterObj, err := validators.ValidateModel(ctx, userCred, KubeClusterManager, &input.CloudKubeClusterId)
	if err != nil {
		return nil, err
	}
	cluster := clusterObj.(*SKubeCluster)
	for i := range input.NetworkIds {
		_, err = validators.ValidateModel(ctx, userCred, NetworkManager, &input.NetworkIds[i])
		if err != nil {
			return nil, err
		}
	}
	if len(input.NetworkIds) == 0 {
		return nil, httperrors.NewMissingParameterError("network_ids")
	}
	if len(input.InstanceTypes) == 0 {
		return nil, httperrors.NewMissingParameterError("instance_types")
	}

	if len(input.KeypairId) > 0 {
		keypairObj, err := validators.ValidateModel(ctx, userCred, KeypairManager, &input.KeypairId)
		if err != nil {
			return nil, err
		}
		keypair := keypairObj.(*SKeypair)
		input.PublicKey = keypair.PublicKey
	} else {
		_, input.PublicKey, err = sshkeys.GetSshAdminKeypair(ctx)
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetSshAdminKeypair"))
		}
	}

	if input.DesiredInstanceCount > 0 {
		if input.MinInstanceCount > input.DesiredInstanceCount {
			return nil, httperrors.NewOutOfRangeError("min_instance_count must less or equal to desired_instance_count")
		}
		if input.MaxInstanceCount < input.DesiredInstanceCount {
			return nil, httperrors.NewOutOfRangeError("max_instance_count must greater than or equal to desired_instance_count")
		}
	}

	region, err := cluster.GetRegion()
	if err != nil {
		return nil, err
	}
	return region.GetDriver().ValidateCreateKubeNodePoolData(ctx, userCred, ownerId, &input)
}

func (self *SKubeNodePool) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartKubeNodePoolCreateTask(ctx, userCred, data)
}

func (self *SKubeNodePool) StartKubeNodePoolCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	params := data.(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "KubeNodePoolCreateTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_CREATING, "")
	return task.ScheduleRun(nil)
}

func (self *SKubeCluster) SyncKubeNodePools(ctx context.Context, userCred mcclient.TokenCredential, exts []cloudprovider.ICloudKubeNodePool) compare.SyncResult {
	lockman.LockRawObject(ctx, KubeNodePoolManager.KeywordPlural(), self.Id)
	defer lockman.ReleaseRawObject(ctx, KubeNodePoolManager.KeywordPlural(), self.Id)

	result := compare.SyncResult{}

	dbPools, err := self.GetNodePools()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SKubeNodePool, 0)
	commondb := make([]SKubeNodePool, 0)
	commonext := make([]cloudprovider.ICloudKubeNodePool, 0)
	added := make([]cloudprovider.ICloudKubeNodePool, 0)

	err = compare.CompareSets(dbPools, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudKubeNodePool(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		_, err := self.newFromCloudKubeNodePool(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SKubeNodePool) SyncWithCloudKubeNodePool(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeNodePool) error {
	cluster, err := self.GetKubeCluster()
	if err != nil {
		return errors.Wrapf(err, "GetKubeCluster")
	}
	_, err = db.UpdateWithLock(ctx, self, func() error {
		self.Status = ext.GetStatus()

		networkIds := ext.GetNetworkIds()
		netIds := api.SKubeNetworkIds{}
		for i := range networkIds {
			netObj, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkIds[i], func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				wires := WireManager.Query().SubQuery()
				vpcs := VpcManager.Query().SubQuery()
				return q.Join(wires, sqlchemy.Equals(wires.Field("id"), q.Field("wire_id"))).
					Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id"))).
					Filter(sqlchemy.Equals(vpcs.Field("manager_id"), cluster.ManagerId))
			})
			if err != nil {
				break
			}
			netIds = append(netIds, netObj.GetId())
		}
		if len(networkIds) == len(netIds) && len(netIds) > 0 {
			self.NetworkIds = &netIds
		}
		instanceTypes := api.SInstanceTypes{}
		for _, instanceType := range ext.GetInstanceTypes() {
			instanceTypes = append(instanceTypes, instanceType)
		}
		if len(instanceTypes) > 0 {
			self.InstanceTypes = &instanceTypes
		}
		if minSize := ext.GetMinInstanceCount(); minSize > 0 {
			self.MinInstanceCount = minSize
		}
		if maxSize := ext.GetMaxInstanceCount(); maxSize > 0 {
			self.MaxInstanceCount = maxSize
		}
		if desiredSize := ext.GetDesiredInstanceCount(); desiredSize > 0 {
			self.DesiredInstanceCount = desiredSize
		}
		if rootSize := ext.GetRootDiskSizeGb(); rootSize > 0 {
			self.RootDiskSizeGb = rootSize
		}

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "UpdateWithLock")
	}

	if account := cluster.GetCloudaccount(); account != nil {
		syncMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}

	return nil
}

func (self *SKubeCluster) newFromCloudKubeNodePool(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeNodePool) (*SKubeNodePool, error) {
	pool := SKubeNodePool{}
	pool.SetModelManager(KubeNodePoolManager, &pool)

	pool.Name = ext.GetName()
	pool.Status = ext.GetStatus()
	pool.CloudKubeClusterId = self.Id
	pool.ExternalId = ext.GetGlobalId()

	networkIds := ext.GetNetworkIds()
	netIds := api.SKubeNetworkIds{}
	for i := range networkIds {
		netObj, err := db.FetchByExternalIdAndManagerId(NetworkManager, networkIds[i], func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			wires := WireManager.Query().SubQuery()
			vpcs := VpcManager.Query().SubQuery()
			return q.Join(wires, sqlchemy.Equals(wires.Field("id"), q.Field("wire_id"))).
				Join(vpcs, sqlchemy.Equals(vpcs.Field("id"), wires.Field("vpc_id"))).
				Filter(sqlchemy.Equals(vpcs.Field("manager_id"), self.ManagerId))
		})
		if err != nil {
			break
		}
		netIds = append(netIds, netObj.GetId())
	}
	if len(netIds) > 0 {
		pool.NetworkIds = &netIds
	}

	pool.MinInstanceCount = ext.GetMinInstanceCount()
	pool.MaxInstanceCount = ext.GetMaxInstanceCount()
	pool.DesiredInstanceCount = ext.GetDesiredInstanceCount()
	pool.RootDiskSizeGb = ext.GetRootDiskSizeGb()
	instanceTypes := api.SInstanceTypes{}
	for _, instanceType := range ext.GetInstanceTypes() {
		instanceTypes = append(instanceTypes, instanceType)
	}
	pool.InstanceTypes = &instanceTypes

	err := KubeNodePoolManager.TableSpec().Insert(ctx, &pool)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncMetadata(ctx, userCred, &pool, ext, false)

	return &pool, nil
}

func (self *SKubeNodePool) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("kube node pool delete do nothing")
	return self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
}

func (self *SKubeNodePool) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SKubeNodePool) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartKubeNodePoolDeleteTask(ctx, userCred, "")
}

func (self *SKubeNodePool) StartKubeNodePoolDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "KubeNodePoolDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (manager *SKubeNodePoolManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}
