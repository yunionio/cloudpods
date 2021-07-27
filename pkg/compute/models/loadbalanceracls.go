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
	"crypto/md5"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerAclEntry struct {
	Cidr    string
	Comment string
}

func (aclEntry *SLoadbalancerAclEntry) Validate(data *jsonutils.JSONDict) error {
	if strings.Index(aclEntry.Cidr, "/") > 0 {
		_, ipNet, err := net.ParseCIDR(aclEntry.Cidr)
		if err != nil {
			return err
		}
		// normalize from 192.168.1.3/24 to 192.168.1.0/24
		aclEntry.Cidr = ipNet.String()
	} else {
		ip := net.ParseIP(aclEntry.Cidr).To4()
		if ip == nil {
			return httperrors.NewInputParameterError("invalid addr %s", aclEntry.Cidr)
		}
	}
	if commentLimit := 128; len(aclEntry.Comment) > commentLimit {
		return httperrors.NewInputParameterError("comment too long (%d>=%d)",
			len(aclEntry.Comment), commentLimit)
	}
	for _, r := range aclEntry.Comment {
		if !unicode.IsPrint(r) {
			return httperrors.NewInputParameterError("comment contains non-printable char: %v", r)
		}
	}
	return nil
}

type SLoadbalancerAclEntries []*SLoadbalancerAclEntry

func (aclEntries *SLoadbalancerAclEntries) String() string {
	return jsonutils.Marshal(aclEntries).String()
}

func (aclEntries *SLoadbalancerAclEntries) IsZero() bool {
	if len([]*SLoadbalancerAclEntry(*aclEntries)) == 0 {
		return true
	}
	return false
}

func (aclEntries *SLoadbalancerAclEntries) Validate(data *jsonutils.JSONDict) error {
	found := map[string]bool{}
	for _, aclEntry := range *aclEntries {
		if err := aclEntry.Validate(data); err != nil {
			return err
		}
		if _, ok := found[aclEntry.Cidr]; ok {
			// error so that the user has a chance to deal with comments
			return httperrors.NewInputParameterError("acl cidr duplicate %s", aclEntry.Cidr)
		}
		found[aclEntry.Cidr] = true
	}
	return nil
}

func (aclEntries *SLoadbalancerAclEntries) Fingerprint() string {
	cidrs := []string{}
	for _, acl := range *aclEntries {
		cidrs = append(cidrs, acl.Cidr)
	}

	sort.Strings(cidrs)

	s := strings.Join(cidrs, "")
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

type SLoadbalancerAclManager struct {
	SLoadbalancerLogSkipper

	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager

	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var LoadbalancerAclManager *SLoadbalancerAclManager

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SLoadbalancerAclEntries{}), func() gotypes.ISerializable {
		return &SLoadbalancerAclEntries{}
	})
	LoadbalancerAclManager = &SLoadbalancerAclManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SLoadbalancerAcl{},
			"loadbalanceracls_tbl",
			"loadbalanceracl",
			"loadbalanceracls",
		),
	}
	LoadbalancerAclManager.SetVirtualObject(LoadbalancerAclManager)
}

type SLoadbalancerAcl struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	AclEntries  *SLoadbalancerAclEntries `list:"user" update:"user" create:"required"`
	Fingerprint string                   `name:"fingerprint" width:"64" charset:"ascii" nullable:"false" index:"true" list:"user" update:"user" create:"required"`
}

func loadbalancerAclsValidateAclEntries(data *jsonutils.JSONDict, update bool) (*jsonutils.JSONDict, error) {
	aclEntries := SLoadbalancerAclEntries{}
	aclEntriesV := validators.NewStructValidator("acl_entries", &aclEntries)
	if update {
		aclEntriesV.Optional(true)
	}
	err := aclEntriesV.Validate(data)
	if err != nil {
		return nil, err
	}

	data.Set("fingerprint", jsonutils.NewString(aclEntries.Fingerprint()))
	return data, nil
}

func (man *SLoadbalancerAclManager) FetchByFingerPrint(projectId string, fingerprint string) (*SLoadbalancerAcl, error) {
	ret := &SLoadbalancerAcl{}
	q := man.Query().IsFalse("pending_deleted")
	q = q.Equals("tenant_id", projectId).Equals("fingerprint", fingerprint).Asc("created_at").Limit(1)
	err := q.First(ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (man *SLoadbalancerAclManager) CountByFingerPrint(projectId string, fingerprint string) int {
	q := man.Query().IsFalse("pending_deleted")
	return q.Equals("tenant_id", projectId).Equals("fingerprint", fingerprint).Asc("created_at").Count()
}

func (man *SLoadbalancerAclManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := loadbalancerAclsValidateAclEntries(data, false)
	if err != nil {
		return nil, err
	}

	input := apis.SharableVirtualResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal SharableVirtualResourceCreateInput fail %s", err)
	}
	input, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	managerIdV := validators.NewModelIdOrNameValidator("manager", "cloudprovider", ownerId)
	managerIdV.Optional(true)
	if err := managerIdV.Validate(data); err != nil {
		return nil, err
	}

	regionV := validators.NewModelIdOrNameValidator("cloudregion", "cloudregion", ownerId)
	regionV.Default("default")
	if err := regionV.Validate(data); err != nil {
		return nil, err
	}
	region := regionV.Model.(*SCloudregion)
	return region.GetDriver().ValidateCreateLoadbalancerAclData(ctx, userCred, data)
}

func (lbacl *SLoadbalancerAcl) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if !data.Contains("public_scope") {
		if db.IsAdminAllowPerform(userCred, lbacl, "public") && ownerId.GetProjectDomainId() == userCred.GetProjectDomainId() {
			lbacl.SetShare(rbacutils.ScopeSystem)
		} else if db.IsDomainAllowPerform(userCred, lbacl, "public") && ownerId.GetProjectId() == userCred.GetProjectId() && consts.GetNonDefaultDomainProjects() {
			// only if non_default_domain_projects turned on, share to domain
			lbacl.SetShare(rbacutils.ScopeDomain)
		} else {
			lbacl.SetShare(rbacutils.ScopeNone)
		}

		data.(*jsonutils.JSONDict).Set("public_scope", jsonutils.NewString(lbacl.PublicScope))
	}

	return lbacl.SSharableVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (lbacl *SLoadbalancerAcl) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbacl *SLoadbalancerAcl) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := loadbalancerAclsValidateAclEntries(data, true)
	if err != nil {
		return nil, err
	}
	input := apis.SharableVirtualResourceBaseUpdateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = lbacl.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (lbacl *SLoadbalancerAcl) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SSharableVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	acls, err := lbacl.GetCachedAcls()
	if err != nil {
		log.Errorf("SLoadbalancerAcl PostUpdate %s", err)
	}

	for i := range acls {
		acl := acls[i]
		acl.SetModelManager(CachedLoadbalancerAclManager, &acl)
		err = acl.StartLoadBalancerAclSyncTask(ctx, userCred, "")
		if err != nil {
			log.Errorf("SLoadbalancerAcl PostUpdate %s", err)
		}
	}
}

func (lbacl *SLoadbalancerAcl) StartLoadBalancerAclSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclSyncTask", lbacl, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbacl *SLoadbalancerAcl) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbacl.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	lbacl.SetStatus(userCred, api.LB_STATUS_ENABLED, "")
}

func (manager *SLoadbalancerAclManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerAclDetails {
	rows := make([]api.LoadbalancerAclDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.LoadbalancerAclDetails{
			SharableVirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:            managerRows[i],
			CloudregionResourceInfo:        regionRows[i],
		}
	}

	for i := range objs {
		q := LoadbalancerListenerManager.Query().IsFalse("pending_deleted").Equals("acl_id", objs[i].(*SLoadbalancerAcl).GetId())
		ownerId, queryScope, err := db.FetchCheckQueryOwnerScope(ctx, userCred, query, LoadbalancerListenerManager, policy.PolicyActionList, true)
		if err != nil {
			log.Errorf("FetchCheckQueryOwnerScope error: %v", err)
			return rows
		}

		q = LoadbalancerListenerManager.FilterByOwner(q, ownerId, queryScope)
		count, err := q.CountWithError()
		if err != nil {
			log.Errorf("db.CountWithError error: %v", err)
		}

		rows[i].LbListenerCount = count
	}

	return rows
}

func (lbacl *SLoadbalancerAcl) AllowPerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return lbacl.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lbacl, "patch")
}

// PerformPatch patches acl entries by adding then deleting the specified acls.
// This is intended mainly for command line operations.
func (lbacl *SLoadbalancerAcl) PerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	aclEntries := gotypes.DeepCopy(*lbacl.AclEntries).(SLoadbalancerAclEntries)
	{
		adds := SLoadbalancerAclEntries{}
		addsV := validators.NewStructValidator("adds", &adds)
		addsV.Optional(true)
		err := addsV.Validate(data)
		if err != nil {
			return nil, err
		}
		for _, add := range adds {
			found := false
			for _, aclEntry := range aclEntries {
				if aclEntry.Cidr == add.Cidr {
					found = true
					aclEntry.Comment = add.Comment
					break
				}
			}
			if !found {
				aclEntries = append(aclEntries, add)
			}
		}
	}
	{
		dels := SLoadbalancerAclEntries{}
		delsV := validators.NewStructValidator("dels", &dels)
		delsV.Optional(true)
		err := delsV.Validate(data)
		if err != nil {
			return nil, err
		}
		for _, del := range dels {
			for i := len(aclEntries) - 1; i >= 0; i-- {
				aclEntry := aclEntries[i]
				if aclEntry.Cidr == del.Cidr {
					aclEntries = append(aclEntries[:i], aclEntries[i+1:]...)
					break
				}
			}
		}
	}
	diff, err := db.Update(lbacl, func() error {
		// todo: sync diff to clouds
		lbacl.AclEntries = &aclEntries
		return nil
	})
	if err != nil {
		return nil, err
	}
	db.OpsLog.LogEvent(lbacl, db.ACT_UPDATE, diff, userCred)
	return nil, nil
}

func (lbacl *SLoadbalancerAcl) ValidateDeleteCondition(ctx context.Context) error {
	men := []db.IModelManager{
		LoadbalancerListenerManager,
	}

	lbaclId := lbacl.Id
	for _, man := range men {
		t := man.TableSpec().Instance()
		pdF := t.Field("pending_deleted")
		n, err := t.Query().
			Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
			Equals("acl_id", lbaclId).
			CountWithError()
		if err != nil {
			return httperrors.NewInternalServerError("get acl count fail %s", err)
		}
		if n > 0 {
			return httperrors.NewResourceBusyError("acl %s is still referred to by %d %s", lbaclId, n, man.KeywordPlural())
		}
	}

	return nil
}

func (lbacl *SLoadbalancerAcl) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbacl, "purge")
}

func (lbacl *SLoadbalancerAcl) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, lbacl.Delete(ctx, userCred)
}

func (lbacl *SLoadbalancerAcl) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !lbacl.PendingDeleted {
		caches, err := lbacl.GetCachedAcls()
		if err != nil {
			return errors.Wrap(err, "GetCachedAcls")
		}

		for i := range caches {
			err := caches[i].MarkPendingDelete(userCred)
			if err != nil {
				return errors.Wrap(err, "MarkPendingDelete")
			}
		}

		return lbacl.DoPendingDelete(ctx, userCred)
	}

	return nil
}

func (lbacl *SLoadbalancerAcl) StartLoadBalancerAclDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerAclDeleteTask", lbacl, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbacl *SLoadbalancerAcl) GetCachedAcls() ([]SCachedLoadbalancerAcl, error) {
	ret := []SCachedLoadbalancerAcl{}
	q := CachedLoadbalancerAclManager.Query().Equals("acl_id", lbacl.Id)
	err := db.FetchModelObjects(CachedLoadbalancerAclManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (man *SLoadbalancerAclManager) SyncLoadbalancerAcls(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, acls []cloudprovider.ICloudLoadbalancerAcl, syncRange *SSyncRange) compare.SyncResult {
	// todo: implement me
	return compare.SyncResult{}
}

func (manager *SLoadbalancerAclManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (manager *SLoadbalancerAclManager) InitializeData() error {
	// sync acl to  acl cache
	acls := []SLoadbalancerAcl{}
	cachedAcls := CachedLoadbalancerAclManager.Query("acl_id").SubQuery()
	q := manager.Query().IsNotEmpty("external_id").IsNotEmpty("cloudregion_id").NotIn("id", cachedAcls)
	if err := q.All(&acls); err != nil {
		return err
	}

	for i := range acls {
		acl := acls[i]
		aclObj := jsonutils.Marshal(acl)
		cachedAcl := &SCachedLoadbalancerAcl{}
		err := aclObj.Unmarshal(cachedAcl)
		if err != nil {
			return err
		}
		cachedAcl.Id = ""
		cachedAcl.AclId = acl.Id
		err = CachedLoadbalancerAclManager.TableSpec().Insert(context.TODO(), cachedAcl)
		if err != nil {
			return err
		}
	}

	return nil
}

// 负载均衡ACL规则列表
func (manager *SLoadbalancerAclManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LoadbalancerAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	if len(input.Fingerprint) > 0 {
		q = q.In("fingerprint", input.Fingerprint)
	}

	return q, nil
}

func (manager *SLoadbalancerAclManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LoadbalancerAclListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SLoadbalancerAclManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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

func (manager *SLoadbalancerAclManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
