package options

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/compute/consts"
	"yunion.io/x/pkg/utils"
)

type LoadbalancerBackendGroupCreateOptions struct {
	NAME         string
	LOADBALANCER string
	Type         string   `choices:"default|normal|master_slave"`
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
			if utils.IsInStringArray(value[1], []string{consts.LB_BACKEND_GUEST, consts.LB_BACKEND_HOST}) {
				return nil, fmt.Errorf("invalid backend type %s only support %s %s", value[1], consts.LB_BACKEND_GUEST, consts.LB_BACKEND_HOST)
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
	params, err := optionsStructToParams(opts)
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
	ID string
}

type LoadbalancerBackendGroupUpdateOptions struct {
	ID   string
	Name string
}

type LoadbalancerBackendGroupDeleteOptions struct {
	ID string
}

type LoadbalancerBackendGroupListOptions struct {
	BaseListOptions
	Loadbalancer string
}
