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
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

type purgePair struct {
	manager db.IModelManager
	key     string
	q       *sqlchemy.SQuery
}

func (self *purgePair) queryIds() ([]string, error) {
	ids := []string{}
	sq := self.q.SubQuery()
	q := sq.Query(sq.Field(self.key)).Distinct()
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return ids, nil
		}
		return ids, errors.Wrap(err, "Query")
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return ids, errors.Wrap(err, "rows.Scan")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (self *purgePair) purgeAll(ctx context.Context) error {
	purgeIds, err := self.queryIds()
	if err != nil {
		return errors.Wrapf(err, "Query ids")
	}
	if len(purgeIds) == 0 {
		return nil
	}
	var purge = func(ids []string) error {
		vars := []interface{}{}
		placeholders := make([]string, len(ids))
		for i := range placeholders {
			placeholders[i] = "?"
			vars = append(vars, ids[i])
		}
		placeholder := strings.Join(placeholders, ",")
		sql := fmt.Sprintf(
			"delete from %s where %s in (%s)",
			self.manager.TableSpec().Name(), self.key, placeholder,
		)
		lockman.LockRawObject(ctx, self.manager.Keyword(), "purge")
		defer lockman.ReleaseRawObject(ctx, self.manager.Keyword(), "purge")

		_, err = sqlchemy.GetDB().Exec(
			sql, vars...,
		)
		if err != nil {
			return errors.Wrapf(err, strings.ReplaceAll(sql, "?", "%s"), vars...)
		}
		return nil
	}
	var splitByLen = func(data []string, splitLen int) [][]string {
		var result [][]string
		for i := 0; i < len(data); i += splitLen {
			end := i + splitLen
			if end > len(data) {
				end = len(data)
			}
			result = append(result, data[i:end])
		}
		return result
	}
	idsArr := splitByLen(purgeIds, 100)
	for i := range idsArr {
		err = purge(idsArr[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SCloudaccount) purge(ctx context.Context) error {
	cloudproviders := CloudproviderManager.Query("id").Equals("cloudaccount_id", self.Id)
	cloudusers := ClouduserManager.Query("id").Equals("cloudaccount_id", self.Id)
	ups := ClouduserPolicyManager.Query("row_id").In("clouduser_id", cloudusers.SubQuery())
	ugs := CloudgroupUserManager.Query("row_id").In("clouduser_id", cloudusers.SubQuery())
	cloudgroups := CloudgroupManager.Query("id").Equals("cloudaccount_id", self.Id)
	gps := CloudgroupPolicyManager.Query("row_id").In("cloudgroup_id", cloudgroups.SubQuery())
	gus := CloudgroupUserManager.Query("row_id").In("cloudgroup_id", cloudgroups.SubQuery())
	samlproviders := SAMLProviderManager.Query("id").Equals("cloudaccount_id", self.Id)
	roles := CloudroleManager.Query("id").Equals("cloudaccount_id", self.Id)
	samlusers := SamluserManager.Query("id").In("cloudgroup_id", cloudgroups.SubQuery())
	policies := CloudpolicyManager.Query("id").Equals("cloudaccount_id", self.Id)

	pairs := []purgePair{
		{manager: CloudpolicyManager, key: "id", q: policies},
		{manager: SamluserManager, key: "id", q: samlusers},
		{manager: CloudroleManager, key: "id", q: roles},
		{manager: SAMLProviderManager, key: "id", q: samlproviders},
		{manager: CloudgroupPolicyManager, key: "row_id", q: gps},
		{manager: CloudgroupUserManager, key: "row_id", q: gus},
		{manager: CloudgroupManager, key: "id", q: cloudgroups},
		{manager: ClouduserPolicyManager, key: "row_id", q: ups},
		{manager: CloudgroupUserManager, key: "row_id", q: ugs},
		{manager: ClouduserManager, key: "id", q: cloudusers},
		{manager: CloudproviderManager, key: "id", q: cloudproviders},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return self.SStandaloneResourceBase.Delete(ctx, nil)
}

func (self *SClouduser) purge(ctx context.Context) error {
	ups := ClouduserPolicyManager.Query("row_id").Equals("clouduser_id", self.Id)
	gps := CloudgroupUserManager.Query("row_id").Equals("clouduser_id", self.Id)

	pairs := []purgePair{
		{manager: ClouduserPolicyManager, key: "row_id", q: ups},
		{manager: CloudgroupUserManager, key: "row_id", q: gps},
	}

	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}

	return self.SStatusDomainLevelResourceBase.Delete(ctx, nil)
}

func (self *SCloudprovider) purge(ctx context.Context) error {
	cloudusers := ClouduserManager.Query("id").Equals("manager_id", self.Id)
	ups := ClouduserPolicyManager.Query("row_id").In("clouduser_id", cloudusers.SubQuery())
	ugs := CloudgroupUserManager.Query("row_id").In("clouduser_id", cloudusers.SubQuery())
	cloudgroups := CloudgroupManager.Query("id").Equals("manager_id", self.Id)
	gps := CloudgroupPolicyManager.Query("row_id").In("cloudgroup_id", cloudgroups.SubQuery())
	gus := CloudgroupUserManager.Query("row_id").In("cloudgroup_id", cloudgroups.SubQuery())
	samlproviders := SAMLProviderManager.Query("id").Equals("manager_id", self.Id)
	samlusers := SamluserManager.Query("id").In("cloudgroup_id", cloudgroups.SubQuery())
	policies := CloudpolicyManager.Query("id").Equals("manager_id", self.Id)
	roles := CloudroleManager.Query("id").Equals("manager_id", self.Id)

	pairs := []purgePair{
		{manager: CloudpolicyManager, key: "id", q: policies},
		{manager: SamluserManager, key: "id", q: samlusers},
		{manager: CloudroleManager, key: "id", q: roles},
		{manager: SAMLProviderManager, key: "id", q: samlproviders},
		{manager: CloudgroupPolicyManager, key: "row_id", q: gps},
		{manager: CloudgroupUserManager, key: "row_id", q: gus},
		{manager: CloudgroupManager, key: "id", q: cloudgroups},
		{manager: ClouduserPolicyManager, key: "row_id", q: ups},
		{manager: CloudgroupUserManager, key: "row_id", q: ugs},
		{manager: ClouduserManager, key: "id", q: cloudusers},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}

	return self.SStandaloneAnonResourceBase.Delete(ctx, nil)
}

func (self *SCloudgroup) purge(ctx context.Context) error {
	ups := CloudgroupPolicyManager.Query("row_id").Equals("cloudgroup_id", self.Id)
	gps := CloudgroupUserManager.Query("row_id").Equals("cloudgroup_id", self.Id)
	saml := SamluserManager.Query("id").Equals("cloudgroup_id", self.Id)

	pairs := []purgePair{
		{manager: SamluserManager, key: "id", q: saml},
		{manager: ClouduserPolicyManager, key: "row_id", q: ups},
		{manager: CloudgroupUserManager, key: "row_id", q: gps},
	}

	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}

	return self.SStatusInfrasResourceBase.Delete(ctx, nil)
}
