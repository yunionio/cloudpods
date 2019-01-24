package modules

import (
	"net/http"
	"strconv"

	"fmt"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

type SServerManager struct {
	ResourceManager
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212668.html
// v.1.1 新增支持创建包年/包月的弹性云服务器。！！但是不支持查询等调用 https://support.huaweicloud.com/api-ecs/zh-cn_topic_0093055772.html
func NewServerManager(regionId, projectId string, signer auth.Signer) *SServerManager {
	return &SServerManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "server",
		KeywordPlural: "servers",

		ResourceKeyword: "cloudservers",
	}}
}

func (self *SServerManager) List(querys map[string]string) (*responses.ListResult, error) {
	if offset, exists := querys["offset"]; !exists {
		// 华为云分页参数各式各样。cloudserver offset从1开始。部分其他接口从0开始。
		// 另外部分接口使用pager分页 或者 maker分页
		querys["offset"] = "1"
	} else {
		n, err := strconv.Atoi(offset)
		if err != nil {
			return nil, fmt.Errorf("offset is invalid: %s", offset)
		}
		querys["offset"] = strconv.Itoa(n + 1)
	}
	return self.ListInContextWithSpec(nil, "detail", querys, self.KeywordPlural)
}

/*
返回job id 或者 order id

https://support.huaweicloud.com/api-ecs/zh-cn_topic_0093055772.html
创建按需的弹性云服务 ——> job_id 任务ID （返回数据uuid举例："70a599e0-31e7-49b7-b260-868f441e862b"）
包年包月机器  --> order_id (返回数据举例： "CS1711152257C60TL")
*/
func (self *SServerManager) AsyncCreate(params jsonutils.JSONObject) (string, error) {
	origin_version := self.version
	self.version = "v1.1"
	defer func() { self.version = origin_version }()

	ret, err := self.CreateInContextWithSpec(nil, "", params, "")
	if err != nil {
		return "", err
	}

	// 按需机器
	jobId, err := ret.GetString("job_id")
	if err == nil {
		return jobId, nil
	}

	// 包年包月机器
	return ret.GetString("order_id")
}

func (self *SServerManager) Create(params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("not supported.please use AsyncCreate")
}

// 不推荐使用这个manager
func NewNovaServerManager(regionId, projectId string, signer auth.Signer) *SServerManager {
	return &SServerManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2.1",
		Keyword:       "server",
		KeywordPlural: "servers",

		ResourceKeyword: "servers",
	}}
}
