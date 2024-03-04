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
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

var _ db.IModelManager = (*SOrganizationNodeManager)(nil)
var _ db.IModel = (*SOrganizationNode)(nil)

type SOrganizationNodeManager struct {
	db.SStandaloneResourceBaseManager
	db.SPendingDeletedBaseManager
}

var OrganizationNodeManager *SOrganizationNodeManager

func init() {
	OrganizationNodeManager = &SOrganizationNodeManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SOrganizationNode{},
			"organization_nodes_tbl",
			"organization_node",
			"organization_nodes",
		),
	}
	OrganizationNodeManager.SetVirtualObject(OrganizationNodeManager)
	OrganizationNodeManager.TableSpec().AddIndex(true, "deleted", "org_id", "full_label", "level")
}

type SOrganizationNode struct {
	db.SStandaloneResourceBase `name:""`
	db.SPendingDeletedBase

	OrgId string `width:"36" charset:"ascii" list:"user" create:"admin_required"`

	FullLabel string `width:"180" charset:"utf8" list:"user" create:"admin_required"`

	Level int `list:"user" create:"admin_required"`

	Weight int `list:"user" create:"admin_required" update:"admin" default:"1"`
}

func (orgNode *SOrganizationNode) GetOrganization() (*SOrganization, error) {
	return OrganizationManager.fetchOrganizationById(orgNode.OrgId)
}

func (orgNode *SOrganizationNode) GetChildCount() (int, error) {
	q := OrganizationNodeManager.Query().Startswith("full_label", orgNode.FullLabel+api.OrganizationLabelSeparator)
	cnt, err := q.CountWithError()
	if err != nil {
		return 0, errors.Wrap(err, "CountWithError")
	}
	return cnt, nil
}

func (orgNode *SOrganizationNode) GetDirectChildCount() (int, error) {
	q := OrganizationNodeManager.Query().Startswith("full_label", orgNode.FullLabel+api.OrganizationLabelSeparator)
	q = q.Equals("level", orgNode.Level+1)
	cnt, err := q.CountWithError()
	if err != nil {
		return 0, errors.Wrap(err, "CountWithError")
	}
	return cnt, nil
}

func generateId(orgId string, fullLabel string, level int) string {
	h := sha256.New()
	h.Write([]byte(orgId))
	h.Write([]byte(fullLabel))
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(level))
	h.Write(bs)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (manager *SOrganizationNodeManager) ensureNode(ctx context.Context, orgId string, fullLabel string, weight *int, desc string) (*SOrganizationNode, error) {
	labels := api.SplitLabel(fullLabel)
	level := len(labels)
	label := labels[level-1]
	id := generateId(orgId, fullLabel, level)
	obj, err := manager.FetchById(id)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return nil, errors.Wrap(err, "FetchById")
		}
		// not exist
		if weight == nil {
			one := 1
			weight = &one
		}
	} else {
		// exist
		if weight == nil {
			weight = &obj.(*SOrganizationNode).Weight
		}
		if len(desc) == 0 {
			desc = obj.(*SOrganizationNode).Description
		}
	}
	node := &SOrganizationNode{
		OrgId:     orgId,
		FullLabel: fullLabel,
		Level:     level,
		Weight:    *weight,
	}
	node.Description = desc
	node.Name = label
	node.Id = id

	err = manager.TableSpec().InsertOrUpdate(ctx, node)
	if err != nil {
		return nil, errors.Wrap(err, "InsertOrUpdate")
	}

	return node, nil
}

func (orgNode *SOrganizationNode) PerformBind(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationNodePerformBindInput,
) (jsonutils.JSONObject, error) {
	orgNode.startOrganizationBindTask(ctx, userCred, input, true)
	return nil, nil
}

func (orgNode *SOrganizationNode) PerformUnbind(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationNodePerformBindInput,
) (jsonutils.JSONObject, error) {
	orgNode.startOrganizationBindTask(ctx, userCred, input, false)
	return nil, nil
}

func (orgNode *SOrganizationNode) startOrganizationBindTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.OrganizationNodePerformBindInput,
	bind bool,
) error {
	params := jsonutils.NewDict()
	params.Set("input", jsonutils.Marshal(input))
	params.Set("bind", jsonutils.NewBool(bind))
	task, err := taskman.TaskManager.NewTask(ctx, "OrganizationBindTask", orgNode, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (orgNode *SOrganizationNode) BindTargetIds(ctx context.Context, userCred mcclient.TokenCredential, input api.OrganizationNodePerformBindInput, bind bool) error {
	org, err := orgNode.GetOrganization()
	if err != nil {
		return errors.Wrap(err, "GetOrganization")
	}
	tags := orgNode.GetTags(org)
	if !bind {
		for k := range tags {
			tags[k] = "None"
		}
	}
	var errs []error
	for _, id := range input.TargetId {
		err := orgNode.bindTargetId(ctx, userCred, input.ResourceType, id, tags)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func (orgNode *SOrganizationNode) bindTargetId(ctx context.Context, userCred mcclient.TokenCredential, resType string, idstr string, tags map[string]string) error {
	org, err := OrganizationManager.fetchOrganizationById(orgNode.OrgId)
	if err != nil {
		return errors.Wrap(err, "fetchOrganizationById")
	}
	switch org.Type {
	case api.OrgTypeProject:
		return orgNode.bindProject(ctx, userCred, idstr, tags)
	case api.OrgTypeDomain:
		return orgNode.bindDomain(ctx, userCred, idstr, tags)
	case api.OrgTypeObject:
		return orgNode.bindObject(ctx, userCred, resType, idstr, tags)
	}
	return errors.Wrapf(errors.ErrNotSupported, "cannot bind to %s %s", resType, idstr)
}

func (orgNode *SOrganizationNode) bindProject(ctx context.Context, userCred mcclient.TokenCredential, idstr string, tags map[string]string) error {
	projObj, err := ProjectManager.FetchById(idstr)
	if err != nil {
		return errors.Wrapf(err, "ProjectManager.FetchById %s", idstr)
	}
	return projObj.(*SProject).SetOrganizationMetadataAll(ctx, tags, userCred)
}

func (orgNode *SOrganizationNode) bindDomain(ctx context.Context, userCred mcclient.TokenCredential, idstr string, tags map[string]string) error {
	domainObj, err := DomainManager.FetchById(idstr)
	if err != nil {
		return errors.Wrapf(err, "DomainManager.FetchById %s", idstr)
	}
	return domainObj.(*SDomain).SetOrganizationMetadataAll(ctx, tags, userCred)
}

func (orgNode *SOrganizationNode) bindObject(ctx context.Context, userCred mcclient.TokenCredential, resType, idstr string, tags map[string]string) error {
	s := auth.GetSession(ctx, userCred, consts.GetRegion())
	module, err := modulebase.GetModule(s, resType)
	if err != nil {
		return errors.Wrap(err, "GetModule")
	}
	input := jsonutils.Marshal(tags)
	_, err = module.PerformAction(s, idstr, "set-org-metadata", input)
	if err != nil {
		return errors.Wrapf(err, "PerformAction set-org-metadata %s", idstr)
	}
	return nil
}

func (orgNode *SOrganizationNode) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := orgNode.SStandaloneResourceBase.GetShortDesc(ctx)
	desc.Set("level", jsonutils.NewInt(int64(orgNode.Level)))
	desc.Set("full_label", jsonutils.NewString(orgNode.FullLabel))
	return desc
}

func (orgNode *SOrganizationNode) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.OrganizationNodeUpdateInput,
) (api.OrganizationNodeUpdateInput, error) {
	var err error
	input.StandaloneResourceBaseUpdateInput, err = orgNode.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	// not allow to update name
	if len(input.Name) > 0 && input.Name != orgNode.Name {
		return input, errors.Wrap(httperrors.ErrForbidden, "not allow to update name")
	}
	return input, nil
}

func tagSetList2Conditions(tagsetList tagutils.TTagSetList, keys []string, q *sqlchemy.SQuery) sqlchemy.ICondition {
	for i := range keys {
		if !strings.HasPrefix(keys[i], "org:") {
			keys[i] = "org:" + keys[i]
		}
	}
	conds := make([]sqlchemy.ICondition, 0)
	paths := tagutils.TagSetList2Paths(tagsetList, keys)
	for i := range paths {
		for j := range paths[i] {
			label := api.JoinLabels(paths[i][:j+1]...)
			conds = append(conds, sqlchemy.Equals(q.Field("full_label"), label))
			if j == len(paths[i])-1 {
				labelSlash := label + api.OrganizationLabelSeparator
				conds = append(conds, sqlchemy.Startswith(q.Field("full_label"), labelSlash))
			}
		}
	}
	if len(conds) > 0 {
		return sqlchemy.OR(conds...)
	}
	return nil
}

// 项目列表
func (manager *SOrganizationNodeManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.OrganizationNodeListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}

	if len(query.OrgId) > 0 {
		orgObj, err := OrganizationManager.FetchByIdOrName(ctx, userCred, query.OrgId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(OrganizationManager.Keyword(), query.OrgId)
			} else {
				return nil, errors.Wrapf(err, "FetchByIdOrName %s", query.OrgId)
			}
		}
		org := orgObj.(*SOrganization)
		q = q.Equals("org_id", org.GetId())

		var cond sqlchemy.ICondition
		switch org.Type {
		case api.OrgTypeDomain:
			if !query.PolicyDomainTags.IsEmpty() {
				cond = tagSetList2Conditions(query.PolicyDomainTags, org.GetKeys(), q)
			}
		case api.OrgTypeProject:
			if !query.PolicyProjectTags.IsEmpty() {
				cond = tagSetList2Conditions(query.PolicyProjectTags, org.GetKeys(), q)
			}
		case api.OrgTypeObject:
			if !query.PolicyObjectTags.IsEmpty() {
				cond = tagSetList2Conditions(query.PolicyObjectTags, org.GetKeys(), q)
			}
		}
		if cond != nil {
			q = q.Filter(cond)
		}
	}

	if len(query.OrgType) > 0 {
		orgSubQ := OrganizationManager.Query().SubQuery()
		q = q.Join(orgSubQ, sqlchemy.Equals(q.Field("org_id"), orgSubQ.Field("id")))
		q = q.Filter(sqlchemy.Equals(orgSubQ.Field("type"), query.OrgType))
	}

	if query.Level != 0 {
		q = q.Equals("level", query.Level)
	}

	return q, nil
}

func (manager *SOrganizationNodeManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.OrganizationNodeListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SOrganizationNodeManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (orgNode *SOrganizationNode) GetTags(org *SOrganization) map[string]string {
	tags := make(map[string]string)
	keys := api.SplitLabel(org.Keys)
	values := api.SplitLabel(orgNode.FullLabel)
	for i := 0; i < orgNode.Level; i++ {
		k := db.ORGANIZATION_TAG_PREFIX + keys[i]
		v := values[i]
		tags[k] = v
	}
	return tags
}

func (orgNode *SOrganizationNode) GetTagSet(org *SOrganization) tagutils.TTagSet {
	tags := make(tagutils.TTagSet, 0)
	keys := api.SplitLabel(org.Keys)
	values := api.SplitLabel(orgNode.FullLabel)
	for i := 0; i < orgNode.Level; i++ {
		tag := tagutils.STag{
			Key:   db.ORGANIZATION_TAG_PREFIX + keys[i],
			Value: values[i],
		}
		tags = append(tags, tag)
	}
	return tags
}

func (manager *SOrganizationNodeManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SOrganizationNodeDetails {
	rows := make([]api.SOrganizationNodeDetails, len(objs))
	orgIds := make([]string, 0)
	for i := range rows {
		orgNode := objs[i].(*SOrganizationNode)
		if !utils.IsInArray(orgNode.OrgId, orgIds) {
			orgIds = append(orgIds, orgNode.OrgId)
		}
	}

	orgMaps := make(map[string]SOrganization)
	err := db.FetchModelObjectsByIds(OrganizationManager, "id", orgIds, &orgMaps)
	if err != nil {
		log.Errorf("FetchModelObjectsByIds fail %s", err)
		return rows
	}

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		orgNode := objs[i].(*SOrganizationNode)
		if org, ok := orgMaps[orgNode.OrgId]; ok {
			rows[i] = api.SOrganizationNodeDetails{
				SOrganizationNode: api.SOrganizationNode{
					OrgId:     orgNode.OrgId,
					FullLabel: orgNode.FullLabel,
					Level:     orgNode.Level,
					Weight:    orgNode.Weight,
				},
				StandaloneResourceDetails: stdRows[i],
				Organization:              org.Name,
				Tags:                      orgNode.GetTagSet(&org),
				Type:                      org.Type,
			}
			rows[i].Id = orgNode.Id
		}
	}

	return rows
}

func (manager *SOrganizationNodeManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	q = manager.SStandaloneResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)
	q = manager.SPendingDeletedBaseManager.FilterBySystemAttributes(manager.GetIStandaloneModelManager(), q, userCred, query, scope)
	return q
}

func (orgNode *SOrganizationNode) ValidateDeleteCondition(ctx context.Context, info *api.ProjectDetails) error {
	childCnt, err := orgNode.GetDirectChildCount()
	if err != nil {
		return errors.Wrap(err, "GetDirectChildCount")
	}
	if childCnt > 0 {
		return errors.Wrapf(httperrors.ErrNotEmpty, "childnodes %d", childCnt)
	}
	return orgNode.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

// fake delete
func (orgNode *SOrganizationNode) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !orgNode.PendingDeleted {
		err := orgNode.SPendingDeletedBase.MarkPendingDelete(orgNode.GetIStandaloneModel(), ctx, userCred, "")
		if err != nil {
			return errors.Wrap(err, "MarkPendingDelete")
		}
	}
	return nil
}

func (manager *SOrganizationNodeManager) fetchOrgNodesInfo(ctx context.Context, userCred mcclient.TokenCredential, nodeIds []string, isList bool) ([]api.SOrganizationNodeInfo, error) {
	q := manager.Query().In("id", nodeIds)
	nodes := make([]SOrganizationNode, 0)
	err := db.FetchModelObjects(manager, q, &nodes)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	infs := make([]interface{}, 0)
	for i := range nodes {
		infs = append(infs, &nodes[i])
	}
	nodeDetails := manager.FetchCustomizeColumns(ctx, userCred, nil, infs, nil, isList)
	ret := make([]api.SOrganizationNodeInfo, 0)
	for i := range nodeDetails {
		node := nodeDetails[i]
		ret = append(ret, api.SOrganizationNodeInfo{
			Id:           node.Id,
			FullLabel:    node.FullLabel,
			OrgId:        node.OrgId,
			Organization: node.Organization,
			Tags:         node.Tags,
			Type:         node.Type,
		})
	}
	return ret, nil
}
