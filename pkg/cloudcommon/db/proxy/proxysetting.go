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

package proxy

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"

	"golang.org/x/net/http/httpproxy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SProxySettingManager struct {
	db.SInfrasResourceBaseManager
}

var ProxySettingManager *SProxySettingManager

func init() {
	ProxySettingManager = &SProxySettingManager{
		SInfrasResourceBaseManager: db.NewInfrasResourceBaseManager(
			SProxySetting{},
			"proxysettings_tbl",
			"proxysetting",
			"proxysettings",
		),
	}
	ProxySettingManager.SetVirtualObject(ProxySettingManager)
}

type SProxySetting struct {
	db.SInfrasResourceBase

	HTTPProxy  string `create:"admin_optional" list:"admin" update:"admin"`
	HTTPSProxy string `create:"admin_optional" list:"admin" update:"admin"`
	NoProxy    string `create:"admin_optional" list:"admin" update:"admin"`
}

func (man *SProxySettingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data proxyapi.ProxySettingCreateInput) (proxyapi.ProxySettingCreateInput, error) {
	var err error
	data.InfrasResourceBaseCreateInput, err = man.SInfrasResourceBaseManager.ValidateCreateData(
		ctx,
		userCred,
		ownerId,
		query,
		data.InfrasResourceBaseCreateInput,
	)
	return data, err
}

func (ps *SProxySetting) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data proxyapi.ProxySettingUpdateInput) (proxyapi.ProxySettingUpdateInput, error) {
	if ps.Id == proxyapi.ProxySettingId_DIRECT {
		return data, httperrors.NewConflictError("DIRECT setting cannot be changed")
	}
	var err error
	data.InfrasResourceBaseUpdateInput, err = ps.SInfrasResourceBase.ValidateUpdateData(
		ctx,
		userCred,
		query,
		data.InfrasResourceBaseUpdateInput,
	)
	return data, err
}

func (ps *SProxySetting) HttpTransportProxyFunc() httputils.TransportProxyFunc {
	cfg := &httpproxy.Config{
		HTTPProxy:  ps.HTTPProxy,
		HTTPSProxy: ps.HTTPSProxy,
		NoProxy:    ps.NoProxy,
	}
	proxyFunc := cfg.ProxyFunc()
	return func(req *http.Request) (*url.URL, error) {
		return proxyFunc(req.URL)
	}
}

func (ps *SProxySetting) ValidateDeleteCondition(ctx context.Context) error {
	for _, man := range referrersMen {
		t := man.TableSpec().Instance()
		n, err := t.Query().
			Equals("proxy_setting_id", ps.Id).
			CountWithError()
		if err != nil {
			return httperrors.NewInternalServerError("get proxysetting refcount fail %s", err)
		}
		if n > 0 {
			return httperrors.NewResourceBusyError("proxysetting %s is still referred to by %d %s",
				ps.Id, n, man.KeywordPlural())
		}
	}
	return ps.SInfrasResourceBase.ValidateDeleteCondition(ctx)
}

func (man *SProxySettingManager) InitializeData() error {
	_, err := man.FetchById(proxyapi.ProxySettingId_DIRECT)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}

	m, err := db.NewModelObject(man)
	if err != nil {
		return err
	}
	ps := m.(*SProxySetting)
	ps.Id = proxyapi.ProxySettingId_DIRECT
	ps.Name = proxyapi.ProxySettingId_DIRECT
	ps.Description = "Connect directly"
	if err := man.TableSpec().Insert(ps); err != nil {
		return err
	}
	return nil
}

var referrersMen []db.IModelManager

func RegisterReferrer(man db.IModelManager) {
	referrersMen = append(referrersMen, man)
}

func ValidateProxySettingResourceInput(userCred mcclient.TokenCredential, input proxyapi.ProxySettingResourceInput) (*SProxySetting, proxyapi.ProxySettingResourceInput, error) {
	m, err := ProxySettingManager.FetchByIdOrName(userCred, input.ProxySettingId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", ProxySettingManager.Keyword(), input.ProxySettingId)
		} else {
			return nil, input, errors.Wrapf(err, "ProxySettingManager.FetchByIdOrName")
		}
	}
	input.ProxySettingId = m.GetId()
	return m.(*SProxySetting), input, nil
}
