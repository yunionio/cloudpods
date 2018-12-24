package aliyun

import (
	"errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

func processCommonRequest(client *sdk.Client, req *requests.CommonRequest) (response *responses.CommonResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("client.ProcessCommonRequest error: %s", r)
			// debug.PrintStack()
			response = nil
			jsonError := jsonutils.NewDict()
			jsonError.Add(jsonutils.NewString("SignatureNonceUsed"), "Code")
			err = errors.New(jsonError.String())
		}
	}()
	return client.ProcessCommonRequest(req)
}
