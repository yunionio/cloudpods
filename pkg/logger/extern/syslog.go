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
	"log/syslog"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

var (
	sysLog *syslog.Writer
)

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
	sysLog, err = syslog.Dial(proto, addr, syslog.LOG_ERR|syslog.LOG_INFO, tag)
	if err != nil {
		return errors.Wrap(err, "syslog.Dial")
	}
	log.Infof("Start syslog to %s://%s@%s", proto, addr, tag)
	return nil
}

func Error(msg string) {
	if sysLog != nil {
		sysLog.Err(msg)
	}
}

func Info(msg string) {
	if sysLog != nil {
		sysLog.Info(msg)
	}
}
