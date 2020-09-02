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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	SuggestSysRuleConfigManager *SSuggestSysRuleConfigManager
)

func init() {
	SuggestSysRuleConfigManager = &SSuggestSysRuleConfigManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			&SSuggestSysRuleConfig{},
			"suggestsysruleconfig_tbl",
			"suggestsysruleconfig",
			"suggestsysruleconfigs",
		),
	}
	SuggestSysRuleConfigManager.SetVirtualObject(SuggestSysRuleConfigManager)
}

// +onecloud:swagger-gen-model-singular=suggestsysruleconfig
// +onecloud:swagger-gen-model-plural=suggestsysruleconfigs
type SSuggestSysRuleConfigManager struct {
	db.SStandaloneResourceBaseManager
	db.SScopedResourceBaseManager
}

type SSuggestSysRuleConfig struct {
	db.SStandaloneResourceBase
	db.SScopedResourceBase

	// RuleId is SSuggestSysRule model object id
	// RuleId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	// Type is suggestsysrules driver type
	Type string `width:"256" charset:"ascii" list:"user" create:"optional"`
	// ResourceType is suggestsysrules driver resource type
	ResourceType string `width:"256" charset:"ascii" list:"user" create:"optional"`
	// ResourceId is suggest alert result resource id
	ResourceId string `width:"256" charset:"ascii" list:"user" create:"optional"`
	// IgnoreAlert means whether or not show SSuggestSysAlert results for current scope
	IgnoreAlert bool `nullable:"false" default:"false" list:"user" create:"optional" update:"user"`
}

// InitInitScopeSuggestConfigs init default configs to project, domain and system scope
func (man *SSuggestSysRuleConfigManager) InitScopeConfigs(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if err := man.InitSystemScopeConfig(ctx, userCred); err != nil {
		log.Errorf("init system scope suggest config error: %v", err)
	}
	domains, projects, err := FetchAllRemoteDomainProjects(ctx)
	if err != nil {
		log.Errorf("fetch remote domain projects: %v", err)
		return
	}
	if err := man.InitDomainScopeConfig(ctx, userCred, domains); err != nil {
		log.Errorf("init domain scope suggest config error: %v", err)
	}
	if err := man.InitProjectScopeConfig(ctx, userCred, projects); err != nil {
		log.Errorf("init project scope suggest config error: %v", err)
	}
}

const (
	SUGGEST_SCOPE_CONFIG = "suggest_scope_config"
)

func (man *SSuggestSysRuleConfigManager) initScopeConfig(ctx context.Context, userCred mcclient.TokenCredential, scope rbacutils.TRbacScope, tenant *db.STenant) error {
	drivers := GetSuggestSysRuleDrivers()
	config := jsonutils.NewDict()
	configJSON := tenant.GetMetadataJson(SUGGEST_SCOPE_CONFIG, userCred)
	if configJSON != nil {
		if err := configJSON.Unmarshal(config); err != nil {
			return errors.Wrap(err, "unmarshal metadata config")
		}
	}
	for _, drv := range drivers {
		if ok, _ := config.Bool(string(drv.GetType())); ok {
			continue
		}
		if err := man.createFromDriver(ctx, scope, drv, tenant); err != nil {
			return errors.Wrapf(err, "init scope %s from driver %s", scope, drv.GetType())
		}
		config.Set(string(drv.GetType()), jsonutils.JSONTrue)
	}

	if err := tenant.SetMetadata(ctx, SUGGEST_SCOPE_CONFIG, config, userCred); err != nil {
		return errors.Wrap(err, "set suggest scope config metadata")
	}
	return nil
}

func (man *SSuggestSysRuleConfigManager) InitSystemScopeConfig(ctx context.Context, userCred mcclient.TokenCredential) error {
	systemFakeTenantId := "monitor.fake.tenant"
	systemFakeTenant, err := db.TenantCacheManager.Save(ctx, systemFakeTenantId, systemFakeTenantId, systemFakeTenantId, systemFakeTenantId)
	if err != nil {
		return errors.Wrap(err, "save system fake tenant")
	}
	return man.initScopeConfig(ctx, userCred, rbacutils.ScopeSystem, systemFakeTenant)
}

func (man *SSuggestSysRuleConfigManager) InitDomainScopeConfig(ctx context.Context, userCred mcclient.TokenCredential, domains []*db.STenant) error {
	errs := make([]error, 0)
	for _, domain := range domains {
		if err := man.initScopeConfig(ctx, userCred, rbacutils.ScopeDomain, domain); err != nil {
			errs = append(errs, errors.Wrapf(err, "init domain %s", domain.GetId()))
		}
	}
	return errors.NewAggregate(errs)
}

func (man *SSuggestSysRuleConfigManager) InitProjectScopeConfig(ctx context.Context, userCred mcclient.TokenCredential, projects []*db.STenant) error {
	for _, project := range projects {
		if err := man.initScopeConfig(ctx, userCred, rbacutils.ScopeProject, project); err != nil {
			return errors.Wrapf(err, "init project %v", project)
		}
	}
	return nil
}

func (man *SSuggestSysRuleConfigManager) createFromDriver(ctx context.Context, scope rbacutils.TRbacScope, drv ISuggestSysRuleDriver, project *db.STenant) error {
	config := new(SSuggestSysRuleConfig)
	drvType := string(drv.GetType())
	name := fmt.Sprintf("%s-%s", strings.ToLower(drvType), scope)
	if scope != rbacutils.ScopeSystem && project == nil {
		return errors.Errorf("scope %s not allow nil project", scope)
	}
	config.Name = name
	config.Type = drvType
	config.ResourceType = string(drv.GetResourceType())
	config.IgnoreAlert = false
	switch scope {
	case rbacutils.ScopeDomain:
		config.DomainId = project.GetId()
	case rbacutils.ScopeProject:
		config.DomainId = project.GetDomainId()
		config.ProjectId = project.GetId()
	}
	config.SetModelManager(man, config)

	ownerId := config.GetOwnerId()
	data := monitor.SuggestSysRuleConfigCreateInput{}
	data.Scope = string(scope)
	data.ProjectDomainId = ownerId.GetProjectDomainId()
	data.ProjectId = ownerId.GetProjectId()
	// HACK parentId, ref SScopedResourceBaseManager.FetchUniqValues
	uniqValues := man.FetchUniqValues(ctx, data.JSON(data))
	if err := db.NewNameValidator(man, ownerId, name, uniqValues); err != nil {
		return errors.Wrapf(err, "validate name for %q, domain %q, project %q", scope, ownerId.GetProjectDomainId(), ownerId.GetProjectId())
	}

	if err := man.TableSpec().Insert(ctx, config); err != nil {
		return errors.Wrapf(err, "insert config %#v", config)
	}
	if _, err := db.Update(config, func() error {
		if scope == rbacutils.ScopeSystem {
			config.DomainId = ""
			config.ProjectId = ""
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "update scope info")
	}
	return nil
}

func (man *SSuggestSysRuleConfigManager) AllowGetPropertySupportTypes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (man *SSuggestSysRuleConfigManager) GetPropertySupportTypes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*monitor.SuggestSysRuleConfigSupportTypes, error) {
	ret := &monitor.SuggestSysRuleConfigSupportTypes{
		Types:         make([]monitor.SuggestDriverType, 0),
		ResourceTypes: make([]string, 0),
	}
	drivers := GetSuggestSysRuleDrivers()
	for _, drv := range drivers {
		ret.Types = append(ret.Types, drv.GetType())
	}
	ret.ResourceTypes = GetSuggestSysRuleResourceTypes().List()
	return ret, nil
}

func (man *SSuggestSysRuleConfigManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeNone
}

func (man *SSuggestSysRuleConfigManager) GetConfigsByScope(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, ignoreAlert bool) ([]SSuggestSysRuleConfig, error) {
	q := man.Query().Equals("ignore_alert", ignoreAlert)
	configs := make([]SSuggestSysRuleConfig, 0)
	switch scope {
	case rbacutils.ScopeSystem:
		q = q.IsNullOrEmpty("domain_id").IsNullOrEmpty("tenant_id")
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", userCred.GetProjectDomainId()).IsNullOrEmpty("tenant_id")
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", userCred.GetProjectId())
	}
	if err := db.FetchModelObjects(man, q, &configs); err != nil {
		return nil, err
	}
	return configs, nil
}

func (man *SSuggestSysRuleConfigManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *monitor.SuggestSysRuleConfigCreateInput) (*monitor.SuggestSysRuleConfigCreateInput, error) {
	var err error
	input.StandaloneResourceCreateInput, err = man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.ScopedResourceCreateInput, err = man.SScopedResourceBaseManager.ValidateCreateData(man, ctx, userCred, ownerId, query, input.ScopedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	if input.Type != nil {
		drvs := GetSuggestSysRuleDrivers()
		if _, ok := drvs[*input.Type]; !ok {
			if err != nil {
				return nil, httperrors.NewNotFoundError("not found type %q", *input.Type)
			}
		}
	}
	if input.ResourceType != nil {
		if !GetSuggestSysRuleResourceTypes().Has(string(*input.ResourceType)) {
			return nil, httperrors.NewNotFoundError("not found resource_type %q", *input.ResourceType)
		}
		if input.Type != nil {
			drv := SuggestSysRuleManager.GetDriver(*input.Type)
			if drv == nil {
				return nil, httperrors.NewNotFoundError("not found driver by type %q", *input.Type)
			}
			if drv.GetResourceType() != *input.ResourceType {
				return nil, httperrors.NewNotAcceptableError("type %q resource type not match input %q", drv.GetType(), *input.ResourceType)
			}
		}
	}
	if input.ResourceId != nil && input.ResourceType == nil {
		return nil, httperrors.NewNotAcceptableError("resource type must provided when resource_id specified")
	}
	if input.Type == nil && input.ResourceType == nil {
		return nil, httperrors.NewNotAcceptableError("type or resource_type must provided")
	}
	return input, nil
}

func (man *SSuggestSysRuleConfigManager) GetRuleByType(drvType monitor.SuggestDriverType) (*SSuggestSysRule, error) {
	drv := SuggestSysRuleManager.GetDriver(drvType)
	if drv == nil {
		return nil, httperrors.NewInputParameterError("not support type %q", drvType)
	}
	rule, err := SuggestSysRuleManager.GetRuleByType(drvType)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, httperrors.NewNotFoundError("not found rule by type %q", drvType)
	}
	return rule, nil
}

func (conf *SSuggestSysRuleConfig) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := conf.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	if err := conf.SScopedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	return nil
}

func (conf *SSuggestSysRuleConfig) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	scope, _ := data.GetString("scope")
	_, err := db.Update(conf, func() error {
		switch rbacutils.TRbacScope(scope) {
		case rbacutils.ScopeSystem:
			conf.DomainId = ""
			conf.ProjectId = ""
		case rbacutils.ScopeDomain:
			conf.ProjectId = ""
		}
		return nil
	})
	if err != nil {
		log.Errorf("post update %s scope info error: %v", conf.GetName(), err)
	}
}

func (conf *SSuggestSysRuleConfig) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *monitor.SuggestSysRuleConfigUpdateInput) (*monitor.SuggestSysRuleConfigUpdateInput, error) {
	return input, nil
}

func (man *SSuggestSysRuleConfigManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.SuggestSysRuleConfigDetails {
	rows := make([]monitor.SuggestSysRuleConfigDetails, len(objs))
	stdRows := man.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := man.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.SuggestSysRuleConfigDetails{
			StandaloneResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:    scopedRows[i],
		}
		rows[i] = objs[i].(*SSuggestSysRuleConfig).getMoreColumns(rows[i])
	}
	return rows
}

func (conf *SSuggestSysRuleConfig) getMoreColumns(out monitor.SuggestSysRuleConfigDetails) monitor.SuggestSysRuleConfigDetails {
	if conf.Type != "" {
		rule, err := conf.GetRule()
		if err != nil {
			log.Errorf("Get config %q rule error: %v", conf.GetName(), err)
			return out
		}
		if rule == nil {
			return out
		}
		out.RuleId = rule.GetId()
		out.Rule = rule.GetName()
		out.RuleEnabled = rule.GetEnabled()
	}
	return out
}

func (conf *SSuggestSysRuleConfig) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (monitor.SuggestSysRuleConfigDetails, error) {
	return monitor.SuggestSysRuleConfigDetails{}, nil
}

func (conf *SSuggestSysRuleConfig) GetRule() (*SSuggestSysRule, error) {
	if conf.Type == "" {
		return nil, nil
	}
	return SuggestSysRuleManager.GetRuleByType(monitor.SuggestDriverType(conf.Type))
}

func (man *SSuggestSysRuleConfigManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query monitor.SuggestSysRuleConfigListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = man.SScopedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemFilter")
	}
	if query.Type != nil {
		q.Equals("type", *query.Type)
	}
	if query.ResourceType != nil {
		q.Equals("resource_type", *query.ResourceType)
	}
	if query.IgnoreAlert != nil {
		if *query.IgnoreAlert {
			q.IsTrue("ignore_alert")
		} else {
			q.IsFalse("ignore_alert")
		}
	}
	return q, nil
}

func (man *SSuggestSysRuleConfigManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query monitor.SuggestSysRuleConfigListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = man.SScopedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ScopedResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (conf *SSuggestSysRuleConfig) AllowPerformToggleAlert(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return conf.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, conf, "toggle-alert")
}

func (conf *SSuggestSysRuleConfig) PerformToggleAlert(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONDict, data jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	var ignoreAlert bool
	if conf.IgnoreAlert {
		ignoreAlert = false
	} else {
		ignoreAlert = true
	}
	if _, err := db.Update(conf, func() error {
		conf.IgnoreAlert = ignoreAlert
		return nil
	}); err != nil {
		return nil, err
	}
	return nil, nil
}

func (conf *SSuggestSysRuleConfig) ShouldIgnoreAlert(alert *SSuggestSysAlert) bool {
	if !conf.IgnoreAlert {
		return false
	}
	drv := alert.GetDriver()
	if conf.ResourceId == "" {
		if conf.Type == alert.Type {
			return true
		}
		if conf.ResourceType == string(drv.GetResourceType()) {
			return true
		}
	} else {
		if conf.ResourceId != alert.ResId {
			return false
		}
		if conf.ResourceType == string(drv.GetResourceType()) {
			return true
		}
	}
	return false
}
