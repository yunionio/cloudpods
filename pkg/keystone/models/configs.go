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
	"sort"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

// +onecloud:swagger-gen-ignore
type SConfigOptionManager struct {
	db.SResourceBaseManager
	db.SRecordChecksumResourceBaseManager
	IsSensitive bool
}

var (
	SensitiveConfigManager   *SConfigOptionManager
	WhitelistedConfigManager *SConfigOptionManager
)

func init() {
	SensitiveConfigManager = &SConfigOptionManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SConfigOption{},
			"sensitive_config",
			"sensitive_config",
			"sensitive_configs",
		),
		SRecordChecksumResourceBaseManager: *db.NewRecordChecksumResourceBaseManager(),
		IsSensitive:                        true,
	}
	SensitiveConfigManager.SetVirtualObject(SensitiveConfigManager)

	WhitelistedConfigManager = &SConfigOptionManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SConfigOption{},
			"whitelisted_config",
			"whitelisted_config",
			"whitelisted_configs",
		),
		SRecordChecksumResourceBaseManager: *db.NewRecordChecksumResourceBaseManager(),
		IsSensitive:                        false,
	}
	WhitelistedConfigManager.SetVirtualObject(WhitelistedConfigManager)
}

/*
+-----------+--------------+------+-----+---------+-------+
| Field     | Type         | Null | Key | Default | Extra |
+-----------+--------------+------+-----+---------+-------+
| domain_id | varchar(64)  | NO   | PRI | NULL    |       |
| group     | varchar(255) | NO   | PRI | NULL    |       |
| option    | varchar(255) | NO   | PRI | NULL    |       |
| value     | text         | NO   |     | NULL    |       |
+-----------+--------------+------+-----+---------+-------+
*/

type SConfigOption struct {
	db.SResourceBase
	db.SRecordChecksumResourceBase

	ResType string `width:"32" charset:"ascii" nullable:"false" default:"identity_provider" primary:"true"`
	ResId   string `name:"domain_id" width:"64" charset:"ascii" primary:"true"`
	Group   string `width:"191" charset:"utf8" primary:"true"`
	Option  string `width:"191" charset:"utf8" primary:"true"`

	Value jsonutils.JSONObject `nullable:"false"`
}

func (manager *SConfigOptionManager) fetchConfigs(model db.IModel, groups []string, options []string) (TConfigOptions, error) {
	return manager.fetchConfigs2(model.Keyword(), model.GetId(), groups, options)
}

func (manager *SConfigOptionManager) fetchConfigs2(resType string, resId string, groups []string, options []string) (TConfigOptions, error) {
	q := manager.Query()
	if len(resType) > 0 {
		q = q.Equals("res_type", resType)
	}
	if len(resId) > 0 {
		q = q.Equals("domain_id", resId)
	}
	if len(groups) > 0 {
		q = q.In("group", groups)
	}
	if len(options) > 0 {
		q = q.In("option", options)
	}
	opts := make(TConfigOptions, 0)
	err := db.FetchModelObjects(manager, q, &opts)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	sort.Sort(opts)
	return opts, nil
}

func config2map(opts []SConfigOption) api.TConfigs {
	conf := make(api.TConfigs)
	for i := range opts {
		opt := opts[i]
		if _, ok := conf[opt.Group]; !ok {
			conf[opt.Group] = make(map[string]jsonutils.JSONObject)
		}
		conf[opt.Group][opt.Option] = opt.Value
	}
	return conf
}

func (manager *SConfigOptionManager) deleteConfigs(model db.IModel) ([]sChangeOption, error) {
	return manager.syncConfigs(model, nil)
}

type sChangeOption struct {
	Group  string               `json:"group"`
	Option string               `json:"option"`
	Value  jsonutils.JSONObject `json:"value"`
	OValue jsonutils.JSONObject `json:"ovalue"`
}

func (manager *SConfigOptionManager) updateConfigs(model db.IModel, newOpts TConfigOptions) ([]sChangeOption, error) {
	return manager.syncRemoveConfigs(model, newOpts, false)
}

func (manager *SConfigOptionManager) removeConfigs(model db.IModel, newOpts TConfigOptions) ([]sChangeOption, error) {
	oldOpts, err := manager.fetchConfigs(model, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "fetchOldConfigs")
	}
	changed := make([]sChangeOption, 0)
	for i := range newOpts {
		newOpts[i].Value = nil
	}
	_, updated1, _, _ := compareConfigOptions(oldOpts, newOpts)
	for i := range updated1 {
		_, err := db.Update(&updated1[i], func() error {
			return updated1[i].MarkDelete()
		})
		if err != nil {
			return changed, errors.Wrap(err, "Delete")
		}
		changed = append(changed, sChangeOption{
			Group:  updated1[i].Group,
			Option: updated1[i].Option,
			OValue: updated1[i].Value,
		})
	}
	return changed, nil
}

func (manager *SConfigOptionManager) syncConfigs(model db.IModel, newOpts TConfigOptions) ([]sChangeOption, error) {
	return manager.syncRemoveConfigs(model, newOpts, true)
}

func (manager *SConfigOptionManager) syncRemoveConfigs(model db.IModel, newOpts TConfigOptions, remove bool) ([]sChangeOption, error) {
	oldOpts, err := manager.fetchConfigs(model, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "fetchOldConfigs")
	}
	changed := make([]sChangeOption, 0)
	deleted, updated1, updated2, added := compareConfigOptions(oldOpts, newOpts)
	if remove {
		for i := range deleted {
			_, err := db.Update(&deleted[i], func() error {
				return deleted[i].MarkDelete()
			})
			if err != nil {
				return changed, errors.Wrap(err, "Delete")
			}
			changed = append(changed, sChangeOption{
				Group:  deleted[i].Group,
				Option: deleted[i].Option,
				OValue: deleted[i].Value,
			})
		}
	}
	for i := range updated1 {
		ovalue := updated1[i].Value
		_, err = db.Update(&updated1[i], func() error {
			updated1[i].Value = updated2[i].Value
			return nil
		})
		if err != nil {
			return changed, errors.Wrap(err, "Update")
		}
		changed = append(changed, sChangeOption{
			Group:  updated1[i].Group,
			Option: updated1[i].Option,
			OValue: ovalue,
			Value:  updated2[i].Value,
		})
	}
	for i := range added {
		err = manager.TableSpec().InsertOrUpdate(context.TODO(), &added[i])
		if err != nil {
			return changed, errors.Wrap(err, "Insert")
		}
		changed = append(changed, sChangeOption{
			Group:  added[i].Group,
			Option: added[i].Option,
			Value:  added[i].Value,
		})
	}
	return changed, nil
}

func filterOptions(opts TConfigOptions, whiteList map[string][]string, blackList map[string][]string) TConfigOptions {
	if whiteList == nil && blackList == nil {
		return opts
	}
	retOpts := make(TConfigOptions, 0)
	for _, opt := range opts {
		if whiteList != nil {
			if v, ok := whiteList[opt.Group]; ok && utils.IsInStringArray(opt.Option, v) {
				retOpts = append(retOpts, opt)
			}
		} else if blackList != nil {
			if v, ok := blackList[opt.Group]; ok && utils.IsInStringArray(opt.Option, v) {
			} else {
				retOpts = append(retOpts, opt)
			}
		}
	}
	return retOpts
}

func getConfigOptions(conf api.TConfigs, model db.IModel, sensitiveList map[string][]string) (TConfigOptions, TConfigOptions) {
	options := make(TConfigOptions, 0)
	sensitive := make(TConfigOptions, 0)
	for group, groupConf := range conf {
		for optKey, optVal := range groupConf {
			opt := &SConfigOption{
				ResType: model.Keyword(),
				ResId:   model.GetId(),
				Group:   group,
				Option:  optKey,
				Value:   optVal,
			}
			opt.SetModelManager(WhitelistedConfigManager, opt)
			if v, ok := sensitiveList[group]; ok && utils.IsInStringArray(optKey, v) {
				sensitive = append(sensitive, *opt)
			} else {
				options = append(options, *opt)
			}
		}
	}
	sort.Sort(options)
	sort.Sort(sensitive)
	return options, sensitive
}

type TConfigOptions []SConfigOption

func (opts TConfigOptions) Validate() error {
	for _, opt := range opts {
		if opt.Option == "time_zone" && opt.Group == "default" && !gotypes.IsNil(opt.Value) {
			timeZone, _ := opt.Value.GetString()
			_, err := time.LoadLocation(timeZone)
			if err != nil {
				return httperrors.NewInputParameterError("invalid time_zone %s", timeZone)
			}
		}
	}
	return nil
}

func (opts TConfigOptions) Len() int {
	return len(opts)
}

func (opts TConfigOptions) Swap(i, j int) {
	opts[i], opts[j] = opts[j], opts[i]
}

func (opts TConfigOptions) Less(i, j int) bool {
	return compareConfigOption(opts[i], opts[j]) < 0
}

func compareConfigOption(opt1, opt2 SConfigOption) int {
	if opt1.Group < opt2.Group {
		return -1
	} else if opt1.Group > opt2.Group {
		return 1
	}
	if opt1.Option < opt2.Option {
		return -1
	} else if opt1.Option > opt2.Option {
		return 1
	}
	return 0
}

func compareConfigOptions(opts1, opts2 TConfigOptions) (deleted, updated1, updated2, added TConfigOptions) {
	deleted = make(TConfigOptions, 0)
	updated1 = make(TConfigOptions, 0)
	updated2 = make(TConfigOptions, 0)
	added = make(TConfigOptions, 0)

	i := 0
	j := 0

	for i < len(opts1) && j < len(opts2) {
		cmp := compareConfigOption(opts1[i], opts2[j])
		if cmp < 0 {
			deleted = append(deleted, opts1[i])
			i += 1
		} else if cmp > 0 {
			added = append(added, opts2[j])
			j += 1
		} else {
			if (opts1[i].Value == nil && opts2[j].Value != nil) ||
				(opts1[i].Value != nil && opts2[j].Value == nil) ||
				(opts1[i].Value != nil && opts2[j].Value != nil && !opts1[i].Value.Equals(opts2[j].Value)) {
				updated1 = append(updated1, opts1[i])
				updated2 = append(updated2, opts2[j])
			}
			i += 1
			j += 1
		}
	}
	if i < len(opts1) {
		deleted = append(deleted, opts1[i:]...)
	}
	if j < len(opts2) {
		added = append(added, opts2[j:]...)
	}
	return
}

func (manager *SConfigOptionManager) getDriver(idpId string) (string, error) {
	idp, _ := db.NewModelObject(IdentityProviderManager)
	idp.(*SIdentityProvider).Id = idpId
	opts, err := manager.fetchConfigs(idp, []string{"identity"}, []string{"driver"})
	if err != nil {
		return "", errors.Wrap(err, "WhitelistedConfigManager.fetchConfigs")
	}
	if len(opts) == 1 {
		return opts[0].Value.GetString()
	}
	return api.IdentityDriverSQL, nil
}

func GetConfigs(model db.IModel, sensitive bool, whiteList, blackList map[string][]string) (api.TConfigs, error) {
	opts, err := WhitelistedConfigManager.fetchConfigs(model, nil, nil)
	if err != nil {
		return nil, err
	}
	opts = filterOptions(opts, whiteList, blackList)
	if sensitive {
		opts2, err := SensitiveConfigManager.fetchConfigs(model, nil, nil)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opts2...)
	}
	if len(opts) == 0 {
		return nil, nil
	}

	return config2map(opts), nil
}

func saveConfigs(userCred mcclient.TokenCredential, action string, model db.IModel, opts api.TConfigs, whiteList map[string][]string, blackList map[string][]string, sensitiveConfs map[string][]string) (bool, error) {
	var err error
	changed := make([]sChangeOption, 0)
	changedSensitive := make([]sChangeOption, 0)
	whiteListedOpts, sensitiveOpts := getConfigOptions(opts, model, sensitiveConfs)
	whiteListedOpts = filterOptions(whiteListedOpts, whiteList, blackList)
	if action == "update" {
		err = whiteListedOpts.Validate()
		if err != nil {
			return false, err
		}
		changed, err = WhitelistedConfigManager.updateConfigs(model, whiteListedOpts)
		if err != nil {
			return false, errors.Wrap(err, "WhitelistedConfigManager.updateConfig")
		}
		changedSensitive, err = SensitiveConfigManager.updateConfigs(model, sensitiveOpts)
		if err != nil {
			return false, errors.Wrap(err, "SensitiveConfigManager.updateConfig")
		}
	} else if action == "remove" {
		changed, err = WhitelistedConfigManager.removeConfigs(model, whiteListedOpts)
		if err != nil {
			return false, errors.Wrap(err, "WhitelistedConfigManager.updateConfig")
		}
		changedSensitive, err = SensitiveConfigManager.removeConfigs(model, sensitiveOpts)
		if err != nil {
			return false, errors.Wrap(err, "SensitiveConfigManager.updateConfig")
		}
	} else {
		changed, err = WhitelistedConfigManager.syncConfigs(model, whiteListedOpts)
		if err != nil {
			return false, errors.Wrap(err, "WhitelistedConfigManager.syncConfig")
		}
		changedSensitive, err = SensitiveConfigManager.syncConfigs(model, sensitiveOpts)
		if err != nil {
			return false, errors.Wrap(err, "SensitiveConfigManager.syncConfig")
		}
	}
	maskedValue := jsonutils.NewString("*")
	for i := range changedSensitive {
		if changedSensitive[i].OValue != nil {
			changedSensitive[i].OValue = maskedValue
		}
		if changedSensitive[i].Value != nil {
			changedSensitive[i].Value = maskedValue
		}
	}
	changed = append(changed, changedSensitive...)
	if len(changed) > 0 {
		notes := jsonutils.Marshal(changed)
		if userCred == nil {
			userCred = getDefaultAdminCred()
		} else {
			logclient.AddSimpleActionLog(model, logclient.ACT_CHANGE_CONFIG, notes, userCred, true)
		}
		db.OpsLog.LogEvent(model, db.ACT_CHANGE_CONFIG, notes, userCred)
	}
	return len(changed) > 0, nil
}

type dbServiceConfigSession struct {
	config  *jsonutils.JSONDict
	service *SService

	commonService *SService
}

func NewServiceConfigSession() common_options.IServiceConfigSession {
	return &dbServiceConfigSession{}
}

func (s *dbServiceConfigSession) Merge(opts interface{}, serviceType string, serviceVersion string) bool {
	merged := false
	s.config = jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	s.service, _ = ServiceManager.fetchServiceByType(serviceType)
	if s.service != nil {
		serviceConf, err := GetConfigs(s.service, false, nil, nil)
		if err != nil {
			log.Errorf("GetConfigs for %s fail: %s", serviceType, err)
		} else if serviceConf != nil {
			serviceConfJson := jsonutils.Marshal(serviceConf["default"])
			s.config.Update(serviceConfJson)
			merged = true
		} else {
			// not initialized
			// uploadConfig(s.service, s.config)
		}
	}
	s.commonService, _ = ServiceManager.fetchServiceByType(consts.COMMON_SERVICE)
	if s.commonService != nil {
		commonConf, err := GetConfigs(s.commonService, false, nil, nil)
		if err != nil {
			log.Errorf("GetConfigs for %s fail: %s", consts.COMMON_SERVICE, err)
		} else if commonConf != nil {
			commonConfJson := jsonutils.Marshal(commonConf["default"])
			s.config.Update(commonConfJson)
			merged = true
		} else {
			// common not initialized
			// uploadConfig(commonService, s.config)
		}
	}
	if merged {
		err := s.config.Unmarshal(opts)
		if err == nil {
			return true
		}
		log.Errorf("s.config.Unmarshal fail %s", err)
	}
	return false
}

func (s *dbServiceConfigSession) Upload() {
	if s.service == nil {
		return
	}
	uploadConfig(s.service, s.config)
	uploadConfig(s.commonService, s.config)
}

func (s *dbServiceConfigSession) IsRemote() bool {
	return false
}

func uploadConfig(service *SService, config jsonutils.JSONObject) {
	nconf := jsonutils.NewDict()
	nconf.Add(config, "default")
	tconf := api.TConfigs{}
	err := nconf.Unmarshal(tconf)
	if err != nil {
		log.Errorf("nconf.Unmarshal fail %s", err)
		return
	}
	if service.isCommonService() {
		_, err = saveConfigs(nil, "", service, tconf, api.CommonWhitelistOptionMap, nil, nil)
	} else {
		_, err = saveConfigs(nil, "", service, tconf, nil, api.MergeServiceConfigOptions(api.CommonWhitelistOptionMap, api.ServiceBlacklistOptionMap), nil)
	}
	if err != nil {
		log.Errorf("saveConfigs fail %s", err)
		return
	}
}
