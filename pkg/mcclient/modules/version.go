package modules

import (
	"io/ioutil"

	"github.com/yunionio/onecloud/pkg/mcclient"
)

func GetVersion(s *mcclient.ClientSession, serviceType string) (string, error) {
	man := &BaseManager{serviceType: serviceType}
	resp, err := man.rawRequest(s, "GET", "/version", nil, nil)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
