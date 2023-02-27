package websocket

import (
	apis "yunion.io/x/onecloud/pkg/apis/websocket"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

/*
添加新manager注意事项：
1. version字段   -- 在endpoint中注册的url如果携带版本。例如http://x.x.x.x/api/v1，那么必须标注对应version字段。否者可能导致yunionapi报资源not found的错误。
*/

func newWebsocketManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_WEBSOCKET, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}
