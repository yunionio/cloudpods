// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package smsdriver

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
)

// 无需修改,用于格式化鉴权头域,给"X-WSSE"参数赋值
const WSSE_HEADER_FORMAT = "UsernameToken Username=\"%s\",PasswordDigest=\"%s\",Nonce=\"%s\",Created=\"%s\""

// 无需修改,用于格式化鉴权头域,给"Authorization"参数赋值
const AUTH_HEADER_VALUE = "WSSE realm=\"SDP\",profile=\"UsernameToken\",type=\"Appkey\""

type SHuaweiSMSDriver struct{}

func (d *SHuaweiSMSDriver) Name() string {
	return DriverHuawei
}

func (d *SHuaweiSMSDriver) Verify(config *api.NotifyConfig) error {
	return d.Send(api.SSMSSendParams{
		RemoteTemplate:      config.VerifiyCode,
		To:                  config.PhoneNumber,
		RemoteTemplateParam: api.SRemoteTemplateParam{Code: "0000"},
	}, true, config)
}

func (d *SHuaweiSMSDriver) Send(args api.SSMSSendParams, isVerify bool, config *api.NotifyConfig) error {
	if isVerify {
		args.AppKey = config.AccessKeyId
		args.AppSecret = config.AccessKeySecret
		args.Signature = config.Signature
	} else {
		args.AppKey = models.ConfigMap[api.MOBILE].Content.AccessKeyId
		args.AppSecret = models.ConfigMap[api.MOBILE].Content.AccessKeySecret
		args.Signature = models.ConfigMap[api.MOBILE].Content.Signature
	}
	args.TemplateId = strings.Split(args.RemoteTemplate, "/")[1]
	args.From = strings.Split(args.RemoteTemplate, "/")[0]
	return d.sendSms(args)
}

func (d *SHuaweiSMSDriver) sendSms(args api.SSMSSendParams) error {
	uri := models.ConfigMap[api.MOBILE].Content.ServiceUrl + HuaweiSendUri + "?"
	header := http.Header{}
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("Authorization", AUTH_HEADER_VALUE)
	header.Set("X-WSSE", buildWsseHeader(args.AppKey, args.AppSecret))
	params := url.Values{}
	params.Set("from", args.From)
	params.Set("to", args.To)
	params.Set("templateId", args.TemplateId)
	params.Set("templateParas", args.TemplateParas)
	params.Set("signature", args.Signature)
	resp, err := sendRequest(uri, httputils.POST, header, params, nil)
	if err != nil {
		return errors.Wrap(err, "huawei sendRequest")
	}
	code, _ := resp.GetString("code")
	if code != "000000" {
		return errors.Wrap(errors.ErrInvalidFormat, resp.PrettyString())
	}
	return nil
}

func buildWsseHeader(appKey, appSecret string) string {
	var cTime = time.Now().Format("2006-01-02T15:04:05Z")
	var nonce = uuid.NewV4().String()
	nonce = strings.ReplaceAll(nonce, "-", "")

	h := sha256.New()
	h.Write([]byte(nonce + cTime + appSecret))
	passwordDigestBase64Str := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return fmt.Sprintf(WSSE_HEADER_FORMAT, appKey, passwordDigestBase64Str, nonce, cTime)
}

func init() {
	models.SMSRegister(&SHuaweiSMSDriver{})
}
