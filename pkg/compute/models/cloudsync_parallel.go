// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/compute/options"
)

// runSyncTasksParallel 在单 region DoSync 内对无依赖任务做有限并发：
// Wave2/4 资源类型、Wave3 多 zone、zone 内多 host、host 内 VM 外设、
// 以及 LB / RDS 等实例外设。
// 使用独立 semaphore（errgroup.SetLimit），禁止复用 syncWorkers region 池。
// 并发上限取 CloudRegionResourceSyncConcurrency。
func runSyncTasksParallel(ctx context.Context, tasks []func()) {
	if len(tasks) == 0 {
		return
	}
	limit := options.Options.CloudRegionResourceSyncConcurrency
	if limit <= 0 {
		limit = 1
	}
	if limit == 1 || len(tasks) == 1 {
		for _, task := range tasks {
			if err := CheckCloudaccountSyncCanceled(ctx); err != nil {
				log.Infof("runSyncTasksParallel aborted: %v", err)
				return
			}
			task()
		}
		return
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	for i := range tasks {
		task := tasks[i]
		g.Go(func() error {
			if err := CheckCloudaccountSyncCanceled(gctx); err != nil {
				return err
			}
			task()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		log.Infof("runSyncTasksParallel finished with: %v", err)
	}
}
