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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

type ResponseParser struct{}

var (
	legendFormat *regexp.Regexp
)

func init() {
	legendFormat = regexp.MustCompile(`\[\[(\w+)(\.\w+)*\]\]*|\$\s*(\w+?)*`)
}

func (rp *ResponseParser) Parse(response *Response, query *Query) *tsdb.QueryResult {
	queryRes := tsdb.NewQueryResult()

	for _, result := range response.Results {
		queryRes.Series = append(queryRes.Series, rp.transformRowsV2(result.Series, queryRes, query)...)
	}

	return queryRes
}

func (rp *ResponseParser) transformRows(rows []Row, queryResult *tsdb.QueryResult, query *Query) tsdb.TimeSeriesSlice {
	var result tsdb.TimeSeriesSlice
	for _, row := range rows {
		for columnIndex, column := range row.Columns {
			if column == "time" {
				continue
			}

			var points tsdb.TimeSeriesPoints
			for _, valuePair := range row.Values {
				point, err := rp.parseTimepoint(valuePair, columnIndex)
				if err == nil {
					points = append(points, point)
				}
			}
			result = append(result, &tsdb.TimeSeries{
				Name:   rp.formatSerieName(row, column, query),
				Points: points,
				Tags:   row.Tags,
			})
		}
	}

	return result
}

func (rp *ResponseParser) transformRowsV2(rows []Row, queryResult *tsdb.QueryResult, query *Query) tsdb.TimeSeriesSlice {
	var result tsdb.TimeSeriesSlice
	for _, row := range rows {
		col := ""
		columns := make([]string, 0)
		for _, column := range row.Columns {
			if column == "time" {
				continue
			}
			columns = append(columns, column)
			if col == "" {
				col = column
				continue
			}
			col = fmt.Sprintf("%s-%s", col, column)
		}
		columns = append(columns, "time")
		var points tsdb.TimeSeriesPoints
		for _, valuePair := range row.Values {
			point, err := rp.parseTimepointV2(valuePair)
			if err == nil {
				points = append(points, point)
			}
		}
		result = append(result, &tsdb.TimeSeries{
			Name:    rp.formatSerieName(row, col, query),
			Columns: columns,
			Points:  points,
			Tags:    row.Tags,
		})
	}

	return result
}

func (rp *ResponseParser) transformRowToTable(row Row, table *tsdb.Table) *tsdb.Table {
	for _, col := range row.Columns {
		table.Columns = append(table.Columns, tsdb.TableColumn{
			Text: col})
	}
	table.Rows = make([]tsdb.RowValues, len(row.Values))
	for _, value := range row.Values {
		rowvalue := tsdb.RowValues(value)
		table.Rows = append(table.Rows, rowvalue)
	}
	return table
}

func (rp *ResponseParser) formatSerieName(row Row, column string, query *Query) string {
	if query.Alias == "" {
		return rp.buildSerieNameFromQuery(row, column)
	}

	nameSegment := strings.Split(row.Name, ".")

	result := legendFormat.ReplaceAllFunc([]byte(query.Alias), func(in []byte) []byte {
		aliasFormat := string(in)
		aliasFormat = strings.Replace(aliasFormat, "[[", "", 1)
		aliasFormat = strings.Replace(aliasFormat, "]]", "", 1)
		aliasFormat = strings.Replace(aliasFormat, "$", "", 1)

		if aliasFormat == "m" || aliasFormat == "measurement" {
			return []byte(query.Measurement)
		}
		if aliasFormat == "col" {
			return []byte(column)
		}

		pos, err := strconv.Atoi(aliasFormat)
		if err == nil && len(nameSegment) >= pos {
			return []byte(nameSegment[pos])
		}

		if !strings.HasPrefix(aliasFormat, "tag_") {
			return in
		}

		tagKey := strings.Replace(aliasFormat, "tag_", "", 1)
		tagValue, exist := row.Tags[tagKey]
		if exist {
			return []byte(tagValue)
		}

		return in
	})

	return string(result)
}

func (rp *ResponseParser) buildSerieNameFromQuery(row Row, column string) string {
	/*var tags []string

	for k, v := range row.Tags {
		tags = append(tags, fmt.Sprintf("%s: %s", k, v))
	}

	tagText := ""
	if len(tags) > 0 {
		tagText = fmt.Sprintf(" { %s }", strings.Join(tags, " "))
	}

	return fmt.Sprintf("%s.%s%s", row.Name, column, tagText)*/
	return fmt.Sprintf("%s.%s", row.Name, column)
}

func (rp *ResponseParser) parseTimepoint(valuePair []interface{}, valuePosition int) (tsdb.TimePoint, error) {
	var value *float64 = rp.parseValue(valuePair[valuePosition])

	timestampNumber, _ := valuePair[0].(json.Number)
	timestamp, err := timestampNumber.Float64()
	if err != nil {
		return tsdb.TimePoint{}, err
	}

	return tsdb.NewTimePoint(value, timestamp), nil
}

func (rp *ResponseParser) parseTimepointV2(valuePair []interface{}) (tsdb.TimePoint, error) {
	timepoint := make(tsdb.TimePoint, 0)
	for i := 1; i < len(valuePair); i++ {
		timepoint = append(timepoint, rp.parseValueV2(valuePair[i]))
	}
	timestampNumber, _ := valuePair[0].(json.Number)
	timestamp, err := timestampNumber.Float64()
	if err != nil {
		return tsdb.TimePoint{}, err
	}
	timepoint = append(timepoint, timestamp)
	return timepoint, nil
}

func (rp *ResponseParser) parseValue(value interface{}) *float64 {
	number, ok := value.(json.Number)
	if !ok {
		return nil
	}

	fvalue, err := number.Float64()
	if err == nil {
		return &fvalue
	}

	ivalue, err := number.Int64()
	if err == nil {
		ret := float64(ivalue)
		return &ret
	}

	return nil
}

func (rp *ResponseParser) parseValueV2(value interface{}) interface{} {
	number, ok := value.(json.Number)
	if !ok {
		return value
	}

	fvalue, err := number.Float64()
	if err == nil {
		return &fvalue
	}

	ivalue, err := number.Int64()
	if err == nil {
		ret := float64(ivalue)
		return &ret
	}
	return number.String()
}
