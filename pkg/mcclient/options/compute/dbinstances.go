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

package compute

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DBInstanceCreateOptions struct {
	NAME               string   `help:"DBInstance Name"`
	InstanceType       string   `help:"InstanceType for DBInstance"`
	VcpuCount          int      `help:"Core of cpu for DBInstance"`
	VmemSizeMb         int      `help:"Memory size of DBInstance"`
	Port               int      `help:"Port of DBInstance"`
	Category           string   `help:"Category of DBInstance"`
	Network            string   `help:"Network of DBInstance"`
	Address            string   `help:"Address of DBInstance"`
	Engine             string   `help:"Engine of DBInstance"`
	EngineVersion      string   `help:"EngineVersion of DBInstance Engine"`
	StorageType        string   `help:"StorageTyep of DBInstance"`
	Secgroup           string   `help:"Secgroup name or Id for DBInstance"`
	Zone               string   `help:"ZoneId or name for DBInstance"`
	DiskSizeGB         int      `help:"Storage size for DBInstance"`
	Duration           string   `help:"Duration for DBInstance"`
	AllowDelete        *bool    `help:"not lock dbinstance" `
	Tags               []string `help:"Tags info,prefix with 'user:', eg: user:project=default" json:"-"`
	DBInstancebackupId string   `help:"create dbinstance from backup" json:"dbinstancebackup_id"`
	MultiAz            bool     `help:"deploy rds with multi az"`
}

func (opts *DBInstanceCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	if opts.AllowDelete != nil && *opts.AllowDelete {
		params.Add(jsonutils.JSONFalse, "disable_delete")
	}
	Tagparams := jsonutils.NewDict()
	for _, tag := range opts.Tags {
		info := strings.Split(tag, "=")
		if len(info) == 2 {
			if len(info[0]) == 0 {
				return nil, fmt.Errorf("invalidate tag info %s", tag)
			}
			Tagparams.Add(jsonutils.NewString(info[1]), info[0])
		} else if len(info) == 1 {
			Tagparams.Add(jsonutils.NewString(info[0]), info[0])
		} else {
			return nil, fmt.Errorf("invalidate tag info %s", tag)
		}
	}
	params.Add(Tagparams, "__meta__")
	return params, nil
}

type DBInstanceListOptions struct {
	options.BaseListOptions
	BillingType string `help:"billing type" choices:"postpaid|prepaid"`
	IpAddr      []string
	SecgroupId  string
}

func (opts *DBInstanceListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type DBInstanceIdOptions struct {
	ID string `help:"DBInstance Id"`
}

func (opts *DBInstanceIdOptions) GetId() string {
	return opts.ID
}

func (opts *DBInstanceIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DBInstanceRenewOptions struct {
	DBInstanceIdOptions
	DURATION string `help:"Duration of renew, ADMIN only command"`
}

func (opts *DBInstanceRenewOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(opts.DURATION), "duration")
	return params, nil
}

type DBInstanceUpdateOptions struct {
	DBInstanceIdOptions
	Name        string
	Description string
	Delete      string `help:"Lock or not lock dbinstance" choices:"enable|disable"`
}

func (opts *DBInstanceUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	if len(opts.Delete) > 0 {
		if opts.Delete == "disable" {
			params.Add(jsonutils.JSONTrue, "disable_delete")
		} else {
			params.Add(jsonutils.JSONFalse, "disable_delete")
		}
	}
	return params, nil
}

type DBInstanceChangeConfigOptions struct {
	DBInstanceIdOptions
	DiskSizeGb   int64  `help:"Change DBInstance storage size"`
	VcpuCount    int64  `help:"Change DBInstance vcpu count"`
	VmemSizeMb   int64  `help:"Change DBInstance vmem size mb"`
	InstanceType string `help:"Change DBInstance instanceType"`
	Category     string `help:"Change DBInstance category"`
}

func (opts *DBInstanceChangeConfigOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Category) > 0 {
		params.Add(jsonutils.NewString(opts.Category), "category")
	}
	if len(opts.InstanceType) > 0 {
		params.Add(jsonutils.NewString(opts.InstanceType), "instance_type")
	}
	if opts.DiskSizeGb > 0 {
		params.Add(jsonutils.NewInt(opts.DiskSizeGb), "disk_size_gb")
	}
	if opts.VcpuCount > 0 {
		params.Add(jsonutils.NewInt(opts.VcpuCount), "vcpu_count")
	}
	if opts.VmemSizeMb > 0 {
		params.Add(jsonutils.NewInt(opts.VmemSizeMb), "vmeme_size_mb")
	}
	return params, nil
}

type DBInstancePublicConnectionOptions struct {
	DBInstanceIdOptions
	IS_OPEN string `help:"Open Or Close public connection" choices:"true|false"`
}

func (opts *DBInstancePublicConnectionOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]bool{"open": opts.IS_OPEN == "true"}), nil
}

type DBInstanceRecoveryOptions struct {
	DBInstanceIdOptions
	BACKUP    string
	Databases []string
}

func (opts *DBInstanceRecoveryOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("dbinstancebackup", jsonutils.NewString(opts.BACKUP))
	dbs := jsonutils.NewDict()
	for _, database := range opts.Databases {
		if len(database) > 0 {
			dbInfo := strings.Split(database, ":")
			if len(dbInfo) == 1 {
				dbs.Add(jsonutils.NewString(dbInfo[0]), dbInfo[0])
			} else if len(dbInfo) == 2 {
				dbs.Add(jsonutils.NewString(dbInfo[1]), dbInfo[0])
			} else {
				return nil, fmt.Errorf("Invalid dbinfo: %s", database)
			}
		}
	}
	if dbs.Length() > 0 {
		params.Add(dbs, "databases")
	}
	return params, nil
}

type DBInstanceDeleteOptions struct {
	DBInstanceIdOptions
	KeepBackup bool `help:"Keep dbinstance manual backup after delete dbinstance"`
}

func (opts *DBInstanceDeleteOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if opts.KeepBackup {
		params.Add(jsonutils.JSONTrue, "keep_backup")
	}
	return params, nil
}

type DBInstanceChangeOwnerOptions struct {
	DBInstanceIdOptions
	PROJECT string `help:"Project ID or change" json:"tenant"`
}

func (opts *DBInstanceChangeOwnerOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type DBInstanceRemoteUpdateOptions struct {
	DBInstanceIdOptions
	api.DBInstanceRemoteUpdateInput
}

func (opts *DBInstanceRemoteUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type DBInstanceSetSecgroupOptions struct {
	DBInstanceIdOptions
	SECGROUP_IDS []string `help:"Security Group Ids"`
}

func (opts *DBInstanceSetSecgroupOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string][]string{"secgroup_ids": opts.SECGROUP_IDS}), nil
}
