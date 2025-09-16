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

// +onecloud:swagger-gen-model-singular=cloud_kube_node
// +onecloud:swagger-gen-model-plural=cloud_kube_nodes
type SKubeNodeManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
}

var KubeNodeManager *SKubeNodeManager

func init() {
	KubeNodeManager = &SKubeNodeManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SKubeNode{},
			"cloud_kube_nodes_tbl",
			"cloud_kube_node",
			"cloud_kube_nodes",
		),
	}
	KubeNodeManager.SetVirtualObject(KubeNodeManager)
}

type SKubeNode struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	CloudKubeNodePoolId string `width:"36" charset:"ascii" name:"cloud_kube_node_pool_id" nullable:"false" list:"user" create:"required" index:"true"`
	CloudKubeClusterId  string `width:"36" charset:"ascii" name:"cloud_kube_cluster_id" nullable:"false" list:"user" create:"required" index:"true"`
}

func (manager *SKubeNodeManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{KubeClusterManager},
	}
}

func (manager *SKubeNodeManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SKubeNode) GetCloudproviderId() string {
	cluster, err := self.GetKubeCluster()
	if err != nil {
		return ""
	}
	return cluster.ManagerId
}

func (self *SKubeNode) GetKubeCluster() (*SKubeCluster, error) {
	cluster, err := KubeClusterManager.FetchById(self.CloudKubeClusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "KubeClusterManager.FetchById")
	}
	return cluster.(*SKubeCluster), nil
}

func (self *SKubeNode) GetOwnerId() mcclient.IIdentityProvider {
	cluster, err := self.GetKubeCluster()
	if err != nil {
		log.Errorf("failed to get cluster for node %s(%s)", self.Name, self.Id)
		return nil
	}
	return cluster.GetOwnerId()
}

func (manager *SKubeNodeManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
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

func (manager *SKubeNodeManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
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

func (self *SKubeNode) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.KubeNodeUpdateInput) (api.KubeNodeUpdateInput, error) {
	var err error
	input.StatusStandaloneResourceBaseUpdateInput, err = self.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrapf(err, "SStatusStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SKubeNodeManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.KubeNodeDetails {
	rows := make([]api.KubeNodeDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	clusterIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.KubeNodeDetails{
			StatusStandaloneResourceDetails: stdRows[i],
		}
		node := objs[i].(*SKubeNode)
		clusterIds[i] = node.CloudKubeClusterId
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

// Kube Node列表
func (manager *SKubeNodeManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KubeNodeListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SKubeNodeManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KubeNodeListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SKubeNodeManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

type sKubeNode struct {
	Name               string
	CloudKubeClusterId string `json:"cloud_kube_cluster_id"`
}

func (self *SKubeNode) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(sKubeNode{Name: self.Name, CloudKubeClusterId: self.CloudKubeClusterId})
}

func (manager *SKubeNodeManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	info := sKubeNode{}
	data.Unmarshal(&info)
	return jsonutils.Marshal(info)
}

func (manager *SKubeNodeManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	info := sKubeNode{}
	values.Unmarshal(&info)
	if len(info.CloudKubeClusterId) > 0 {
		q = q.Equals("cloud_kube_cluster_id", info.CloudKubeClusterId)
	}
	if len(info.Name) > 0 {
		q = q.Equals("name", info.Name)
	}
	return q
}

func (manager *SKubeNodeManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.KubeNodeCreateInput) (api.KubeNodeCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implemented")
}

func (self *SKubeCluster) SyncKubeNodes(ctx context.Context, userCred mcclient.TokenCredential, exts []cloudprovider.ICloudKubeNode) compare.SyncResult {
	lockman.LockRawObject(ctx, KubeNodeManager.KeywordPlural(), self.Id)
	defer lockman.ReleaseRawObject(ctx, KubeNodeManager.KeywordPlural(), self.Id)

	result := compare.SyncResult{}

	dbNodes, err := self.GetNodes()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SKubeNode, 0)
	commondb := make([]SKubeNode, 0)
	commonext := make([]cloudprovider.ICloudKubeNode, 0)
	added := make([]cloudprovider.ICloudKubeNode, 0)

	err = compare.CompareSets(dbNodes, exts, &removed, &commondb, &commonext, &added)
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
		err = commondb[i].SyncWithCloudKubeNode(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		_, err := self.newFromCloudKubeNode(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SKubeNode) SyncWithCloudKubeNode(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeNode) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = ext.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "UpdateWithLock")
	}

	cluster, err := self.GetKubeCluster()
	if err == nil {
		if account := cluster.GetCloudaccount(); account != nil {
			syncMetadata(ctx, userCred, self, ext, account.ReadOnly)
		}
	}

	return nil
}

func (self *SKubeCluster) GetNodePoolIdByExternalId(id string) (*SKubeNodePool, error) {
	q := KubeNodePoolManager.Query().Equals("cloud_kube_cluster_id", self.Id).Equals("external_id", id)
	pools := []SKubeNodePool{}
	err := db.FetchModelObjects(KubeNodePoolManager, q, &pools)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModeObjects")
	}
	if len(pools) == 1 {
		return &pools[0], nil
	}
	if len(pools) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}
	return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, id)
}

func (self *SKubeCluster) newFromCloudKubeNode(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeNode) (*SKubeNode, error) {
	node := SKubeNode{}
	node.SetModelManager(KubeNodeManager, &node)

	node.Name = ext.GetName()
	node.Status = ext.GetStatus()
	node.CloudKubeClusterId = self.Id
	node.ExternalId = ext.GetGlobalId()
	if poolId := ext.GetINodePoolId(); len(poolId) > 0 {
		pool, _ := self.GetNodePoolIdByExternalId(poolId)
		if pool != nil {
			node.CloudKubeNodePoolId = pool.Id
		}
	}

	err := KubeNodeManager.TableSpec().Insert(ctx, &node)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncMetadata(ctx, userCred, &node, ext, false)

	return &node, nil
}

func (self *SKubeNode) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("kube node delete do nothing")
	return self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
}

func (self *SKubeNode) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SKubeNode) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartKubeNodeDeleteTask(ctx, userCred, "")
}

func (self *SKubeNode) StartKubeNodeDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "KubeNodeDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (manager *SKubeNodeManager) ListItemExportKeys(ctx context.Context,
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
