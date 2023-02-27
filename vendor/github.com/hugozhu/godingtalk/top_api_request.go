/*
 * Author Kevin Zhu
 *
 * Direct questions, comments to <ipandtcp@gmail.com>
 */

package godingtalk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	topAPIRootURL = "https://eco.taobao.com/router/rest"
	formDataType  = "application/x-www-form-urlencoded;charset=utf-8"
)

type TopAPIResponse interface {
	checkError() error
}

type topAPIErrResponse struct {
	ERR struct {
		Code      int    `json:"code"`
		Msg       string `json:"msg"`
		SubCode   string    `json:"sub_code"`
		SubMsg    string `json:"sub_msg"`
		RequestID string `json:"request_id"`
	} `json:"error_response"`
}

func (data *topAPIErrResponse) checkError() (err error) {
	if data.ERR.Code != 0 || len(data.ERR.SubCode) != 0 {
		err = fmt.Errorf("%#v", data.ERR)
	}
	return err
}

func (c *DingTalkClient) topAPIRequest(requestForm url.Values, respData TopAPIResponse) error {
	requestForm.Set("v", "2.0")
	requestForm.Set("format", "json")
	requestForm.Set("simplify", "true")

	err := c.RefreshAccessToken()
	if err != nil {
		return err
	}
	requestForm.Set("session", c.AccessToken)
	if requestForm.Get("timestamp") == "" {
		requestForm.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	}
	if c.PartnerID != "" {
		requestForm.Set("partner_id", c.PartnerID)
	}

	v := bytes.NewBuffer([]byte(requestForm.Encode()))

	req, _ := http.NewRequest("POST", topAPIRootURL, v)
	req.Header.Set("Content-Type", formDataType)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("Server error: " + resp.Status)
	}

	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)

	if err == nil {
		err := json.Unmarshal(buf, &respData)
		if err != nil {
			return err
		}
		return respData.checkError()
	}
	return err
}
