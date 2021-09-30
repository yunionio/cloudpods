package ctyun

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SCrmUser struct {
	LoginName   string `json:"loginName"`
	LoginEmail  string `json:"loginEmail"`
	RootUserid  string `json:"rootUserid"`
	CreateDate  int64  `json:"createDate"`
	AccountType int64  `json:"accountType"`
	Status      int64  `json:"status"`
	Province    string `json:"province"`
	City        string `json:"city"`
	Mobilephone string `json:"mobilephone"`
	Postpaid    int64  `json:"postpaid"`
	Channel     int64  `json:"channel"`
	AuditStatus string `json:"auditStatus"`
	AuditMsg    string `json:"auditMsg"`
}

func (self *SRegion) getCustiomInfo(t string, crmBizId string, accountId string) jsonutils.JSONObject {
	if len(t) == 0 {
		return nil
	}

	customeInfo := jsonutils.NewDict()
	//customeInfo.Set("name", jsonutils.NewString(""))
	//customeInfo.Set("email", jsonutils.NewString(""))
	//customeInfo.Set("phone", jsonutils.NewString(""))
	indentity := jsonutils.NewDict()
	if len(crmBizId) > 0 {
		indentity.Set("crmBizId", jsonutils.NewString(crmBizId))
	}

	if len(accountId) > 0 {
		indentity.Set("accountId", jsonutils.NewString(accountId))
	}

	if len(t) > 0 {
		customeInfo.Set("type", jsonutils.NewString(t))
		customeInfo.Set("identity", indentity)
	}

	return customeInfo
}

func (self *SRegion) GetCrmUser(t string, crmBizId string, accountId string) (*SCrmUser, error) {
	params := map[string]string{
		"regionId": self.GetId(),
	}

	customInfo := self.getCustiomInfo(t, crmBizId, accountId)
	if customInfo != nil {
		params["customInfo"] = customInfo.String()
	}

	resp, err := self.client.DoGet("/apiproxy/v3/queryCRM", params)
	if err != nil {
		return nil, errors.Wrap(err, "Region.queryCRM.DoGet")
	}

	user := &SCrmUser{}
	err = resp.Unmarshal(user, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "Region.queryCRM.Unmarshal")
	}

	return user, nil
}
