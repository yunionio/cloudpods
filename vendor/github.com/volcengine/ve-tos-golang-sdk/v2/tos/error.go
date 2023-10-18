package tos

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

var (
	InputIsNilClientError             = newTosClientError("input is nil. ", nil)
	InputInvalidClientError           = newTosClientError("input data is invalid. ", nil)
	InvalidPartNumber                 = newTosClientError("input part number is invalid. ", nil)
	InvalidUploadID                   = newTosClientError("input upload id is invalid. ", nil)
	InvalidBucketNameLength           = newTosClientError("invalid bucket name, the length must be [3, 63]", nil)
	InvalidBucketNameCharacter        = newTosClientError("invalid bucket name, the character set is illegal", nil)
	InvalidBucketNameStartingOrEnding = newTosClientError("invalid bucket name, the bucket name can be neither starting with '-' nor ending with '-'", nil)
	InvalidObjectNameLength           = newTosClientError("invalid object name, the length must be [1, 696]", nil)
	InvalidObjectNameStartingOrEnding = newTosClientError("invalid object name, the object name can not start with '\\'", nil)
	InvalidObjectNameCharacterSet     = newTosClientError("invalid object name, the character set is illegal", nil)
	InvalidACL                        = newTosClientError("invalid acl type", nil)
	InvalidStorageClass               = newTosClientError("invalid storage class", nil)
	InvalidGrantee                    = newTosClientError("invalid grantee type", nil)
	InvalidCanned                     = newTosClientError("invalid canned type", nil)
	InvalidAzRedundancy               = newTosClientError("invalid az redundancy type", nil)
	InvalidMetadataDirective          = newTosClientError("invalid metadata directive type", nil)
	InvalidPermission                 = newTosClientError("invalid permission type", nil)
	InvalidSSECAlgorithm              = newTosClientError("invalid encryption-decryption algorithm", nil)
	InvalidPartSize                   = newTosClientError("invalid part size, the size must be [5242880, 5368709120]", nil)
	InvalidSrcFilePath                = newTosClientError("invalid file path, the file does not exist", nil)
	InvalidFilePartNum                = newTosClientError("unsupported part number, the maximum is 10000", nil)
	InvalidMarshal                    = newTosClientError("unable to do serialization/deserialization", nil)
	InvalidPreSignedURLExpires        = newTosClientError("invalid pre signed url expires, the time must be less 604800 seconds.", nil)
	InvalidPreSignedConditions        = newTosClientError("invalid pre signed url conditions.", nil)
	InvalidFilePath                   = newTosClientError("invalid file path", nil)
	InvalidCheckpointFilePath         = newTosClientError("invalid checkpoint file path", nil)
	CrcCheckFail                      = newTosClientError("crc check not equal", nil)
	InvalidS3Endpoint                 = newTosClientError("do not support s3 endpoint, please use tos endpoint", nil)
	NotSupportSeek                    = newTosClientError("reader not support seek", nil)
	ProxyNotSupportHttps              = newTosClientError("proxy not support https", nil)
	ProxyUrlInvalid                   = newTosClientError("proxy url invalid", nil)
	NotificationConfigurationsInvalid = newTosClientError("Notification Configurations invalid", nil)
	InvalidCompleteAllPartsLength     = newTosClientError("Should not specify both complete all and Parts", nil)
	InvalidPartsLength                = newTosClientError("You must specify at least one part", nil)
	InvlidDeleteMultiObjectsLength    = newTosClientError("You must specify at least one object", nil)
)

type TosError struct {
	Message string
}

func (e *TosError) Error() string {
	return e.Message
}

// for simplify code
func newTosClientError(message string, cause error) *TosClientError {
	return &TosClientError{
		TosError: TosError{
			Message: message,
		},
		Cause: cause,
	}
}

type TosClientError struct {
	TosError
	Cause error
}

func (t *TosClientError) withCause(err error) error {
	t.Cause = err
	return t
}

// try to unmarshal server error from response
func newTosServerError(res *Response) error {
	data, err := ioutil.ReadAll(io.LimitReader(res.Body, 64<<10)) // avoid too large
	if err != nil && len(data) <= 0 {
		return &TosServerError{
			TosError:    TosError{"tos: server returned an empty body"},
			RequestInfo: res.RequestInfo(),
		}
	}
	se := Error{StatusCode: res.StatusCode}
	if err = json.Unmarshal(data, &se); err != nil {
		return &TosServerError{
			TosError:    TosError{"tos: server returned an invalid body"},
			RequestInfo: res.RequestInfo(),
		}
	}
	return &TosServerError{
		TosError:    TosError{se.Message},
		RequestInfo: res.RequestInfo(),
		Code:        se.Code,
		HostID:      se.HostID,
		Resource:    se.Resource,
	}
}

// 服务端错误定义参考：https://www.volcengine.com/docs/6349/74874
type TosServerError struct {
	TosError    `json:"TosError"`
	RequestInfo `json:"RequestInfo"`
	Code        string `json:"Code,omitempty"`
	HostID      string `json:"HostID,omitempty"`
	Resource    string `json:"Resource,omitempty"`
}

type Error struct {
	StatusCode int    `json:"-"`
	Code       string `json:"Code,omitempty"`
	Message    string `json:"Message,omitempty"`
	RequestID  string `json:"RequestId,omitempty"`
	HostID     string `json:"HostId,omitempty"`
	Resource   string `json:"Resource,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("tos: request error: StatusCode=%d, Code=%s, Message=%q, RequestID=%s, HostID=%s",
		e.StatusCode, e.Code, e.Message, e.RequestID, e.HostID)
}

// Code return error code saved in TosServerError
func Code(err error) string {
	if er, ok := err.(*TosServerError); ok {
		return er.Code
	}
	return ""
}

// StatueCode return status code saved in TosServerError or UnexpectedStatusCodeError
//
// Deprecated: use StatusCode instead
func StatueCode(err error) int {
	return StatusCode(err)
}

// StatusCode return status code saved in TosServerError or UnexpectedStatusCodeError
func StatusCode(err error) int {
	if er, ok := err.(*TosServerError); ok {
		return er.StatusCode
	}
	if er, ok := err.(*UnexpectedStatusCodeError); ok {
		return er.StatusCode
	}
	return 0
}

func RequestID(err error) string {
	switch ev := err.(type) {
	case *TosServerError:
		return ev.RequestID
	case *UnexpectedStatusCodeError:
		return ev.RequestID
	case *ChecksumError:
		return ev.RequestID
	case *SerializeError:
		return ev.RequestID
	}
	return ""
}

type UnexpectedStatusCodeError struct {
	StatusCode    int    `json:"StatusCode,omitempty"`
	ExpectedCodes []int  `json:"ExpectedCodes,omitempty"`
	RequestID     string `json:"RequestId,omitempty"`
	expectedCodes [2]int
	responseMsg   string
	err           Error
}

func NewUnexpectedStatusCodeError(statusCode int, expectedCode int, expectedCodes ...int) *UnexpectedStatusCodeError {
	err := UnexpectedStatusCodeError{
		StatusCode: statusCode,
	}
	err.ExpectedCodes = err.expectedCodes[:0]
	err.ExpectedCodes = append(err.ExpectedCodes, expectedCode)
	err.ExpectedCodes = append(err.ExpectedCodes, expectedCodes...)
	return &err
}

func (us *UnexpectedStatusCodeError) WithRequestBody(res *Response) *UnexpectedStatusCodeError {
	data, err := ioutil.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil || len(data) <= 0 {
		return us
	}
	us.responseMsg = string(data)
	se := Error{StatusCode: res.StatusCode}
	err = json.Unmarshal(data, &se)
	if err != nil {
		return us
	}
	us.err = se
	return us
}

func (us *UnexpectedStatusCodeError) WithRequestID(requestID string) *UnexpectedStatusCodeError {
	us.RequestID = requestID
	return us
}

func (us *UnexpectedStatusCodeError) GoString() string {
	if us.responseMsg != "" {
		return fmt.Sprintf("tos.UnexpectedStatusCodeError{StatusCode:%d, ExpectedCodes:%v, RequestID:%s, ResponseErr:%s}",
			us.StatusCode, us.ExpectedCodes, us.RequestID, us.responseMsg)
	}
	return fmt.Sprintf("tos.UnexpectedStatusCodeError{StatusCode:%d, ExpectedCodes:%v, RequestID:%s}",
		us.StatusCode, us.ExpectedCodes, us.RequestID)
}

func (us *UnexpectedStatusCodeError) Error() string {
	if us.responseMsg != "" {
		return fmt.Sprintf("tos: unexpected status code error: StatusCode=%d, ExpectedCodes=%v, RequestID=%s, ResponseErr:%s",
			us.StatusCode, us.ExpectedCodes, us.RequestID, us.responseMsg)
	}
	return fmt.Sprintf("tos: unexpected status code error: StatusCode=%d, ExpectedCodes=%v, RequestID=%s",
		us.StatusCode, us.ExpectedCodes, us.RequestID)
}

type ChecksumError struct {
	RequestID        string `json:"RequestId,omitempty"`
	ExpectedChecksum string `json:"ExpectedChecksum,omitempty"`
	ActualChecksum   string `json:"ActualChecksum,omitempty"`
}

func (ce *ChecksumError) Error() string {
	return fmt.Sprintf("tos: checksum error: RequestID=%s, ExpectedChecksum=%s, ActualChecksum=%s",
		ce.RequestID, ce.ExpectedChecksum, ce.ActualChecksum)
}

type SerializeError struct {
	RequestID string `json:"RequestId,omitempty"`
	Message   string `json:"Message,omitempty"`
}

func (se *SerializeError) Error() string {
	return fmt.Sprintf("tos: serialize error: RequestID=%s, Message=%q", se.RequestID, se.Message)
}

func checkError(res *Response, readBody bool, okCode int, okCodes ...int) error {
	if res.StatusCode == okCode {
		return nil
	}
	for _, code := range okCodes {
		if res.StatusCode == code {
			return nil
		}
	}
	defer res.Close()
	if readBody && res.StatusCode >= http.StatusBadRequest && res.Body != nil {
		return newTosServerError(res)
		// fall through
	}
	unexpected := NewUnexpectedStatusCodeError(res.StatusCode, okCode, okCodes...).
		WithRequestID(res.RequestInfo().RequestID)
	if readBody && res.Body != nil {
		unexpected = unexpected.WithRequestBody(res)
	}
	return &TosServerError{
		TosError:    TosError{unexpected.Error()},
		RequestInfo: res.RequestInfo(),
		Code:        unexpected.err.Code,
		HostID:      unexpected.err.HostID,
		Resource:    unexpected.err.Resource,
	}
}

// StatusCodeClassifier classifies Errors.
// If the error is nil, it returns NoRetry;
// if the error is TimeoutException or can be interpreted as TosServerError, and the StatusCode is 5xx or 429, it returns Retry;
// otherwise, it returns NoRetry.
type StatusCodeClassifier struct{}

// Classify implements the classifier interface.
func (classifier StatusCodeClassifier) Classify(err error) retryAction {
	if err == nil {
		return NoRetry
	}
	e, ok := err.(*TosServerError)
	if ok {
		if e.StatusCode >= 500 || e.StatusCode == 429 {
			return Retry
		}
	}
	cErr, ok := err.(*TosClientError)
	if ok {
		_, ok = cErr.Cause.(interface{ Timeout() bool })
		if ok {
			return Retry
		}
	}

	return NoRetry
}

// ServerErrorClassifier classify errors returned by POST method.
// If the error is nil, it returns NoRetry;
// if the error can be interpreted as TosServerError and its StatusCode is 5xx or 429, it returns Retry;
// otherwise, it returns NoRetry.
type ServerErrorClassifier struct{}

// Classify implements the classifier interface.
func (classifier ServerErrorClassifier) Classify(err error) retryAction {
	if err == nil {
		return NoRetry
	}
	e, ok := err.(*TosServerError)
	if ok {
		if e.StatusCode >= 500 || e.StatusCode == 429 {
			return Retry
		}
	}
	return NoRetry
}

type NoRetryClassifier struct{}

// Classify implements the classifier interface.
func (classifier NoRetryClassifier) Classify(_ error) retryAction {
	return NoRetry
}
