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
	ret := struct {
		Items []SXpnHost
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret.Items, nil
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
	ret := struct {
		Resources []SXpnResource
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret.Resources, nil
}
