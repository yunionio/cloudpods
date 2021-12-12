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

package clickhouse

import (
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"
)

type sTTL struct {
	// number of time interval
	Count int
	// TTL in month, day or hour
	Unit string
}

type sColumnTTL struct {
	sTTL

	ColName string
}

func parseTTL(ttl string) (sTTL, error) {
	ret := sTTL{}
	if len(ttl) == 0 {
		return ret, errors.Wrap(errors.ErrInvalidStatus, "not valid ttl")
	}
	unit := ""
	switch ttl[len(ttl)-1] {
	case 'h':
		unit = "HOUR"
	case 'd':
		unit = "DAY"
	case 'm':
		unit = "MONTH"
	}
	if len(unit) == 0 {
		unit = "MONTH"
	} else {
		ttl = ttl[:len(ttl)-1]
	}
	intv, err := strconv.ParseInt(ttl, 10, 64)
	if err != nil {
		return ret, errors.Wrap(errors.ErrInvalidStatus, "not valid ttl")
	}
	ret.Count = int(intv)
	ret.Unit = unit
	return ret, nil
}

// created_at + INTERVAL 3 MONTH
func parseTTLExpression(expr string) (sColumnTTL, error) {
	parts := strings.Split(expr, " ")
	ret := sColumnTTL{}
	if len(parts) == 5 && parts[1] == "+" && strings.HasPrefix(parts[2], "INT") {
		ret.ColName = parts[0]
		if ret.ColName[0] == '`' || ret.ColName[0] == '\'' {
			ret.ColName = ret.ColName[1 : len(ret.ColName)-1]
		}
		switch parts[4] {
		case "MONTH", "DAY", "HOUR":
			ret.Unit = parts[4]
		default:
			return ret, errors.Wrap(errors.ErrInvalidStatus, "invalid time unit, MONTH, DAY or HOUR only")
		}
		var err error
		ret.Count, err = strconv.Atoi(parts[3])
		if err != nil {
			return ret, errors.Wrap(err, "invalid interval count")
		}
		return ret, nil
	} else if len(parts) == 3 && parts[1] == "+" && strings.HasPrefix(parts[2], "toInterval") {
		ret.ColName = parts[0]
		if ret.ColName[0] == '`' || ret.ColName[0] == '\'' {
			ret.ColName = ret.ColName[1 : len(ret.ColName)-1]
		}
		intvlCnts := strings.Split(parts[2][len("toInterval"):len(parts[2])-1], "(")
		cnt, err := strconv.Atoi(intvlCnts[1])
		if err != nil {
			return ret, errors.Wrapf(err, "strconv.Atoi %s", intvlCnts[1])
		}
		ret.Count = cnt
		switch intvlCnts[0] {
		case "Year":
			ret.Unit = "MONTH"
			ret.Count *= 12
		case "Quarter":
			ret.Unit = "MONTH"
			ret.Count *= 3
		case "Month":
			ret.Unit = "MONTH"
		case "Week":
			ret.Unit = "DAY"
			ret.Count *= 7
		case "Day":
			ret.Unit = "DAY"
		case "Hour":
			ret.Unit = "HOUR"
		default:
			return ret, errors.Wrapf(errors.ErrInvalidStatus, "invalid interval %s", intvlCnts[0])
		}
		return ret, nil
	} else {
		return ret, errors.Wrapf(errors.ErrInvalidStatus, "invalid format %s", expr)
	}
}
