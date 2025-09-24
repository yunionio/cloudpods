package retry

import (
	"errors"
	"github.com/ks3sdklib/aws-sdk-go/aws/awserr"
	"github.com/ks3sdklib/aws-sdk-go/internal/apierr"
)

// ShouldRetry 判断是否需要重试
// 重试条件：
// 1.状态码为5xx
// 2.状态码在retryErrorCodes中
// 3.错误码在retryableCodes中
func ShouldRetry(err error) bool {
	var requestError *apierr.RequestError
	if errors.As(err, &requestError) {
		if requestError.StatusCode() >= 500 {
			return true
		}

		for _, code := range retryErrorCodes {
			if requestError.StatusCode() == code {
				return true
			}
		}
	}

	if err, ok := err.(awserr.Error); ok {
		return IsCodeRetryable(err.Code())
	}

	return false
}

// 重试错误码
var retryErrorCodes = []int{
	408, // RequestTimeout
	429, // TooManyRequests
}

// retryableCodes is a collection of service response codes which are retry-able
// without any further action.
var retryableCodes = map[string]struct{}{
	"RequestError":                           {},
	"ProvisionedThroughputExceededException": {},
	"Throttling":                             {},
}

// credsExpiredCodes is a collection of error codes which signify the credentials
// need to be refreshed. Expired tokens require refreshing of credentials, and
// resigning before the request can be retried.
var credsExpiredCodes = map[string]struct{}{
	"ExpiredToken":          {},
	"ExpiredTokenException": {},
	"RequestExpired":        {},
}

func IsCodeRetryable(code string) bool {
	if _, ok := retryableCodes[code]; ok {
		return true
	}

	return IsCodeExpiredCreds(code)
}

func IsCodeExpiredCreds(code string) bool {
	_, ok := credsExpiredCodes[code]
	return ok
}
