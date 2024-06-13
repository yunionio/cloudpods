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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SConfigManager struct {
	db.SDomainLevelResourceBaseManager
}

var ConfigManager *SConfigManager
var ConfigMap map[string]SConfig

func init() {
	ConfigManager = &SConfigManager{
		SDomainLevelResourceBaseManager: db.NewDomainLevelResourceBaseManager(
			SConfig{},
			"configs_tbl",
			"notifyconfig",
			"notifyconfigs",
		),
	}
	ConfigManager.SetVirtualObject(ConfigManager)
}

type SConfig struct {
	db.SDomainLevelResourceBase

	Type        string                    `width:"15" nullable:"false" create:"required" get:"domain" list:"domain" index:"true"`
	Content     *api.SNotifyConfigContent `nullable:"false" create:"required" update:"domain" get:"domain" list:"domain"`
	Attribution string                    `width:"8" nullable:"false" default:"system" get:"domain" list:"domain" create:"optional"`
}

func (cm *SConfigManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ConfigCreateInput) (api.ConfigCreateInput, error) {
	var err error
	input.DomainLevelResourceCreateInput, err = cm.SDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.DomainLevelResourceCreateInput)
	if err != nil {
		return input, err
	}
	if !utils.IsInStringArray(input.Type, GetSenderTypes()) {
		return input, httperrors.NewInputParameterError("unkown type %s allow: %s", input.Type, GetSenderTypes())
	}
	if !utils.IsInStringArray(input.Attribution, []string{api.CONFIG_ATTRIBUTION_SYSTEM, api.CONFIG_ATTRIBUTION_DOMAIN}) {
		return input, httperrors.NewInputParameterError("invalid attribution, need %q or %q", api.CONFIG_ATTRIBUTION_SYSTEM, api.CONFIG_ATTRIBUTION_DOMAIN)
	}
	if input.Content == nil {
		return input, httperrors.NewMissingParameterError("content")
	}
	config, err := cm.Config(input.Type, input.ProjectDomainId, input.Attribution)
	if config != nil {
		return input, httperrors.NewDuplicateResourceError("duplicate type %q", input.Type)
	}
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return input, err
	}
	driver := GetDriver(input.Type)
	// validate
	if input.Type != api.MOBILE {
		message, err := driver.ValidateConfig(ctx, api.NotifyConfig{
			SNotifyConfigContent: *input.Content,
			Attribution:          input.Attribution,
			DomainId:             input.ProjectDomainId,
		})
		if err != nil {
			return input, errors.Wrapf(err, message)
		}
	}
	if len(input.Name) == 0 {
		input.Name = input.Type
	}
	return input, nil
}

func (manager *SConfigManager) GetPropertyCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	q := manager.Query()
	configs := []SConfig{}
	err := db.FetchModelObjects(manager, q, &configs)
	if err != nil {
		return nil, err
	}
	ret := struct {
		System []string            `json:"system,allowempty"`
		Domain map[string][]string `json:"domain,allowempty"`
	}{
		System: []string{},
		Domain: map[string][]string{},
	}

	for _, config := range configs {
		switch config.Attribution {
		case api.CONFIG_ATTRIBUTION_SYSTEM:
			ret.System = append(ret.System, config.Type)
		case api.CONFIG_ATTRIBUTION_DOMAIN:
			_, ok := ret.Domain[config.DomainId]
			if !ok {
				ret.Domain[config.DomainId] = []string{}
			}
			ret.Domain[config.DomainId] = append(ret.Domain[config.DomainId], config.Type)
		}
	}
	return jsonutils.Marshal(ret), nil
}

func (c *SConfig) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SDomainLevelResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	err := c.StartRepullSubcontactTask(ctx, userCred, false)
	if err != nil {
		log.Errorf("unable to StartRepullSubcontactTask: %v", err)
	}
	driver := GetDriver(c.Type)
	driver.RegisterConfig(*c)
}

func (c *SConfig) GetNotifyConfig() api.NotifyConfig {
	return api.NotifyConfig{
		SNotifyConfigContent: *c.Content,
		Attribution:          c.Attribution,
		DomainId:             c.DomainId,
	}
}

func (c *SConfig) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ConfigUpdateInput) (api.ConfigUpdateInput, error) {
	q := ConfigManager.Query()
	q = q.Equals("type", c.Type)
	confs := []SConfig{}
	err := db.FetchModelObjects(ConfigManager, q, &confs)
	if err != nil {
		return input, errors.Wrapf(err, "config type:%s", c.Type)
	}
	if len(confs) == 0 {
		return input, errors.Wrapf(errors.ErrNotFound, "config type:%s", c.Type)
	}

	// check if changed
	if input.Content != nil {
		driver := GetDriver(c.Type)
		if c.Type != api.MOBILE {
			message, err := driver.ValidateConfig(ctx, api.NotifyConfig{
				SNotifyConfigContent: *input.Content,
				Attribution:          c.Attribution,
				DomainId:             c.DomainId,
			})
			if err != nil {
				return input, errors.Wrapf(err, message)
			}
		}
	}
	return input, nil
}

func (c *SConfig) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	config := c.GetNotifyConfig()
	ConfigMap[fmt.Sprintf("%s-%s", c.Type, c.DomainId)] = SConfig{
		Content: &config.SNotifyConfigContent,
	}
	err := c.StartRepullSubcontactTask(ctx, userCred, false)
	if err != nil {
		log.Errorf("unable to StartRepullSubcontactTask: %v", err)
	}
}

func (c *SConfig) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	c.SStandaloneResourceBase.PreDelete(ctx, userCred)
	key := fmt.Sprintf("%s-%s", c.Type, c.DomainId)
	if c.Type == api.MOBILE || c.Type == api.EMAIL {
		key = c.Type
	}
	delete(ConfigMap, key)
}

func (c *SConfig) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := c.SStandaloneResourceBase.CustomizeDelete(ctx, userCred, query, data)
	if err != nil {
		return err
	}
	return c.StartRepullSubcontactTask(ctx, userCred, true)
}

func (c *SConfig) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (c *SConfig) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	delete(ConfigMap, fmt.Sprintf("%s-%s", c.Type, c.DomainId))
	return c.SDomainLevelResourceBase.Delete(ctx, userCred)
}

func (self *SConfig) GetReceivers() ([]SReceiver, error) {
	subq := SubContactManager.Query("receiver_id").Equals("type", self.Type).SubQuery()
	q := ReceiverManager.Query()
	if self.Attribution == api.CONFIG_ATTRIBUTION_DOMAIN {
		q = q.Equals("domain_id", self.DomainId)
	} else {
		// The system-level config update should not affect the receiver under the domain with config
		configq := ConfigManager.Query("domain_id").Equals("type", self.Type).Equals("attribution", api.CONFIG_ATTRIBUTION_DOMAIN).SubQuery()
		q = q.NotIn("domain_id", configq)
	}
	q = q.Join(subq, sqlchemy.Equals(q.Field("id"), subq.Field("receiver_id")))
	ret := []SReceiver{}
	err := db.FetchModelObjects(ReceiverManager, q, &ret)
	return ret, err
}

func (c *SConfig) StartRepullSubcontactTask(ctx context.Context, userCred mcclient.TokenCredential, del bool) error {
	taskData := jsonutils.NewDict()
	if del {
		taskData.Set("deleted", jsonutils.JSONTrue)
	}
	task, err := taskman.TaskManager.NewTask(ctx, "RepullSuncontactTask", c, userCred, taskData, "", "")
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

func (cm *SConfigManager) allContactType(domainid string) ([]string, error) {
	q := cm.Query("type")
	q = q.Equals("domain_id", domainid)
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
	q, err := self.SDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, input.DomainLevelResourceListInput)
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

func (manager *SConfigManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	return manager.SDomainLevelResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
}

func (cm *SConfigManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ConfigDetails {
	sRows := cm.SDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]api.ConfigDetails, len(objs))
	for i := range rows {
		rows[i].DomainLevelResourceDetails = sRows[i]
	}
	return rows
}

func (cm *SConfigManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := cm.SDomainLevelResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return q, nil
	}
	return q, nil
}

func (cm *SConfigManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ConfigListInput) (*sqlchemy.SQuery, error) {
	q, err := cm.SDomainLevelResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainLevelResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (cm *SConfigManager) PerformValidate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ConfigValidateInput) (api.ConfigValidateOutput, error) {
	output := api.ConfigValidateOutput{}
	if !utils.IsInStringArray(input.Type, GetSenderTypes()) {
		return output, httperrors.NewInputParameterError("unkown type %q", input.Type)
	}
	if input.Content == nil {
		return output, httperrors.NewMissingParameterError("content")
	}
	// validate
	driver := GetDriver(input.Type)
	message, err := driver.ValidateConfig(ctx, api.NotifyConfig{
		SNotifyConfigContent: *input.Content,
		DomainId:             userCred.GetDomainId(),
	})
	if err != nil {
		return output, errors.Wrapf(err, message)
	}
	output.IsValid = true
	output.Message = message
	return output, nil
}

func (confManager *SConfigManager) InitializeData() error {
	q := confManager.Query()
	res := []SConfig{}
	err := db.FetchModelObjects(confManager, q, &res)
	if err != nil {
		return errors.Wrap(err, "init configMap err")
	}
	ConfigMap = make(map[string]SConfig)
	for _, config := range res {
		driver := GetDriver(config.Type)
		driver.RegisterConfig(config)
		err := driver.GetAccessToken(context.Background(), config.DomainId)
		if err != nil {
			session := auth.GetAdminSession(context.Background(), options.Options.Region)
			logclient.AddSimpleActionLog(&config, logclient.ACT_INIT_NOTIFY_CONFIGMAP, err, session.GetToken(), false)
		}
	}
	log.Infoln("init ConfigMap:", jsonutils.Marshal(ConfigMap))
	return nil
}

func (cm *SConfigManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	switch scope {
	case rbacscope.ScopeDomain, rbacscope.ScopeProject:
		q = q.Equals("attribution", api.CONFIG_ATTRIBUTION_DOMAIN)
		if owner != nil {
			q = q.Equals("domain_id", owner.GetProjectDomainId())
		}
	}
	return q
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

func (self *SConfigManager) Config(contactType, domainId string, attribution string) (*SConfig, error) {
	q := self.Query()
	q = q.Equals("type", contactType).Equals("attribution", attribution)
	if attribution == api.CONFIG_ATTRIBUTION_DOMAIN {
		q = q.Equals("domain_id", domainId)
	}
	var config SConfig
	err := q.First(&config)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to fetch SConfig by type %s and domain %s", contactType, domainId)
	}
	return &config, nil
}

func (self *SConfigManager) HasSystemConfig(contactType string) (bool, error) {
	q := self.Query().Equals("type", contactType).Equals("attribution", api.CONFIG_ATTRIBUTION_SYSTEM)
	c, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return c > 0, nil
}
