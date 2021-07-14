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

package azure

import (
	"context"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// todo: 目前不支持 入站 NAT 规则
// todo： HTTP 设置(端口+协议，其余信息丢弃) + 后端池  = onecloud 路径的路由 的 onecloud 后端服务器组
/*
应用型LB： urlPathMaps（defaultBackendAddressPool+defaultBackendHttpSettings+requestRoutingRules+httpListeners）= Onecloud监听器
4层LB: loadBalancingRules（前端） = Onecloud监听器
*/
type SLoadBalancerListener struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase

	lb   *SLoadbalancer
	lbrs []cloudprovider.ICloudLoadbalancerListenerRule

	/*
		IPVersion    string
		FrontendIP   string // 前端IP
		FrontendIPId string // 前端IP ID
	*/
	fp *FrontendIPConfiguration

	/*
		    Protocol     string // 监听协议
			HostName     string // 监听器 监听域名
			FrontendPort int    // 前端端口
	*/
	listener *HTTPListener

	/*
		   Protocol     string // 后端协议
		   BackendPort  int    // 后端端口
			IdleTimeoutInMinutes      int    // 空闲超时(分钟)
			LoadDistribution          string // 会话保持方法SourceIP|SourceIPProtocol
			CookieBasedAffinity       string // cookies 关联
			CookieName                string // cookie 名称
			EnabledConnectionDraining bool   //   排出超时这应用于通过 API 调用从后端池明确删除的后端实例，以及运行状况探测报告的运行不正常的实例。
			ConnectionDrainingSec     int    // 排出超时时长(秒)
			RequestTimeout            int    // 请求超时时间(秒)
		    probe                            // 健康检查
	*/
	backendSetting *BackendHTTPSettingsCollection

	//
	backendGroup *BackendAddressPool

	/*
			redirectType string // 重定向类型
		    targetListener      // 重定向到监听
	*/
	redirect *RedirectConfiguration

	/*
	 "protocol": "Http",
	 "host": "baidu.com",
	 "path": "/test",
	 "interval": 30,
	 "timeout": 30,
	 "unhealthyThreshold": 3,
	 "pickHostNameFromBackendHttpSettings": false,
	 "minServers": 0,
	 "match": {
	 	"body": "500",
	 	"statusCodes": [
	 		"200-399"
	 	]
	 },
	*/
	healthcheck *Probe

	Name              string
	ID                string
	ProvisioningState string
	IPVersion         string
	Protocol          string // 监听协议
	LoadDistribution  string // 调度算法
	FrontendPort      int    // 前端端口
	BackendPort       int    // 后端端口
	ClientIdleTimeout int    // 客户端连接超时
	EnableFloatingIP  bool   // 浮动 IP
	EnableTcpReset    bool

	rules []PathRule
}

func (self *SLoadBalancerListener) GetId() string {
	return self.ID
}

func (self *SLoadBalancerListener) GetName() string {
	return self.Name
}

func (self *SLoadBalancerListener) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadBalancerListener) GetStatus() string {
	switch self.ProvisioningState {
	case "Succeeded", "Updating", "Deleting":
		return api.LB_STATUS_ENABLED
	case "Failed":
		return api.LB_STATUS_DISABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadBalancerListener) Refresh() error {
	lbl, err := self.lb.GetILoadBalancerListenerById(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetILoadBalancerListenerById")
	}

	err = jsonutils.Update(self, lbl)
	if err != nil {
		return errors.Wrap(err, "refresh.Update")
	}

	self.lbrs = nil
	return nil
}

func (self *SLoadBalancerListener) IsEmulated() bool {
	return false
}

func (self *SLoadBalancerListener) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadBalancerListener) GetTags() (map[string]string, error) {
	if self.fp != nil {
		if self.fp.Properties.PublicIPAddress != nil && len(self.fp.Properties.PublicIPAddress.ID) > 0 {
			eip, _ := self.lb.GetIEIP()
			if eip != nil {
				return map[string]string{"FrontendIP": eip.GetIpAddr()}, nil
			}
		}

		if len(self.fp.Properties.PrivateIPAddress) > 0 {
			return map[string]string{"FrontendIP": self.fp.Properties.PrivateIPAddress}, nil
		}
	}

	return map[string]string{}, nil
}

func (self *SLoadBalancerListener) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

func (self *SLoadBalancerListener) GetProjectId() string {
	return getResourceGroup(self.GetId())
}

func (self *SLoadBalancerListener) GetListenerType() string {
	switch strings.ToLower(self.Protocol) {
	case "tcp":
		return api.LB_LISTENER_TYPE_TCP
	case "udp":
		return api.LB_LISTENER_TYPE_UDP
	case "http":
		return api.LB_LISTENER_TYPE_HTTP
	case "https":
		return api.LB_LISTENER_TYPE_HTTPS
	default:
		return ""
	}
}

func (self *SLoadBalancerListener) GetListenerPort() int {
	return int(self.FrontendPort)
}

func (self *SLoadBalancerListener) GetScheduler() string {
	switch self.LoadDistribution {
	case "SourceIPProtocol", "SourceIP":
		return api.LB_SCHEDULER_SCH
	default:
		return ""
	}
}

func (self *SLoadBalancerListener) GetAclStatus() string {
	return api.LB_BOOL_OFF
}

func (self *SLoadBalancerListener) GetAclType() string {
	return ""
}

func (self *SLoadBalancerListener) GetAclId() string {
	return ""
}

func (self *SLoadBalancerListener) GetEgressMbps() int {
	return 0
}

func (self *SLoadBalancerListener) GetHealthCheck() string {
	// if self.probe != nil
	if self.healthcheck != nil {
		return api.LB_BOOL_ON
	}

	return api.LB_BOOL_OFF
}

func (self *SLoadBalancerListener) GetHealthCheckType() string {
	if self.healthcheck == nil {
		return ""
	}
	switch strings.ToLower(self.healthcheck.Properties.Protocol) {
	case "tcp":
		return api.LB_HEALTH_CHECK_TCP
	case "udp":
		return api.LB_HEALTH_CHECK_UDP
	case "http":
		return api.LB_HEALTH_CHECK_HTTP
	case "https":
		return api.LB_HEALTH_CHECK_HTTPS
	default:
		return ""
	}
}

func (self *SLoadBalancerListener) GetHealthCheckTimeout() int {
	if self.healthcheck == nil {
		return 0
	}
	switch self.GetHealthCheckType() {
	case api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS:
		return self.healthcheck.Properties.Timeout
	}

	return self.healthcheck.Properties.IntervalInSeconds
}

func (self *SLoadBalancerListener) GetHealthCheckInterval() int {
	if self.healthcheck == nil {
		return 0
	}
	switch self.GetHealthCheckType() {
	case api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS:
		return self.healthcheck.Properties.Interval
	}

	return self.healthcheck.Properties.IntervalInSeconds
}

func (self *SLoadBalancerListener) GetHealthCheckRise() int {
	return 0
}

func (self *SLoadBalancerListener) GetHealthCheckFail() int {
	if self.healthcheck == nil {
		return 0
	}
	switch self.GetHealthCheckType() {
	case api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS:
		return self.healthcheck.Properties.UnhealthyThreshold
	}

	return self.healthcheck.Properties.IntervalInSeconds
}

func (self *SLoadBalancerListener) GetHealthCheckReq() string {
	return ""
}

func (self *SLoadBalancerListener) GetHealthCheckExp() string {
	return ""
}

func (self *SLoadBalancerListener) GetBackendGroupId() string {
	if self.backendGroup != nil {
		return self.backendGroup.ID + "::" + strconv.Itoa(self.BackendPort)
	}

	return ""
}

func (self *SLoadBalancerListener) GetBackendServerPort() int {
	return self.BackendPort
}

func (self *SLoadBalancerListener) GetHealthCheckDomain() string {
	if self.healthcheck == nil {
		return ""
	}
	switch self.GetHealthCheckType() {
	case api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS:
		return self.healthcheck.Properties.Host
	}

	return ""
}

func (self *SLoadBalancerListener) GetHealthCheckURI() string {
	if self.healthcheck == nil {
		return ""
	}
	switch self.GetHealthCheckType() {
	case api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS:
		return self.healthcheck.Properties.Path
	}

	return ""
}

/*
todo: 和onecloud code不兼容？
与此处输入的 HTTP 状态代码或代码范围相匹配的响应将被视为成功。请输入逗号分隔的代码列表(例如 200, 201)或者输入一个代码范围(例如 220-226)
*/
func (self *SLoadBalancerListener) GetHealthCheckCode() string {
	return ""
}

func (self *SLoadBalancerListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancerListenerRule")
}

func (self *SLoadBalancerListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	lbrs, err := self.GetILoadbalancerListenerRules()
	if err != nil {
		return nil, errors.Wrap(err, "GetILoadbalancerListenerRules")
	}

	for i := range lbrs {
		if lbrs[i].GetId() == ruleId {
			return lbrs[i], nil
		}
	}

	return nil, errors.Wrap(err, "GetILoadBalancerListenerRuleById")
}

func (self *SLoadBalancerListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	if self.lbrs != nil {
		return self.lbrs, nil
	}

	irules := []cloudprovider.ICloudLoadbalancerListenerRule{}
	for i := range self.rules {
		r := self.rules[i]
		var redirect *RedirectConfiguration
		var lbbg *SLoadbalancerBackendGroup
		if r.Properties.RedirectConfiguration != nil {
			redirect = self.lb.getRedirectConfiguration(r.Properties.RedirectConfiguration.ID)
		} else {
			if r.Properties.BackendAddressPool == nil || r.Properties.BackendHTTPSettings == nil {
				continue
			}

			pool := self.lb.getBackendAddressPool(r.Properties.BackendAddressPool.ID)
			if pool == nil {
				log.Debugf("getBackendAddressPool %s not found", r.Properties.BackendAddressPool.ID)
				continue
			}
			backsetting := self.lb.getBackendHTTPSettingsCollection(r.Properties.BackendHTTPSettings.ID)
			if backsetting == nil {
				log.Debugf("getBackendHTTPSettingsCollection %s not found", r.Properties.BackendHTTPSettings.ID)
				continue
			}

			lbbg = &SLoadbalancerBackendGroup{
				lb:           self.lb,
				Pool:         *pool,
				DefaultPort:  backsetting.Properties.Port,
				HttpSettings: backsetting,
				BackendIps:   pool.Properties.BackendIPConfigurations,
			}
		}

		domain := ""
		if self.listener != nil {
			domain = self.listener.Properties.HostName
		}

		rule := SLoadbalancerListenerRule{
			SResourceBase: multicloud.SResourceBase{},
			listener:      self,
			lbbg:          lbbg,
			redirect:      redirect,
			Name:          r.Name,
			ID:            r.ID,
			Domain:        domain,
			Properties:    r.Properties,
		}
		irules = append(irules, &rule)
	}

	return irules, nil
}

func (self *SLoadBalancerListener) GetStickySession() string {
	if self.backendSetting == nil {
		return api.LB_BOOL_OFF
	}

	if self.backendSetting.Properties.CookieBasedAffinity == "Enabled" {
		return api.LB_BOOL_ON
	} else {
		return api.LB_BOOL_OFF
	}
}

func (self *SLoadBalancerListener) GetStickySessionType() string {
	if self.GetStickySession() == api.LB_BOOL_ON {
		return api.LB_STICKY_SESSION_TYPE_INSERT
	}
	return ""
}

func (self *SLoadBalancerListener) GetStickySessionCookie() string {
	if self.backendSetting == nil {
		return ""
	}

	return self.backendSetting.Properties.AffinityCookieName
}

func (self *SLoadBalancerListener) GetStickySessionCookieTimeout() int {
	if self.backendSetting == nil {
		return 0
	}

	if self.backendSetting.Properties.ConnectionDraining.Enabled {
		_sec := strings.TrimSpace(self.backendSetting.Properties.ConnectionDraining.DrainTimeoutInSEC)
		sec, _ := strconv.ParseInt(_sec, 10, 64)
		return int(sec)
	}

	return 0
}

func (self *SLoadBalancerListener) XForwardedForEnabled() bool {
	return false
}

func (self *SLoadBalancerListener) GzipEnabled() bool {
	return false
}

func (self *SLoadBalancerListener) GetCertificateId() string {
	if self.listener != nil {
		return self.listener.Properties.SSLCertificate.ID
	}

	return ""
}

func (self *SLoadBalancerListener) GetTLSCipherPolicy() string {
	return ""
}

func (self *SLoadBalancerListener) HTTP2Enabled() bool {
	return self.lb.Properties.EnableHttp2
}

func (self *SLoadBalancerListener) Start() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Start")
}

func (self *SLoadBalancerListener) Stop() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Stop")
}

func (self *SLoadBalancerListener) Sync(ctx context.Context, listener *cloudprovider.SLoadbalancerListener) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Sync")
}

func (self *SLoadBalancerListener) Delete(ctx context.Context) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadBalancerListener) GetRedirect() string {
	if self.redirect != nil {
		return api.LB_REDIRECT_RAW
	}

	return api.LB_REDIRECT_OFF
}

func (self *SLoadBalancerListener) GetRedirectCode() int64 {
	if self.redirect == nil {
		return 0
	}

	switch self.redirect.Properties.RedirectType {
	case "Permanent":
		return api.LB_REDIRECT_CODE_301
	case "Found":
		return api.LB_REDIRECT_CODE_302
	case "Temporary", "SeeOther":
		return api.LB_REDIRECT_CODE_307
	default:
		return 0
	}
}

func (self *SLoadBalancerListener) getRedirectUrl() *url.URL {
	if self.redirect == nil {
		return nil
	}

	if len(self.redirect.Properties.TargetUrl) == 0 {
		return nil
	}

	_url := self.redirect.Properties.TargetUrl
	if matched, _ := regexp.MatchString("^\\w{0,5}://", _url); !matched {
		_url = "http://" + _url
	}

	u, err := url.Parse(_url)
	if err != nil {
		log.Debugf("url Parse %s : %s", self.redirect.Properties.TargetUrl, err)
		return nil
	}

	return u
}
func (self *SLoadBalancerListener) GetRedirectScheme() string {
	u := self.getRedirectUrl()
	if u == nil {
		return ""
	}

	return strings.ToLower(u.Scheme)
}

func (self *SLoadBalancerListener) GetRedirectHost() string {
	u := self.getRedirectUrl()
	if u == nil {
		if self.redirect != nil && len(self.redirect.Properties.TargetListener.ID) > 0 {
			segs := strings.Split(self.redirect.Properties.TargetListener.ID, "/")
			return segs[len(segs)-1]
		}
		return ""
	}

	return u.Host
}

func (self *SLoadBalancerListener) GetRedirectPath() string {
	u := self.getRedirectUrl()
	if u == nil {
		return ""
	}

	return u.Path
}

func (self *SLoadBalancerListener) GetClientIdleTimeout() int {
	return self.ClientIdleTimeout
}

func (self *SLoadBalancerListener) GetBackendConnectTimeout() int {
	if self.backendSetting == nil {
		return 0
	}

	return self.backendSetting.Properties.RequestTimeout
}
