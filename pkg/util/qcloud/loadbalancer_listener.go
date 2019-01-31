package qcloud

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

var HTTP_CODES = []string{
	models.LB_HEALTH_CHECK_HTTP_CODE_1xx,
	models.LB_HEALTH_CHECK_HTTP_CODE_2xx,
	models.LB_HEALTH_CHECK_HTTP_CODE_3xx,
	models.LB_HEALTH_CHECK_HTTP_CODE_4xx,
	models.LB_HEALTH_CHECK_HTTP_CODE_5xx,
}

type certificate struct {
	SSLMode  string `json:"SSLMode"`
	CERTCAID string `json:"CertCaId"`
	CERTID   string `json:"CertId"`
}

/*
健康检查状态码（仅适用于HTTP/HTTPS转发规则）。可选值：1~31，默认 31。
1 表示探测后返回值 1xx 表示健康，2 表示返回 2xx 表示健康，4 表示返回 3xx 表示健康，8 表示返回 4xx 表示健康，16 表示返回 5xx 表示健康。
若希望多种码都表示健康，则将相应的值相加。
*/
type healthCheck struct {
	HTTPCheckDomain string `json:"HttpCheckDomain"`
	HealthSwitch    int64  `json:"HealthSwitch"`
	HTTPCheckPath   string `json:"HttpCheckPath"`
	HTTPCheckMethod string `json:"HttpCheckMethod"`
	UnHealthNum     int    `json:"UnHealthNum"`
	IntervalTime    int    `json:"IntervalTime"`
	HTTPCode        int    `json:"HttpCode"` // 健康检查状态码（仅适用于HTTP/HTTPS转发规则）。可选值：1~31，默认 31。
	HealthNum       int    `json:"HealthNum"`
	TimeOut         int    `json:"TimeOut"`
}

type SLBListener struct {
	lb *SLoadbalancer

	Protocol          string            `json:"Protocol"` // 监听器协议类型，取值 TCP | UDP | HTTP | HTTPS | TCP_SSL
	Certificate       certificate       `json:"Certificate"`
	SniSwitch         int64             `json:"SniSwitch"`   // 是否开启SNI特性（本参数仅对于HTTPS监听器有意义）
	HealthCheck       healthCheck       `json:"HealthCheck"` // 仅适用于TCP/UDP/TCP_SSL监听器
	ListenerID        string            `json:"ListenerId"`
	ListenerName      string            `json:"ListenerName"`
	Rules             []SLBListenerRule `json:"Rules"` // 监听器下的全部转发规则（本参数仅对于HTTP/HTTPS监听器有意义）
	Scheduler         string            `json:"Scheduler"`
	SessionExpireTime int               `json:"SessionExpireTime"` // 会话保持时间，单位：秒。可选值：30~3600，默认 0，表示不开启。此参数仅适用于TCP/UDP监听器。
	Port              int               `json:"Port"`
	ClassicListener   bool              // 这个字段是在qcloud返回字段基础上,额外增加的字段。用于区分listener 是否是classic。
}

// todo: 腾讯云后端端口不是与listener绑定的
func (self *SLBListener) GetBackendServerPort() int {
	return 0
}

func (self *SLBListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	panic("implement me")
}

func (self *SLBListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := self.GetILoadbalancerListenerRules()
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if rule.GetId() == ruleId {
			return rule, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SLBListener) Start() error {
	panic("implement me")
}

func (self *SLBListener) Stop() error {
	panic("implement me")
}

func (self *SLBListener) Sync(listener *cloudprovider.SLoadbalancerListener) error {
	panic("implement me")
}

func (self *SLBListener) Delete() error {
	panic("implement me")
}

// https://cloud.tencent.com/document/api/214/30694#ClassicalListener
type SLBClassicListener struct {
	InstancePort  int64  `json:"InstancePort"`
	CERTCAID      string `json:"CertCaId"`
	Status        int64  `json:"Status"`
	CERTID        string `json:"CertId"`
	Protocol      string `json:"Protocol"`
	TimeOut       int    `json:"TimeOut"`
	HTTPHash      string `json:"HttpHash"` // 公网固定IP型的 HTTP、HTTPS 协议监听器的轮询方法。wrr 表示按权重轮询，ip_hash 表示根据访问的源 IP 进行一致性哈希方式来分发
	UnhealthNum   int    `json:"UnhealthNum"`
	IntervalTime  int    `json:"IntervalTime"`
	ListenerID    string `json:"ListenerId"`
	ListenerPort  int    `json:"ListenerPort"`
	HTTPCheckPath string `json:"HttpCheckPath"`
	HealthNum     int    `json:"HealthNum"`
	ListenerName  string `json:"ListenerName"`
	HealthSwitch  int64  `json:"HealthSwitch"`
	SSLMode       string `json:"SSLMode"`
	SessionExpire int    `json:"SessionExpire"`
	HTTPCode      int    `json:"HttpCode"`
}

func (self *SLBClassicListener) ToLBListener() SLBListener {
	// 转换之后丢弃了 InstancePort、Status、HttpHash
	return SLBListener{
		Protocol: self.Protocol,
		Certificate: certificate{
			SSLMode:  self.SSLMode,
			CERTCAID: self.CERTCAID,
			CERTID:   self.CERTID,
		},
		HealthCheck: healthCheck{
			HTTPCheckDomain: "",
			HealthSwitch:    self.HealthSwitch,
			HTTPCheckPath:   self.HTTPCheckPath,
			HTTPCheckMethod: "",
			UnHealthNum:     self.UnhealthNum,
			IntervalTime:    self.IntervalTime,
			HTTPCode:        self.HTTPCode,
			HealthNum:       self.HealthNum,
			TimeOut:         self.TimeOut,
		},
		ListenerID:        self.ListenerID,
		ListenerName:      self.ListenerName,
		Rules:             nil,
		Scheduler:         self.HTTPHash,
		SessionExpireTime: self.SessionExpire,
		Port:              self.ListenerPort,
		ClassicListener:   true,
	}
}

func (self *SLBListener) GetId() string {
	return self.ListenerID
}

func (self *SLBListener) GetName() string {
	return self.ListenerName
}

func (self *SLBListener) GetGlobalId() string {
	return self.ListenerID
}

// 腾讯云负载均衡没有启用禁用操作
func (self *SLBListener) GetStatus() string {
	return models.LB_STATUS_ENABLED
}

func (self *SLBListener) Refresh() error {
	panic("implement me")
}

func (self *SLBListener) IsEmulated() bool {
	return false
}

func (self *SLBListener) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLBListener) GetListenerType() string {
	switch self.Protocol {
	case "TCP":
		return models.LB_LISTENER_TYPE_TCP
	case "UDP":
		return models.LB_LISTENER_TYPE_UDP
	case "HTTP":
		return models.LB_LISTENER_TYPE_HTTP
	case "HTTPS":
		return models.LB_LISTENER_TYPE_HTTPS
	case "TCP_SSL":
		return models.LB_LISTENER_TYPE_TCP
	default:
		return ""
	}
}

func (self *SLBListener) GetListenerPort() int {
	return self.Port
}

func (self *SLBListener) GetScheduler() string {
	switch strings.ToLower(self.Scheduler) {
	case "wrr":
		return models.LB_SCHEDULER_WRR
	case "ip_hash":
		return models.LB_SCHEDULER_SCH
	case "least_conn":
		return models.LB_SCHEDULER_WLC
	default:
		return ""
	}
}

func (self *SLBListener) GetAclStatus() string {
	return ""
}

func (self *SLBListener) GetAclType() string {
	return ""
}

func (self *SLBListener) GetAclId() string {
	return ""
}

func (self *SLBListener) GetHealthCheck() string {
	if self.HealthCheck.HealthSwitch == 0 {
		return models.LB_HEALTH_CHECK_DISABLE
	} else {
		return models.LB_HEALTH_CHECK_ENABLE
	}
}

// todo: ?待确认
func (self *SLBListener) GetHealthCheckType() string {
	if len(self.HealthCheck.HTTPCheckMethod) > 0 {
		return models.LB_HEALTH_CHECK_HTTP
	} else {
		return models.LB_HEALTH_CHECK_TCP
	}
}

func (self *SLBListener) GetHealthCheckTimeout() int {
	return self.HealthCheck.TimeOut
}

func (self *SLBListener) GetHealthCheckInterval() int {
	return self.HealthCheck.IntervalTime
}

func (self *SLBListener) GetHealthCheckRise() int {
	return self.HealthCheck.HealthNum
}

func (self *SLBListener) GetHealthCheckFail() int {
	return self.HealthCheck.UnHealthNum
}

func (self *SLBListener) GetHealthCheckReq() string {
	return ""
}

func (self *SLBListener) GetHealthCheckExp() string {
	return ""
}

func (self *SLBListener) GetBackendGroup() *SLBBackendGroup {
	t := self.GetListenerType()
	// http、https类型的监听不能直接绑定服务器
	if t == models.LB_LISTENER_TYPE_HTTP || t == models.LB_LISTENER_TYPE_HTTPS {
		return nil
	} else {
		return &SLBBackendGroup{lb: self.lb, listener: self}
	}
}

func (self *SLBListener) GetBackendGroupId() string {
	bg := self.GetBackendGroup()
	if bg == nil {
		return ""
	}

	return bg.GetId()
}

func (self *SLBListener) GetHealthCheckDomain() string {
	return self.HealthCheck.HTTPCheckDomain
}

func (self *SLBListener) GetHealthCheckURI() string {
	return self.HealthCheck.HTTPCheckPath
}

func (self *SLBListener) GetHealthCheckCode() string {
	codes := []string{}
	for i := uint8(0); i < 5; i++ {
		n := 1 << i
		if (self.HealthCheck.HTTPCode & n) == n {
			codes = append(codes, HTTP_CODES[i])
		}
	}

	return strings.Join(codes, ",")
}

// 仅http、https类型监听包含rules
func (self *SLBListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules := self.Rules
	iRules := []cloudprovider.ICloudLoadbalancerListenerRule{}
	for i := 0; i < len(rules); i++ {
		rules[i].listener = self
		iRules = append(iRules, &rules[i])
	}
	return iRules, nil
}

// todo: ??
func (self *SLBListener) GetStickySession() string {
	if self.SessionExpireTime == 0 {
		return models.LB_BOOL_OFF
	} else {
		return models.LB_BOOL_ON
	}
}

// 支持基于 cookie 插入的会话保持能力 https://cloud.tencent.com/document/product/214/6154
func (self *SLBListener) GetStickySessionType() string {
	return models.LB_STICKY_SESSION_TYPE_INSERT
}

// todo: 腾讯云不支持？
func (self *SLBListener) GetStickySessionCookie() string {
	return ""
}

func (self *SLBListener) GetStickySessionCookieTimeout() int {
	return self.SessionExpireTime
}

/*
7层负载均衡系统提供 X-Forwarded-For 的方式获取访问者真实 IP，LB 侧默认开启

https://cloud.tencent.com/document/product/214/6151
七层转发获取来访真实IP的方法 https://cloud.tencent.com/document/product/214/3728
*/
func (self *SLBListener) XForwardedForEnabled() bool {
	switch self.GetListenerType() {
	case models.LB_LISTENER_TYPE_HTTP, models.LB_LISTENER_TYPE_HTTPS:
		return true
	default:
		return false
	}
}

// HTTP/HTTPS协议默认支持用户开启gzip压缩功能
// 负载均衡开启Gzip配置及检测方法说明 https://cloud.tencent.com/document/product/214/5404
func (self *SLBListener) GzipEnabled() bool {
	switch self.GetListenerType() {
	case models.LB_LISTENER_TYPE_HTTP, models.LB_LISTENER_TYPE_HTTPS:
		return true
	default:
		return false
	}
}

func (self *SLBListener) GetCertificateId() string {
	return self.Certificate.CERTID
}

// todo: ??
func (self *SLBListener) GetTLSCipherPolicy() string {
	return ""
}

// todo: ?
// 负载均衡能力说明 https://cloud.tencent.com/document/product/214/6534
func (self *SLBListener) HTTP2Enabled() bool {
	return true
}

func (self *SRegion) GetLoadbalancerListeners(lbid string, t LB_TYPE, protocol string) ([]SLBListener, error) {
	params := map[string]string{"LoadBalancerId": lbid}
	if len(protocol) > 0 {
		params["Protocol"] = protocol
	}

	listeners := []SLBListener{}
	if t == LB_TYPE_CLASSIC {
		resp, err := self.clbRequest("DescribeClassicalLBListeners", params)
		if err != nil {
			return nil, err
		}

		clisteners := []SLBClassicListener{}
		err = resp.Unmarshal(&clisteners, "Listeners")
		if err != nil {
			return nil, err
		}

		for _, l := range clisteners {
			listeners = append(listeners, l.ToLBListener())
		}

		return listeners, nil
	}

	resp, err := self.clbRequest("DescribeListeners", params)
	if err != nil {
		return nil, err
	}

	err = resp.Unmarshal(&listeners, "Listeners")
	if err != nil {
		return nil, err
	}

	return listeners, nil
}
