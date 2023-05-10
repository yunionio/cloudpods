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
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

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

func (self *SKubeNodePool) GetOwnerId() mcclient.IIdentityProvider {
	cluster, err := self.GetKubeCluster()
	if err != nil {
		log.Errorf("failed to get cluster for node pool %s(%s)", self.Name, self.Id)
		return nil
	}
	return cluster.GetOwnerId()
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

func (manager *SKubeNodePoolManager) FilterByOwner(q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
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

func (manager *SKubeNodePoolManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.KubeNodePoolCreateInput) (api.KubeNodePoolCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implemented")
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
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = ext.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "UpdateWithLock")
	}

	syncMetadata(ctx, userCred, self, ext)

	return nil
}

func (self *SKubeCluster) newFromCloudKubeNodePool(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeNodePool) (*SKubeNodePool, error) {
	pool := SKubeNodePool{}
	pool.SetModelManager(KubeNodePoolManager, &pool)

	pool.Name = ext.GetName()
	pool.Status = ext.GetStatus()
	pool.CloudKubeClusterId = self.Id
	pool.ExternalId = ext.GetGlobalId()

	err := KubeNodePoolManager.TableSpec().Insert(ctx, &pool)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncMetadata(ctx, userCred, &pool, ext)

	return &pool, nil
}

func (self *SKubeNodePool) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("kube node pool delete do nothing")
	return self.SetStatus(userCred, apis.STATUS_DELETING, "")
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
