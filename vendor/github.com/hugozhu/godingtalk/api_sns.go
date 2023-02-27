//普通钉钉用户账号开放相关接口
package godingtalk

import(
	"net/url"
)

//获取钉钉开放应用ACCESS_TOKEN
//TODO:
//    根据和赤司(钉钉开发者)的沟通，ACCESS_TOKEN只有两个小时的有效期
//    但是目前接口貌似没有返回过期时间相关的信息，因此所有相关的调用都需要强制刷新
func (c *DingTalkClient) RefreshSnsAccessToken() error {
	var data AccessTokenResponse

	params := url.Values{}
	params.Add("appid", c.SnsAppID)
	params.Add("appsecret", c.SnsAppSecret)

	err := c.httpRPC("sns/gettoken", params, nil, &data)
	if err==nil {
		c.SnsAccessToken = data.AccessToken
	}
	return err
}

//获取用户授权的持久授权码返回信息
type SnsPersistentCodeResponse struct {
	OAPIResponse
	UnionID        string `json:"unionid"`
	OpenID         string `json:"openid"`
	PersistentCode string `json:"persistent_code"`
}

//获取用户授权的持久授权码
func (c *DingTalkClient) GetSnsPersistentCode(tmpAuthCode string) (string, string, string, error) {
	c.RefreshSnsAccessToken()

	params := url.Values{}
	params.Add("access_token", c.SnsAccessToken)

	request := map[string]interface{}{
		"tmp_auth_code": tmpAuthCode,
	}

	var data SnsPersistentCodeResponse
	err := c.httpRequest("sns/get_persistent_code", params, request, &data)
	if err!=nil {
		return "","","",err
	}
	return data.UnionID, data.OpenID, data.PersistentCode, nil
}


type SnsTokenResponse struct {
	OAPIResponse
	Expires  int    `json:"expires_in"`
	SnsToken string `json:"sns_token"`
}

//获取用户授权的SNS_TOKEN
func (c *DingTalkClient) GetSnsToken(openid, persistentCode string) (string, error) {
	c.RefreshSnsAccessToken()

	params := url.Values{}
	params.Add("access_token", c.SnsAccessToken)

	request := map[string]interface{}{
		"openid": openid,
		"persistent_code": persistentCode,
	}

	var data SnsTokenResponse
	err := c.httpRequest("sns/get_sns_token", params, request, &data)
	if err!=nil {
		return "", err
	}
	return data.SnsToken, err
}

type SnsUserInfoResponse struct {
	OAPIResponse

	CorpInfo []struct{
		CorpName string `json:"corp_name"`
		IsAuth bool `json:"is_auth"`
		IsManager bool `json:"is_manager"`
		RightsLevel int `json:"rights_level"`
	} `json:"corp_info"`

	UserInfo struct {
		MaskedMobile string `json:"marskedMobile"`
		Nick string `json:"nick"`
		OpenID string `json:"openid"`
		UnionID string `json:"unionid"`
		DingID string `json:"dingId"`
	} `json:"user_info"`
}

//获取用户授权的个人信息
func (c *DingTalkClient) GetSnsUserInfo(snsToken string) (SnsUserInfoResponse, error) {
	c.RefreshSnsAccessToken()

	params := url.Values{}
	params.Add("sns_token", snsToken)

	var data SnsUserInfoResponse
	err := c.httpRequest("sns/getuserinfo", params, nil, &data)
	return data, err
}
