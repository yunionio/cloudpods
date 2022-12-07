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
