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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

const (
	OutputMaxBytes   = 64*1024*1024 - 1
	PlaybookMaxBytes = 64*1024 - 1
)

type ansiblePlaybookOutputRecorder interface {
	db.IModel
	getMaxOutputLength() int
	getOutput() string
	setOutput(string)
}

type ansiblePlaybookOutputWriter struct {
	rec ansiblePlaybookOutputRecorder
}

func (w *ansiblePlaybookOutputWriter) Write(p []byte) (n int, err error) {
	rec := w.rec
	_, err = db.Update(rec, func() error {
		cur := rec.getOutput()
		i := len(p) + len(cur) - rec.getMaxOutputLength()
		if i > 0 {
			// truncate to preserve the tail
			rec.setOutput(cur[i:] + string(p))
		} else {
			rec.setOutput(cur + string(p))
		}
		return nil
	})
	if err != nil {
		log.Errorf("%s %s(%s): record output: %v", rec.Keyword(), rec.GetName(), rec.GetId(), err)
		return 0, err
	}
	return len(p), nil
}
