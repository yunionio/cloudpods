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

import "yunion.io/x/pkg/errors"

const (
	DriverKey = "sms_driver"

	DriverAliyun = "smsaliyun"
	DriverHuawei = "smshuawei"
)

const (
	ACCESS_KEY_ID     = "access_key_id"
	ACCESS_KEY_SECRET = "access_key_secret"
	SIGNATURE         = "signature"
	SERVICE_URL       = "service_url"
)

var (
	ErrAccessKeyIdNotFound   = errors.Error("AccessKeyId not found")
	ErrSignatureDoesNotMatch = errors.Error("AccessKeySecret does not match with the accessKeyId")
	ErrSignnameInvalid       = errors.Error("Invalid signature (does not exist or is blackened)")
	ErrDriverNotFound        = errors.Error("Driver not found")
)

const (
	ACESS_KEY_ID_BP     = "accessKeyId"
	ACESS_KEY_SECRET_BP = "accessKeySecret"

	NEED_REMOTE_TEMPLATE = "remote template is needed in aliyun sms"

	ACCESSKEYID_NOTFOUND = "InvalidAccessKeyId.NotFound"
	SIGN_DOESNOTMATCH    = "SignatureDoesNotMatch"
	SIGHNTURE_ILLEGAL    = "isv.SMS_SIGNATURE_ILLEGAL"
	TEMPLATE_ILLGAL      = "isv.SMS_TEMPLATE_ILLEGAL"
)

const (
	HuaweiSendUri = "/sms/batchSendSms/v1"
)
