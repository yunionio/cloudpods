/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package errors

const (
	DefaultClientErrorStatus = 400
	DefaultClientErrorCode   = "SDK.ClientError"

	UnsupportedCredentialCode    = "SDK.UnsupportedCredential"
	UnsupportedCredentialMessage = "Specified credential (type = %s) is not supported, please check"

	CanNotResolveEndpointCode    = "SDK.CanNotResolveEndpoint"
	CanNotResolveEndpointMessage = "Can not resolve endpoint(param = %s), please check the user guide\n %s"

	UnsupportedParamPositionCode    = "SDK.UnsupportedParamPosition"
	UnsupportedParamPositionMessage = "Specified param position (%s) is not supported, please upgrade sdk and retry"

	AsyncFunctionNotEnabledCode    = "SDK.AsyncFunctionNotEnabled"
	AsyncFunctionNotEnabledMessage = "Async function is not enabled in client, please invoke 'client.EnableAsync' function"

	UnknownRequestTypeCode    = "SDK.UnknownRequestType"
	UnknownRequestTypeMessage = "Unknown Request Type: %s"

	MissingParamCode = "SDK.MissingParam"
	InvalidParamCode = "SDK.InvalidParam"
)

type ClientError struct {
	errorCode   string
	message     string
	originError error
}

func NewClientError(errorCode, message string, originErr error) Error {
	return &ClientError{
		errorCode:   errorCode,
		message:     message,
		originError: originErr,
	}
}

func (err *ClientError) Error() string {
	if err.originError != nil {
		return err.originError.Error()
	} else {
		return ""
	}
}

func (err *ClientError) OriginError() error {
	return err.originError
}

func (*ClientError) HttpStatus() int {
	return DefaultClientErrorStatus
}

func (err *ClientError) ErrorCode() string {
	if err.errorCode == "" {
		return DefaultClientErrorCode
	} else {
		return err.errorCode
	}
}

func (err *ClientError) Message() string {
	return err.message
}
