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
	"encoding/json"
	"regexp"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SAliyunSMSDriver struct{}

var parser = regexp.MustCompile(`\+(\d*) (.*)`)

func (d *SAliyunSMSDriver) Name() string {
	return DriverAliyun
}

func (d *SAliyunSMSDriver) Verify(config *api.NotifyConfig) error {
	return d.Send(api.SSMSSendParams{
		RemoteTemplate:      config.VerifiyCode,
		To:                  config.PhoneNumber,
		RemoteTemplateParam: api.SRemoteTemplateParam{Code: "0000"},
	}, true, config)
}

func (d *SAliyunSMSDriver) Send(args api.SSMSSendParams, isVerify bool, config *api.NotifyConfig) error {
	if isVerify {
		args.AppKey = config.AccessKeyId
		args.AppSecret = config.AccessKeySecret
		args.Signature = config.Signature
	} else {
		args.AppKey = models.ConfigMap[api.MOBILE].Content.AccessKeyId
		args.AppSecret = models.ConfigMap[api.MOBILE].Content.AccessKeySecret
		args.Signature = models.ConfigMap[api.MOBILE].Content.Signature
	}
	args.TemplateId = args.RemoteTemplate
	return d.sendSms(args)
}

func (d *SAliyunSMSDriver) sendSms(args api.SSMSSendParams) error {
	// lock and update
	client, err := sdk.NewClientWithAccessKey("default", args.AppKey, args.AppSecret)
	if err != nil {
		return errors.Wrap(err, "NewClientWithAccessKey")
	}
	m := parser.FindStringSubmatch(args.To)
	if len(m) > 0 {
		if m[1] == "86" {
			args.To = m[2]
		} else {
			args.To = m[1] + m[2]
		}
	}
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = "dysmsapi.aliyuncs.com"
	request.Version = "2017-05-25"
	request.ApiName = "SendSms"
	request.QueryParams["RegionId"] = "default"
	request.QueryParams["PhoneNumbers"] = args.To
	request.QueryParams["SignName"] = args.Signature

	request.QueryParams["TemplateCode"] = args.TemplateId
	request.QueryParams["TemplateParam"] = jsonutils.Marshal(args.RemoteTemplateParam).String()

	return d.checkResponseAndError(client.ProcessCommonRequest(request))
}

func (d *SAliyunSMSDriver) checkResponseAndError(rep *responses.CommonResponse, err error) error {
	if err != nil {
		serr, ok := err.(*sdkerrors.ServerError)
		if !ok {
			return err
		}
		if serr.ErrorCode() == ACCESSKEYID_NOTFOUND {
			return ErrAccessKeyIdNotFound
		}
		if serr.ErrorCode() == SIGN_DOESNOTMATCH {
			return ErrSignatureDoesNotMatch
		}
		return err
	}

	type RepContent struct {
		Message string
		Code    string
	}
	respContent := rep.GetHttpContentBytes()
	rc := RepContent{}
	err = json.Unmarshal(respContent, &rc)
	if err != nil {
		return errors.Wrap(err, "json.Unmarshal")
	}
	if rc.Code == "OK" {
		return nil
	}
	if rc.Code == SIGHNTURE_ILLEGAL {
		return ErrSignnameInvalid
	} else if rc.Code == TEMPLATE_ILLGAL {
		return ErrSignnameInvalid
	}
	return errors.Error(rc.Message)
}

func init() {
	models.SMSRegister(&SAliyunSMSDriver{})
}
