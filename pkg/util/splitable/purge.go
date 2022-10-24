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
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

var (
	splitableManager sync.Map
)

func registerSplitable(splitable *SSplitTableSpec) {
	splitableManager.Store(splitable.Name(), splitable)
}

func PurgeAll() error {
	errs := make([]error, 0)
	splitableManager.Range(func(k, v interface{}) bool {
		log.Infof("purge splitable %s", k)
		_, err := v.(*SSplitTableSpec).Purge(nil)
		if err != nil {
			errs = append(errs, err)
		}
		return true
	})
	return errors.NewAggregate(errs)
}

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
	metaMax := len(metas)
	if len(tables) == 0 {
		// keep maxSegments if no table specified
		metaMax -= t.maxSegments
	} else {
		// cannot delete the last one
		metaMax -= 1
	}
	ret := []string{}
	for i := 0; i < metaMax; i += 1 {
		if len(tables) == 0 || utils.IsInStringArray(metas[i].Table, tables) {
			dropSQL := fmt.Sprintf("DROP TABLE `%s`", metas[i].Table)
			log.Infof("Ready to drop table: %s", dropSQL)
			tblSpec := t.GetTableSpec(metas[i])
			err := tblSpec.Drop()
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
