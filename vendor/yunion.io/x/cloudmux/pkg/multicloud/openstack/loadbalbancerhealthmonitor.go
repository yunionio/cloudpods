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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancerHealthmonitorCreateParams struct {
	Name           string   `json:"name,omitempty"`
	AdminStateUp   bool     `json:"admin_state_up"`
	PoolID         string   `json:"pool_id,omitempty"`
	Delay          *int     `json:"delay,omitempty"`
	ExpectedCodes  string   `json:"expected_codes,omitempty"`
	MaxRetries     *int     `json:"max_retries,omitempty"`
	HTTPMethod     string   `json:"http_method,omitempty"`
	Timeout        *int     `json:"timeout,omitempty"`
	URLPath        string   `json:"url_path,omitempty"`
	Type           string   `json:"type,omitempty"`
	MaxRetriesDown *int     `json:"max_retries_down,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	HTTPVersion    *float64 `json:"http_version,omitempty"`
	DomainName     string   `json:"domain_name,omitempty"`
}

type SLoadbalancerHealthmonitorUpdateParams struct {
	Name           string   `json:"name,omitempty"`
	AdminStateUp   bool     `json:"admin_state_up"`
	Delay          *int     `json:"delay,omitempty"`
	ExpectedCodes  string   `json:"expected_codes,omitempty"`
	MaxRetries     *int     `json:"max_retries,omitempty"`
	HTTPMethod     string   `json:"http_method,omitempty"`
	Timeout        *int     `json:"timeout,omitempty"`
	URLPath        string   `json:"url_path,omitempty"`
	MaxRetriesDown *int     `json:"max_retries_down,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	HTTPVersion    *float64 `json:"http_version,omitempty"`
	DomainName     string   `json:"domain_name,omitempty"`
}

type SLoadbalancerHealthmonitor struct {
	multicloud.SResourceBase
	OpenStackTags
	region             *SRegion
	ProjectID          string    `json:"project_id"`
	Name               string    `json:"name"`
	AdminStateUp       bool      `json:"admin_state_up"`
	PoolIds            []SPoolID `json:"pools"`
	CreatedAt          string    `json:"created_at"`
	ProvisioningStatus string    `json:"provisioning_status"`
	UpdatedAt          string    `json:"updated_at"`
	Delay              int       `json:"delay"`
	ExpectedCodes      string    `json:"expected_codes"`
	MaxRetries         int       `json:"max_retries"`
	HTTPMethod         string    `json:"http_method"`
	Timeout            int       `json:"timeout"`
	MaxRetriesDown     int       `json:"max_retries_down"`
	URLPath            string    `json:"url_path"`
	Type               string    `json:"type"`
	ID                 string    `json:"id"`
	OperatingStatus    string    `json:"operating_status"`
	Tags               []string  `json:"tags"`
	HTTPVersion        float64   `json:"http_version"`
	DomainName         string    `json:"domain_name"`
}

func (region *SRegion) GetLoadbalancerHealthmonitorById(healthmonitorId string) (*SLoadbalancerHealthmonitor, error) {
	body, err := region.lbGet(fmt.Sprintf("/v2/lbaas/healthmonitors/%s", healthmonitorId))
	if err != nil {
		return nil, errors.Wrapf(err, `region.lbGet(/v2/lbaas/healthmonitors/%s)`, healthmonitorId)
	}
	healthmonitor := SLoadbalancerHealthmonitor{}
	healthmonitor.region = region
	return &healthmonitor, body.Unmarshal(&healthmonitor, "healthmonitor")
}
func (region *SRegion) CreateLoadbalancerHealthmonitor(poolId string, healthcheck *cloudprovider.SLoadbalancerHealthCheck) (*SLoadbalancerHealthmonitor, error) {
	type CreateParams struct {
		Healthmonitor SLoadbalancerHealthmonitorCreateParams `json:"healthmonitor"`
	}
	params := CreateParams{}
	params.Healthmonitor.AdminStateUp = true
	params.Healthmonitor.Delay = &healthcheck.HealthCheckInterval
	params.Healthmonitor.Timeout = &healthcheck.HealthCheckTimeout

	params.Healthmonitor.MaxRetries = &healthcheck.HealthCheckRise
	params.Healthmonitor.MaxRetriesDown = &healthcheck.HealthCheckFail
	params.Healthmonitor.PoolID = poolId

	switch healthcheck.HealthCheckType {
	case api.LB_HEALTH_CHECK_TCP:
		params.Healthmonitor.Type = "TCP"
	case api.LB_HEALTH_CHECK_UDP:
		params.Healthmonitor.Type = "UDP-CONNECT"
	case api.LB_HEALTH_CHECK_HTTP:
		params.Healthmonitor.Type = "HTTP"
	case api.LB_HEALTH_CHECK_HTTPS:
		params.Healthmonitor.Type = "HTTPS"
	case api.LB_HEALTH_CHECK_PING:
		params.Healthmonitor.Type = "PING"
	default:
		params.Healthmonitor.Type = "PING"
	}

	if params.Healthmonitor.Type == "HTTP" || params.Healthmonitor.Type == "HTTPS" {
		params.Healthmonitor.HTTPMethod = "GET"
		httpVersion := 1.1
		params.Healthmonitor.HTTPVersion = &httpVersion
		params.Healthmonitor.DomainName = healthcheck.HealthCheckDomain
		params.Healthmonitor.URLPath = healthcheck.HealthCheckURI
		params.Healthmonitor.ExpectedCodes = ToOpenstackHealthCheckHttpCode(healthcheck.HealthCheckHttpCode)
	}

	body, err := region.lbPost("/v2/lbaas/healthmonitors", jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, `region.lbPost("/v2/lbaas/healthmonitors", jsonutils.Marshal(params))`)
	}
	shealthmonitor := SLoadbalancerHealthmonitor{}
	shealthmonitor.region = region
	err = body.Unmarshal(&shealthmonitor, "healthmonitor")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal(&shealthmonitor, healthmonitor)")
	}
	return &shealthmonitor, nil
}

func (region *SRegion) UpdateLoadbalancerHealthmonitor(healthmonitorId string, healthcheck *cloudprovider.SLoadbalancerHealthCheck) (*SLoadbalancerHealthmonitor, error) {
	type UpdateParams struct {
		Healthmonitor SLoadbalancerHealthmonitorUpdateParams `json:"healthmonitor"`
	}
	params := UpdateParams{}
	params.Healthmonitor.AdminStateUp = true
	params.Healthmonitor.Delay = &healthcheck.HealthCheckInterval
	params.Healthmonitor.Timeout = &healthcheck.HealthCheckTimeout

	params.Healthmonitor.MaxRetries = &healthcheck.HealthCheckRise
	params.Healthmonitor.MaxRetriesDown = &healthcheck.HealthCheckFail

	if healthcheck.HealthCheckType == api.LB_HEALTH_CHECK_HTTP || healthcheck.HealthCheckType == api.LB_HEALTH_CHECK_HTTPS {
		params.Healthmonitor.HTTPMethod = "GET"
		httpVersion := 1.1
		params.Healthmonitor.HTTPVersion = &httpVersion
		params.Healthmonitor.DomainName = healthcheck.HealthCheckDomain
		params.Healthmonitor.URLPath = healthcheck.HealthCheckURI
		params.Healthmonitor.ExpectedCodes = ToOpenstackHealthCheckHttpCode(healthcheck.HealthCheckHttpCode)
	}

	body, err := region.lbUpdate(fmt.Sprintf("/v2/lbaas/healthmonitors/%s", healthmonitorId), jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrapf(err, `region.lbUpdate(/v2/lbaas/healthmonitors/%s), jsonutils.Marshal(params))`, healthmonitorId)
	}
	shealthmonitor := SLoadbalancerHealthmonitor{}
	err = body.Unmarshal(&shealthmonitor, "healthmonitor")
	shealthmonitor.region = region
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal(&shealthmonitor, healthmonitor)")
	}
	return &shealthmonitor, nil
}

func (region *SRegion) DeleteLoadbalancerHealthmonitor(healthmonitorId string) error {
	_, err := region.lbDelete(fmt.Sprintf("/v2/lbaas/healthmonitors/%s", healthmonitorId))
	if err != nil {
		return errors.Wrapf(err, `region.lbDelete(/v2/lbaas/healthmonitors/%s )`, healthmonitorId)
	}
	return nil
}

func (healthmonitor *SLoadbalancerHealthmonitor) GetName() string {
	return healthmonitor.Name
}

func (healthmonitor *SLoadbalancerHealthmonitor) GetId() string {
	return healthmonitor.ID
}

func (healthmonitor *SLoadbalancerHealthmonitor) GetGlobalId() string {
	return healthmonitor.ID
}

func (healthmonitor *SLoadbalancerHealthmonitor) GetStatus() string {
	switch healthmonitor.ProvisioningStatus {
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

func (healthmonitor *SLoadbalancerHealthmonitor) Refresh() error {
	newhealthmonitor, err := healthmonitor.region.GetLoadbalancerHealthmonitorById(healthmonitor.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(healthmonitor, newhealthmonitor)
}

func (healthmonitor *SLoadbalancerHealthmonitor) IsEmulated() bool {
	return false
}
