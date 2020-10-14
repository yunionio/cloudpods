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

	"yunion.io/x/jsonutils"
)

func doStatsHandler(stats *jsonutils.JSONArray, method, path string, hi, total *SHandlerInfo) {
	s := jsonutils.NewDict()
	s.Add(jsonutils.NewString(method), "method")
	s.Add(jsonutils.NewString(path), "path")
	s.Add(jsonutils.NewString(hi.GetName(nil)), "name")
	s.Add(jsonutils.NewInt(hi.counter2XX.hit), "hit.2XX")
	s.Add(jsonutils.NewFloat64(hi.counter2XX.duration), "duration.2XX")
	s.Add(jsonutils.NewInt(hi.counter4XX.hit), "hit.4XX")
	s.Add(jsonutils.NewFloat64(hi.counter4XX.duration), "duration.4XX")
	s.Add(jsonutils.NewInt(hi.counter5XX.hit), "hit.5XX")
	s.Add(jsonutils.NewFloat64(hi.counter5XX.duration), "duration.5XX")
	total.counter2XX.hit += hi.counter2XX.hit
	total.counter2XX.duration += hi.counter2XX.duration
	total.counter4XX.hit += hi.counter4XX.hit
	total.counter4XX.duration += hi.counter4XX.duration
	total.counter5XX.hit += hi.counter5XX.hit
	total.counter5XX.duration += hi.counter5XX.duration
	stats.Add(s)
}

func StatisticHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	app := AppContextApp(ctx)
	stats := jsonutils.NewArray()
	total := newHandlerInfo("", nil, nil, nil, "", nil)
	doStatsHandler(stats, "*", "*", &app.defHandlerInfo, total)
	for method, root := range app.roots {
		root.Walk(func(path string, data interface{}) {
			hi := data.(*SHandlerInfo)
			doStatsHandler(stats, method, path, hi, total)
		})
	}
	result := jsonutils.NewDict()
	result.Add(stats, "paths")
	result.Add(jsonutils.NewInt(total.counter2XX.hit), "hit.2XX")
	result.Add(jsonutils.NewFloat64(total.counter2XX.duration), "duration.2XX")
	result.Add(jsonutils.NewInt(total.counter4XX.hit), "hit.4XX")
	result.Add(jsonutils.NewFloat64(total.counter4XX.duration), "duration.4XX")
	result.Add(jsonutils.NewInt(total.counter5XX.hit), "hit.5XX")
	result.Add(jsonutils.NewFloat64(total.counter5XX.duration), "duration.5XX")
	fmt.Fprintf(w, result.String())
}
