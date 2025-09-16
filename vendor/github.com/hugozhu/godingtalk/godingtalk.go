package godingtalk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	//VERSION is SDK version
	VERSION = "0.3"
)

//DingTalkClient is the Client to access DingTalk Open API
type DingTalkClient struct {
	CorpID     string
	CorpSecret string

	AgentID     string
	PartnerID   string
	AccessToken string
	HTTPClient  *http.Client
	Cache       Cache

	//社交相关的属性
	SnsAppID       string
	SnsAppSecret   string
	SnsAccessToken string
}

//Unmarshallable is
type Unmarshallable interface {
	checkError() error
	getWriter() io.Writer
}

//OAPIResponse is
type OAPIResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (data *OAPIResponse) checkError() (err error) {
	if data.ErrCode != 0 {
		err = fmt.Errorf("%d: %s", data.ErrCode, data.ErrMsg)
	}
	return err
}

func (data *OAPIResponse) getWriter() io.Writer {
	return nil
}

//MessageResponse is
type MessageResponse struct {
	OAPIResponse
	MessageID string `json:"messageId"`
}

//MessageResponse is
type MessageReadListResponse struct {
	OAPIResponse
	NextCursor     int64    `json:"next_cursor"`
	ReadUserIdList []string `json:"readUserIdList"`
}

//AccessTokenResponse is
type AccessTokenResponse struct {
	OAPIResponse
	AccessToken string `json:"access_token"`
	Expires     int    `json:"expires_in"`
	Created     int64
}

//CreatedAt is when the access token is generated
func (e *AccessTokenResponse) CreatedAt() int64 {
	return e.Created
}

//ExpiresIn is how soon the access token is expired
func (e *AccessTokenResponse) ExpiresIn() int {
	return e.Expires
}

//JsAPITicketResponse is
type JsAPITicketResponse struct {
	OAPIResponse
	Ticket  string
	Expires int `json:"expires_in"`
	Created int64
}

//CreatedAt is when the ticket is generated
func (e *JsAPITicketResponse) CreatedAt() int64 {
	return e.Created
}

//ExpiresIn is how soon the ticket is expired
func (e *JsAPITicketResponse) ExpiresIn() int {
	return e.Expires
}

//NewDingTalkClient creates a DingTalkClient instance
func NewDingTalkClient(corpID string, corpSecret string) *DingTalkClient {
	c := new(DingTalkClient)
	c.CorpID = corpID
	c.CorpSecret = corpSecret
	c.HTTPClient = &http.Client{
		Timeout: 30 * time.Second,
	}
	c.Cache = NewFileCache(".auth_file")
	return c
}

//RefreshAccessToken is to get a valid access token
func (c *DingTalkClient) RefreshAccessToken(p ...interface{}) error {
	var data AccessTokenResponse
	err := c.Cache.Get(&data)
	if err == nil {
		c.AccessToken = data.AccessToken
		return nil
	}

	params := url.Values{}

	useAppKey := false

	if len(p) > 0 {
		useAppKey = p[0].(bool)
	}

	if useAppKey {
		params.Add("corpid", c.CorpID)
		params.Add("corpsecret", c.CorpSecret)
	} else {
		params.Add("appkey", c.CorpID)
		params.Add("appsecret", c.CorpSecret)
	}

	err = c.httpRPC("gettoken", params, nil, &data)
	if err == nil {
		c.AccessToken = data.AccessToken
		data.Expires = data.Expires | 7200
		data.Created = time.Now().Unix()
		err = c.Cache.Set(&data)
	}
	return err
}

//GetJsAPITicket is to get a valid ticket for JS API
func (c *DingTalkClient) GetJsAPITicket() (ticket string, err error) {
	var data JsAPITicketResponse
	cache := NewFileCache(".jsapi_ticket")
	err = cache.Get(&data)
	if err == nil {
		return data.Ticket, err
	}
	err = c.httpRPC("get_jsapi_ticket", nil, nil, &data)
	if err == nil {
		ticket = data.Ticket
		cache.Set(&data)
	}
	return ticket, err
}

//GetConfig is to return config in json
func (c *DingTalkClient) GetConfig(nonceStr string, timestamp string, url string) string {
	ticket, _ := c.GetJsAPITicket()
	config := map[string]string{
		"url":       url,
		"nonceStr":  nonceStr,
		"agentId":   c.AgentID,
		"timeStamp": timestamp,
		"corpId":    c.CorpID,
		"ticket":    ticket,
		"signature": Sign(ticket, nonceStr, timestamp, url),
	}
	bytes, _ := json.Marshal(&config)
	return string(bytes)
}

//Sign is 签名
func Sign(ticket string, nonceStr string, timeStamp string, url string) string {
	s := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%s&url=%s", ticket, nonceStr, timeStamp, url)
	return sha1Sign(s)
}
