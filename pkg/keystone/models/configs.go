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
	"database/sql"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type SConfigOptionManager struct {
	db.SResourceBaseManager
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
		IsSensitive: true,
	}
	SensitiveConfigManager.SetVirtualObject(SensitiveConfigManager)
	WhitelistedConfigManager = &SConfigOptionManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SConfigOption{},
			"whitelisted_config",
			"whitelisted_config",
			"whitelisted_configs",
		),
		IsSensitive: false,
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

	ResType string `width:"32" charset:"ascii" nullable:"false" default:"identity_provider" primary:"true"`
	ResId   string `name:"domain_id" width:"64" charset:"ascii" primary:"true"`
	Group   string `width:"255" charset:"utf8" primary:"true"`
	Option  string `width:"255" charset:"utf8" primary:"true"`

	Value jsonutils.JSONObject `nullable:"false"`
}

func (manager *SConfigOptionManager) fetchConfigs(model db.IModel, groups []string, options []string) (TConfigOptions, error) {
	q := manager.Query().Equals("res_type", model.Keyword()).Equals("domain_id", model.GetId())
	if len(groups) > 0 {
		if len(groups) == 1 {
			q = q.Equals("group", groups[0])
		} else {
			q = q.In("group", groups)
		}
	}
	if len(options) > 0 {
		if len(options) == 1 {
			q = q.Equals("option", options[0])
		} else {
			q = q.In("option", options)
		}
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

func (manager *SConfigOptionManager) deleteConfigs(model db.IModel) error {
	return manager.syncConfigs(model, nil)
}

func (manager *SConfigOptionManager) updateConfigs(newOpts TConfigOptions) error {
	for i := range newOpts {
		err := manager.TableSpec().InsertOrUpdate(&newOpts[i])
		if err != nil {
			return errors.Wrap(err, "Insert")
		}
	}
	return nil
}

func (manager *SConfigOptionManager) removeConfigs(model db.IModel, newOpts TConfigOptions) error {
	oldOpts, err := manager.fetchConfigs(model, nil, nil)
	if err != nil {
		return errors.Wrap(err, "fetchOldConfigs")
	}
	_, updated1, _, _ := compareConfigOptions(oldOpts, newOpts)
	for i := range updated1 {
		_, err := db.Update(&updated1[i], func() error {
			return updated1[i].MarkDelete()
		})
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	return nil
}

func (manager *SConfigOptionManager) syncConfigs(model db.IModel, newOpts TConfigOptions) error {
	oldOpts, err := manager.fetchConfigs(model, nil, nil)
	if err != nil {
		return errors.Wrap(err, "fetchOldConfigs")
	}
	deleted, updated1, updated2, added := compareConfigOptions(oldOpts, newOpts)
	for i := range deleted {
		_, err := db.Update(&deleted[i], func() error {
			return deleted[i].MarkDelete()
		})
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	for i := range updated1 {
		_, err = db.Update(&updated1[i], func() error {
			updated1[i].Value = updated2[i].Value
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "Update")
		}
	}
	for i := range added {
		err = manager.TableSpec().InsertOrUpdate(&added[i])
		if err != nil {
			return errors.Wrap(err, "Insert")
		}
	}
	return nil
}

func getConfigOptions(conf api.TConfigs, model db.IModel, whiteList map[string][]string, blackList map[string][]string, sensitiveList map[string][]string) (TConfigOptions, TConfigOptions) {
	options := make(TConfigOptions, 0)
	sensitive := make(TConfigOptions, 0)
	for group, groupConf := range conf {
		for optKey, optVal := range groupConf {
			opt := SConfigOption{}
			opt.ResType = model.Keyword()
			opt.ResId = model.GetId()
			opt.Group = group
			opt.Option = optKey
			opt.Value = optVal
			if v, ok := sensitiveList[group]; ok && utils.IsInStringArray(optKey, v) {
				sensitive = append(sensitive, opt)
			} else {
				if whiteList != nil {
					if v, ok := whiteList[group]; ok && utils.IsInStringArray(optKey, v) {
						options = append(options, opt)
					}
				} else if blackList != nil {
					if v, ok := blackList[group]; ok && utils.IsInStringArray(optKey, v) {
					} else {
						options = append(options, opt)
					}
				} else {
					options = append(options, opt)
				}
			}
		}
	}
	sort.Sort(options)
	sort.Sort(sensitive)
	return options, sensitive
}

type TConfigOptions []SConfigOption

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
			updated1 = append(updated1, opts1[i])
			updated2 = append(updated2, opts2[j])
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

func GetConfigs(model db.IModel, all bool) (api.TConfigs, error) {
	opts, err := WhitelistedConfigManager.fetchConfigs(model, nil, nil)
	if err != nil {
		return nil, err
	}
	if all {
		opts2, err := SensitiveConfigManager.fetchConfigs(model, nil, nil)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opts2...)
	}
	return config2map(opts), nil
}

func saveConfigs(action string, model db.IModel, opts api.TConfigs, whiteList map[string][]string, blackList map[string][]string, sensitiveConfs map[string][]string) error {
	whiteListedOpts, sensitiveOpts := getConfigOptions(opts, model, whiteList, blackList, sensitiveConfs)
	if action == "update" {
		err := WhitelistedConfigManager.updateConfigs(whiteListedOpts)
		if err != nil {
			return errors.Wrap(err, "WhitelistedConfigManager.updateConfig")
		}
		err = SensitiveConfigManager.updateConfigs(sensitiveOpts)
		if err != nil {
			return errors.Wrap(err, "SensitiveConfigManager.updateConfig")
		}
	} else if action == "remove" {
		err := WhitelistedConfigManager.removeConfigs(model, whiteListedOpts)
		if err != nil {
			return errors.Wrap(err, "WhitelistedConfigManager.updateConfig")
		}
		err = SensitiveConfigManager.removeConfigs(model, sensitiveOpts)
		if err != nil {
			return errors.Wrap(err, "SensitiveConfigManager.updateConfig")
		}
	} else {
		err := WhitelistedConfigManager.syncConfigs(model, whiteListedOpts)
		if err != nil {
			return errors.Wrap(err, "WhitelistedConfigManager.syncConfig")
		}
		err = SensitiveConfigManager.syncConfigs(model, sensitiveOpts)
		if err != nil {
			return errors.Wrap(err, "SensitiveConfigManager.syncConfig")
		}
	}
	return nil
}

type dbServiceConfigSession struct {
	config  *jsonutils.JSONDict
	service *SService
}

func NewServiceConfigSession() common_options.IServiceConfigSession {
	return &dbServiceConfigSession{}
}

func (s *dbServiceConfigSession) Merge(opts interface{}, serviceType string, serviceVersion string) bool {
	merged := false
	s.config = jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	s.service, _ = ServiceManager.fetchServiceByType(serviceType)
	if s.service != nil {
		serviceConf, err := GetConfigs(s.service, false)
		if err != nil {
			log.Errorf("GetConfigs for %s fail: %s", serviceType, err)
		} else {
			serviceConfJson := jsonutils.Marshal(serviceConf["default"])
			s.config.Update(serviceConfJson)
			merged = true
		}
	}
	commonService, _ := ServiceManager.fetchServiceByType(consts.COMMON_SERVICE)
	if commonService != nil {
		commonConf, err := GetConfigs(commonService, false)
		if err != nil {
			log.Errorf("GetConfigs for %s fail: %s", consts.COMMON_SERVICE, err)
		} else {
			commonConfJson := jsonutils.Marshal(commonConf["default"])
			s.config.Update(commonConfJson)
			merged = true
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
	nconf := jsonutils.NewDict()
	nconf.Add(s.config, "default")
	tconf := api.TConfigs{}
	err := nconf.Unmarshal(tconf)
	if err != nil {
		log.Errorf("nconf.Unmarshal fail %s", err)
		return
	}
	err = saveConfigs("", s.service, tconf, nil, api.ServiceBlacklistOptionMap, nil)
	if err != nil {
		log.Errorf("saveConfigs fail %s", err)
		return
	}
}
