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

package google

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

func (self *SGoogleClient) bigqueryPost(resource string, params map[string]string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	resource = fmt.Sprintf("projects/%s/%s", self.projectId, resource)
	return jsonRequest(self.client, "POST", GOOGLE_BIGQUERY_DOMAIN, GOOGLE_BIGQUERY_API_VERSION, resource, params, body, self.debug)
}

func (self *SGoogleClient) bigqueryList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_BIGQUERY_DOMAIN, GOOGLE_BIGQUERY_API_VERSION, resource, params, nil, self.debug)
}

func (self *SGoogleClient) bigqueryGet(resource string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_BIGQUERY_DOMAIN, GOOGLE_BIGQUERY_API_VERSION, resource, nil, nil, self.debug)
}

func (region *SRegion) BigQuery(sql string) ([]jsonutils.JSONObject, error) {
	req := struct {
		Kind         string `json:"kind"`
		Query        string `json:"query"`
		MaxResults   string `json:"maxResults"`
		UseLegacySql bool   `json:"useLegacySql"`
	}{
		Kind:         "query",
		Query:        sql,
		MaxResults:   "1",
		UseLegacySql: false,
	}
	resp, err := region.client.bigqueryPost("queries", nil, jsonutils.Marshal(req))
	if err != nil {
		return nil, errors.Wrap(err, "bigqueryPost")
	}
	result := SBigQueryResult{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Unmarshal")
	}
	rows, err := result.GetRows()
	if err != nil {
		return nil, errors.Wrap(err, "GetRows")
	}
	return rows, nil
}

type SBigQueryField struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Mode string `json:"mode"`

	Fields []SBigQueryField `json:"fields"`
}

type SBigQuerySchema struct {
	Fields []SBigQueryField `json:"fields"`
}

type SBigQueryJobReference struct {
}

type SBigQueryResult struct {
	CacheHit            bool                   `json:"cacheHit"`
	JobComplete         bool                   `json:"jobComplete"`
	JobReference        SBigQueryJobReference  `json:"jobReference"`
	Kind                string                 `json:"kind"`
	Rows                []jsonutils.JSONObject `json:"rows"`
	Schema              SBigQuerySchema        `json:"schema"`
	TotalBytesProcessed int64                  `json:"totalBytesProcessed"`
	totalRows           int64                  `json:"totalRows"`
}

func (res SBigQueryResult) GetRows() ([]jsonutils.JSONObject, error) {
	rows := make([]jsonutils.JSONObject, 0)
	for _, r := range res.Rows {
		row, err := res.Schema.Parse(r)
		if err != nil {
			return nil, errors.Wrap(err, "Schema.Parse")
		}
		rows = append(rows, jsonutils.Marshal(row))
	}
	return rows, nil
}

func (schema SBigQuerySchema) Parse(r jsonutils.JSONObject) (map[string]jsonutils.JSONObject, error) {
	f := SBigQueryField{
		Type:   "RECORD",
		Mode:   "NULLABLE",
		Name:   "",
		Fields: schema.Fields,
	}
	nr := jsonutils.NewDict()
	nr.Add(r, "v")
	return f.Parse(nr, "")
}

func (f SBigQueryField) Parse(r jsonutils.JSONObject, prefix string) (map[string]jsonutils.JSONObject, error) {
	if len(prefix) > 0 {
		prefix = prefix + "." + f.Name
	} else {
		prefix = f.Name
	}
	ret := make(map[string]jsonutils.JSONObject)
	switch f.Type {
	case "RECORD":
		if f.Mode == "REPEATED" {
			items, err := r.GetArray("v")
			if err != nil {
				return nil, errors.Wrapf(err, "GetArray v %s", r)
			}
			nf := f
			nf.Mode = "NULLABLE"
			val := jsonutils.NewArray()
			for _, item := range items {
				obj, err := nf.Parse(item, "")
				if err != nil {
					return nil, errors.Wrap(err, "Parse items")
				}
				val.Add(jsonutils.Marshal(obj))
			}
			ret[prefix] = val
		} else {
			items, err := r.GetArray("v", "f")
			if err != nil {
				return nil, errors.Wrap(err, "GetArray v.f")
			}
			if len(items) != len(f.Fields) {
				return nil, errors.Wrap(errors.ErrServer, "inconsistent items and fields")
			}
			for i := range f.Fields {
				res, err := f.Fields[i].Parse(items[i], prefix)
				if err != nil {
					return nil, errors.Wrapf(err, "Parse %s", prefix)
				}
				for k, v := range res {
					ret[k] = v
				}
			}
		}
	default:
		v, err := r.Get("v")
		if err != nil {
			return nil, errors.Wrap(err, "Get v")
		}
		ret[prefix] = v
	}
	return ret, nil
}
