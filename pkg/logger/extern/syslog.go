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

package extern

import (
	"fmt"
	"log/syslog"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
)

var (
	sysLog          ISyslogWriter
	syslogWorkerMan *appsrv.SWorkerManager
)

func init() {
	syslogWorkerMan = appsrv.NewWorkerManager("syslogSenderWorkerManager", 1, 50, false)
}

func InitSyslog(url string) error {
	proto := "tcp"
	addr := url
	tag := "cloudaudit"

	if strings.HasPrefix(url, "tcp://") {
		proto = "tcp"
		addr = url[6:]
	} else if strings.HasPrefix(url, "udp://") {
		proto = "udp"
		addr = url[6:]
	}

	if strings.Contains(addr, "@") {
		parts := strings.Split(addr, "@")
		addr = parts[0]
		tag = parts[1]
	}

	var err error
	sysLog, err = syslog.Dial(proto, addr, 0, tag)
	if err != nil {
		return errors.Wrap(err, "syslog.Dial")
	}
	log.Infof("Start syslog to %s://%s@%s", proto, addr, tag)
	return nil
}

func Emergency(msg string) {
	if sysLog == nil {
		return
	}
	sendSyslog("emerg", msg)
}

func Alert(msg string) {
	if sysLog == nil {
		return
	}
	sendSyslog("alert", msg)
}

func Critical(msg string) {
	if sysLog == nil {
		return
	}
	sendSyslog("crit", msg)
}

func Error(msg string) {
	if sysLog == nil {
		return
	}
	sendSyslog("err", msg)
}

func Warning(msg string) {
	if sysLog == nil {
		return
	}
	sendSyslog("warning", msg)
}

func Notice(msg string) {
	if sysLog == nil {
		return
	}
	sendSyslog("notice", msg)
}

func Info(msg string) {
	if sysLog == nil {
		return
	}
	sendSyslog("info", msg)
}

func Debug(msg string) {
	if sysLog == nil {
		return
	}
	sendSyslog("debug", msg)
}

func sendSyslog(facility string, msg string) {
	log.Infof("%s", msg)
	syslogWorkerMan.Run(&syslogTask{
		facility: facility,
		msg:      msg,
	}, nil, nil)
}

type syslogTask struct {
	facility string
	msg      string
}

func (t *syslogTask) Run() {
	if sysLog == nil {
		return
	}
	switch t.facility {
	case "info":
		sysLog.Info(t.msg)
	case "debug":
		sysLog.Debug(t.msg)
	case "emerg":
		sysLog.Emerg(t.msg)
	case "warning":
		sysLog.Warning(t.msg)
	case "notice":
		sysLog.Notice(t.msg)
	case "err":
		sysLog.Err(t.msg)
	case "crit":
		sysLog.Crit(t.msg)
	case "Alert":
		sysLog.Alert(t.msg)
	default:
		sysLog.Info(t.msg)
	}
}

func (t *syslogTask) Dump() string {
	return fmt.Sprintf("syslogTask %v %s", t.facility, t.msg)
}
