package db

import (
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

var globalTables map[string]IModelManager

func RegisterModelManager(modelMan IModelManager) {
	if globalTables == nil {
		globalTables = make(map[string]IModelManager)
	}
	mustCheckModelManager(modelMan)
	log.Infof("Register model %s", modelMan.Keyword())
	globalTables[modelMan.Keyword()] = modelMan
}

func mustCheckModelManager(modelMan IModelManager) {
	allowedTags := map[string][]string{
		"create": {"required", "optional", "admin_required", "admin_optional"},
		"search": {"user", "admin"},
		"get":    {"user", "admin"},
		"list":   {"user", "admin"},
		"update": {"user", "admin"},
	}
	for _, col := range modelMan.TableSpec().Columns() {
		tags := col.Tags()
		for tagName, allowedValues := range allowedTags {
			v, ok := tags[tagName]
			if !ok {
				continue
			}
			if !utils.IsInStringArray(v, allowedValues) {
				msg := fmt.Sprintf("model manager %s: column %s has invalid tag %s:\"%s\", expecting %v",
					modelMan.KeywordPlural(), col.Name(), tagName, v, allowedValues)
				panic(msg)
			}
		}
	}
}

func CheckSync(autoSync bool) bool {
	log.Infof("Start check database ...")
	allSqls := make([]string, 0)
	for modelName, modelMan := range globalTables {
		log.Infof("# check table of model %s", modelName)
		tableSpec := modelMan.TableSpec()
		sqls := tableSpec.SyncSQL()
		for _, sql := range sqls {
			allSqls = append(allSqls, sql)
		}
	}
	if len(allSqls) > 0 {
		if autoSync {
			err := commitSqlDIffs(allSqls)
			if err == nil {
				return true
			}
		}
		for _, sql := range allSqls {
			fmt.Println(sql)
		}
		log.Fatalf("Database not in sync!")
		return false
	} else {
		log.Infof("Database is in SYNC!!!")
		return true
	}
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
