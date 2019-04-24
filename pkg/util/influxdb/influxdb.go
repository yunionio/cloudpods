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

package influxdb

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SInfluxdb struct {
	accessUrl string
	client    *http.Client
	dbName    string
}

func NewInfluxdb(accessUrl string) *SInfluxdb {
	inst := SInfluxdb{
		accessUrl: accessUrl,
		client:    httputils.GetDefaultClient(),
	}
	return &inst
}

type dbResult struct {
	Name    string
	Columns []string
	Values  [][]jsonutils.JSONObject
}

func (db *SInfluxdb) query(sql string) ([][]dbResult, error) {
	nurl := fmt.Sprintf("%s/query?q=%s", db.accessUrl, url.QueryEscape(sql))
	_, body, err := httputils.JSONRequest(db.client, context.Background(), "POST", nurl, nil, nil, false)
	if err != nil {
		return nil, err
	}
	log.Debugf("influx query: %s %s", db.accessUrl, body)
	results, err := body.GetArray("results")
	if err != nil {
		return nil, err
	}
	rets := make([][]dbResult, len(results))
	for i := range results {
		series, err := results[i].Get("series")
		if err == nil {
			ret := make([]dbResult, 0)
			err = series.Unmarshal(&ret)
			if err != nil {
				return nil, err
			}
			rets[i] = ret
		}
	}
	return rets, nil
}

func (db *SInfluxdb) SetDatabase(dbName string) error {
	dbs, err := db.GetDatabases()
	if err != nil {
		return err
	}
	if !utils.IsInStringArray(dbName, dbs) {
		err = db.CreateDatabase(dbName)
		if err != nil {
			return err
		}
		return nil
	}
	db.dbName = dbName
	return nil
}

func (db *SInfluxdb) CreateDatabase(dbName string) error {
	_, err := db.query(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return err
	}
	return nil
}

func (db *SInfluxdb) GetDatabases() ([]string, error) {
	results, err := db.query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	res := results[0][0]
	ret := make([]string, len(res.Values))
	for i := range res.Values {
		ret[i], _ = res.Values[i][0].GetString()
	}
	return ret, nil
}

type SRetentionPolicy struct {
	Name               string
	Duration           string
	ShardGroupDuration string
	ReplicaN           int
	Default            bool
}

func (rp *SRetentionPolicy) String(dbName string) string {
	var buf strings.Builder
	buf.WriteString("RETENTION POLICY \"")
	buf.WriteString(rp.Name)
	buf.WriteString("\" ON \"")
	buf.WriteString(dbName)
	buf.WriteString("\" DURATION ")
	buf.WriteString(rp.Duration)
	buf.WriteString(fmt.Sprintf(" REPLICATION %d", rp.ReplicaN))
	if len(rp.ShardGroupDuration) > 0 {
		buf.WriteString(fmt.Sprintf(" SHARD DURATION %s", rp.ShardGroupDuration))
	}
	if rp.Default {
		buf.WriteString(" DEFAULT")
	}
	return buf.String()
}

func (db *SInfluxdb) GetRetentionPolicies() ([]SRetentionPolicy, error) {
	results, err := db.query(fmt.Sprintf("SHOW RETENTION POLICIES ON %s", db.dbName))
	if err != nil {
		return nil, err
	}
	res := results[0][0]
	ret := make([]SRetentionPolicy, len(res.Values))
	for i := range res.Values {
		tmpDict := jsonutils.NewDict()
		for j := range res.Columns {
			tmpDict.Add(res.Values[i][j], res.Columns[j])
		}
		err = tmpDict.Unmarshal(&ret[i])
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (db *SInfluxdb) CreateRetentionPolicy(rp SRetentionPolicy) error {
	_, err := db.query(fmt.Sprintf("CREATE %s", rp.String(db.dbName)))
	return err
}

func (db *SInfluxdb) AlterRetentionPolicy(rp SRetentionPolicy) error {
	_, err := db.query(fmt.Sprintf("ALTER %s", rp.String(db.dbName)))
	return err
}

func (db *SInfluxdb) SetRetentionPolicy(rp SRetentionPolicy) error {
	rps, err := db.GetRetentionPolicies()
	if err != nil {
		return err
	}
	find := false
	for i := range rps {
		if rps[i].Name == rp.Name {
			find = true
			break
		}
	}
	if find {
		return db.AlterRetentionPolicy(rp)
	} else {
		return db.CreateRetentionPolicy(rp)
	}
}
