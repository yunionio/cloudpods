package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SPolicyManager struct {
	ResourceManager
}

var Policies SPolicyManager

func translateS2C(s jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ss := s.(*jsonutils.JSONDict)
	ret := ss.CopyIncludes("id", "type")
	blobStr, err := ss.GetString("blob")
	if err != nil {
		return nil, err
	}
	blobJson, err := jsonutils.ParseString(blobStr)
	if err != nil {
		return nil, err
	}
	ret.Add(jsonutils.NewString(blobJson.YAMLString()), "policy")
	return ret, nil
}

func translateC2S(s jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	if s.Contains("policy") {
		blobYaml, err := s.GetString("policy")
		if err != nil {
			return nil, err
		}
		blobJson, err := jsonutils.ParseYAML(blobYaml)
		if err != nil {
			return nil, err
		}
		ret.Add(jsonutils.NewString(blobJson.String()), "blob")
	}
	if s.Contains("type") {
		typeStr, err := s.GetString("type")
		if err != nil {
			return nil, err
		}
		ret.Add(jsonutils.NewString(typeStr), "type")
	}
	return ret, nil
}

func (pm *SPolicyManager) Get(s *mcclient.ClientSession, id string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	result, err := pm.GetById(s, id, nil)
	if err == nil {
		return result, nil
	}
	jsonErr := err.(*httputils.JSONClientError)
	if jsonErr.Code != 404 {
		return nil, err
	}
	return pm.GetByName(s, id, query)
}

func (pm *SPolicyManager) GetId(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (string, error) {
	obj, err := pm.Get(s, id, params)
	if err != nil {
		return "", err
	}
	return obj.GetString("id")
}

func (pm *SPolicyManager) GetById(s *mcclient.ClientSession, id string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	result, err := pm.ResourceManager.GetById(s, id, query)
	if err != nil {
		return nil, err
	}
	return translateS2C(result)
}

func (pm *SPolicyManager) GetByName(s *mcclient.ClientSession, name string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(name), "type")
	listResult, err := Policies.List(s, params)
	if err != nil {
		return nil, err
	}
	if listResult.Total == 1 {
		return listResult.Data[0], nil
	} else if listResult.Total == 0 {
		return nil, &httputils.JSONClientError{Code: 404, Class: "NotFound",
			Details: fmt.Sprintf("%d not found", name)}
	} else {
		return nil, &httputils.JSONClientError{Code: 409, Class: "Conflict",
			Details: fmt.Sprintf("multiple %d found", name)}
	}
}

func (pm *SPolicyManager) Create(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, err := translateC2S(params)
	if err != nil {
		return nil, err
	}
	result, err := pm.ResourceManager.Create(s, data)
	if err != nil {
		return nil, err
	}
	return translateS2C(result)
}

func (pm *SPolicyManager) Patch(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, err := translateC2S(params)
	if err != nil {
		return nil, err
	}
	result, err := pm.ResourceManager.Patch(s, id, data)
	if err != nil {
		return nil, err
	}
	return translateS2C(result)
}

func (pm *SPolicyManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*ListResult, error) {
	results, err := pm.ResourceManager.List(s, params)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(results.Data); i += 1 {
		val, _ := translateS2C(results.Data[i])
		results.Data[i] = val
	}
	return results, nil
}

func init() {
	Policies = SPolicyManager{NewIdentityV3Manager("policy", "policies",
		[]string{"id", "type", "policy"},
		[]string{})}

	register(&Policies)
}
