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

package appsrv

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v3/process"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

func ProcessStatsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ret := struct {
		ProcessStat apis.ProcessStats `json:"process_stats"`
	}{}
	ret.ProcessStat.GoroutineNum = runtime.NumGoroutine()
	ret.ProcessStat.MemSize = m.Alloc
	process, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		fmt.Fprintf(w, jsonutils.Marshal(ret).String())
		return
	}
	ret.ProcessStat.CpuPercent, _ = process.CPUPercent()
	ret.ProcessStat.MemPercent, _ = process.MemoryPercent()
	fmt.Fprintf(w, jsonutils.Marshal(ret).String())
}
