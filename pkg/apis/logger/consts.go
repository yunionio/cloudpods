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

package logger

import "yunion.io/x/onecloud/pkg/apis"

const (
	SERVICE_TYPE = apis.SERVICE_TYPE_LOG
)

type TEventSeverity string

// 风险级别 0 紧急(Emergency) 1 警报(Alert) 2 关键(Critical) 3 错误(Error) 4 警告(Warning) 5 通知(Notice) 6 信息(informational) 7 调试(debug)
const (
	SeverityEmergency = TEventSeverity("EMERGENCY")
	SeverityAlert     = TEventSeverity("ALERT")
	SeverityCritical  = TEventSeverity("CRITICAL")
	SeverityError     = TEventSeverity("ERROR")
	SeverityWarning   = TEventSeverity("WARNING")
	SeverityNotice    = TEventSeverity("NOTICE")
	SeverityInfo      = TEventSeverity("INFO")
	SeverityDebug     = TEventSeverity("DEBUG")
)

type TEventKind string

const (
	KindNormal   = TEventKind("NORMAL")
	KindAbnormal = TEventKind("ABNORMAL")
	KindIllegal  = TEventKind("ILLEGAL")
)
