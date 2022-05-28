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

package compute

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LoadbalancerBackendGroupCreateOptions struct {
	NAME         string
	Loadbalancer string
	Type         string   `choices:"default|normal|master_slave"`
	ProtocolType string   `help:"Huawei backendgroup protocol type" choices:"tcp|udp|http"`
	Scheduler    string   `help:"Huawei backendgroup scheduler algorithm" choices:"rr|sch|wlc"`
	Backend      []string `help:"backends with separated by ',' e.g. weight:80,port:443,id:01e9d393-d2b8-4d2e-85fb-023b83889070,backend_type:guest" json:"-"`
}

type Backends []*SBackend

type SBackend struct {
	Index       int
	Weight      int
	Port        int
	ID          string
	BackendType string
}

func NewBackend(s string, index int) (*SBackend, error) {
	backend := &SBackend{Index: index}
	for _, part := range strings.Split(s, ",") {
		value := strings.Split(part, ":")
		if len(value) != 2 {
			return nil, fmt.Errorf("invalid input params %s eg: weight:80,port:443,id:01e9d393-d2b8-4d2e-85fb-023b83889070,backend_type:guest", part)
		}
		switch value[0] {
		case "weight":
			weight, err := strconv.Atoi(value[1])
			if err != nil {
				return nil, fmt.Errorf("invalid weight %s error: %v", value[1], err)
			}
			if weight < 0 || weight > 256 {
				return nil, fmt.Errorf("invalid weight range, only support 0 ~ 256")
			}
			backend.Weight = weight
		case "port":
			port, err := strconv.Atoi(value[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port %s error: %v", value[1], err)
			}
			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("invalid port range, only support 1 ~ 65535")
			}
			backend.Port = port
		case "backend_type":
			if utils.IsInStringArray(value[1], []string{api.LB_BACKEND_GUEST, api.LB_BACKEND_HOST}) {
				return nil, fmt.Errorf("invalid backend type %s only support %s %s", value[1], api.LB_BACKEND_GUEST, api.LB_BACKEND_HOST)
			}
			backend.BackendType = value[1]
		case "id":
			backend.ID = value[1]
		default:
			return nil, fmt.Errorf("invalid input type %s", value[0])
		}
	}
	return backend, nil
}

func NewBackends(ss []string) (Backends, error) {
	backends := Backends{}
	for index, s := range ss {
		backend, err := NewBackend(s, index)
		if err != nil {
			return nil, err
		}
		backends = append(backends, backend)
	}
	return backends, nil
}

func (opts *LoadbalancerBackendGroupCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	backends, err := NewBackends(opts.Backend)
	if err != nil {
		return nil, err
	}
	backendJSON := jsonutils.Marshal(backends)
	params.Set("backends", backendJSON)
	return params, nil
}

type LoadbalancerBackendGroupGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerBackendGroupUpdateOptions struct {
	ID   string `json:"-"`
	Name string
}

func (opts *LoadbalancerBackendGroupUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type LoadbalancerBackendGroupDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerBackendGroupIDOptions struct {
	ID string `json:"-"`
}

type LoadbalancerBackendGroupListOptions struct {
	options.BaseListOptions
	Loadbalancer string
	Cloudregion  string
	NoRef        bool
}

func (opts *LoadbalancerBackendGroupListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}
