package godingtalk

type Callback struct {
	OAPIResponse
	Token     string
	AES_KEY   string `json:"aes_key"`
	URL       string
	Callbacks []string `json:"call_back_tag"`
}

//RegisterCallback is 注册事件回调接口
func (c *DingTalkClient) RegisterCallback(callbacks []string, token string, aes_key string, callbackURL string) error {
	var data OAPIResponse
	request := map[string]interface{}{
		"call_back_tag": callbacks,
		"token":         token,
		"aes_key":       aes_key,
		"url":           callbackURL,
	}
	err := c.httpRPC("call_back/register_call_back", nil, request, &data)
	return err
}

//UpdateCallback is 更新事件回调接口
func (c *DingTalkClient) UpdateCallback(callbacks []string, token string, aes_key string, callbackURL string) error {
	var data OAPIResponse
	request := map[string]interface{}{
		"call_back_tag": callbacks,
		"token":         token,
		"aes_key":       aes_key,
		"url":           callbackURL,
	}
	err := c.httpRPC("call_back/update_call_back", nil, request, &data)
	return err
}

//DeleteCallback is 删除事件回调接口
func (c *DingTalkClient) DeleteCallback() error {
	var data OAPIResponse
	err := c.httpRPC("call_back/delete_call_back", nil, nil, &data)
	return err
}

//ListCallback is 查询事件回调接口
func (c *DingTalkClient) ListCallback() (Callback, error) {
	var data Callback
	err := c.httpRPC("call_back/get_call_back", nil, nil, &data)
	return data, err
}
