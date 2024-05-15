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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SWafRuleStatementManager struct {
	db.SResourceBaseManager
}

var WafRuleStatementManager *SWafRuleStatementManager

func init() {
	WafRuleStatementManager = &SWafRuleStatementManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SWafRuleStatement{},
			"waf_rule_statements_tbl",
			"waf_rule_statement",
			"waf_rule_statements",
		),
	}
	WafRuleStatementManager.SetVirtualObject(WafRuleStatementManager)
}

type SWafRuleStatement struct {
	db.SResourceBase

	Id string `width:"128" charset:"ascii" primary:"true" list:"user"`
	cloudprovider.SWafStatement

	WafRuleId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (self *SWafRuleStatement) BeforeInsert() {
	if len(self.Id) == 0 {
		self.Id = stringutils.UUID4()
	}
}

func (self *SWafRuleStatement) GetId() string {
	return self.Id
}

func (self *SWafRule) GetRuleStatements() ([]SWafRuleStatement, error) {
	q := WafRuleStatementManager.Query().Equals("waf_rule_id", self.Id)
	statements := []SWafRuleStatement{}
	err := db.FetchModelObjects(WafRuleStatementManager, q, &statements)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return statements, nil
}

func (self *SWafRuleStatement) syncWithStatement(ctx context.Context, userCred mcclient.TokenCredential, statement cloudprovider.SWafStatement) error {
	_, err := db.Update(self, func() error {
		self.SWafStatement = statement
		switch self.Type {
		case cloudprovider.WafStatementTypeIPSet:
			if len(self.IPSetId) > 0 {
				ipSet, err := db.FetchByExternalId(WafIPSetManager, self.IPSetId)
				if err != nil {
					log.Errorf("WafIPSetManager(%s) error: %v", self.IPSetId, err)
				} else {
					self.IPSetId = ipSet.GetId()
				}
			}
		case cloudprovider.WafStatementTypeRegexSet:
			if len(self.RegexSetId) > 0 {
				regexSet, err := db.FetchByExternalId(WafRegexSetManager, self.RegexSetId)
				if err != nil {
					log.Errorf("WafRegexSetManager(%s) error: %v", self.RegexSetId, err)
				} else {
					self.RegexSetId = regexSet.GetId()
				}
			}
		}

		return nil
	})
	return err
}

func (self *SWafRule) newFromCloudStatement(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.SWafStatement) error {
	statement := &SWafRuleStatement{}
	statement.SetModelManager(WafRuleStatementManager, statement)
	statement.WafRuleId = self.Id
	statement.SWafStatement = ext
	switch statement.Type {
	case cloudprovider.WafStatementTypeIPSet:
		if len(statement.IPSetId) > 0 {
			ipSet, err := db.FetchByExternalId(WafIPSetManager, statement.IPSetId)
			if err != nil {
				log.Errorf("WafIPSetManager(%s) error: %v", statement.IPSetId, err)
			} else {
				statement.IPSetId = ipSet.GetId()
			}
		}
	case cloudprovider.WafStatementTypeRegexSet:
		if len(statement.RegexSetId) > 0 {
			regexSet, err := db.FetchByExternalId(WafRegexSetManager, statement.RegexSetId)
			if err != nil {
				log.Errorf("WafRegexSetManager(%s) error: %v", statement.RegexSetId, err)
			} else {
				statement.RegexSetId = regexSet.GetId()
			}
		}
	}
	return WafRuleStatementManager.TableSpec().Insert(ctx, statement)
}

func (self *SWafRule) SyncStatements(ctx context.Context, userCred mcclient.TokenCredential, rule cloudprovider.ICloudWafRule) error {
	lockman.LockRawObject(ctx, WafRuleManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, WafRuleManager.Keyword(), self.Id)

	dbStatements, err := self.GetRuleStatements()
	if err != nil {
		return errors.Wrapf(err, "GetRuleStatements")
	}

	exts, err := rule.GetStatements()
	if err != nil {
		return errors.Wrapf(err, "GetStatements")
	}

	result := compare.SyncResult{}

	removed := make([]SWafRuleStatement, 0)
	commondb := make([]SWafRuleStatement, 0)
	commonext := make([]cloudprovider.SWafStatement, 0)
	added := make([]cloudprovider.SWafStatement, 0)
	err = compare.CompareSets(dbStatements, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		return errors.Wrapf(err, "compare.CompareSets")
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].syncWithStatement(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		err := self.newFromCloudStatement(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	log.Debugf("sync statements for rule %s result: %s", self.Name, result.Result())
	return nil
}
