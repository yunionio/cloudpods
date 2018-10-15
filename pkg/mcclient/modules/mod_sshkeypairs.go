package modules

import (
	"fmt"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSshkeypairManager struct {
	ResourceManager
}

func (this *SSshkeypairManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*ListResult, error) {
	url := "/sshkeypairs"
	queryStr := params.QueryString()
	if len(queryStr) > 0 {
		url = fmt.Sprintf("%s?%s", url, queryStr)
	}
	body, err := this._get(s, url, "sshkeypair")
	if err != nil {
		return nil, err
	}
	result := ListResult{Data: []jsonutils.JSONObject{body}}
	return &result, nil
}

var (
	Sshkeypairs SSshkeypairManager
)

func init() {
	Sshkeypairs = SSshkeypairManager{NewComputeManager("sshkeypair", "sshkeypairs",
		[]string{},
		[]string{})}

	registerComputeV2(&Sshkeypairs)
}
