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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SKubeClusterManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var KubeClusterManager *SKubeClusterManager

func init() {
	KubeClusterManager = &SKubeClusterManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SKubeCluster{},
			"cloud_kube_clusters_tbl",
			"cloud_kube_cluster",
			"cloud_kube_clusters",
		),
	}
	KubeClusterManager.SetVirtualObject(KubeClusterManager)
}

type SKubeCluster struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" default:"default"`
	// 本地KubeserverId
	ExternalClusterId string `width:"36" charset:"ascii" nullable:"false" list:"admin"`
}

func (manager *SKubeClusterManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *SKubeCluster) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.KubeClusterUpdateInput) (api.KubeClusterUpdateInput, error) {
	if _, err := self.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput); err != nil {
		return input, err
	}
	return input, nil
}

func (self *SKubeCluster) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return self.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SKubeCluster) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		return nil, errors.Wrap(err, "CloudregionManager.FetchById")
	}
	return region.(*SCloudregion), nil
}

func (manager *SKubeClusterManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.KubeClusterDetails {
	rows := make([]api.KubeClusterDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.KubeClusterDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:                    managerRows[i],
			CloudregionResourceInfo:                regionRows[i],
		}
	}
	return rows
}

func (self *SCloudregion) GetKubeClusters(managerId string) ([]SKubeCluster, error) {
	q := KubeClusterManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	clusters := []SKubeCluster{}
	err := db.FetchModelObjects(KubeClusterManager, q, &clusters)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return clusters, nil
}

func (self *SCloudregion) SyncKubeClusters(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, clusters []cloudprovider.ICloudKubeCluster) ([]SKubeCluster, []cloudprovider.ICloudKubeCluster, compare.SyncResult) {
	lockman.LockRawObject(ctx, KubeClusterManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))
	defer lockman.ReleaseRawObject(ctx, KubeClusterManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))

	localClusters := make([]SKubeCluster, 0)
	remoteClusters := make([]cloudprovider.ICloudKubeCluster, 0)
	result := compare.SyncResult{}

	dbClusters, err := self.GetKubeClusters(provider.Id)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	removed := make([]SKubeCluster, 0)
	commondb := make([]SKubeCluster, 0)
	commonext := make([]cloudprovider.ICloudKubeCluster, 0)
	added := make([]cloudprovider.ICloudKubeCluster, 0)

	err = compare.CompareSets(dbClusters, clusters, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return nil, nil, result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudKubeCluster(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudKubeCluster(ctx, userCred, commonext[i], provider)
		if err != nil {
			result.UpdateError(err)
			continue
		}
		localClusters = append(localClusters, commondb[i])
		remoteClusters = append(remoteClusters, commonext[i])
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		newKubeCluster, err := self.newFromCloudKubeCluster(ctx, userCred, added[i], provider)
		if err != nil {
			result.AddError(err)
			continue
		}
		localClusters = append(localClusters, *newKubeCluster)
		remoteClusters = append(remoteClusters, added[i])
		result.Add()
	}

	return localClusters, remoteClusters, result
}

func (self *SKubeCluster) syncRemoveCloudKubeCluster(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		self.SetStatus(userCred, apis.STATUS_UNKNOWN, "sync to delete")
		return errors.Wrapf(err, "ValidateDeleteCondition")
	}
	return self.RealDelete(ctx, userCred)
}

func (self *SKubeCluster) ImportOrUpdate(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeCluster) error {
	if len(self.ExternalClusterId) == 0 {
		return self.doRemoteImport(ctx, userCred, ext)
	}
	return self.doRemoteUpdate(ctx, userCred, ext)
}

func (self *SKubeCluster) doRemoteImport(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeCluster) error {
	config, err := ext.GetKubeConfig(false, 0)
	if err != nil {
		return errors.Wrapf(err, "GetKubeConfig")
	}

	params := map[string]interface{}{
		"name":                self.Name,
		"project_domain_id":   self.DomainId,
		"domain_id":           self.DomainId,
		"mode":                "import",
		"external_cluster_id": self.GetId(),
		"resource_type":       "guest",
		"import_data": map[string]interface{}{
			"kubeconfig": config.Config,
		},
	}
	s := auth.GetAdminSession(ctx, options.Options.Region)
	resp, err := k8s.KubeClusters.Create(s, jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrapf(err, "Create")
	}
	id, err := resp.GetString("id")
	if err != nil {
		return errors.Wrapf(err, "resp.GetId")
	}
	if _, err := db.Update(self, func() error {
		self.ExternalClusterId = id
		return nil
	}); err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return nil
}

func (self *SKubeCluster) doRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeCluster) error {
	// TODO
	return nil
}

func (self *SKubeCluster) SyncAllWithCloudKubeCluster(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeCluster, provider *SCloudprovider) error {
	err := self.SyncWithCloudKubeCluster(ctx, userCred, ext, provider)
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudKubeCluster")
	}

	err = syncKubeClusterNodePools(ctx, userCred, SSyncResultSet{}, self, ext)
	if err != nil {
		return errors.Wrapf(err, "syncKubeClusterNodePools")
	}

	return syncKubeClusterNodes(ctx, userCred, SSyncResultSet{}, self, ext)
}

func (self *SKubeCluster) SyncWithCloudKubeCluster(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeCluster, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = ext.GetStatus()
		return nil
	})
	if err != nil {
		return err
	}

	syncMetadata(ctx, userCred, self, ext)

	if provider != nil {
		SyncCloudDomain(userCred, self, provider.GetOwnerId())
		self.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SCloudregion) newFromCloudKubeCluster(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudKubeCluster, provider *SCloudprovider) (*SKubeCluster, error) {
	cluster := SKubeCluster{}
	cluster.SetModelManager(KubeClusterManager, &cluster)

	cluster.CloudregionId = self.Id
	cluster.ManagerId = provider.Id
	cluster.ExternalId = ext.GetGlobalId()
	cluster.Enabled = tristate.True
	cluster.Status = ext.GetStatus()

	var err = func() error {
		lockman.LockRawObject(ctx, KubeClusterManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, KubeClusterManager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, KubeClusterManager, userCred, ext.GetName())
		if err != nil {
			return err
		}
		cluster.Name = newName

		return KubeClusterManager.TableSpec().Insert(ctx, &cluster)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncMetadata(ctx, userCred, &cluster, ext)
	SyncCloudDomain(userCred, &cluster, provider.GetOwnerId())

	if provider != nil {
		cluster.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	db.OpsLog.LogEvent(&cluster, db.ACT_CREATE, cluster.GetShortDesc(ctx), userCred)

	return &cluster, nil
}

func (manager *SKubeClusterManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.KubeClusterCreateInput,
) (api.KubeClusterCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implemented")
}

func (self *SKubeCluster) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrap(err, "GetRegion")
	}
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetIRegionById(region.GetExternalId())
}

func (self *SKubeCluster) GetIKubeCluster(ctx context.Context) (cloudprovider.ICloudKubeCluster, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	return iRegion.GetICloudKubeClusterById(self.ExternalId)
}

func (self *SKubeCluster) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	self.SetStatus(userCred, apis.STATUS_DELETING, "")
	return nil
}

func (self *SKubeCluster) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.KubeClusterDeleteInput) error {
	return self.StartDeleteKubeClusterTask(ctx, userCred, input.Retain)
}

func (self *SKubeCluster) GetNodes() ([]SKubeNode, error) {
	nodes := []SKubeNode{}
	q := KubeNodeManager.Query().Equals("cloud_kube_cluster_id", self.Id)
	err := db.FetchModelObjects(KubeNodeManager, q, &nodes)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return nodes, nil
}

func (self *SKubeCluster) GetNodePools() ([]SKubeNodePool, error) {
	pools := []SKubeNodePool{}
	q := KubeNodePoolManager.Query().Equals("cloud_kube_cluster_id", self.Id)
	err := db.FetchModelObjects(KubeNodePoolManager, q, &pools)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return pools, nil
}

func (self *SKubeCluster) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	nodes, err := self.GetNodes()
	if err != nil {
		return errors.Wrapf(err, "GetNodes")
	}
	for i := range nodes {
		err = nodes[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "delete kube node %s", nodes[i].Name)
		}
	}
	pools, err := self.GetNodePools()
	if err != nil {
		return errors.Wrapf(err, "GetNodePools")
	}
	for i := range pools {
		err = pools[i].RealDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "delete kube node pool %s", pools[i].Name)
		}
	}
	if len(self.ExternalClusterId) > 0 {
		s := auth.GetAdminSession(ctx, options.Options.Region)
		_, err = k8s.KubeClusters.PerformAction(s,
			self.ExternalClusterId,
			"purge",
			jsonutils.Marshal(map[string]interface{}{"force": true}),
		)
		if err != nil {
			return errors.Wrapf(err, "Create")
		}
	}
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SKubeCluster) StartDeleteKubeClusterTask(ctx context.Context, userCred mcclient.TokenCredential, isRetail bool) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(isRetail), "retain")
	task, err := taskman.TaskManager.NewTask(ctx, "KubeClusterDeleteTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

// 列出Kube Cluster
func (manager *SKubeClusterManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KubeClusterListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
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

	return q, nil
}

func (manager *SKubeClusterManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "cluster":
		q = q.AppendField(q.Field("name").Label("cluster")).Distinct()
		return q, nil
	default:
		var err error
		q, err = manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
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

	}
	return q, httperrors.ErrNotFound
}

func (manager *SKubeClusterManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KubeClusterListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

// 同步Kube Cluster状态
func (self *SKubeCluster) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.SyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "KubeClusterSyncstatusTask", "")
}

func (cluster *SKubeCluster) GetQuotaKeys() quotas.SDomainRegionalCloudResourceKeys {
	region, _ := cluster.GetRegion()
	manager := cluster.GetCloudprovider()
	ownerId := cluster.GetOwnerId()
	regionKeys := fetchRegionalQuotaKeys(rbacscope.ScopeDomain, ownerId, region, manager)
	keys := quotas.SDomainRegionalCloudResourceKeys{}
	keys.SBaseDomainQuotaKeys = regionKeys.SBaseDomainQuotaKeys
	keys.SRegionalBaseKeys = regionKeys.SRegionalBaseKeys
	keys.SCloudResourceBaseKeys = regionKeys.SCloudResourceBaseKeys
	return keys
}

func (self *SKubeCluster) GetUsages() []db.IUsage {
	if self.Deleted {
		return nil
	}
	//usage := SInfrasQuota{KubeCluster: 1}
	usage := SInfrasQuota{}
	keys := self.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (self *SKubeCluster) GetChangeOwnerCandidateDomainIds() []string {
	candidates := [][]string{
		self.SManagedResourceBase.GetChangeOwnerCandidateDomainIds(),
	}
	return db.ISharableMergeChangeOwnerCandidateDomainIds(self, candidates...)
}

func (manager *SKubeClusterManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (self *SKubeCluster) GetDetailsKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, input api.GetKubeConfigInput) (*cloudprovider.SKubeconfig, error) {
	iCluster, err := self.GetIKubeCluster(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIKubeCluster")
	}
	return iCluster.GetKubeConfig(input.Private, input.ExpireMinutes)
}
