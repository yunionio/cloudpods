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

package qcloud

import (
	"fmt"
	"strings"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SMySQLInstanceAccount struct {
	multicloud.SDBInstanceAccountBase
	QcloudTags
	rds *SMySQLInstance

	Notes              string
	Host               string
	User               string
	ModifyTime         string
	ModifyPasswordTime string
	CreateTime         string
}

func (self *SMySQLInstanceAccount) GetName() string {
	return self.User
}

func (self *SMySQLInstanceAccount) GetHost() string {
	return self.Host
}

func (self *SMySQLInstanceAccount) ResetPassword(password string) error {
	return self.rds.region.ModifyMySQLAccountPassword(self.rds.InstanceId, password, map[string]string{self.User: self.Host})
}

func (self *SMySQLInstanceAccount) Delete() error {
	return self.rds.region.DeleteMySQLAccounts(self.rds.InstanceId, map[string]string{self.User: self.Host})
}

type sPrivilege struct {
	Database  string
	Privilege string
	User      string
	Host      string
}

func (p sPrivilege) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s-%s", p.User, p.Host, p.Database, p.Privilege)
}

func (p sPrivilege) GetDBName() string {
	return p.Database
}

func (p sPrivilege) GetPrivilege() string {
	return p.Privilege
}

func (self *SRegion) GrantAccountPrivilege(instanceId, user, host, database, privilege string) error {
	privileges := []string{}
	switch privilege {
	case api.DATABASE_PRIVILEGE_RW:
		privileges = api.QCLOUD_RW_PRIVILEGE_SET
	case api.DATABASE_PRIVILEGE_R:
		privileges = api.QCLOUD_R_PRIVILEGE_SET
	default:
		return fmt.Errorf("unknow privilege %s", privilege)
	}
	priv, err := self.DescribeAccountPrivileges(instanceId, user, host)
	if err != nil {
		return errors.Wrapf(err, "DescribeAccountPrivileges")
	}
	params := map[string]string{
		"InstanceId":      instanceId,
		"Accounts.0.User": user,
		"Accounts.0.Host": host,
	}
	for i, p := range priv.GlobalPrivileges {
		params[fmt.Sprintf("GlobalPrivileges.%d", i)] = p
	}
	find := false
	for i, p := range priv.DatabasePrivileges {
		params[fmt.Sprintf("DatabasePrivileges.%d.Database", i)] = p.Database
		if database == p.Database {
			p.Privileges = privileges
			find = true
		}
		for j, v := range p.Privileges {
			params[fmt.Sprintf("DatabasePrivileges.%d.Privileges.%d", i, j)] = v
		}
	}
	if !find {
		params[fmt.Sprintf("DatabasePrivileges.%d.Database", len(priv.DatabasePrivileges))] = database
		for j, v := range privileges {
			params[fmt.Sprintf("DatabasePrivileges.%d.Privileges.%d", len(priv.DatabasePrivileges), j)] = v
		}
	}
	for i, p := range priv.TablePrivileges {
		params[fmt.Sprintf("TablePrivileges.%d.Database", i)] = p.Database
		params[fmt.Sprintf("TablePrivileges.%d.Table", i)] = p.Table
		for j, v := range p.Privileges {
			params[fmt.Sprintf("TablePrivileges.%d.Privileges.%d", i, j)] = v
		}
	}
	for i, p := range priv.ColumnPrivileges {
		params[fmt.Sprintf("ColumnPrivileges.%d.Database", i)] = p.Database
		params[fmt.Sprintf("ColumnPrivileges.%d.Table", i)] = p.Table
		params[fmt.Sprintf("ColumnPrivileges.%d.Column", i)] = p.Column
		for j, v := range p.Privileges {
			params[fmt.Sprintf("ColumnPrivileges.%d.Privileges.%d", i, j)] = v
		}
	}
	resp, err := self.cdbRequest("ModifyAccountPrivileges", params)
	if err != nil {
		return errors.Wrapf(err, "ModifyAccountPrivileges")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	return self.waitAsyncAction("ModifyAccountPrivileges", instanceId, asyncRequestId)
}

func (self *SRegion) RevokeAccountPrivilege(instanceId, user, host, database string) error {
	priv, err := self.DescribeAccountPrivileges(instanceId, user, host)
	if err != nil {
		return errors.Wrapf(err, "DescribeAccountPrivileges")
	}
	params := map[string]string{
		"InstanceId":      instanceId,
		"Accounts.0.User": user,
		"Accounts.0.Host": host,
	}
	for i, p := range priv.GlobalPrivileges {
		params[fmt.Sprintf("GlobalPrivileges.%d", i)] = p
	}
	idx := 0
	for _, p := range priv.DatabasePrivileges {
		if p.Database == database {
			continue
		}
		params[fmt.Sprintf("DatabasePrivileges.%d.Database", idx)] = p.Database
		for i, v := range p.Privileges {
			params[fmt.Sprintf("DatabasePrivileges.%d.Privileges.%d", idx, i)] = v
		}
	}
	for i, p := range priv.TablePrivileges {
		params[fmt.Sprintf("TablePrivileges.%d.Database", i)] = p.Database
		params[fmt.Sprintf("TablePrivileges.%d.Table", i)] = p.Table
		for j, v := range p.Privileges {
			params[fmt.Sprintf("TablePrivileges.%d.Privileges.%d", i, j)] = v
		}
	}
	for i, p := range priv.ColumnPrivileges {
		params[fmt.Sprintf("ColumnPrivileges.%d.Database", i)] = p.Database
		params[fmt.Sprintf("ColumnPrivileges.%d.Table", i)] = p.Table
		params[fmt.Sprintf("ColumnPrivileges.%d.Column", i)] = p.Column
		for j, v := range p.Privileges {
			params[fmt.Sprintf("ColumnPrivileges.%d.Privileges.%d", i, j)] = v
		}
	}
	resp, err := self.cdbRequest("ModifyAccountPrivileges", params)
	if err != nil {
		return errors.Wrapf(err, "ModifyAccountPrivileges")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	return self.waitAsyncAction("ModifyAccountPrivileges", instanceId, asyncRequestId)
}

func (self *SMySQLInstanceAccount) GrantPrivilege(database, privilege string) error {
	return self.rds.region.GrantAccountPrivilege(self.rds.InstanceId, self.User, self.Host, database, privilege)
}

func (self *SMySQLInstanceAccount) RevokePrivilege(database string) error {
	return self.rds.region.RevokeAccountPrivilege(self.rds.InstanceId, self.User, self.Host, database)
}

func (self *SMySQLInstanceAccount) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	if utils.IsInStringArray(self.User, []string{"mysql.infoschema", "mysql.session", "mysql.sys"}) {
		return []cloudprovider.ICloudDBInstanceAccountPrivilege{}, nil
	}
	priv, err := self.rds.region.DescribeAccountPrivileges(self.rds.InstanceId, self.User, self.Host)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeAccountPrivileges")
	}
	ret := []cloudprovider.ICloudDBInstanceAccountPrivilege{}
	rwSet := set.New(set.ThreadSafe)
	for _, p := range api.QCLOUD_RW_PRIVILEGE_SET {
		rwSet.Add(p)
	}
	rSet := set.New(set.ThreadSafe)
	for _, p := range api.QCLOUD_R_PRIVILEGE_SET {
		rSet.Add(p)
	}
	for _, p := range priv.DatabasePrivileges {
		pSet := set.New(set.ThreadSafe)
		for _, v := range p.Privileges {
			pSet.Add(v)
		}
		priv := strings.Join(p.Privileges, ",")
		if pSet.IsEqual(rSet) {
			priv = api.DATABASE_PRIVILEGE_R
		} else if pSet.IsEqual(rwSet) {
			priv = api.DATABASE_PRIVILEGE_RW
		}
		privilege := &sPrivilege{
			Database:  p.Database,
			User:      self.User,
			Host:      self.Host,
			Privilege: priv,
		}
		ret = append(ret, privilege)
	}
	return ret, nil
}

func (self *SRegion) ModifyMySQLAccountPassword(instanceId string, password string, users map[string]string) error {
	params := map[string]string{
		"InstanceId":  instanceId,
		"NewPassword": password,
	}
	idx := 0
	for user, host := range users {
		params[fmt.Sprintf("Accounts.%d.user", idx)] = user
		params[fmt.Sprintf("Accounts.%d.host", idx)] = host
		idx++
	}
	resp, err := self.cdbRequest("ModifyAccountPassword", params)
	if err != nil {
		return errors.Wrapf(err, "ModifyAccountPassword")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	return self.waitAsyncAction("ModifyAccountPassword", instanceId, asyncRequestId)
}

func (self *SRegion) DeleteMySQLAccounts(instanceId string, users map[string]string) error {
	params := map[string]string{
		"InstanceId": instanceId,
	}
	idx := 0
	for user, host := range users {
		params[fmt.Sprintf("Accounts.%d.user", idx)] = user
		params[fmt.Sprintf("Accounts.%d.host", idx)] = host
		idx++
	}
	resp, err := self.cdbRequest("DeleteAccounts", params)
	if err != nil {
		return errors.Wrapf(err, "DeleteAccounts")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	return self.waitAsyncAction("DeleteAccounts", instanceId, asyncRequestId)
}

func (self *SRegion) DescribeMySQLAccounts(instanceId string, offset, limit int) ([]SMySQLInstanceAccount, int, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	params := map[string]string{
		"InstanceId": instanceId,
		"Offset":     fmt.Sprintf("%d", offset),
		"Limit":      fmt.Sprintf("%d", limit),
	}
	resp, err := self.cdbRequest("DescribeAccounts", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeAccounts")
	}
	ret := []SMySQLInstanceAccount{}
	err = resp.Unmarshal(&ret, "Items")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Float("TotalCount")
	return ret, int(totalCount), nil
}

func (self *SMySQLInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	accounts := []SMySQLInstanceAccount{}
	for {
		part, total, err := self.region.DescribeMySQLAccounts(self.InstanceId, len(accounts), 100)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeMySQLAccounts")
		}
		accounts = append(accounts, part...)
		if len(accounts) >= total {
			break
		}
	}
	ret := []cloudprovider.ICloudDBInstanceAccount{}
	for i := range accounts {
		if len(accounts[i].User) > 0 { // 忽略用户为空的用户
			accounts[i].rds = self
			ret = append(ret, &accounts[i])
		}
	}
	return ret, nil
}

func (self *SRegion) CreateMySQLAccount(instanceId string, opts *cloudprovider.SDBInstanceAccountCreateConfig) error {
	params := map[string]string{
		"InstanceId":      instanceId,
		"Password":        opts.Password,
		"Accounts.0.User": opts.Name,
		"Accounts.0.Host": opts.Host,
		"Description":     opts.Description,
	}
	resp, err := self.cdbRequest("CreateAccounts", params)
	if err != nil {
		return errors.Wrapf(err, "CreateAccounts")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	return self.waitAsyncAction("CreateAccounts", instanceId, asyncRequestId)
}

type SDatabasePrivilege struct {
	Privileges []string
	Database   string
}

type STablePrivilege struct {
	Database   string
	Table      string
	Privileges []string
}

type SColumnPrivilege struct {
	Database   string
	Table      string
	Column     string
	Privileges []string
}

type SAccountPrivilege struct {
	GlobalPrivileges   []string
	DatabasePrivileges []SDatabasePrivilege
	TablePrivileges    []STablePrivilege
	ColumnPrivileges   []SColumnPrivilege
}

func (self *SRegion) DescribeAccountPrivileges(instanceId string, user, host string) (*SAccountPrivilege, error) {
	params := map[string]string{
		"InstanceId": instanceId,
		"User":       user,
		"Host":       host,
	}
	resp, err := self.cdbRequest("DescribeAccountPrivileges", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeAccountPrivileges")
	}
	priv := &SAccountPrivilege{}
	err = resp.Unmarshal(priv)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return priv, nil
}
