package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCapabilityManager struct {
	ResourceManager
}

func (this *SCapabilityManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*ListResult, error) {
	body, err := this._get(s, "/capabilities", "")
	if err != nil {
		return nil, err
	}
	result := ListResult{Data: []jsonutils.JSONObject{body}}
	return &result, nil
}

var (
	Capabilities SCapabilityManager
)

func init() {
	Capabilities = SCapabilityManager{
		ResourceManager: NewComputeManager("capability", "capabilities", []string{}, []string{}),
	}
	registerCompute(&Capabilities)
}