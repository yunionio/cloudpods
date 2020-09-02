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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	notifyv2 "yunion.io/x/onecloud/pkg/notify"
	"yunion.io/x/onecloud/pkg/notify/oldmodels"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SConfigManager struct {
	db.SStandaloneResourceBaseManager
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

	Type    string               `width:"15" nullable:"false" create:"required" get:"admin" list:"admin"`
	Content jsonutils.JSONObject `nullable:"false" create:"required" update:"admin" get:"admin" list:"admin"`
}

func (cm *SConfigManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ConfigCreateInput) (api.ConfigCreateInput, error) {
	var err error
	input.StandaloneResourceCreateInput, err = cm.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}
	if !utils.IsInStringArray(input.Type, []string{api.EMAIL, api.MOBILE, api.DINGTALK, api.FEISHU, api.WEBCONSOLE, api.WORKWX, api.FEISHU_ROBOT, api.DINGTALK_ROBOT, api.WORKWX_ROBOT}) {
		return input, httperrors.NewInputParameterError("unkown type %q", input.Type)
	}
	if input.Content == nil {
		return input, httperrors.NewMissingParameterError("content")
	}
	config, err := cm.GetConfigByType(input.Type)
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

func (c *SConfig) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ConfigUpdateInput) (api.ConfigUpdateInput, error) {
	// validate
	configs := make(map[string]string)
	err := input.Content.Unmarshal(&configs)
	if err != nil {
		return input, err
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

func (c *SConfig) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	configMap := make(map[string]string)
	err := c.Content.Unmarshal(&configMap)
	if err != nil {
		log.Errorf("unable to unmarshal: %v", err)
		return
	}
	NotifyService.RestartService(ctx, configMap, c.Type)
}

func (cm *SConfigManager) AllowPerformGetTypes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (cm *SConfigManager) PerformGetTypes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ConfigManagerGetTypesInput) (api.ConfigManagerGetTypesOutput, error) {
	output := api.ConfigManagerGetTypesOutput{}
	allContactType, err := cm.allContactType()
	if err != nil {
		return output, err
	}
	var judge func(string) bool
	switch input.Robot {
	case "only":
		judge = func(ctype string) bool {
			return strings.Contains(ctype, "robot")
		}
	case "yes":
		judge = func(ctype string) bool {
			return true
		}
	default:
		judge = func(ctype string) bool {
			return !strings.Contains(ctype, "robot")
		}
	}
	for _, ctype := range allContactType {
		if judge(ctype) {
			output.Types = append(output.Types, ctype)
		}
	}
	return output, nil
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
	q = q.NotEquals("type", api.WEBCONSOLE)
	if len(input.Type) > 0 {
		q.Filter(sqlchemy.Equals(q.Field("type"), input.Type))
	}
	return q, nil
}

func (cm *SConfigManager) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.ConfigDetails, error) {
	return api.ConfigDetails{}, nil
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
	rows := make([]api.ConfigDetails, len(objs))
	for i := range rows {
		rows[i].StandaloneResourceDetails = sRows[i]
	}
	return rows
}

func (cm *SConfigManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := cm.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
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
	return nil
}

// Fetch all SConfig struct which type is contactType.
func (self *SConfigManager) GetConfigByType(contactType string) (*SConfig, error) {
	var config SConfig
	q := self.Query()
	q.Filter(sqlchemy.Equals(q.Field("type"), contactType))
	err := q.First(&config)
	if err != nil {
		return nil, errors.Wrap(err, "fail to fetch SConfigs by type")
	}
	return &config, nil
}

func (self *SConfigManager) GetConfig(contactType string) (notifyv2.SConfig, error) {
	config, err := self.GetConfigByType(contactType)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]string)
	err = config.Content.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrap(err, "fail unmarshal config content")
	}
	return ret, nil
}

func (self *SConfigManager) SetConfig(contactType string, config notifyv2.SConfig) error {
	content := jsonutils.Marshal(config)
	sConfig := &SConfig{
		Type:    contactType,
		Content: content,
	}
	return self.TableSpec().InsertOrUpdate(context.Background(), sConfig)
}
