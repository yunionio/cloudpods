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

package cloudcommon

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
)

// convert clickhouse sqlstr v1 to v2
// v1: tcp://192.168.222.4:9000?database=yunionmeter&read_timeout=10&write_timeout=20
// v2: clickhouse://username:password@host1:9000,host2:9000/database?dial_timeout=200ms&max_execution_time=60
func clickhouseSqlStrV1ToV2(sqlstr string) (string, error) {
	if strings.HasPrefix(sqlstr, "clickhouse://") {
		// already v2 format
		return sqlstr, nil
	}
	queryPos := strings.IndexByte(sqlstr, '?')
	if queryPos <= 0 {
		return "", errors.Wrap(httperrors.ErrInputParameter, "no query string")
	}
	hostPart := sqlstr[len("tcp://"):queryPos]
	qs, err := jsonutils.ParseQueryString(sqlstr[queryPos+1:])
	if err != nil {
		return "", errors.Wrap(err, "ParseQueryString")
	}
	dbname, _ := qs.GetString("database")
	if len(dbname) == 0 {
		return "", errors.Wrap(httperrors.ErrInputParameter, "empty database")
	}
	uname, _ := qs.GetString("username")
	pword, _ := qs.GetString("password")
	if len(uname) > 0 {
		if len(pword) > 0 {
			hostPart = fmt.Sprintf("%s:%s@%s", uname, pword, hostPart)
		} else {
			hostPart = fmt.Sprintf("%s@%s", uname, hostPart)
		}
	}
	return fmt.Sprintf("clickhouse://%s/%s?dial_timeout=200ms&max_execution_time=60", hostPart, dbname), nil
}

func clickhouseSqlStrV2ToV1(sqlstr string) (string, error) {
	if strings.HasPrefix(sqlstr, "tcp://") {
		// already v1 format
		return sqlstr, nil
	}
	hostPart := sqlstr[len("clickhouse://"):]
	queryPos := strings.IndexByte(hostPart, '?')
	if queryPos > 0 {
		hostPart = hostPart[:queryPos]
	}
	slashPos := strings.IndexByte(hostPart, '/')
	if slashPos <= 0 {
		return "", errors.Wrap(httperrors.ErrInputParameter, "no database part")
	}
	qs := make(map[string]string)
	qs["database"] = hostPart[slashPos+1:]
	hostPart = hostPart[:slashPos]
	atPos := strings.IndexByte(hostPart, '@')
	if atPos > 0 {
		authPart := hostPart[:atPos]
		hostPart = hostPart[atPos+1:]
		colonPos := strings.IndexByte(authPart, ':')
		if colonPos > 0 {
			qs["username"] = authPart[:colonPos]
			qs["password"] = authPart[colonPos+1:]
		} else {
			qs["username"] = authPart
		}
	}
	return fmt.Sprintf("tcp://%s?%s&read_timeout=10&write_timeout=20", hostPart, jsonutils.Marshal(qs).QueryString()), nil
}

func validateClickhouseV2Str(sqlstr string) error {
	if !strings.HasPrefix(sqlstr, "clickhouse://") {
		return errors.Wrapf(httperrors.ErrInputParameter, "must start with clickhouse://")
	}
	qsPos := strings.IndexByte(sqlstr, '?')
	if qsPos >= 0 {
		sqlstr = sqlstr[:qsPos]
	}
	slashPos := strings.IndexByte(sqlstr, '/')
	if slashPos <= 0 {
		return errors.Wrapf(httperrors.ErrInputParameter, "missing db slash")
	}
	dbName := sqlstr[slashPos+1:]
	if len(dbName) == 0 {
		return errors.Wrapf(httperrors.ErrInputParameter, "empty database name")
	}
	return nil
}

func validateClickhouseV1Str(sqlstr string) error {
	if !strings.HasPrefix(sqlstr, "tcp://") {
		return errors.Wrapf(httperrors.ErrInputParameter, "must start with tcp://")
	}
	qsPos := strings.IndexByte(sqlstr, '?')
	if qsPos <= 0 {
		return errors.Wrapf(httperrors.ErrInputParameter, "mising query string")
	}
	qsPart := sqlstr[qsPos+1:]
	qs, err := jsonutils.ParseQueryString(qsPart)
	if err != nil {
		return errors.Wrap(err, "ParseQueryString")
	}
	dbName, _ := qs.GetString("database")
	if len(dbName) == 0 {
		return errors.Wrapf(httperrors.ErrInputParameter, "empty database name")
	}
	return nil
}
