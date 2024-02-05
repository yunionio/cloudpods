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
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/http/httpproxy"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/rbacscope"

	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

	HTTPProxy  string `create:"domain_optional" list:"domain" update:"domain"`
	HTTPSProxy string `create:"domain_optional" list:"domain" update:"domain"`
	NoProxy    string `create:"domain_optional" list:"domain" update:"domain"`
}

func (man *SProxySettingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data proxyapi.ProxySettingCreateInput) (proxyapi.ProxySettingCreateInput, error) {
	var err error
	if err := data.ProxySetting.Sanitize(); err != nil {
		return data, httperrors.NewInputParameterError("%s", err)
	}
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
	if err := data.ProxySetting.Sanitize(); err != nil {
		return data, httperrors.NewInputParameterError("%s", err)
	}
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

func (ps *SProxySetting) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if ps.Id == proxyapi.ProxySettingId_DIRECT {
		return httperrors.NewConflictError("DIRECT setting cannot be deleted")
	}
	for _, man := range referrersMen {
		n, err := man.Query().
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
	return ps.SInfrasResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (man *SProxySettingManager) test(ctx context.Context, urls map[string]string) (jsonutils.JSONObject, error) {
	type TestURLResult struct {
		Ok     bool   `json:"ok"`
		Reason string `json:"reason"`
	}
	r := map[string]TestURLResult{}
	for k, v := range urls {
		if v == "" {
			r[k] = TestURLResult{Ok: true}
			continue
		}
		u, err := url.Parse(v)
		if err != nil {
			r[k] = TestURLResult{
				Reason: err.Error(),
			}
		} else if u == nil {
			r[k] = TestURLResult{
				Reason: fmt.Sprintf("bad url: %q", v),
			}
		} else {
			host := u.Hostname()
			port := u.Port()
			if port == "" {
				switch u.Scheme {
				case "http":
					port = "80"
				case "https":
					port = "443"
				case "socks5":
					port = "1080"
				default:
					r[k] = TestURLResult{
						Reason: fmt.Sprintf("bad url scheme: %s", u.Scheme),
					}
					continue
				}
			}
			addr := net.JoinHostPort(host, port)
			conn, err := net.DialTimeout("tcp", addr, 7*time.Second)
			if err != nil {
				r[k] = TestURLResult{
					Reason: err.Error(),
				}
			} else {
				r[k] = TestURLResult{Ok: true}
				conn.Close()
			}
		}
	}
	return jsonutils.Marshal(r), nil
}

func (ps *SProxySetting) PerformTest(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	urls := map[string]string{
		"http_proxy":  ps.HTTPProxy,
		"https_proxy": ps.HTTPSProxy,
	}
	return ProxySettingManager.test(ctx, urls)
}

func (man *SProxySettingManager) PerformTest(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data proxyapi.ProxySettingTestInput) (jsonutils.JSONObject, error) {
	urls := map[string]string{
		"http_proxy":  data.HttpProxy,
		"https_proxy": data.HttpsProxy,
	}
	return man.test(ctx, urls)
}

func (man *SProxySettingManager) InitializeData() error {
	psObj, err := man.FetchById(proxyapi.ProxySettingId_DIRECT)
	if err == nil {
		ps := psObj.(*SProxySetting)
		if !ps.IsPublic || ps.PublicScope != string(rbacscope.ScopeSystem) {
			_, err = db.Update(ps, func() error {
				ps.IsPublic = true
				ps.PublicScope = string(rbacscope.ScopeSystem)
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "Update")
			}
		}
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
	ps.IsPublic = true
	ps.PublicScope = string(rbacscope.ScopeSystem)
	if err := man.TableSpec().Insert(context.Background(), ps); err != nil {
		return err
	}
	return nil
}

var referrersMen []db.IModelManager

func RegisterReferrer(man db.IModelManager) {
	referrersMen = append(referrersMen, man)
}

func ValidateProxySettingResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input proxyapi.ProxySettingResourceInput) (*SProxySetting, proxyapi.ProxySettingResourceInput, error) {
	m, err := ProxySettingManager.FetchByIdOrName(ctx, userCred, input.ProxySettingId)
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
