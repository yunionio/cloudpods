package aliyun

import (
	aliyunerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
)

func isError(err error, code string) bool {
	aliyunErr, ok := err.(aliyunerrors.Error)
	if !ok {
		return false
	}
	return aliyunErr.ErrorCode() == code
}
