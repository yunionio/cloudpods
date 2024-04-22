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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

var _ db.IModelManager = (*SOrganizationManager)(nil)
var _ db.IModel = (*SOrganization)(nil)

type SOrganizationManager struct {
	SEnabledIdentityBaseResourceManager
	db.SSharableBaseResourceManager
	db.SStatusResourceBaseManager

	cache *db.SCacheManager[SOrganization]
}

var OrganizationManager *SOrganizationManager

func init() {
	OrganizationManager = &SOrganizationManager{
		SEnabledIdentityBaseResourceManager: NewEnabledIdentityBaseResourceManager(
			SOrganization{},
			"organizations_tbl",
			"organization",
			"organizations",
		),
	}
	OrganizationManager.SetVirtualObject(OrganizationManager)
	OrganizationManager.cache = db.NewCacheManager[SOrganization](OrganizationManager)
}

type SOrganization struct {
	SEnabledIdentityBaseResource
	db.SSharableBaseResource
	db.SStatusResourceBase

	Type api.TOrgType `width:"32" charset:"ascii" list:"user" create:"admin_required"`

	Keys string `width:"256" charset:"utf8" list:"user" create:"admin_optional"`

	Level int `list:"user" create:"admin_optional"`
}

func (manager *SOrganizationManager) fetchOrganizationById(orgId string) (*SOrganization, error) {
	org, err := manager.cache.FetchById(orgId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			obj, err := manager.FetchById(orgId)
			if err != nil {
				return nil, errors.Wrap(err, "manager.FetchById")
			}
			manager.cache.Invalidate()
			org = obj.(*SOrganization)
		} else {
			return nil, errors.Wrapf(err, "cache.FetchById %s", orgId)
		}
	}
	return org, nil
}

// 组织列表
func (manager *SOrganizationManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.OrganizationListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query.EnabledIdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.ListItemFilter")
	}
	q, err = manager.SSharableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SharableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	q, err = manager.SStatusResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}

	if len(query.Type) > 0 {
		q = q.In("type", query.Type)
	}

	if len(query.Key) > 0 {
		q = q.Contains("keys", query.Key)
	}

	return q, nil
}

func (manager *SOrganizationManager) ExtendListQuery(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.OrganizationListInput,
) (*sqlchemy.SQuery, error) {
	return q, nil
}

func (manager *SOrganizationManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.OrganizationListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledIdentityBaseResourceManager.OrderByExtraFields(ctx, q, userCred, query.EnabledIdentityBaseResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.OrderByExtraFields")
	}
	q, err = manager.SStatusResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SOrganizationManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledIdentityBaseResourceManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SOrganizationManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SOrganizationDetails {
	rows := make([]api.SOrganizationDetails, len(objs))
	infRows := manager.SEnabledIdentityBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	sharedRows := manager.SSharableBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		// org := objs[i].(*SOrganization)
		rows[i] = api.SOrganizationDetails{
			EnabledIdentityBaseResourceDetails: infRows[i],
			SharableResourceBaseInfo:           sharedRows[i],
		}
	}
	return rows
}

func (manager *SOrganizationManager) FetchOrgnaizations(filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) ([]SOrganization, error) {
	q := manager.Query()
	q = filter(q)
	ret := make([]SOrganization, 0)
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return ret, nil
}

func (org *SOrganization) getNodesQuery() *sqlchemy.SQuery {
	return OrganizationNodeManager.Query().Equals("org_id", org.Id)
}

func (org *SOrganization) getNodesCount() (int, error) {
	q := org.getNodesQuery()
	return q.CountWithError()
}

func (org *SOrganization) getNodes() ([]SOrganizationNode, error) {
	q := org.getNodesQuery()
	q = q.Asc("level")
	q = q.Asc("weight")
	q = q.Asc("full_label")
	nodes := make([]SOrganizationNode, 0)
	err := db.FetchModelObjects(OrganizationNodeManager, q, &nodes)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return nodes, nil
}

func (org *SOrganization) getNode(fullLabel string) (*SOrganizationNode, error) {
	q := org.getNodesQuery()
	q = q.Equals("full_label", fullLabel)

	node := &SOrganizationNode{}
	err := q.First(node)
	if err != nil {
		return nil, errors.Wrap(err, "First")
	}
	node.SetModelManager(OrganizationNodeManager, node)
	return node, nil
}

func (org *SOrganization) ValidateDeleteCondition(ctx context.Context, info *api.ProjectDetails) error {
	if org.GetEnabled() {
		return errors.Wrap(httperrors.ErrInvalidStatus, "organization enabled")
	}
	return org.SEnabledIdentityBaseResource.ValidateDeleteCondition(ctx, nil)
}

func (org *SOrganization) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := org.removeAll(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "removeAll")
	}
	OrganizationManager.cache.Delete(org)
	// pending delete
	err = org.SEnabledIdentityBaseResource.Delete(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "SEnabledIdentityBaseResource.Delete")
	}
	return nil
}

func (manager *SOrganizationManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.OrganizationCreateInput,
) (api.OrganizationCreateInput, error) {
	var err error

	input.EnabledIdentityBaseResourceCreateInput, err = manager.SEnabledIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledIdentityBaseResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledIdentityBaseResourceManager.ValidateCreateData")
	}
	input.SharableResourceBaseCreateInput, err = db.SharableManagerValidateCreateData(manager, ctx, userCred, ownerId, query, input.SharableResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SharableManagerValidateCreateData")
	}

	if !api.IsValidOrgType(input.Type) {
		return input, errors.Wrapf(httperrors.ErrInputParameter, "invalid organization type %s", input.Type)
	}

	// allow empty key
	// if len(input.Key) == 0 {
	//	return input, errors.Wrap(httperrors.ErrInputParameter, "empty key")
	// }

	input.Level = len(input.Key)
	// keys should be uniq
	keyMap := make(map[string]struct{})
	for _, k := range input.Key {
		if _, ok := keyMap[k]; ok {
			return input, errors.Wrapf(httperrors.ErrInputParameter, "duplicate key %s", k)
		} else {
			keyMap[k] = struct{}{}
		}
	}
	input.Keys = api.JoinLabels(input.Key...)

	return input, nil
}

func (org *SOrganization) CustomizeCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	org.SetShare(rbacscope.ScopeSystem)
	org.Enabled = tristate.False
	org.Status = api.OrganizationStatusReady
	return org.SEnabledIdentityBaseResource.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (org *SOrganization) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	org.SEnabledIdentityBaseResource.PostCreate(ctx, userCred, ownerId, query, data)
	OrganizationManager.cache.Update(org)
}

func (org *SOrganization) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationUpdateInput,
) (api.OrganizationUpdateInput, error) {
	return input, nil
}

func (org *SOrganization) PostUpdate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	org.SEnabledIdentityBaseResource.PostUpdate(ctx, userCred, query, data)
	OrganizationManager.cache.Update(org)
}

func (org *SOrganization) PerformAddLevel(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationPerformAddLevelsInput,
) (jsonutils.JSONObject, error) {
	keys := api.SplitLabel(org.Keys)

	for _, nk := range input.Key {
		if len(nk) == 0 {
			return nil, errors.Wrap(httperrors.ErrInputParameter, "empty key")
		}
		if utils.IsInArray(nk, keys) {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "key %s duplicated", nk)
		} else {
			keys = append(keys, nk)
		}
	}

	if len(input.Tags) > 0 {
		if len(input.Tags) != len(keys) {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "inconsist level of key %d and tags %d", len(keys), len(input.Tags))
		}
		for _, k := range keys {
			if _, ok := input.Tags[k]; !ok {
				return nil, errors.Wrapf(httperrors.ErrInputParameter, "missing value for key %s", k)
			}
		}
	}

	_, err := db.Update(org, func() error {
		org.Keys = api.JoinLabels(keys...)
		org.Level = len(keys)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "Update")
	}

	OrganizationManager.cache.Update(org)

	db.OpsLog.LogEvent(org, db.ACT_UPDATE, org.GetShortDesc(ctx), userCred)
	logclient.AddSimpleActionLog(org, logclient.ACT_UPDATE, org.GetShortDesc(ctx), userCred, true)

	if len(input.Tags) > 0 {
		_, err := org.PerformAddNode(ctx, userCred, query, input.OrganizationPerformAddNodeInput)
		if err != nil {
			return nil, errors.Wrap(err, "AddNode")
		}
	}

	return nil, nil
}

func (org *SOrganization) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := org.SEnabledIdentityBaseResource.GetShortDesc(ctx)
	desc.Set("status", jsonutils.NewString(org.Status))
	desc.Set("keys", jsonutils.NewString(org.Keys))
	desc.Set("level", jsonutils.NewInt(int64(org.Level)))
	return desc
}

func (org *SOrganization) GetKeys() []string {
	return api.SplitLabel(org.Keys)
}

func (org *SOrganization) PerformAddNode(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationPerformAddNodeInput,
) (jsonutils.JSONObject, error) {
	labels := make([]string, 0)

	for _, k := range org.GetKeys() {
		if label, ok := input.Tags[k]; !ok {
			break
		} else {
			labels = append(labels, label)
		}
	}

	for i := 0; i < len(labels); i++ {
		var weight *int
		var desc string
		if i == len(labels)-1 {
			weight = &input.Weight
			desc = input.Description
		}
		_, err := OrganizationNodeManager.ensureNode(ctx, org.Id, api.JoinLabels(labels[0:i+1]...), weight, desc)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to insert node %s", api.JoinLabels(labels[i:i]...))
		}
	}

	return nil, nil
}

func (org *SOrganization) PerformSync(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationPerformSyncInput,
) (jsonutils.JSONObject, error) {
	if input.Reset != nil && *input.Reset {
		err := org.removeAll(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "removeAll")
		}
	}
	err := org.startOrganizationSyncTask(ctx, userCred, input.ResourceType)
	if err != nil {
		return nil, errors.Wrap(err, "startOrganizationSyncTask")
	}
	return nil, nil
}

func (org *SOrganization) removeAll(ctx context.Context, userCred mcclient.TokenCredential) error {
	nodes, err := org.getNodes()
	if err != nil {
		return errors.Wrap(err, "getNodes")
	}
	for i := range nodes {
		err := nodes[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Delete %s", nodes[i].FullLabel)
		}
	}
	return nil
}

func (org *SOrganization) SetStatus(ctx context.Context, userCred mcclient.TokenCredential, status string, reason string) error {
	return db.StatusBaseSetStatus(ctx, org, userCred, status, reason)
}

func (org *SOrganization) startOrganizationSyncTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	resourceType string,
) error {
	org.SetStatus(ctx, userCred, api.OrganizationStatusSync, "start sync task")
	params := jsonutils.NewDict()
	if len(resourceType) > 0 {
		params.Add(jsonutils.NewString(resourceType), "resource_type")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "OrganizationSyncTask", org, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (org *SOrganization) SyncTags(ctx context.Context, userCred mcclient.TokenCredential, resourceType string) error {
	switch org.Type {
	case api.OrgTypeProject:
		return org.syncProjectTags(ctx, userCred)
	case api.OrgTypeDomain:
		return org.syncDomainTags(ctx, userCred)
	case api.OrgTypeObject:
		return org.syncObjectTags(ctx, userCred, resourceType)
	}
	return nil
}

func (org *SOrganization) syncProjectTags(ctx context.Context, userCred mcclient.TokenCredential) error {
	return org.syncIModelManagerTags(ctx, userCred, ProjectManager)
}

func (org *SOrganization) syncIModelManagerTags(ctx context.Context, userCred mcclient.TokenCredential, manager db.IStandaloneModelManager) error {
	query := jsonutils.NewDict()
	query.Set("scope", jsonutils.NewString("system"))
	keys := api.SplitLabel(org.Keys)
	userKeys := make([]string, len(keys))
	for i := range keys {
		userKeys[i] = fmt.Sprintf("%s%s", db.USER_TAG_PREFIX, keys[i])
	}
	orgKeys := make([]string, len(keys))
	for i := range keys {
		orgKeys[i] = fmt.Sprintf("%s%s", db.ORGANIZATION_TAG_PREFIX, keys[i])
	}
	{
		tagValMaps, err := db.GetTagValueCountMap(manager, manager.Keyword(), "id", "", userKeys, ctx, userCred, query)
		if err != nil {
			return errors.Wrap(err, "GetTagValueCountMap")
		}
		for i := range tagValMaps {
			labels, err := org.syncTagValueMap(ctx, tagValMaps[i])
			if err != nil {
				return errors.Wrap(err, "syncTagValueMap")
			}
			err = db.CopyTags(ctx, manager.Keyword(), userKeys, labels, orgKeys)
			if err != nil {
				return errors.Wrap(err, "copyTags")
			}
		}
	}
	{
		tagValMaps, err := db.GetTagValueCountMap(manager, manager.Keyword(), "id", "", orgKeys, ctx, userCred, query)
		if err != nil {
			return errors.Wrap(err, "GetTagValueCountMap")
		}
		for i := range tagValMaps {
			_, err := org.syncTagValueMap(ctx, tagValMaps[i])
			if err != nil {
				return errors.Wrap(err, "syncTagValueMap")
			}
		}
	}
	return nil
}

func (org *SOrganization) syncDomainTags(ctx context.Context, userCred mcclient.TokenCredential) error {
	return org.syncIModelManagerTags(ctx, userCred, DomainManager)
}

func (org *SOrganization) syncObjectTags(ctx context.Context, userCred mcclient.TokenCredential, resourceType string) error {
	// XXX TODO QJ
	return errors.Wrap(httperrors.ErrNotImplemented, "not ready yet")
}

func (org *SOrganization) syncTagValueMap(ctx context.Context, tagVal map[string]string) ([]string, error) {
	labels := make([]string, 0)
	for i := 0; i < org.Level; i++ {
		key := db.TagValueKey(i)
		if val, ok := tagVal[key]; ok && val != tagutils.NoValue {
			labels = append(labels, val)
			log.Debugf("%d %s %s %s", i, key, val, strings.Join(labels, "/"))
			_, err := OrganizationNodeManager.ensureNode(ctx, org.Id, api.JoinLabels(labels...), nil, "")
			if err != nil {
				return nil, errors.Wrapf(err, "fail to insert node %s", api.JoinLabels(labels...))
			}
		} else {
			break
		}
	}
	return labels, nil
}

func (org *SOrganization) PerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformEnableInput,
) (jsonutils.JSONObject, error) {
	if !org.GetEnabled() {
		_, err := org.SEnabledIdentityBaseResource.PerformEnable(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformEnable")
		}
		OrganizationManager.cache.Update(org)
		// disable other org of the same type
		otherOrgs, err := OrganizationManager.FetchOrgnaizations(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			q = q.Equals("type", org.Type)
			q = q.NotEquals("id", org.Id)
			q = q.IsTrue("enabled")
			return q
		})
		if err != nil {
			return nil, errors.Wrap(err, "FetchOrgnaizations")
		}
		for i := range otherOrgs {
			_, err := otherOrgs[i].PerformDisable(ctx, userCred, query, apis.PerformDisableInput{})
			if err != nil {
				return nil, errors.Wrap(err, "PerformDisable")
			}
		}
	}
	return nil, nil
}

func (org *SOrganization) PerformDisable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformDisableInput,
) (jsonutils.JSONObject, error) {
	if org.GetEnabled() {
		_, err := org.SEnabledIdentityBaseResource.PerformDisable(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBase.PerformDisable")
		}
		OrganizationManager.cache.Update(org)
	}
	return nil, nil
}

func (org *SOrganization) getProjectOrganization(tags map[string]string) (*api.SProjectOrganization, error) {
	keys := api.SplitLabel(org.Keys)
	ret := api.SProjectOrganization{
		Id:    org.Id,
		Name:  org.Name,
		Keys:  keys,
		Nodes: make([]api.SProjectOrganizationNode, 0, len(keys)),
	}
	labels := make([]string, 0, len(keys))
	for _, k := range keys {
		if val, ok := tags[k]; ok {
			labels = append(labels, val)
			fullLabel := api.JoinLabels(labels...)
			node, err := org.getNode(fullLabel)
			if err != nil {
				return nil, errors.Wrapf(err, "getNode %s", fullLabel)
			}
			ret.Nodes = append(ret.Nodes, api.SProjectOrganizationNode{
				Id:     node.Id,
				Labels: api.SplitLabel(fullLabel),
			})
		} else {
			break
		}
	}
	return &ret, nil
}

func (org *SOrganization) PerformClean(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationPerformCleanInput,
) (jsonutils.JSONObject, error) {
	err := org.removeAll(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "removeAll")
	}
	return nil, nil
}
