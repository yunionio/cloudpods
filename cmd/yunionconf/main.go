package main

/*
Yunion Conf Service
“参数服务”在服务器端为指定用户持久化存储和管理个性化参数，例如 控制台的配置，列表的colume配置等，从而实现产品的个性化配置。
*/

import (
	"yunion.io/x/onecloud/pkg/yunionconf/service"
)

func main() {
	service.StartService()
}
