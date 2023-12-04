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

package huawei

import (
	"fmt"
	"net/url"
)

type GaussDBNoSQL struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Port      string `json:"port"`
	Mode      string `json:"mode"`
	Region    string `json:"region"`
	Datastore struct {
		Type           string `json:"type"`
		Version        string `json:"version"`
		PatchAvailable bool   `json:"patch_available"`
		WholeVersion   string `json:"whole_version"`
	} `json:"datastore"`
	Engine          string `json:"engine"`
	Created         string `json:"created"`
	Updated         string `json:"updated"`
	DbUserName      string `json:"db_user_name"`
	VpcId           string `json:"vpc_id"`
	SubnetId        string `json:"subnet_id"`
	SecurityGroupId string `json:"security_group_id"`
	BackupStrategy  struct {
		StartTime string `json:"start_time"`
		KeepDays  int    `json:"keep_days"`
	} `json:"backup_strategy"`
	PayMode           string `json:"pay_mode"`
	MaintenanceWindow string `json:"maintenance_window"`
	BackupSpaceUsage  struct {
		BackupUsage int `json:"backup_usage"`
	} `json:"backup_space_usage"`
	Groups []struct {
		Id     string `json:"id"`
		Status string `json:"status"`
		Volume struct {
			Size     int    `json:"size"`
			Used     string `json:"used"`
			GiftSize string `json:"gift_size"`
		} `json:"volume"`
		Nodes []struct {
			Id               string `json:"id"`
			Name             string `json:"name"`
			Status           string `json:"status"`
			SubnetId         string `json:"subnet_id"`
			PrivateIP        string `json:"private_ip"`
			SpecCode         string `json:"spec_code"`
			AvailabilityZone string `json:"availability_zone"`
			SupportReduce    bool   `json:"support_reduce"`
		} `json:"nodes"`
	} `json:"groups"`
	EnterpriseProjectId string   `json:"enterprise_project_id"`
	TimeZone            string   `json:"time_zone"`
	Actions             []string `json:"actions"`
	LbIPAddress         string   `json:"lb_ip_address"`
	LbPort              string   `json:"lb_port"`
}

func (self *SRegion) ListGaussNoSQLInstances() ([]GaussDBNoSQL, error) {
	query := url.Values{}
	ret := []GaussDBNoSQL{}
	for {
		resp, err := self.list(SERVICE_GAUSSDB_NOSQL, "instances", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Instances  []GaussDBNoSQL
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Instances...)
		if len(ret) >= part.TotalCount || len(part.Instances) == 0 {
			break
		}
		query.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}
