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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SConfigOptionManager struct {
	db.SResourceBaseManager
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
	}
	SensitiveConfigManager.SetVirtualObject(SensitiveConfigManager)
	WhitelistedConfigManager = &SConfigOptionManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SConfigOption{},
			"whitelisted_config",
			"whitelisted_config",
			"whitelisted_configs",
		),
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

	DomainId string `width:"64" charset:"ascii" primary:"true"`
	Group    string `width:"255" charset:"utf8" primary:"true"`
	Option   string `width:"255" charset:"utf8" primary:"true"`

	Value jsonutils.JSONObject `nullable:"false"`
}

func (manager *SConfigOptionManager) fetchConfigs(domainId string, groups []string, options []string) (TConfigOptions, error) {
	q := manager.Query().Equals("domain_id", domainId)
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

func config2map(opts []SConfigOption) TDomainConfigs {
	conf := make(TDomainConfigs)
	for i := range opts {
		opt := opts[i]
		if _, ok := conf[opt.Group]; !ok {
			conf[opt.Group] = make(map[string]jsonutils.JSONObject)
		}
		conf[opt.Group][opt.Option] = opt.Value
	}
	return conf
}

func (manager *SConfigOptionManager) deleteConfig(ctx context.Context, userCred mcclient.TokenCredential, domainId string) error {
	return manager.syncConfig(ctx, userCred, domainId, nil)
}

func (manager *SConfigOptionManager) syncConfig(ctx context.Context, userCred mcclient.TokenCredential, domainId string, newOpts TConfigOptions) error {
	oldOpts, err := manager.fetchConfigs(domainId, nil, nil)
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

type TDomainConfigs map[string]map[string]jsonutils.JSONObject

func (conf TDomainConfigs) getConfigOptions(domainId string, sensitiveList map[string]string) (TConfigOptions, TConfigOptions) {
	options := make(TConfigOptions, 0)
	sensitive := make(TConfigOptions, 0)
	for group, groupConf := range conf {
		for optKey, optVal := range groupConf {
			opt := SConfigOption{}
			opt.DomainId = domainId
			opt.Group = group
			opt.Option = optKey
			opt.Value = optVal
			if v, ok := sensitiveList[group]; ok && v == optKey {
				sensitive = append(sensitive, opt)
			} else {
				options = append(options, opt)
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
