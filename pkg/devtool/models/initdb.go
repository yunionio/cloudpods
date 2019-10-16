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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

func InitDB() error {
	for _, manager := range []db.IModelManager{
		CronjobManager,
		DevtoolTemplateManager,
	} {
		err := manager.InitializeData()
		if err != nil {
			log.Errorf("Manager %s initializeData fail %s", manager.Keyword(), err)
			return err
		} else {
			log.Infof("Manager %s initializeData PASS!", manager.Keyword())
		}
	}
	return nil
}
