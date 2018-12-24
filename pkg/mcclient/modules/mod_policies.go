package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SPolicyManager struct {
	ResourceManager
}

var Policies SPolicyManager

func policyReadFilter(session *mcclient.ClientSession, s jsonutils.JSONObject, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ss := s.(*jsonutils.JSONDict)
	ret := ss.CopyIncludes("id", "type")
	blobStr, _ := ss.GetString("blob")
	if len(blobStr) > 0 {
		blobJson, _ := jsonutils.ParseString(blobStr)
		var format string
		if query != nil {
			format, _ = query.GetString("format")
		}
		if format == "yaml" {
			var policy string
			if blobJson != nil {
				policy = blobJson.YAMLString()
			}
			ret.Add(jsonutils.NewString(policy), "policy")
		} else {
			ret.Add(blobJson, "policy")
		}
	}
	return ret, nil
}

func policyWriteFilter(session *mcclient.ClientSession, s jsonutils.JSONObject, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	if s.Contains("policy") {
		blobJson, err := s.Get("policy")
		if err != nil {
			return nil, err
		}
		switch blob := blobJson.(type) {
		case *jsonutils.JSONString:
			blobStr, _ := blob.GetString()
			blobJson, err = jsonutils.ParseYAML(blobStr)
			if err != nil {
				return nil, err
			}
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
