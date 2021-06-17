package notify

import "yunion.io/x/onecloud/pkg/apis"

type RobotCreateInput struct {
	apis.SharableVirtualResourceCreateInput
	// description: robot type
	// enum: feishu,dingtalk,workwx,webhook
	// example: webhook
	Type string `json:"type"`
	// description: address
	// example: http://helloworld.io/test/webhook
	Address string `json:"address"`
	// description: Language preference
	// example: zh_CN
	Lang string `json:"lang"`
}

type RobotDetails struct {
	apis.SharableVirtualResourceDetails
}

type RobotListInput struct {
	apis.SharableVirtualResourceListInput
	apis.EnabledResourceBaseListInput
	// description: robot type
	// enum: feishu,dingtalk,workwx,webhook
	// example: webhook
	Type string `json:"type"`
	// description: Language preference
	// example: en
	Lang string `json:"lang"`
}

type RobotUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput
	// description: address
	// example: http://helloworld.io/test/webhook
	Address string `json:"address"`
	// description: Language preference
	// example: en
	Lang string `json:"lang"`
}
