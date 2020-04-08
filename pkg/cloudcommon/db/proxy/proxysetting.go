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

	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SProxySettingManager struct {
	db.SStandaloneResourceBaseManager
	db.SDomainizedResourceBaseManager
}

var ProxySettingManager *SProxySettingManager

func init() {
	ProxySettingManager = &SProxySettingManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SProxySetting{},
			"proxysettings_tbl",
			"proxysetting",
			"proxysettings",
		),
	}
	ProxySettingManager.SetVirtualObject(ProxySettingManager)
}

type SProxySetting struct {
	db.SStandaloneResourceBase
	db.SDomainizedResourceBase

	HTTPProxy  string `create:"admin_optional" list:"admin" update:"admin"`
	HTTPSProxy string `create:"admin_optional" list:"admin" update:"admin"`
	NoProxy    string `create:"admin_optional" list:"admin" update:"admin"`
}

func (man *SProxySettingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data proxyapi.ProxySettingCreateInput) (proxyapi.ProxySettingCreateInput, error) {
	if err := data.ProxySetting.Sanitize(); err != nil {
		return data, httperrors.NewInputParameterError("%s", err)
	}
	return data, nil
}

func (ps *SProxySetting) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	ps.DomainId = ownerId.GetProjectDomainId()
	return ps.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (ps *SProxySetting) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data proxyapi.ProxySettingUpdateInput) (proxyapi.ProxySettingUpdateInput, error) {
	if ps.Id == proxyapi.ProxySettingId_DIRECT {
		return data, httperrors.NewConflictError("DIRECT setting cannot be changed")
	}
	if err := data.ProxySetting.Sanitize(); err != nil {
		return data, httperrors.NewInputParameterError("%s", err)
	}
	return data, nil
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
	if ps.Id == proxyapi.ProxySettingId_DIRECT {
		return httperrors.NewConflictError("DIRECT setting cannot be deleted")
	}
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
	return nil
}

func (ps *SProxySetting) AllowPerformTest(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, ps, "test")
}

func (ps *SProxySetting) PerformTest(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	type TestURLResult struct {
		Ok     bool   `json:"ok"`
		Reason string `json:"reason"`
	}
	var (
		r = map[string]TestURLResult{}
		m = map[string]string{
			"http_proxy":  ps.HTTPProxy,
			"https_proxy": ps.HTTPSProxy,
		}
	)
	for k, v := range m {
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
