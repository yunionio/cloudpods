package etcd

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func NewCloudirManager(keyword, keywordPlural string, columns, adminColumns []string) modules.ResourceManager {
	return modules.NewResourceManager(
		"cloudir",
		keyword,
		keywordPlural,
		columns,
		adminColumns,
	)
}
