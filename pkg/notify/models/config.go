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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	notifyv2 "yunion.io/x/onecloud/pkg/notify"
	"yunion.io/x/onecloud/pkg/notify/oldmodels"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SConfigManager struct {
	db.SStandaloneResourceBaseManager
	db.SDomainizedResourceBaseManager
}

var ConfigManager *SConfigManager

func init() {
	ConfigManager = &SConfigManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SConfig{},
			"configs_tbl",
			"notifyconfig",
			"notifyconfigs",
		),
	}
	ConfigManager.SetVirtualObject(ConfigManager)
}

type SConfig struct {
	db.SStandaloneResourceBase
	db.SDomainizedResourceBase

	Type        string               `width:"15" nullable:"false" create:"required" get:"domain" list:"domain" index:"true"`
	Content     jsonutils.JSONObject `nullable:"false" create:"required" update:"domain" get:"domain" list:"domain"`
	Attribution string               `width:"8" nullable:"false" default:"system" get:"domain" list:"domain" create:"optional"`
}

func (cm *SConfigManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ConfigCreateInput) (api.ConfigCreateInput, error) {
	var err error
	input.StandaloneResourceCreateInput, err = cm.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}
	if len(input.ProjectDomainId) > 0 {
		_, input.DomainizedResourceInput, err = db.ValidateDomainizedResourceInput(ctx, input.DomainizedResourceInput)
		if err != nil {
			return input, err
		}
	}
	if !utils.IsInStringArray(input.Type, []string{api.EMAIL, api.MOBILE, api.DINGTALK, api.FEISHU, api.WEBCONSOLE, api.WORKWX}) {
		return input, httperrors.NewInputParameterError("unkown type %q", input.Type)
	}
	if !utils.IsInStringArray(input.Attribution, []string{api.CONFIG_ATTRIBUTION_SYSTEM, api.CONFIG_ATTRIBUTION_DOMAIN}) {
		return input, httperrors.NewInputParameterError("invalid attribution, need %q or %q", api.CONFIG_ATTRIBUTION_SYSTEM, api.CONFIG_ATTRIBUTION_DOMAIN)
	}
	if input.Attribution == api.CONFIG_ATTRIBUTION_SYSTEM {
		allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), ConfigManager.KeywordPlural(), policy.PolicyActionCreate)
		if allowScope != rbacutils.ScopeSystem {
			return input, httperrors.NewInputParameterError("No permission to set %q attribution", api.CONFIG_ATTRIBUTION_SYSTEM)
		}
	}
	if input.Content == nil {
		return input, httperrors.NewMissingParameterError("content")
	}
	config, err := cm.Config(input.Type, input.ProjectDomainId)
	if err == nil && config != nil {
		return input, httperrors.NewDuplicateResourceError("duplicate type %q", input.Type)
	}
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return input, err
	}
	// validate
	configs := make(map[string]string)
	err = input.Content.Unmarshal(&configs)
	if err != nil {
		return input, err
	}
	isValid, message, err := NotifyService.ValidateConfig(ctx, input.Type, configs)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotImplemented {
			return input, httperrors.NewNotImplementedError("validating config of %s", input.Type)
		}
		return input, err
	}
	if !isValid {
		return input, httperrors.NewInputParameterError(message)
	}
	if len(input.Name) == 0 {
		input.Name = input.Type
	}
	return input, nil
}

func (c *SConfig) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := c.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return err
	}
	if c.Attribution == api.CONFIG_ATTRIBUTION_DOMAIN || c.Attribution == "" {
		c.Attribution = api.CONFIG_ATTRIBUTION_DOMAIN
		c.DomainId, _ = data.GetString("project_domain_id")
		if c.DomainId == "" {
			c.DomainId = userCred.GetProjectDomainId()
		}
	}
	return nil
}

func (c *SConfig) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ConfigUpdateInput) (api.ConfigUpdateInput, error) {
	// validate
	configs := make(map[string]string)
	err := input.Content.Unmarshal(&configs)
	if err != nil {
		return input, err
	}
	// check if changed
	if c.Content.Equals(input.Content) {
		return input, nil
	}
	isValid, message, err := NotifyService.ValidateConfig(ctx, c.Type, configs)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotImplemented {
			return input, httperrors.NewNotImplementedError("validating config of %s", c.Type)
		}
		return input, err
	}
	if !isValid {
		return input, httperrors.NewInputParameterError(message)
	}
	return input, nil
}

func (c *SConfig) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	configMap := make(map[string]string)
	err := c.Content.Unmarshal(&configMap)
	if err != nil {
		log.Errorf("unable to unmarshal: %v", err)
		return
	}
	NotifyService.AddConfig(ctx, c.Type, notifyv2.SConfig{
		Config:   configMap,
		DomainId: c.DomainId,
	})
	err = c.StartRepullSubcontactTask(ctx, userCred)
	if err != nil {
		log.Errorf("unable to StartRepullSubcontactTask: %v", err)
	}
}

func (c *SConfig) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	configMap := make(map[string]string)
	err := c.Content.Unmarshal(&configMap)
	if err != nil {
		log.Errorf("unable to unmarshal: %v", err)
		return
	}
	NotifyService.UpdateConfig(ctx, c.Type, notifyv2.SConfig{
		Config:   configMap,
		DomainId: c.DomainId,
	})
	err = c.StartRepullSubcontactTask(ctx, userCred)
	if err != nil {
		log.Errorf("unable to StartRepullSubcontactTask: %v", err)
	}
}

func (c *SConfig) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	c.SStandaloneResourceBase.PreDelete(ctx, userCred)
	NotifyService.DeleteConfig(ctx, c.Type, c.DomainId)
}
func (c *SConfig) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	err := c.StartRepullSubcontactTask(ctx, userCred)
	if err != nil {
		log.Errorf("unable to StartRepullSubcontactTask: %v", err)
	}
}

func (c *SConfig) StartRepullSubcontactTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "RepullSuncontactTask", c, userCred, nil, "", "")
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

var sortedCTypes = []string{
	api.WEBCONSOLE, api.EMAIL, api.MOBILE, api.DINGTALK, api.FEISHU, api.WORKWX,
}

func sortContactType(ctypes []string) []string {
	ctSet := sets.NewString(ctypes...)
	ret := make([]string, 0, len(ctypes))
	for _, ct := range sortedCTypes {
		if ctSet.Has(ct) {
			ret = append(ret, ct)
		}
	}
	return ret
}

func (cm *SConfigManager) contactTypesQuery(domainId string) *sqlchemy.SQuery {
	q := cm.Query("type").Distinct()
	if domainId == "" {
		q = q.Equals("attribution", api.CONFIG_ATTRIBUTION_SYSTEM)
	} else {
		q = q.Filter(sqlchemy.OR(sqlchemy.AND(sqlchemy.Equals(q.Field("attribution"), api.CONFIG_ATTRIBUTION_DOMAIN), sqlchemy.Equals(q.Field("domain_id"), domainId)), sqlchemy.Equals(q.Field("attribution"), api.CONFIG_ATTRIBUTION_SYSTEM)))
	}
	return q
}

func (cm *SConfigManager) availableContactTypes(domainId string) ([]string, error) {
	q := cm.Query("type")
	q = q.Filter(sqlchemy.OR(sqlchemy.AND(sqlchemy.Equals(q.Field("attribution"), api.CONFIG_ATTRIBUTION_DOMAIN), sqlchemy.Equals(q.Field("domain_id"), domainId)), sqlchemy.Equals(q.Field("attribution"), api.CONFIG_ATTRIBUTION_SYSTEM)))

	allTypes := make([]struct {
		Type string
	}, 0, 3)
	err := q.All(&allTypes)
	if err != nil {
		return nil, err
	}
	ret := make([]string, len(allTypes))
	for i := range ret {
		ret[i] = allTypes[i].Type
	}
	// De-duplication
	return sets.NewString(ret...).UnsortedList(), nil
}

func (cm *SConfigManager) allContactType() ([]string, error) {
	q := cm.Query("type")
	allTypes := make([]struct {
		Type string
	}, 0, 3)
	err := q.All(&allTypes)
	if err != nil {
		return nil, err
	}
	ret := make([]string, len(allTypes))
	for i := range ret {
		ret[i] = allTypes[i].Type
	}
	return ret, nil
}

func (self *SConfigManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.ConfigListInput) (*sqlchemy.SQuery, error) {
	q, err := self.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = self.SDomainizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.DomainizedResourceListInput)
	if err != nil {
		return nil, err
	}
	q = q.NotEquals("type", api.WEBCONSOLE)
	if len(input.Type) > 0 {
		q.Filter(sqlchemy.Equals(q.Field("type"), input.Type))
	}
	if len(input.Attribution) > 0 {
		q = q.Equals("attribution", input.Attribution)
	}

	return q, nil
}

func (cm *SConfigManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ConfigDetails {
	sRows := cm.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	dRows := cm.SDomainizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]api.ConfigDetails, len(objs))
	for i := range rows {
		rows[i].StandaloneResourceDetails = sRows[i]
		rows[i].DomainizedResourceInfo = dRows[i]
	}
	return rows
}

func (cm *SConfigManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := cm.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return q, nil
	}
	q, err = cm.SDomainizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return q, nil
	}
	return q, nil
}

func (cm *SConfigManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ConfigListInput) (*sqlchemy.SQuery, error) {
	q, err := cm.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = cm.SDomainizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainizedResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (cm *SConfigManager) AllowPerformValidate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, cm, "validate")
}

func (cm *SConfigManager) PerformValidate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ConfigValidateInput) (api.ConfigValidateOutput, error) {
	var (
		output api.ConfigValidateOutput
		err    error
	)
	if !utils.IsInStringArray(input.Type, []string{api.EMAIL, api.MOBILE, api.DINGTALK, api.FEISHU, api.WEBCONSOLE, api.WORKWX, api.FEISHU_ROBOT, api.DINGTALK_ROBOT, api.WORKWX_ROBOT}) {
		return output, httperrors.NewInputParameterError("unkown type %q", input.Type)
	}
	if input.Content == nil {
		return output, httperrors.NewMissingParameterError("content")
	}
	// validate
	configs := make(map[string]string)
	err = input.Content.Unmarshal(&configs)
	if err != nil {
		return output, err
	}
	isValid, message, err := NotifyService.ValidateConfig(ctx, input.Type, configs)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotImplemented {
			return output, httperrors.NewNotImplementedError("validating config of %s", input.Type)
		}
		return output, err
	}
	if !isValid {
		output.IsValid = false
		output.Message = message
	} else {
		output.IsValid = true
	}
	return output, nil
}

func (self *SConfigManager) InitializeData() error {
	ctx := context.Background()
	userCred := auth.AdminCredential()
	// fetch all configs
	configs := make([]oldmodels.SConfig, 0, 5)
	q := oldmodels.ConfigManager.Query()
	err := db.FetchModelObjects(oldmodels.ConfigManager, q, &configs)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}

	// build type==>config map
	tcMap := make(map[string][]*oldmodels.SConfig)
	for i := range configs {
		t := configs[i].Type
		if _, ok := tcMap[t]; !ok {
			tcMap[t] = make([]*oldmodels.SConfig, 0, 3)
		}
		tcMap[t] = append(tcMap[t], &configs[i])
	}

	for t, configs := range tcMap {
		cMap := make(map[string]string)
		if t == api.EMAIL {
			for _, config := range configs {
				switch config.KeyText {
				case "mail.username":
					cMap["username"] = config.ValueText
				case "mail.password":
					cMap["password"] = config.ValueText
				case "mail.smtp.hostname":
					cMap["hostname"] = config.ValueText
				case "mail.smtp.hostport":
					cMap["hostport"] = config.ValueText
				}
			}
		} else {
			for _, config := range configs {
				cMap[config.KeyText] = config.ValueText
			}
		}
		newConfig := SConfig{
			Type:    t,
			Content: jsonutils.Marshal(cMap),
		}
		err := self.TableSpec().Insert(ctx, &newConfig)
		if err != nil {
			return errors.Wrap(err, "TableSpec().Insert")
		}
		for _, config := range configs {
			err := config.Delete(ctx, userCred)
			if err != nil {
				return errors.Wrap(err, "Delete")
			}
		}
	}

	// init webconsole's config
	q = self.Query().Equals("type", api.WEBCONSOLE)
	wsConfigs := make([]SConfig, 0, 2)
	err = db.FetchModelObjects(self, q, &wsConfigs)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	if len(wsConfigs) > 1 {
		for i := 1; i < len(wsConfigs); i++ {
			err := wsConfigs[i].Delete(ctx, userCred)
			if err != nil {
				return errors.Wrap(err, "Delete redundant")
			}
		}
	}

	var config *SConfig
	if len(wsConfigs) > 0 {
		config = &wsConfigs[0]
	} else {
		config = &SConfig{
			Type: api.WEBCONSOLE,
		}
	}
	config.Content = jsonutils.Marshal(map[string]string{
		"auth_uri":          options.Options.AuthURL,
		"admin_user":        options.Options.AdminUser,
		"admin_password":    options.Options.AdminPassword,
		"admin_tenant_name": options.Options.AdminProject,
	})
	err = self.TableSpec().InsertOrUpdate(context.TODO(), config)

	// init config name
	q = self.Query().IsNullOrEmpty("name")
	enConfigs := make([]SConfig, 0)
	err = db.FetchModelObjects(self, q, &enConfigs)
	if err != nil {
		return errors.Wrap(err, "unable to get configs with empty name")
	}
	for i := range enConfigs {
		c := &enConfigs[i]
		name, err := db.GenerateAlterName(c, enConfigs[i].Type)
		if err != nil {
			return errors.Wrap(err, "unable to generate alter name")
		}
		_, err = db.Update(c, func() error {
			c.Name = name
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "unable to update name")
		}
	}
	return nil
}

func (cm *SConfigManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (cm *SConfigManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	switch scope {
	case rbacutils.ScopeDomain, rbacutils.ScopeProject:
		q = q.Equals("attribution", api.CONFIG_ATTRIBUTION_DOMAIN)
		if owner != nil {
			q = q.Equals("domain_id", owner.GetProjectDomainId())
		}
	}
	return q
}

func (cm *SConfigManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, cm)
}

func (c *SConfig) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, c)
}

func (cm *SConfigManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, cm)
}

func (c *SConfig) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, c)
}

// Fetch all SConfig struct which type is contactType.
func (self *SConfigManager) Configs(contactType string) ([]SConfig, error) {
	var configs = make([]SConfig, 0, 2)
	q := self.Query()
	q.Filter(sqlchemy.Equals(q.Field("type"), contactType))
	err := q.All(&configs)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to fetch SConfigs by type %s", contactType)
	}
	return configs, nil
}

func (self *SConfigManager) Config(contactType, domainId string) (*SConfig, error) {
	q := self.Query()
	q = q.Equals("type", contactType)
	if len(domainId) == 0 {
		q = q.Equals("attribution", api.CONFIG_ATTRIBUTION_SYSTEM)
	} else {
		q = q.Equals("domain_id", domainId).Equals("attribution", api.CONFIG_ATTRIBUTION_DOMAIN)
	}
	var config SConfig
	err := q.First(&config)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to fetch SConfig by type %s and domain %s", contactType, domainId)
	}
	return &config, nil
}

func (self *SConfigManager) HasSystemConfig(contactType string) (bool, error) {
	q := self.Query().Equals("type", contactType).Equals("attribution", "system")
	c, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

func (self *SConfigManager) BatchCheckConfig(contactType string, domainIds []string) ([]bool, error) {
	domainIdSet := sets.NewString(domainIds...)
	var configs = make([]SConfig, 0, 2)
	q := self.Query().Equals("type", contactType).Equals("attribution", "domain").In("domain_id", domainIdSet.UnsortedList())
	err := q.All(&configs)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to fetch SConfigs by type %s", contactType)
	}
	for i := range configs {
		if domainIdSet.Has(configs[i].DomainId) {
			domainIdSet.Delete(configs[i].DomainId)
		}
	}
	ret := make([]bool, len(domainIds))
	for i := range domainIds {
		if domainIdSet.Has(domainIds[i]) {
			// no config of domainId, use default one
			ret[i] = false
		}
		ret[i] = true
	}
	return ret, nil
}

func (self *SConfigManager) GetConfigs(contactType string) ([]notifyv2.SConfig, error) {
	configs, err := self.Configs(contactType)
	if err != nil {
		return nil, err
	}
	ret := make([]notifyv2.SConfig, 0, len(configs))
	for i := range configs {
		c := make(map[string]string)
		err := configs[i].Content.Unmarshal(&c)
		if err != nil {
			return nil, errors.Wrap(err, "fail unmarshal config content")
		}
		ret = append(ret, notifyv2.SConfig{
			Config:   c,
			DomainId: configs[i].DomainId,
		})
	}
	return ret, nil
}

func (self *SConfigManager) GetConfig(contactType, domainId string) (notifyv2.SConfig, error) {
	config, err := self.Config(contactType, domainId)
	if err != nil {
		return notifyv2.SConfig{}, err
	}
	ret := make(map[string]string)
	err = config.Content.Unmarshal(&ret)
	if err != nil {
		return notifyv2.SConfig{}, errors.Wrap(err, "fail unmarshal config content")
	}
	return notifyv2.SConfig{
		Config:   ret,
		DomainId: config.DomainId,
	}, nil
}

func (self *SConfigManager) SetConfig(contactType string, config notifyv2.SConfig) error {
	content := jsonutils.Marshal(config.Config)
	sConfig := &SConfig{
		Type:    contactType,
		Content: content,
	}
	sConfig.DomainId = config.DomainId
	if sConfig.DomainId == "" {
		sConfig.Attribution = api.CONFIG_ATTRIBUTION_SYSTEM
	} else {
		sConfig.Attribution = api.CONFIG_ATTRIBUTION_DOMAIN
	}
	return self.TableSpec().InsertOrUpdate(context.Background(), sConfig)
}

func intersection(sa1, sa2 []string) []string {
	set1 := sets.NewString(sa1...)
	set2 := sets.NewString(sa2...)
	return set1.Intersection(set2).UnsortedList()
}

func difference(sa1, sa2 []string) []string {
	set1 := sets.NewString(sa1...)
	set2 := sets.NewString(sa2...)
	return set1.Difference(set2).UnsortedList()
}

func union(sa1, sa2 []string) []string {
	set1 := sets.NewString(sa1...)
	set2 := sets.NewString(sa2...)
	return set1.Union(set2).UnsortedList()
}
