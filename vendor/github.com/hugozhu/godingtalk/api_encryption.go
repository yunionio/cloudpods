package godingtalk

//DataMessage 服务端加密、解密消息
type DataMessage struct {
	OAPIResponse
	Data string
}


//Encrypt is 服务端加密 
func (c *DingTalkClient) Encrypt(str string) (string, error) {
	var data DataMessage
	request := map[string]interface{}{
		"data":  str,
	}
	err := c.httpRPC("encryption/encrypt", nil, request, &data)
    if err!=nil {
        return "", err
    }
	return data.Data, nil
}

//Decrypt is 服务端解密
func (c *DingTalkClient) Decrypt(str string) (string, error) {
	var data DataMessage
	request := map[string]interface{}{
		"data":  str,
	}
	err := c.httpRPC("encryption/decrypt", nil, request, &data)
    if err!=nil {
        return "", err
    }
	return data.Data, nil    
}