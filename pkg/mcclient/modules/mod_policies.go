package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SPolicyManager struct {
	ResourceManager
}

var Policies SPolicyManager

func policyReadFilter(session *mcclient.ClientSession, s jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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

func policyWriteFilter(session *mcclient.ClientSession, s jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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

func init() {
	Policies = SPolicyManager{NewIdentityV3Manager("policy", "policies",
		[]string{"id", "type", "policy"},
		[]string{})}

	Policies.SetReadFilter(policyReadFilter).SetWriteFilter(policyWriteFilter).SetNameField("type")

	register(&Policies)
}
