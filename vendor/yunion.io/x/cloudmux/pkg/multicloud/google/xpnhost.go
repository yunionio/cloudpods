package google

import "fmt"

type SXpnHost struct {
	Kind string
	Name string
}

func (cli *SGoogleClient) GetXpnHosts() ([]SXpnHost, error) {
	res := fmt.Sprintf("projects/%s", cli.projectId)
	resp, err := cli.ecsPost(res, "listXpnHosts", nil, nil)
	if err != nil {
		return nil, err
	}
	ret := []SXpnHost{}
	err = resp.Unmarshal(&ret, "items")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type SXpnResource struct {
	Type string
	Id   string
}

func (cli *SGoogleClient) GetXpnResources(projectId string) ([]SXpnResource, error) {
	res := fmt.Sprintf("projects/%s/getXpnResources", projectId)
	resp, err := cli.ecsList(res, nil)
	if err != nil {
		return nil, err
	}
	ret := []SXpnResource{}
	err = resp.Unmarshal(&ret, "resources")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
