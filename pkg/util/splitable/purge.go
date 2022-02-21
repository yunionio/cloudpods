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

package splitable

import (
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

func (t *SSplitTableSpec) Purge(tables []string) ([]string, error) {
	if t.maxSegments <= 0 {
		return nil, nil
	}
	metas, err := t.GetTableMetas()
	if err != nil {
		return nil, errors.Wrap(err, "GetTableMetas")
	}
	if t.maxSegments >= len(metas) && len(tables) == 0 {
		return nil, nil
	}
	ret := []string{}
	for i := 0; i < len(metas); i += 1 {
		if len(tables) == 0 || utils.IsInStringArray(metas[i].Table, tables) {
			dropSQL := fmt.Sprintf("DROP TABLE `%s`", metas[i].Table)
			log.Infof("Ready to drop table: %s", dropSQL)
			_, err := sqlchemy.Exec(dropSQL)
			if err != nil {
				return ret, errors.Wrap(err, "sqlchemy.Exec")
			}
			_, err = t.metaSpec.Update(&metas[i], func() error {
				metas[i].DeleteAt = time.Now()
				metas[i].Deleted = true
				return nil
			})
			if err != nil {
				return ret, errors.Wrap(err, "metaSpec.Update")
			}
			ret = append(ret, metas[i].Table)
		}
	}
	return ret, nil
}
