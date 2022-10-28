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

package openstack

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancerListenerCreateParams struct {
	Protocol                string         `json:"protocol,omitempty"`
	Description             string         `json:"description,omitempty"`
	AdminStateUp            bool           `json:"admin_state_up,omitempty"`
	ConnectionLimit         *int           `json:"connection_limit,omitempty"`
	ProtocolPort            string         `json:"protocol_port,omitempty"`
	LoadbalancerID          string         `json:"loadbalancer_id,omitempty"`
	DefaultPoolId           string         `json:"default_pool_id,omitempty"`
	Name                    string         `json:"name,omitempty"`
	InsertHeaders           SInsertHeaders `json:"insert_headers,omitempty"`
	DefaultTLSContainerRef  string         `json:"default_tls_container_ref,omitempty"`
	SniContainerRefs        []string       `json:"sni_container_refs,omitempty"`
	TimeoutClientData       *int           `json:"timeout_client_data,omitempty"`
	TimeoutMemberConnect    *int           `json:"timeout_member_connect,omitempty"`
	TimeoutMemberData       *int           `json:"timeout_member_data,omitempty"`
	TimeoutTCPInspect       *int           `json:"timeout_tcp_inspect,omitempty"`
	Tags                    []string       `json:"tags,omitempty"`
	ClientCaTLSContainerRef string         `json:"client_ca_tls_container_ref,omitempty"`
	ClientAuthentication    string         `json:"client_authentication,omitempty"`
	ClientCrlContainerRef   string         `json:"client_crl_container_ref,omitempty"`
	AllowedCidrs            []string       `json:"allowed_cidrs,omitempty"`
	TLSCiphers              string         `json:"tls_ciphers,omitempty"`
	TLSVersions             []string       `json:"tls_versions,omitempty"`
}

type SLoadbalancerListenerUpdateParams struct {
	Description             string         `json:"description,omitempty"`
	AdminStateUp            bool           `json:"admin_state_up,omitempty"`
	ConnectionLimit         *int           `json:"connection_limit,omitempty"`
	DefaultPoolId           string         `json:"default_pool_id,omitempty"`
	Name                    string         `json:"name,omitempty"`
	InsertHeaders           SInsertHeaders `json:"insert_headers,omitempty"`
	DefaultTLSContainerRef  string         `json:"default_tls_container_ref,omitempty"`
	SniContainerRefs        []string       `json:"sni_container_refs,omitempty"`
	TimeoutClientData       *int           `json:"timeout_client_data,omitempty"`
	TimeoutMemberConnect    *int           `json:"timeout_member_connect,omitempty"`
	TimeoutMemberData       *int           `json:"timeout_member_data,omitempty"`
	TimeoutTCPInspect       *int           `json:"timeout_tcp_inspect,omitempty"`
	Tags                    []string       `json:"tags,omitempty"`
	ClientCaTLSContainerRef string         `json:"client_ca_tls_container_ref,omitempty"`
	ClientAuthentication    string         `json:"client_authentication,omitempty"`
	ClientCrlContainerRef   string         `json:"client_crl_container_ref,omitempty"`
	AllowedCidrs            []string       `json:"allowed_cidrs,omitempty"`
	TLSCiphers              string         `json:"tls_ciphers,omitempty"`
	TLSVersions             []string       `json:"tls_versions,omitempty"`
}

type SInsertHeaders struct {
	XForwardedPort string `json:"X-Forwarded-Port"`
	XForwardedFor  string `json:"X-Forwarded-For"`
}

type SLoadbalancerListener struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	OpenStackTags
	region                  *SRegion
	l7policies              []SLoadbalancerL7Policy
	pools                   []SLoadbalancerPool
	Description             string            `json:"description"`
	AdminStateUp            bool              `json:"admin_state_up"`
	ProjectID               string            `json:"project_id"`
	Protocol                string            `json:"protocol"`
	ProtocolPort            int               `json:"protocol_port"`
	ProvisioningStatus      string            `json:"provisioning_status"`
	DefaultTLSContainerRef  string            `json:"default_tls_container_ref"`
	LoadbalancerIds         []SLoadbalancerID `json:"loadbalancers"`
	InsertHeaders           SInsertHeaders    `json:"insert_headers"`
	CreatedAt               string            `json:"created_at"`
	UpdatedAt               string            `json:"updated_at"`
	ID                      string            `json:"id"`
	OperatingStatus         string            `json:"operating_status"`
	DefaultPoolID           string            `json:"default_pool_id"`
	SniContainerRefs        []string          `json:"sni_container_refs"`
	L7PolicieIds            []SL7PolicieID    `json:"l7policies"`
	Name                    string            `json:"name"`
	TimeoutClientData       int               `json:"timeout_client_data"`
	TimeoutMemberConnect    int               `json:"timeout_member_connect"`
	TimeoutMemberData       int               `json:"timeout_member_data"`
	TimeoutTCPInspect       int               `json:"timeout_tcp_inspect"`
	Tags                    []string          `json:"tags"`
	ClientCaTLSContainerRef string            `json:"client_ca_tls_container_ref"`
	ClientAuthentication    string            `json:"client_authentication"`
	ClientCrlContainerRef   string            `json:"client_crl_container_ref"`
	AllowedCidrs            []string          `json:"allowed_cidrs"`
	TLSCiphers              string            `json:"tls_ciphers"`
	TLSVersions             []string          `json:"tls_versions"`
}

func (listener *SLoadbalancerListener) GetName() string {
	if len(listener.Name) == 0 {
		listener.Refresh()
	}
	if len(listener.Name) > 0 {
		return listener.Name
	}
	return fmt.Sprintf("HTTP:%d", listener.ProtocolPort)
}

func (listener *SLoadbalancerListener) GetId() string {
	return listener.ID
}

func (listener *SLoadbalancerListener) GetGlobalId() string {
	return listener.GetId()
}

func (listener *SLoadbalancerListener) GetStatus() string {
	switch listener.ProvisioningStatus {
	case "ACTIVE":
		return api.LB_STATUS_ENABLED
	case "PENDING_CREATE":
		return api.LB_CREATING
	case "PENDING_UPDATE":
		return api.LB_SYNC_CONF
	case "PENDING_DELETE":
		return api.LB_STATUS_DELETING
	case "DELETED":
		return api.LB_STATUS_DELETED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (listener *SLoadbalancerListener) IsEmulated() bool {
	return false
}

func (listener *SLoadbalancerListener) GetEgressMbps() int {

	return 0
}

func (region *SRegion) GetLoadbalancerListeners() ([]SLoadbalancerListener, error) {
	listeners := []SLoadbalancerListener{}
	resource := "/v2/lbaas/listeners"
	query := url.Values{}
	for {
		resp, err := region.lbList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "lbList")
		}
		part := struct {
			Listeners      []SLoadbalancerListener
			ListenersLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		listeners = append(listeners, part.Listeners...)
		marker := part.ListenersLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}

	for i := 0; i < len(listeners); i++ {
		listeners[i].region = region
	}
	for i := 0; i < len(listeners); i++ {
		err := listeners[i].fetchLoadbalancerListenerL7Policies()
		if err != nil {
			return nil, errors.Wrap(err, "listener.fetchLoadbalancerListenerL7Policies()")
		}
	}

	for i := 0; i < len(listeners); i++ {
		err := listeners[i].fetchLoadbalancerPools()
		if err != nil {
			return nil, errors.Wrap(err, "listeners[i].fetchLoadbalancerPools()")
		}
	}

	return listeners, nil
}

func (region *SRegion) GetLoadbalancerListenerbyId(listenerId string) (*SLoadbalancerListener, error) {
	resp, err := region.lbGet(fmt.Sprintf("/v2/lbaas/listeners/%s", listenerId))
	if err != nil {
		return nil, errors.Wrapf(err, "region.Get(/v2/lbaas/listeners/%s)", listenerId)
	}
	listener := SLoadbalancerListener{}
	err = resp.Unmarshal(&listener, "listener")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal(&listener, listener)")
	}
	listener.region = region
	err = listener.fetchLoadbalancerListenerL7Policies()
	if err != nil {
		return nil, errors.Wrap(err, "listener.fetchLoadbalancerListenerL7Policies()")
	}

	err = listener.fetchLoadbalancerPools()
	if err != nil {
		return nil, errors.Wrap(err, "listeners[i].fetchLoadbalancerPools()")

	}
	return &listener, nil
}

func (region *SRegion) CreateLoadbalancerListener(loadbalancerId string, listenerParams *cloudprovider.SLoadbalancerListener) (*SLoadbalancerListener, error) {
	type CreateParams struct {
		Listener SLoadbalancerListenerCreateParams `json:"listener"`
	}
	params := CreateParams{}
	params.Listener.AdminStateUp = true
	params.Listener.LoadbalancerID = loadbalancerId
	params.Listener.DefaultPoolId = listenerParams.BackendGroupID
	params.Listener.Protocol = LB_PROTOCOL_MAP[listenerParams.ListenerType]
	params.Listener.ProtocolPort = strconv.Itoa(listenerParams.ListenerPort)
	if listenerParams.ClientIdleTimeout != 0 {
		// 毫秒单位
		msClientIdleTimeout := listenerParams.ClientIdleTimeout * 1000
		params.Listener.TimeoutClientData = &msClientIdleTimeout
	}
	if listenerParams.BackendConnectTimeout != 0 {
		msBackendConnectTimeout := listenerParams.BackendConnectTimeout * 1000
		params.Listener.TimeoutMemberConnect = &msBackendConnectTimeout
	}
	if listenerParams.BackendIdleTimeout != 0 {
		msBackendIdleTimeout := listenerParams.BackendIdleTimeout * 1000
		params.Listener.TimeoutMemberData = &msBackendIdleTimeout
	}

	params.Listener.Name = listenerParams.Name
	if listenerParams.XForwardedFor {
		params.Listener.InsertHeaders.XForwardedFor = "true"
	}
	body, err := region.lbPost("/v2/lbaas/listeners", jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, "region.Post(/v2/lbaas/listeners)")
	}
	slistener := SLoadbalancerListener{}
	slistener.region = region
	return &slistener, body.Unmarshal(&slistener, "listener")
}

func (listener *SLoadbalancerListener) Refresh() error {
	newlistener, err := listener.region.GetLoadbalancerListenerbyId(listener.ID)
	if err != nil {
		return errors.Wrapf(err, "listener.region.GetLoadbalancerListenerbyId(%s)", listener.ID)
	}
	return jsonutils.Update(listener, newlistener)
}

func (listener *SLoadbalancerListener) GetListenerType() string {
	switch listener.Protocol {
	case "HTTP":
		return api.LB_LISTENER_TYPE_HTTP
	case "HTTPS":
		return api.LB_LISTENER_TYPE_HTTPS
	case "TERMINATED_HTTPS":
		return api.LB_LISTENER_TYPE_TERMINATED_HTTPS
	case "TCP":
		return api.LB_LISTENER_TYPE_TCP
	case "UDP":
		return api.LB_LISTENER_TYPE_UDP
	default:
		return ""
	}
}

func (listener *SLoadbalancerListener) GetListenerPort() int {
	return listener.ProtocolPort
}

func (listener *SLoadbalancerListener) GetBackendGroupId() string {
	return listener.DefaultPoolID
}

func (listener *SLoadbalancerListener) GetBackendServerPort() int {
	return listener.ProtocolPort
}

func (listener *SLoadbalancerListener) GetScheduler() string {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetScheduler():listener.fetchFeaturePool():%s", err)
		return ""
	}
	switch pool.LbAlgorithm {
	case "ROUND_ROBIN":
		return api.LB_SCHEDULER_WRR
	case "LEAST_CONNECTIONS":
		return api.LB_SCHEDULER_WLC
	case "SOURCE_IP":
		return api.LB_SCHEDULER_SCH
	case "SOURCE_IP_PORT":
		return api.LB_SCHEDULER_TCH
	default:
		return ""
	}
}

func (listener *SLoadbalancerListener) GetAclStatus() string {
	if len(listener.AllowedCidrs) > 0 {
		return api.LB_BOOL_ON
	}
	return api.LB_BOOL_OFF
}

func (listener *SLoadbalancerListener) GetAclType() string {
	return api.LB_ACL_TYPE_WHITE
}

func (listener *SLoadbalancerListener) GetAclId() string {
	return ""
}

func (listener *SLoadbalancerListener) GetHealthCheck() string {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetHealthCheck():listener.fetchFeaturePool():%s", err)
		return ""
	}
	if pool.healthmonitor != nil {
		return api.LB_BOOL_ON
	}
	return api.LB_BOOL_OFF

}

func (listener *SLoadbalancerListener) GetHealthCheckType() string {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetHealthCheckType():listener.fetchFeaturePool():%s", err)
		return ""
	}
	if pool.healthmonitor == nil {
		return ""
	}
	switch pool.healthmonitor.Type {
	case "HTTP":
		return api.LB_HEALTH_CHECK_HTTP
	case "HTTPS":
		return api.LB_HEALTH_CHECK_HTTPS
	case "TCP":
		return api.LB_HEALTH_CHECK_TCP
	case "UDP-CONNECT":
		return api.LB_HEALTH_CHECK_UDP
	default:
		return ""
	}
}

func (listener *SLoadbalancerListener) GetHealthCheckDomain() string {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetHealthCheckDomain():listener.fetchFeaturePool():%s", err)
		return ""
	}
	if pool.healthmonitor == nil {
		return ""
	}
	return pool.healthmonitor.DomainName
}

func (listener *SLoadbalancerListener) GetHealthCheckURI() string {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetHealthCheckURI():listener.fetchFeaturePool():%s", err)
		return ""
	}
	if pool.healthmonitor == nil {
		return ""
	}
	return pool.healthmonitor.URLPath
}

func (listener *SLoadbalancerListener) GetHealthCheckCode() string {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetHealthCheckCode():listener.fetchFeaturePool():%s", err)
		return ""
	}
	if pool.healthmonitor == nil {
		return ""
	}
	return pool.healthmonitor.ExpectedCodes
}

func (listener *SLoadbalancerListener) GetHealthCheckRise() int {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetHealthCheckRise():listener.fetchFeaturePool():%s", err)
		return 0
	}
	if pool.healthmonitor == nil {
		return 0
	}
	return pool.healthmonitor.MaxRetries
}

func (listener *SLoadbalancerListener) GetHealthCheckFail() int {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetHealthCheckFail():listener.fetchFeaturePool():%s", err)
		return 0
	}
	if pool.healthmonitor == nil {
		return 0
	}
	return pool.healthmonitor.MaxRetriesDown
}

func (listener *SLoadbalancerListener) GetHealthCheckTimeout() int {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetHealthCheckTimeout():listener.fetchFeaturePool():%s", err)
		return 0
	}
	if pool.healthmonitor == nil {
		return 0
	}
	return pool.healthmonitor.Timeout
}

func (listener *SLoadbalancerListener) GetHealthCheckInterval() int {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetHealthCheckInterval():listener.fetchFeaturePool():%s", err)
		return 0
	}
	if pool.healthmonitor == nil {
		return 0
	}
	return pool.healthmonitor.Delay
}

func (listener *SLoadbalancerListener) GetHealthCheckReq() string {
	return ""
}

func (listener *SLoadbalancerListener) GetHealthCheckExp() string {
	return ""
}

func (listener *SLoadbalancerListener) GetStickySession() string {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetStickySession():listener.fetchFeaturePool():%s", err)
		return api.LB_BOOL_OFF
	}
	stickySession, err := pool.GetStickySession()
	if err != nil {
		log.Errorf("GetStickySession():listener.fetchFeaturePool():%s", err)
		return api.LB_BOOL_OFF
	}
	if stickySession == nil {
		return ""
	}
	return stickySession.StickySession
}

func (listener *SLoadbalancerListener) GetStickySessionType() string {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetStickySession():listener.fetchFeaturePool():%s", err)
		return ""
	}
	stickySession, err := pool.GetStickySession()
	if err != nil {
		log.Errorf("GetStickySession():listener.fetchFeaturePool():%s", err)
		return ""
	}
	if stickySession == nil {
		return ""
	}
	return stickySession.StickySessionType
}

func (listener *SLoadbalancerListener) GetStickySessionCookie() string {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetStickySession():listener.fetchFeaturePool():%s", err)
		return ""
	}
	stickySession, err := pool.GetStickySession()
	if err != nil {
		log.Errorf("GetStickySession():listener.fetchFeaturePool():%s", err)
		return ""
	}
	if stickySession == nil {
		return ""
	}
	return stickySession.StickySessionCookie
}

func (listener *SLoadbalancerListener) GetStickySessionCookieTimeout() int {
	pool, err := listener.fetchFeaturePool()
	if err != nil {
		log.Errorf("GetStickySession():listener.fetchFeaturePool():%s", err)
		return 0
	}
	stickySession, err := pool.GetStickySession()
	if err != nil {
		log.Errorf("GetStickySession():listener.fetchFeaturePool():%s", err)
		return 0
	}
	if stickySession == nil {
		return 0
	}
	return stickySession.StickySessionCookieTimeout
}

func (listener *SLoadbalancerListener) XForwardedForEnabled() bool {
	if listener.InsertHeaders.XForwardedFor == "true" {
		return true
	}
	return false
}

func (listener *SLoadbalancerListener) GzipEnabled() bool {
	return false
}

func (listener *SLoadbalancerListener) GetCertificateId() string {
	return ""
}

func (listener *SLoadbalancerListener) GetTLSCipherPolicy() string {
	return listener.TLSCiphers
}

func (listener *SLoadbalancerListener) HTTP2Enabled() bool {
	return false
}

func (listener *SLoadbalancerListener) fetchLoadbalancerListenerL7Policies() error {
	l7policies := []SLoadbalancerL7Policy{}
	for i := 0; i < len(listener.L7PolicieIds); i++ {
		l7policy, err := listener.region.GetLoadbalancerL7PolicybyId(listener.L7PolicieIds[i].ID)
		if err != nil {
			return errors.Wrapf(err, "listener.region.GetLoadbalancerL7PolicybyId(%s)", listener.L7PolicieIds[i].ID)
		}
		l7policies = append(l7policies, *l7policy)
	}
	listener.l7policies = l7policies
	return nil
}

func (listener *SLoadbalancerListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	iRules := []cloudprovider.ICloudLoadbalancerListenerRule{}
	for i := 0; i < len(listener.l7policies); i++ {
		for j := 0; j < len(listener.l7policies[i].l7rules); j++ {
			iRules = append(iRules, &listener.l7policies[i].l7rules[j])
		}
	}
	return iRules, nil
}

func (region *SRegion) DeleteLoadbalancerListener(listenerId string) error {
	_, err := region.lbDelete(fmt.Sprintf("/v2/lbaas/listeners/%s", listenerId))
	if err != nil {
		return errors.Wrapf(err, `region.lbDelete("/v2/lbaas/listeners/%s")`, listenerId)
	}
	return nil
}

func (listener *SLoadbalancerListener) Delete(ctx context.Context) error {
	waitLbResStatus(listener, 10*time.Second, 1*time.Minute)
	return listener.region.DeleteLoadbalancerListener(listener.ID)
}

func (listener *SLoadbalancerListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	l7policy, err := listener.region.CreateLoadbalancerL7Policy(listener.ID, rule)
	if err != nil {
		return nil, errors.Wrapf(err, `listener.region.CreateLoadbalancerL7Policy(%s, rule)`, listener.ID)
	}
	// async wait
	err = waitLbResStatus(l7policy, 10*time.Second, 8*time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, `waitLbResStatus(l7policy, 10*time.Second, 8*time.Minute)`)
	}

	l7rule, err := listener.region.CreateLoadbalancerL7Rule(l7policy.ID, rule)
	if err != nil {
		return nil, errors.Wrapf(err, `listener.region.CreateLoadbalancerL7Rule(%s, rule)`, l7policy.ID)
	}
	l7rule.policy = l7policy
	// async wait
	err = waitLbResStatus(l7rule, 10*time.Second, 8*time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, `waitLbResStatus(l7rule, 10*time.Second, 8*time.Minute)`)
	}
	return l7rule, nil
}

func (listener *SLoadbalancerListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	for i := 0; i < len(listener.l7policies); i++ {
		for j := 0; j < len(listener.l7policies[i].l7rules); j++ {
			if listener.l7policies[i].l7rules[j].GetId() == ruleId {
				return &listener.l7policies[i].l7rules[j], nil
			}
		}
	}
	return nil, nil
}

func (listener *SLoadbalancerListener) fetchLoadbalancerPools() error {
	pools := []SLoadbalancerPool{}
	if len(listener.DefaultPoolID) > 0 {
		defaultPool, err := listener.region.GetLoadbalancerPoolById(listener.DefaultPoolID)
		if err != nil {
			return errors.Wrapf(err, "listener.region.GetLoadbalancerPoolById(%s)", listener.DefaultPoolID)
		}
		pools = append(pools, *defaultPool)
	}
	for i := 0; i < len(listener.l7policies); i++ {
		if len(listener.l7policies[i].RedirectPoolID) > 0 {
			policyPool, err := listener.region.GetLoadbalancerPoolById(listener.l7policies[i].RedirectPoolID)
			if err != nil {
				return errors.Wrapf(err, "listener.region.GetLoadbalancerPoolById(%s)", listener.l7policies[i].RedirectPoolID)
			}
			pools = append(pools, *policyPool)
		}
	}
	listener.pools = pools
	return nil
}

func (listener *SLoadbalancerListener) fetchFeaturePool() (*SLoadbalancerPool, error) {
	if len(listener.pools) < 1 {
		return nil, fmt.Errorf("can't find pool with healthmonitor")
	}
	for i := 0; i < len(listener.pools); i++ {
		if listener.pools[i].healthmonitor != nil {
			return &listener.pools[i], nil
		}
	}
	return &listener.pools[0], nil
}

func (region *SRegion) UpdateLoadBalancerListenerAdminStateUp(AdminStateUp bool, loadbalancerListenerId string) error {
	params := jsonutils.NewDict()
	poolParam := jsonutils.NewDict()
	poolParam.Add(jsonutils.NewBool(AdminStateUp), "admin_state_up")
	params.Add(poolParam, "listener")
	_, err := region.lbUpdate(fmt.Sprintf("/v2/lbaas/listeners/%s", loadbalancerListenerId), params)
	if err != nil {
		return errors.Wrapf(err, `region.lbUpdate(/v2/lbaas/listeners/%s, params)`, loadbalancerListenerId)
	}
	return nil
}

func (region *SRegion) UpdateLoadBalancerListener(loadbalancerListenerId string, lblis *cloudprovider.SLoadbalancerListener) error {
	type UpdateParams struct {
		Listener SLoadbalancerListenerUpdateParams `json:"listener"`
	}
	params := UpdateParams{}
	params.Listener.AdminStateUp = true
	params.Listener.DefaultPoolId = lblis.BackendGroupID

	if lblis.ClientIdleTimeout != 0 {
		// 毫秒单位
		msClientIdleTimeout := lblis.ClientIdleTimeout * 1000
		params.Listener.TimeoutClientData = &msClientIdleTimeout
	}
	if lblis.BackendConnectTimeout != 0 {
		msBackendConnectTimeout := lblis.BackendConnectTimeout * 1000
		params.Listener.TimeoutMemberConnect = &msBackendConnectTimeout
	}
	if lblis.BackendIdleTimeout != 0 {
		msBackendIdleTimeout := lblis.BackendIdleTimeout * 1000
		params.Listener.TimeoutMemberData = &msBackendIdleTimeout
	}
	params.Listener.Name = lblis.Name
	if lblis.XForwardedFor {
		params.Listener.InsertHeaders.XForwardedFor = "true"
	}
	_, err := region.lbUpdate(fmt.Sprintf("/v2/lbaas/listeners/%s", loadbalancerListenerId), jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrapf(err, `region.lbUpdate(/v2/lbaas/listeners/%s, jsonutils.Marshal(params))`, loadbalancerListenerId)
	}
	return nil
}

func (listener *SLoadbalancerListener) Start() error {
	// ensure listener status
	err := waitLbResStatus(listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, ` waitLbResStatus(listener, 10*time.Second, 8*time.Minute)`)
	}
	err = listener.region.UpdateLoadBalancerListenerAdminStateUp(true, listener.ID)
	if err != nil {
		return errors.Wrapf(err, `listener.region.UpdateLoadBalancerListenerAdminStateUp(true, %s)`, listener.ID)
	}
	err = waitLbResStatus(listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(listener, 10*time.Second, 8*time.Minute)`)
	}
	return nil
}

func (listener *SLoadbalancerListener) Stop() error {
	// ensure listener status
	err := waitLbResStatus(listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, ` waitLbResStatus(listener, 10*time.Second, 8*time.Minute)`)
	}
	err = listener.region.UpdateLoadBalancerListenerAdminStateUp(false, listener.ID)
	if err != nil {
		return errors.Wrapf(err, `listener.region.UpdateLoadBalancerListenerAdminStateUp(false,%s)`, listener.ID)
	}
	err = waitLbResStatus(listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(listener, 10*time.Second, 8*time.Minute)`)
	}
	return nil
}

func (listener *SLoadbalancerListener) Sync(ctx context.Context, lblis *cloudprovider.SLoadbalancerListener) error {
	// ensure listener status
	err := waitLbResStatus(listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, ` waitLbResStatus(listener, 10*time.Second, 8*time.Minute)`)
	}
	err = listener.region.UpdateLoadBalancerListener(listener.ID, lblis)
	if err != nil {
		return errors.Wrapf(err, `listener.region.UpdateLoadBalancerListener(%s, lblis)`, listener.ID)
	}
	err = waitLbResStatus(listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(listener, 10*time.Second, 8*time.Minute)`)
	}
	return nil
}

func (listener *SLoadbalancerListener) GetProjectId() string {
	return listener.ProjectID
}

func (listener *SLoadbalancerListener) GetClientIdleTimeout() int {
	return 0
}

func (listener *SLoadbalancerListener) GetBackendConnectTimeout() int {
	return 0
}
