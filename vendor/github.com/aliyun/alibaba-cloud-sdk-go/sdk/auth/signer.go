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

package auth

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/signers"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"reflect"
)

type Signer interface {
	GetName() string
	GetType() string
	GetVersion() string
	GetAccessKeyId() string
	GetExtraParam() map[string]string
	Sign(stringToSign, secretSuffix string) string
	Shutdown()
}

func NewSignerWithCredential(credential Credential, commonApi func(request *requests.CommonRequest, signer interface{}) (response *responses.CommonResponse, err error)) (signer Signer, err error) {
	switch instance := credential.(type) {
	case *credentials.BaseCredential:
		{
			signer, err = signers.NewSignerV1(instance)
		}
	case *credentials.StsCredential:
		{
			signer, err = signers.NewSignerSts(instance)
		}
	case *credentials.StsAssumeRoleCredential:
		{
			signer, err = signers.NewSignerStsAssumeRole(instance, commonApi)
		}
	case *credentials.KeyPairCredential:
		{
			signer, err = signers.NewSignerKeyPair(instance, commonApi)
		}
	case *credentials.EcsInstanceCredential:
		{
			signer, err = signers.NewSignereEcsInstance(instance, commonApi)
		}
	default:
		message := fmt.Sprintf(errors.UnsupportedCredentialMessage, reflect.TypeOf(credential))
		err = errors.NewClientError(errors.UnsupportedCredentialCode, message, nil)
	}
	return
}

func Sign(request requests.AcsRequest, signer Signer, regionId string) (err error) {
	switch request.GetStyle() {
	case requests.ROA:
		{
			signRoaRequest(request, signer, regionId)
		}
	case requests.RPC:
		{
			signRpcRequest(request, signer, regionId)
		}
	default:
		message := fmt.Sprintf(errors.UnknownRequestTypeMessage, reflect.TypeOf(request))
		err = errors.NewClientError(errors.UnknownRequestTypeCode, message, nil)
	}

	return
}
