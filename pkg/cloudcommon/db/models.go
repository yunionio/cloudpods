package db

import (
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"
)

var globalTables map[string]IModelManager

func RegisterModelManager(modelMan IModelManager) {
	if globalTables == nil {
		globalTables = make(map[string]IModelManager)
	}
	log.Infof("Register model %s", modelMan.Keyword())
	globalTables[modelMan.Keyword()] = modelMan
}

func CheckSync(autoSync bool) bool {
	log.Infof("Start check database ...")
	allSqls := make([]string, 0)
	for modelName, modelMan := range globalTables {
		log.Infof("# check table of model %s", modelName)
		tableSpec := modelMan.TableSpec()
		sqls := tableSpec.SyncSQL()
		for _, sql := range sqls {
			// fmt.Printf("%s\n", sql)
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
			log.Errorf("Exec sql %s failed %s", sql, err)
			return err
		}
	}
	return nil
}
