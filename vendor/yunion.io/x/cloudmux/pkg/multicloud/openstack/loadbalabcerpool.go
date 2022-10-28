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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancerPoolCreateParams struct {
	LbAlgorithm        string               `json:"lb_algorithm,omitempty"`
	Protocol           string               `json:"protocol,omitempty"`
	Description        string               `json:"description,omitempty"`
	AdminStateUp       bool                 `json:"admin_state_up,omitempty"`
	SessionPersistence *SSessionPersistence `json:"session_persistence"`
	LoadbalancerID     string               `json:"loadbalancer_id,omitempty"`
	ListenerID         string               `json:"listener_id,omitempty"`
	Name               string               `json:"name,omitempty"`
	Tags               []string             `json:"tags,omitempty"`
	TLSContainerRef    string               `json:"tls_container_ref,omitempty"`
	CaTLSContainerRef  string               `json:"ca_tls_container_ref,omitempty"`
	CrlContainerRef    string               `json:"crl_container_ref,omitempty"`
	TLSEnabled         *bool                `json:"tls_enabled,omitempty"`
	TLSCiphers         string               `json:"tls_ciphers,omitempty"`
	TLSVersions        []string             `json:"tls_versions,omitempty"`
}

type SLoadbalancerPoolUpdateParams struct {
	LbAlgorithm        string               `json:"lb_algorithm,omitempty"`
	Description        string               `json:"description,omitempty"`
	AdminStateUp       bool                 `json:"admin_state_up,omitempty"`
	SessionPersistence *SSessionPersistence `json:"session_persistence"`
	Name               string               `json:"name,omitempty"`
	Tags               []string             `json:"tags,omitempty"`
	TLSContainerRef    string               `json:"tls_container_ref,omitempty"`
	CaTLSContainerRef  string               `json:"ca_tls_container_ref,omitempty"`
	CrlContainerRef    string               `json:"crl_container_ref,omitempty"`
	TLSEnabled         *bool                `json:"tls_enabled,omitempty"`
	TLSCiphers         string               `json:"tls_ciphers,omitempty"`
	TLSVersions        []string             `json:"tls_versions,omitempty"`
}

type SSessionPersistence struct {
	CookieName string `json:"cookie_name,omitempty"`
	Type       string `json:"type,omitempty"`
}

type SLoadbalancerPool struct {
	multicloud.SResourceBase
	OpenStackTags
	region             *SRegion
	members            []SLoadbalancerMember
	healthmonitor      *SLoadbalancerHealthmonitor
	LbAlgorithm        string              `json:"lb_algorithm"`
	Protocol           string              `json:"protocol"`
	Description        string              `json:"description"`
	AdminStateUp       bool                `json:"admin_state_up"`
	LoadbalancerIds    []SLoadbalancerID   `json:"loadbalancers"`
	CreatedAt          string              `json:"created_at"`
	ProvisioningStatus string              `json:"provisioning_status"`
	UpdatedAt          string              `json:"updated_at"`
	SessionPersistence SSessionPersistence `json:"session_persistence"`
	ListenerIds        []SListenerID       `json:"listeners"`
	MemberIds          []SMemberID         `json:"members"`
	HealthmonitorID    string              `json:"healthmonitor_id"`
	ProjectID          string              `json:"project_id"`
	ID                 string              `json:"id"`
	OperatingStatus    string              `json:"operating_status"`
	Name               string              `json:"name"`
	Tags               []string            `json:"tags"`
	TLSContainerRef    string              `json:"tls_container_ref"`
	CaTLSContainerRef  string              `json:"ca_tls_container_ref"`
	CrlContainerRef    string              `json:"crl_container_ref"`
	TLSEnabled         bool                `json:"tls_enabled"`
	TLSCiphers         string              `json:"tls_ciphers"`
	TLSVersions        []string            `json:"tls_versions"`
}

func ToOpenstackHealthCheckHttpCode(c string) string {
	c = strings.TrimSpace(c)
	segs := strings.Split(c, ",")
	ret := []string{}
	for _, seg := range segs {
		seg = strings.TrimLeft(seg, "http_")
		seg = strings.TrimSpace(seg)
		seg = strings.Replace(seg, "xx", "00", -1)
		ret = append(ret, seg)
	}

	return strings.Join(ret, ",")
}

func ToOnecloudHealthCheckHttpCode(c string) string {
	c = strings.TrimSpace(c)
	segs := strings.Split(c, ",")
	ret := []string{}
	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		seg = strings.Replace(seg, "00", "xx", -1)
		seg = "http_" + seg
		ret = append(ret, seg)
	}

	return strings.Join(ret, ",")
}

func (pool *SLoadbalancerPool) GetILoadbalancer() cloudprovider.ICloudLoadbalancer {
	if len(pool.LoadbalancerIds) != 1 {
		return nil
	}
	loadbalancer, err := pool.region.GetLoadbalancerbyId(pool.LoadbalancerIds[0].ID)
	if err != nil {
		return nil
	}
	return loadbalancer
}

func (pool *SLoadbalancerPool) GetLoadbalancerId() string {
	if len(pool.LoadbalancerIds) != 1 {
		return ""
	}
	return pool.LoadbalancerIds[0].ID
}

func (pool *SLoadbalancerPool) GetProtocolType() string {
	switch pool.Protocol {
	case "TCP":
		return api.LB_LISTENER_TYPE_TCP
	case "UDP":
		return api.LB_LISTENER_TYPE_UDP
	case "HTTP":
		return api.LB_LISTENER_TYPE_HTTP
	default:
		return ""
	}
}

func (pool *SLoadbalancerPool) GetScheduler() string {
	switch pool.LbAlgorithm {
	case "LEAST_CONNECTIONS":
		return api.LB_SCHEDULER_WLC
	case "ROUND_ROBIN":
		return api.LB_SCHEDULER_WRR
	case "SOURCE_IP":
		return api.LB_SCHEDULER_SCH
	case "SOURCE_IP_PORT":
		return api.LB_SCHEDULER_TCH
	default:
		return ""
	}
}

func (pool *SLoadbalancerPool) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	healthCheck := cloudprovider.SLoadbalancerHealthCheck{}
	healthCheck.HealthCheckDomain = pool.healthmonitor.DomainName
	healthCheck.HealthCheckHttpCode = ToOnecloudHealthCheckHttpCode(pool.healthmonitor.ExpectedCodes)
	healthCheck.HealthCheckInterval = pool.healthmonitor.Delay
	healthCheck.HealthCheckRise = pool.healthmonitor.MaxRetries
	healthCheck.HealthCheckFail = pool.healthmonitor.MaxRetriesDown
	healthCheck.HealthCheckTimeout = pool.healthmonitor.Timeout
	switch pool.healthmonitor.Type {
	case "HTTP":
		healthCheck.HealthCheckType = api.LB_HEALTH_CHECK_HTTP
	case "HTTPS":
		healthCheck.HealthCheckType = api.LB_HEALTH_CHECK_HTTPS
	case "TCP":
		healthCheck.HealthCheckType = api.LB_HEALTH_CHECK_TCP
	case "UDP-CONNECT":
		healthCheck.HealthCheckType = api.LB_HEALTH_CHECK_UDP
	default:
		healthCheck.HealthCheckType = ""
	}
	healthCheck.HealthCheckURI = pool.healthmonitor.URLPath
	return &healthCheck, nil
}

func (pool *SLoadbalancerPool) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	if len(pool.SessionPersistence.Type) == 0 {
		return nil, nil
	}

	var stickySessionType string
	switch pool.SessionPersistence.Type {
	case "SOURCE_IP":
		stickySessionType = api.LB_STICKY_SESSION_TYPE_INSERT
	case "HTTP_COOKIE":
		stickySessionType = api.LB_STICKY_SESSION_TYPE_INSERT
	case "APP_COOKIE":
		stickySessionType = api.LB_STICKY_SESSION_TYPE_SERVER
	}

	ret := cloudprovider.SLoadbalancerStickySession{
		StickySession:              api.LB_BOOL_ON,
		StickySessionCookie:        pool.SessionPersistence.CookieName,
		StickySessionType:          stickySessionType,
		StickySessionCookieTimeout: 0,
	}

	return &ret, nil
}

func (pool *SLoadbalancerPool) GetName() string {
	return pool.Name
}

func (pool *SLoadbalancerPool) GetId() string {
	return pool.ID
}

func (pool *SLoadbalancerPool) GetGlobalId() string {
	return pool.ID
}

func (pool *SLoadbalancerPool) GetStatus() string {
	switch pool.ProvisioningStatus {
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

func (pool *SLoadbalancerPool) IsDefault() bool {
	return false
}

func (pool *SLoadbalancerPool) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (pool *SLoadbalancerPool) IsEmulated() bool {
	return false
}

func (region *SRegion) GetLoadbalancerPools() ([]SLoadbalancerPool, error) {
	pools := []SLoadbalancerPool{}
	resource := "/v2/lbaas/pools"
	query := url.Values{}
	for {
		resp, err := region.lbList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "lbList")
		}
		part := struct {
			Pools      []SLoadbalancerPool
			PoolsLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		pools = append(pools, part.Pools...)
		marker := part.PoolsLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}

	for i := 0; i < len(pools); i++ {
		pools[i].region = region
		err := pools[i].fetchLoadbalancerHealthmonitor()
		if err != nil {
			return nil, errors.Wrapf(err, "pools[%d].fetchLoadbalancerHealthmonitor()", i)
		}

	}
	return pools, nil
}

func (region *SRegion) GetLoadbalancerPoolById(poolId string) (*SLoadbalancerPool, error) {
	body, err := region.lbGet(fmt.Sprintf("/v2/lbaas/pools/%s", poolId))
	if err != nil {
		return nil, errors.Wrapf(err, "region.lbGet(fmt.Sprintf(/v2/lbaas/pools/%s)", poolId)
	}
	pool := SLoadbalancerPool{}
	err = body.Unmarshal(&pool, "pool")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal")
	}
	pool.region = region
	err = pool.fetchLoadbalancermembers()
	if err != nil {
		return nil, errors.Wrap(err, "pool.fetchLoadbalancermembers()")
	}
	err = pool.fetchLoadbalancerHealthmonitor()
	if err != nil {
		return nil, errors.Wrap(err, "pool.fetchLoadbalancerHealthmonitor()")
	}
	return &pool, nil
}

func (region *SRegion) CreateLoadbalancerPool(group *cloudprovider.SLoadbalancerBackendGroup) (*SLoadbalancerPool, error) {
	type CreateParams struct {
		Pool SLoadbalancerPoolCreateParams `json:"pool"`
	}
	params := CreateParams{}
	params.Pool.AdminStateUp = true
	params.Pool.LbAlgorithm = LB_ALGORITHM_MAP[group.Scheduler]
	params.Pool.Name = group.Name
	params.Pool.LoadbalancerID = group.LoadbalancerID
	// 绑定规则时不能指定listener
	params.Pool.ListenerID = group.ListenerID
	params.Pool.Protocol = LB_PROTOCOL_MAP[group.ListenType]
	params.Pool.SessionPersistence = nil
	if group.StickySession != nil {
		session := SSessionPersistence{}
		session.Type = LB_STICKY_SESSION_MAP[group.StickySession.StickySessionType]
		if session.Type == "APP_COOKIE" {
			session.CookieName = group.StickySession.StickySessionCookie
		}
		params.Pool.SessionPersistence = &session
	}
	body, err := region.lbPost("/v2/lbaas/pools", jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, "region.lbPost(/v2/lbaas/pools)")
	}
	spool := SLoadbalancerPool{}
	spool.region = region
	err = body.Unmarshal(&spool, "pool")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal(&spool, pool)")
	}
	return &spool, nil
}

func (pool *SLoadbalancerPool) Refresh() error {
	newPool, err := pool.region.GetLoadbalancerPoolById(pool.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(pool, newPool)
}

func (pool *SLoadbalancerPool) fetchLoadbalancermembers() error {
	if len(pool.MemberIds) < 1 {
		return nil
	}
	members, err := pool.region.GetLoadbalancerMenbers(pool.ID)
	if err != nil {
		return err
	}
	pool.members = members
	return nil
}

func (pool *SLoadbalancerPool) fetchLoadbalancerHealthmonitor() error {
	if len(pool.HealthmonitorID) < 1 {
		return nil
	}
	healthmonitor, err := pool.region.GetLoadbalancerHealthmonitorById(pool.HealthmonitorID)
	if err != nil {
		return errors.Wrap(err, "pool.region.GetLoadbalancerHealthmonitorById")
	}
	pool.healthmonitor = healthmonitor
	return nil
}

func (pool *SLoadbalancerPool) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	ibackends := []cloudprovider.ICloudLoadbalancerBackend{}
	for i := 0; i < len(pool.members); i++ {
		ibackends = append(ibackends, &pool.members[i])
	}
	return ibackends, nil
}

func (pool *SLoadbalancerPool) GetILoadbalancerBackendById(memberId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
	for i := 0; i < len(pool.members); i++ {
		if pool.members[i].ID == memberId {
			return &pool.members[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetILoadbalancerBackendById(%s)", memberId)
}

func (region *SRegion) UpdateLoadBalancerPool(poolId string, group *cloudprovider.SLoadbalancerBackendGroup) error {
	type UpdateParams struct {
		Pool SLoadbalancerPoolUpdateParams `json:"pool"`
	}
	params := UpdateParams{}
	params.Pool.AdminStateUp = true
	params.Pool.LbAlgorithm = LB_ALGORITHM_MAP[group.Scheduler]
	params.Pool.Name = group.Name
	if group.StickySession != nil {
		session := SSessionPersistence{}
		session.Type = LB_STICKY_SESSION_MAP[group.StickySession.StickySessionType]
		if session.Type == "APP_COOKIE" {
			session.CookieName = group.StickySession.StickySessionCookie
		}
		params.Pool.SessionPersistence = &session
	}
	_, err := region.lbUpdate(fmt.Sprintf("/v2/lbaas/pools/%s", poolId), jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrapf(err, `region.lbUpdate("/v2/lbaas/pools/%s", jsonutils.Marshal(params))`, poolId)
	}
	return nil
}

func (pool *SLoadbalancerPool) Sync(ctx context.Context, group *cloudprovider.SLoadbalancerBackendGroup) error {
	lb, err := pool.region.GetLoadbalancerbyId(pool.GetLoadbalancerId())
	if err != nil {
		return errors.Wrap(err, "pool.region.GetLoadbalancerbyId(pool.GetLoadbalancerId())")
	}
	// ensure loadbalancer status
	err = waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(lb, 10*time.Second, 8*time.Minute)`)
	}
	// ensure pool status
	err = waitLbResStatus(pool, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(pool, 10*time.Second, 8*time.Minute)`)
	}
	// sync healthmonitor
	healthmonitor := SLoadbalancerHealthmonitor{}
	if len(pool.HealthmonitorID) > 0 {
		oldhealthmonitor, err := pool.region.GetLoadbalancerHealthmonitorById(pool.HealthmonitorID)
		if err != nil {
			return errors.Wrap(err, "pool.region.GetLoadbalancerHealthmonitorById(pool.HealthmonitorID)")
		}
		// 不能更新健康检查类型，需要删除重建
		var sHealthCheckType string
		switch oldhealthmonitor.Type {
		case "HTTP":
			sHealthCheckType = api.LB_HEALTH_CHECK_HTTP
		case "HTTPS":
			sHealthCheckType = api.LB_HEALTH_CHECK_HTTPS
		case "TCP":
			sHealthCheckType = api.LB_HEALTH_CHECK_TCP
		case "UDP-CONNECT":
			sHealthCheckType = api.LB_HEALTH_CHECK_UDP
		default:
			sHealthCheckType = ""
		}

		if sHealthCheckType != group.HealthCheck.HealthCheckType {
			err := pool.region.DeleteLoadbalancerHealthmonitor(pool.HealthmonitorID)
			if err != nil {
				return errors.Wrapf(err, "pool.region.DeleteLoadbalancerHealthmonitor(%s)", pool.HealthmonitorID)
			}
			// 等待删除结束
			err = waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
			if err != nil {
				return errors.Wrap(err, `waitLbResStatus(lb, 10*time.Second, 8*time.Minute)`)
			}

			newhealthmonitor, err := pool.region.CreateLoadbalancerHealthmonitor(pool.ID, group.HealthCheck)
			if err != nil {
				return errors.Wrapf(err, "pool.region.CreateLoadbalancerHealthmonitor(%s,group.HealthCheck)", pool.ID)
			}
			healthmonitor = *newhealthmonitor
		} else {
			// ensure healthmonitor status
			err = waitLbResStatus(oldhealthmonitor, 10*time.Second, 8*time.Minute)
			if err != nil {
				return errors.Wrap(err, `waitLbResStatus(oldhealthmonitor, 10*time.Second, 8*time.Minute)`)
			}
			oldhealthmonitor, err = pool.region.UpdateLoadbalancerHealthmonitor(pool.HealthmonitorID, group.HealthCheck)
			if err != nil {
				return errors.Wrapf(err, `pool.region.UpdateLoadbalancerHealthmonitor(%s, group.HealthCheck)`, pool.HealthmonitorID)
			}
			healthmonitor = *oldhealthmonitor
		}
	} else {
		newhealthmonitor, err := pool.region.CreateLoadbalancerHealthmonitor(pool.ID, group.HealthCheck)
		if err != nil {
			return errors.Wrapf(err, "pool.region.CreateLoadbalancerHealthmonitor(%s, group.HealthCheck)", pool.ID)
		}
		healthmonitor = *newhealthmonitor
	}

	// ensure pool status
	err = waitLbResStatus(pool, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(pool, 10*time.Second, 8*time.Minute)`)
	}
	// sync pool
	err = pool.region.UpdateLoadBalancerPool(pool.ID, group)
	if err != nil {
		return errors.Wrapf(err, `pool.region.UpdateLoadBalancerPool(%s, group)`, pool.ID)
	}

	// wait healthmonitor status
	err = waitLbResStatus(&healthmonitor, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(&healthmonitor, 10*time.Second, 8*time.Minute)`)
	}
	// wait pool status
	err = waitLbResStatus(pool, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(pool,  10*time.Second, 8*time.Minute)`)
	}
	return nil
}

func (region *SRegion) DeleteLoadBalancerPool(poolId string) error {
	_, err := region.lbDelete(fmt.Sprintf("/v2/lbaas/pools/%s", poolId))
	if err != nil {
		return errors.Wrapf(err, "lbDelete(/v2/lbaas/pools/%s)", poolId)
	}
	return nil
}

func (pool *SLoadbalancerPool) Delete(ctx context.Context) error {
	lb, err := pool.region.GetLoadbalancerbyId(pool.GetLoadbalancerId())
	if err != nil {
		return errors.Wrap(err, "pool.region.GetLoadbalancerbyId(pool.GetLoadbalancerId())")
	}
	err = waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(lb, 10*time.Second, 8*time.Minute)`)
	}
	return pool.region.DeleteLoadBalancerPool(pool.ID)
}

func (pool *SLoadbalancerPool) AddBackendServer(serverId string, weight, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	// ensure lb status
	lb, err := pool.region.GetLoadbalancerbyId(pool.GetLoadbalancerId())
	if err != nil {
		return nil, errors.Wrap(err, "pool.region.GetLoadbalancerbyId(pool.GetLoadbalancerId())")
	}
	err = waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, `waitLbResStatus(lb, 10*time.Second, 8*time.Minute)`)
	}
	smemeber, err := pool.region.CreateLoadbalancerMember(pool.ID, serverId, weight, port)
	if err != nil {
		return nil, errors.Wrapf(err, `CreateLoadbalancerMember(%s,%s,%d,%d)`, pool.ID, serverId, weight, port)
	}
	err = waitLbResStatus(smemeber, 10*time.Second, 8*time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, `waitLbResStatus(smemeber,  10*time.Second, 8*time.Minute)`)
	}
	smemeber.region = pool.region
	smemeber.poolID = pool.ID
	pool.members = append(pool.members, *smemeber)
	return smemeber, nil
}

// 不是serverId，是memberId
func (pool *SLoadbalancerPool) RemoveBackendServer(id string, weight, port int) error {
	return pool.region.DeleteLoadbalancerMember(pool.ID, id)
}

func (pool *SLoadbalancerPool) GetProjectId() string {
	return pool.ProjectID
}
