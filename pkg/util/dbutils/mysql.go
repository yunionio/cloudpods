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

package dbutils

import (
	"strings"

	"yunion.io/x/pkg/errors"
)

type SDBConfig struct {
	Hostport string
	Database string
	Username string
	Password string
}

func (cfg SDBConfig) Validate() error {
	errs := make([]error, 0)
	if len(cfg.Hostport) == 0 {
		errs = append(errs, errors.Error("empty host port"))
	}
	if len(cfg.Username) == 0 {
		errs = append(errs, errors.Error("empty username"))
	}
	if len(cfg.Database) == 0 {
		errs = append(errs, errors.Error("empty database"))
	}
	return errors.NewAggregate(errs)
}

func ParseMySQLConnStr(connStr string) SDBConfig {
	// fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", user, passwd, host, port, dburl, query.Encode())
	cfg := SDBConfig{}
	index := strings.Index(connStr, "@tcp(")
	if index > 0 {
		userpass := connStr[:index]
		hostdb := connStr[index+len("@tcp("):]

		index = strings.Index(userpass, ":")
		if index > 0 {
			cfg.Username = userpass[:index]
			cfg.Password = userpass[index+1:]
		} else if index < 0 {
			cfg.Username = userpass
		}
		index = strings.Index(hostdb, ")/")
		if index > 0 {
			cfg.Hostport = hostdb[:index]
			dbstr := hostdb[index+len(")/"):]
			index = strings.Index(dbstr, "?")
			if index > 0 {
				cfg.Database = dbstr[:index]
			} else if index < 0 {
				cfg.Database = dbstr
			}
		}
	}
	return cfg
}
