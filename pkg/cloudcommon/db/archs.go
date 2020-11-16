package db

import (
	"context"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SMultiArchResourceBase struct {
	// 操作系统 CPU 架构
	// example: x86 arm
	OsArch string `width:"16" charset:"ascii" nullable:"true" list:"user" get:"user" create:"optional" update:"domain"`
}

type SMultiArchResourceBaseManager struct{}

func (manager *SMultiArchResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.MultiArchResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.OsArch) > 0 {
		osArchs := []string{query.OsArch}
		if query.OsArch == compute.OS_ARCH_X86 {
			osArchs = append(osArchs, "")
		}
		q = q.In("os_arch", osArchs)
	}
	return q, nil
}
