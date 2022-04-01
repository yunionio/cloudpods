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

package db

import (
	"fmt"
	"os"
	"reflect"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

var globalTables map[string]IModelManager

func GlobalModelManagerTables() map[string]IModelManager {
	return globalTables
}

func RegisterModelManager(modelMan IModelManager) {
	if globalTables == nil {
		globalTables = make(map[string]IModelManager)
	}
	mustCheckModelManager(modelMan)
	if _, ok := globalTables[modelMan.Keyword()]; ok {
		log.Fatalf("keyword %s exists in globalTables!", modelMan.Keyword())
	}
	globalTables[modelMan.Keyword()] = modelMan
}

func mustCheckModelManager(modelMan IModelManager) {
	allowedTags := map[string][]string{
		"create": {
			"required",
			"optional",
			"domain",
			"domain_required",
			"domain_optional",
			"admin_required",
			"admin_optional",
		},
		"search": {"user", "domain", "admin"},
		"get":    {"user", "domain", "admin"},
		"list":   {"user", "domain", "admin"},
		"update": {"user", "domain", "admin"},
	}
	for _, col := range modelMan.TableSpec().Columns() {
		tags := col.Tags()
		for tagName, allowedValues := range allowedTags {
			v, ok := tags[tagName]
			if !ok {
				continue
			}
			if len(v) > 0 && !utils.IsInStringArray(v, allowedValues) {
				msg := fmt.Sprintf("model manager %s: column %s has invalid tag %s:\"%s\", expecting %v",
					modelMan.KeywordPlural(), col.Name(), tagName, v, allowedValues)
				panic(msg)
			}
		}
	}

	if false {
		requiredManagerFuncNames := []string{
			"ListItemFilter",
			"OrderByExtraFields",
			"FetchCustomizeColumns",
		}
		for _, name := range requiredManagerFuncNames {
			manV := reflect.ValueOf(modelMan)
			methV := manV.MethodByName(name)
			if !methV.IsValid() {
				msg := fmt.Sprintf("model manager %T: has no valid %s, likely caused by ambiguity",
					modelMan, name)
				panic(msg)
			}
		}
	}
}

func CheckSync(autoSync bool, enableChecksumTables bool, skipInitChecksum bool) bool {
	log.Infof("Start check database schema ...")
	inSync := true
	var err error
	for modelName, modelMan := range globalTables {
		tableSpec := modelMan.TableSpec()
		dropFKSqls := tableSpec.DropForeignKeySQL()
		if len(dropFKSqls) > 0 {
			log.Infof("model %s drop foreign key constraints!!!", modelName)
			if autoSync {
				if ts, ok := tableSpec.(*sTableSpec); ok {
					err = commitSqlDiffWithName(dropFKSqls, ts.GetDBName())
				} else {
					err = commitSqlDIffs(dropFKSqls)
				}
				if err != nil {
					log.Errorf("commit sql error %s", err)
					return false
				}
			} else {
				for _, sql := range dropFKSqls {
					log.Infof("%s;", sql)
				}
				inSync = false
			}
		}
	}
	for modelName, modelMan := range globalTables {
		tableSpec := modelMan.TableSpec()
		sqls := tableSpec.SyncSQL()
		if len(sqls) > 0 {
			log.Infof("model %s is not in SYNC!!!", modelName)
			if autoSync {
				if ts, ok := tableSpec.(*sTableSpec); ok {
					err = commitSqlDiffWithName(sqls, ts.GetDBName())
				} else {
					err = commitSqlDIffs(sqls)
				}
				if err != nil {
					log.Errorf("commit sql error %s", err)
					return false
				}
			} else {
				for _, sql := range sqls {
					log.Infof("%s;", sql)
				}
				inSync = false
			}
		}

		recordMan, ok := modelMan.(IRecordChecksumModelManager)
		if ok {
			recordMan.SetEnableRecordChecksum(enableChecksumTables)
			if recordMan.EnableRecordChecksum() {
				if len(sqls) > 0 || !skipInitChecksum {
					if err := InjectModelsChecksum(recordMan); err != nil {
						log.Errorf("InjectModelsChecksum for %q error: %v", modelMan.TableSpec().Name(), err)
						return false
					}
				}
			}
		}
	}
	return inSync
}

func EnsureAppSyncDB(app *appsrv.Application, opt *common_options.DBOptions, modelInitDBFunc func() error) {
	// cloudcommon.InitDB(opt)

	if !CheckSync(opt.AutoSyncTable, opt.EnableDBChecksumTables, opt.DBChecksumSkipInit) {
		log.Fatalf("database schema not in sync!")
	}

	if modelInitDBFunc != nil {
		if err := modelInitDBFunc(); err != nil {
			log.Fatalf("model init db: %v", err)
		}
	}

	if opt.ExitAfterDBInit {
		log.Infof("Exiting after db initialization ...")
		os.Exit(0)
	}

	AppDBInit(app)
}

func GetModelManager(keyword string) IModelManager {
	modelMan, ok := globalTables[keyword]
	if ok {
		return modelMan
	} else {
		return nil
	}
}

func commitSqlDIffs(sqls []string) error {
	db := sqlchemy.GetDB()

	for _, sql := range sqls {
		log.Infof("Exec %s", sql)
		_, err := db.Exec(sql)
		if err != nil {
			log.Errorf("Exec sql failed %s\n%s", sql, err)
			return err
		}
	}
	return nil
}

func commitSqlDiffWithName(sqls []string, dbName sqlchemy.DBName) error {
	db := sqlchemy.GetDBWithName(dbName)
	return execSqlDiffWithDb(sqls, db)
}

func execSqlDiffWithDb(sqls []string, db *sqlchemy.SDatabase) error {
	for _, sql := range sqls {
		log.Infof("Exec %s", sql)
		_, err := db.Exec(sql)
		if err != nil {
			log.Errorf("Exec sql failed %s\n%s", sql, err)
			return err
		}
	}
	return nil
}
