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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SElasticcache.Acl
type SElasticcacheAclManager struct {
	db.SStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SElasticcacheResourceBaseManager
}

var ElasticcacheAclManager *SElasticcacheAclManager

func init() {
	ElasticcacheAclManager = &SElasticcacheAclManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SElasticcacheAcl{},
			"elasticcacheacls_tbl",
			"elasticcacheacl",
			"elasticcacheacls",
		),
	}
	ElasticcacheAclManager.SetVirtualObject(ElasticcacheAclManager)
}

type SElasticcacheAcl struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SElasticcacheResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`

	// ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // elastic cache instance id

	// Ip地址白名单列表
	IpList string `width:"256" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required" json:"ip_list"`
}

func (manager *SElasticcacheAclManager) SyncElasticcacheAcls(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, cloudElasticcacheAcls []cloudprovider.ICloudElasticcacheAcl) compare.SyncResult {
	lockman.LockRawObject(ctx, "elastic-cache-acls", elasticcache.Id)
	defer lockman.ReleaseRawObject(ctx, "elastic-cache-acls", elasticcache.Id)

	syncResult := compare.SyncResult{}

	dbAcls, err := elasticcache.GetElasticcacheAcls()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SElasticcacheAcl, 0)
	commondb := make([]SElasticcacheAcl, 0)
	commonext := make([]cloudprovider.ICloudElasticcacheAcl, 0)
	added := make([]cloudprovider.ICloudElasticcacheAcl, 0)
	if err := compare.CompareSets(dbAcls, cloudElasticcacheAcls, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticcacheAcl(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudElasticcacheAcl(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}

		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := manager.newFromCloudElasticcacheAcl(ctx, userCred, elasticcache, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}

		syncResult.Add()
	}
	return syncResult
}

func (self *SElasticcacheAcl) syncRemoveCloudElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudElasticcacheAcl.Remove")
	}
	return self.Delete(ctx, userCred)
}

func (self *SElasticcacheAcl) SyncWithCloudElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, extAcl cloudprovider.ICloudElasticcacheAcl) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.IpList = extAcl.GetIpList()
		self.Status = extAcl.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudElasticcacheAcl.UpdateWithLock")
	}

	return nil
}

func (manager *SElasticcacheAclManager) newFromCloudElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, extAcl cloudprovider.ICloudElasticcacheAcl) (*SElasticcacheAcl, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	acl := SElasticcacheAcl{}
	acl.SetModelManager(manager, &acl)

	acl.ElasticcacheId = elasticcache.GetId()
	acl.Status = extAcl.GetStatus()
	acl.Name = extAcl.GetName()
	acl.ExternalId = extAcl.GetGlobalId()
	acl.IpList = extAcl.GetIpList()

	err := manager.TableSpec().Insert(ctx, &acl)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcacheAcl.Insert")
	}

	return &acl, nil
}

func (self *SElasticcacheAcl) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{"elasticcache_id": self.ElasticcacheId})
}

func (manager *SElasticcacheAclManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	elasticcacheId := jsonutils.GetAnyString(data, []string{"elasticcache_id", "elasticcache"})
	return jsonutils.Marshal(map[string]string{"elasticcache_id": elasticcacheId})
}

func (manager *SElasticcacheAclManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (manager *SElasticcacheAclManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return elasticcacheSubResourceFetchOwnerId(ctx, data)
}

func (manager *SElasticcacheAclManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return elasticcacheSubResourceFetchOwner(q, userCred, scope)
}

func (manager *SElasticcacheAclManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	cacheId, _ := values.GetString("elasticcache_id")
	if len(cacheId) > 0 {
		q = q.Equals("elasticcache_id", cacheId)
	}
	return q
}

func (manager *SElasticcacheAclManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (manager *SElasticcacheAclManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var region *SCloudregion
	if id, _ := data.GetString("elasticcache"); len(id) > 0 {
		ec, err := db.FetchByIdOrName(ElasticcacheManager, userCred, id)
		if err != nil {
			return nil, fmt.Errorf("getting elastic cache instance failed")
		}
		region = ec.(*SElasticcache).GetRegion()

		if region == nil {
			return nil, fmt.Errorf("getting elastic cache region failed")
		}
	} else {
		return nil, httperrors.NewMissingParameterError("elasticcache")
	}

	input := apis.StandaloneResourceCreateInput{}
	var err error
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	return region.GetDriver().ValidateCreateElasticcacheAclData(ctx, userCred, ownerId, data)
}

func (self *SElasticcacheAcl) GetOwnerId() mcclient.IIdentityProvider {
	return ElasticcacheManager.GetOwnerIdByElasticcacheId(self.ElasticcacheId)
}

func (self *SElasticcacheAcl) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.SetStatus(userCred, api.ELASTIC_CACHE_ACL_STATUS_CREATING, "")
	if err := self.StartElasticcacheAclCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("Failed to create elastic cache acl error: %v", err)
	}
}

func (self *SElasticcacheAcl) StartElasticcacheAclCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticcacheAclCreateTask", self, userCred, jsonutils.NewDict(), parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticcacheAcl) GetRegion() *SCloudregion {
	ieb, err := db.FetchById(ElasticcacheManager, self.ElasticcacheId)
	if err != nil {
		return nil
	}

	return ieb.(*SElasticcache).GetRegion()
}

func (self *SElasticcacheAcl) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	// todo: fix me self.IsOwner(userCred) ||
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SElasticcacheAcl) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ips, err := data.GetString("ip_list")
	if err != nil || ips == "" {
		return nil, httperrors.NewMissingParameterError("ip_list")
	}

	ipV := validators.NewIPv4AddrValidator("ip")
	cidrV := validators.NewIPv4PrefixValidator("ip")
	_ips := strings.Split(ips, ",")
	for _, ip := range _ips {
		params := jsonutils.NewDict()
		params.Set("ip", jsonutils.NewString(ip))
		if strings.Contains(ip, "/") {
			if err := cidrV.Validate(params); err != nil {
				return nil, err
			}
		} else {
			if err := ipV.Validate(params); err != nil {
				return nil, err
			}
		}
	}

	return data, nil
}

func (self *SElasticcacheAcl) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SetStatus(userCred, api.ELASTIC_CACHE_ACL_STATUS_UPDATING, "")
	if err := self.StartUpdateElasticcacheAclTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("ElasticcacheAcl %s", err.Error())
	}

	return
}

func (self *SElasticcacheAcl) StartUpdateElasticcacheAclTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticcacheAclUpdateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticcacheAcl) ValidateDeleteCondition(ctx context.Context) error {
	return nil
}

func (self *SElasticcacheAcl) ValidatePurgeCondition(ctx context.Context) error {
	return nil
}

func (self *SElasticcacheAcl) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	self.SetStatus(userCred, api.ELASTIC_CACHE_ACL_STATUS_DELETING, "")
	return self.StartDeleteElasticcacheAclTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SElasticcacheAcl) StartDeleteElasticcacheAclTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticcacheAclDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticcacheAcl) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

// 弹性缓存ACL规则列表
func (manager *SElasticcacheAclManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ElasticcacheAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SElasticcacheResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemFilter")
	}

	if len(input.IpList) > 0 {
		q = q.Contains("ip_list", input.IpList)
	}

	return q, nil
}

func (manager *SElasticcacheAclManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ElasticcacheAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SElasticcacheResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SElasticcacheAclManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SElasticcacheResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SElasticcacheAcl) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ElasticcacheAclDetails, error) {
	return api.ElasticcacheAclDetails{}, nil
}

func (manager *SElasticcacheAclManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticcacheAclDetails {
	rows := make([]api.ElasticcacheAclDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	elasticRows := manager.SElasticcacheResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	cacheIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.ElasticcacheAclDetails{
			StandaloneResourceDetails: stdRows[i],
			ElasticcacheResourceInfo:  elasticRows[i],
		}
		acl := objs[i].(*SElasticcacheAcl)
		cacheIds[i] = acl.ElasticcacheId
	}

	caches := make(map[string]SElasticcache)
	err := db.FetchStandaloneObjectsByIds(ElasticcacheManager, cacheIds, &caches)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail: %v", err)
		return rows
	}

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if cache, ok := caches[cacheIds[i]]; ok {
			virObjs[i] = &cache
			rows[i].ProjectId = cache.ProjectId
		}
	}

	projRows := ElasticcacheManager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, virObjs, fields, isList)
	for i := range rows {
		rows[i].ProjectizedResourceInfo = projRows[i]
	}

	return rows
}

func (manager *SElasticcacheAclManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SElasticcacheResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SElasticcacheResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}
