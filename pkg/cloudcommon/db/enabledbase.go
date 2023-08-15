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
	"context"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SEnabledResourceBaseManager struct{}

type SEnabledResourceBase struct {
	// 资源是否启用
	Enabled tristate.TriState `default:"false" list:"user" create:"optional"`
}

type IEnabledBaseInterface interface {
	SetEnabled(enabled bool)
	GetEnabled() bool
}

type IEnabledBase interface {
	IModel
	IEnabledBaseInterface
}

func (m *SEnabledResourceBase) SetEnabled(enabled bool) {
	if enabled {
		m.Enabled = tristate.True
	} else {
		m.Enabled = tristate.False
	}
}

func (m *SEnabledResourceBase) GetEnabled() bool {
	return m.Enabled.Bool()
}

func EnabledPerformEnable(model IEnabledBase, ctx context.Context, userCred mcclient.TokenCredential, enabled bool) error {
	if model.GetEnabled() == enabled {
		return nil
	}
	_, err := Update(model, func() error {
		model.SetEnabled(enabled)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	if enabled {
		OpsLog.LogEvent(model, ACT_ENABLE, "", userCred)
		logclient.AddSimpleActionLog(model, logclient.ACT_ENABLE, nil, userCred, true)
	} else {
		OpsLog.LogEvent(model, ACT_DISABLE, "", userCred)
		logclient.AddSimpleActionLog(model, logclient.ACT_DISABLE, nil, userCred, true)
	}
	return nil
}

func (manager *SEnabledResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.EnabledResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	if query.Enabled != nil {
		if *query.Enabled {
			q = q.IsTrue("enabled")
		} else {
			q = q.IsFalse("enabled")
		}
	}
	return q, nil
}
