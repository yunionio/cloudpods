// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package promql

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/influxdata/promql/v2/pkg/labels"
)

// Value is a generic interface for values resulting from a query evaluation.
type Value interface {
	Type() ValueType
	String() string
}

func (Matrix) Type() ValueType { return ValueTypeMatrix }
func (Vector) Type() ValueType { return ValueTypeVector }
func (Scalar) Type() ValueType { return ValueTypeScalar }
func (String) Type() ValueType { return ValueTypeString }

// ValueType describes a type of a value.
type ValueType string

// The valid value types.
const (
	ValueTypeNone   = "none"
	ValueTypeVector = "vector"
	ValueTypeScalar = "scalar"
	ValueTypeMatrix = "matrix"
	ValueTypeString = "string"
)

// String represents a string value.
type String struct {
	T int64
	V string
}

func (s String) String() string {
	return s.V
}

func (s String) MarshalJSON() ([]byte, error) {
	return json.Marshal([...]interface{}{float64(s.T) / 1000, s.V})
}

// Scalar is a data point that's explicitly not associated with a metric.
type Scalar struct {
	T int64
	V float64
}

func (s Scalar) String() string {
	v := strconv.FormatFloat(s.V, 'f', -1, 64)
	return fmt.Sprintf("scalar: %v @[%v]", v, s.T)
}

func (s Scalar) MarshalJSON() ([]byte, error) {
	v := strconv.FormatFloat(s.V, 'f', -1, 64)
	return json.Marshal([...]interface{}{float64(s.T) / 1000, v})
}

// Series is a stream of data points belonging to a metric.
type Series struct {
	Metric labels.Labels `json:"metric"`
	Points []Point       `json:"values"`
}

func (s Series) String() string {
	vals := make([]string, len(s.Points))
	for i, v := range s.Points {
		vals[i] = v.String()
	}
	return fmt.Sprintf("%s =>\n%s", s.Metric, strings.Join(vals, "\n"))
}

// Point represents a single data point for a given timestamp.
type Point struct {
	T int64
	V float64
}

func (p Point) String() string {
	v := strconv.FormatFloat(p.V, 'f', -1, 64)
	return fmt.Sprintf("%v @[%v]", v, p.T)
}

// MarshalJSON implements json.Marshaler.
func (p Point) MarshalJSON() ([]byte, error) {
	v := strconv.FormatFloat(p.V, 'f', -1, 64)
	return json.Marshal([...]interface{}{float64(p.T) / 1000, v})
}

// Sample is a single sample belonging to a metric.
type Sample struct {
	Point

	Metric labels.Labels
}

func (s Sample) String() string {
	return fmt.Sprintf("%s => %s", s.Metric, s.Point)
}

func (s Sample) MarshalJSON() ([]byte, error) {
	v := struct {
		M labels.Labels `json:"metric"`
		V Point         `json:"value"`
	}{
		M: s.Metric,
		V: s.Point,
	}
	return json.Marshal(v)
}

// Vector is basically only an alias for model.Samples, but the
// contract is that in a Vector, all Samples have the same timestamp.
type Vector []Sample

func (vec Vector) String() string {
	entries := make([]string, len(vec))
	for i, s := range vec {
		entries[i] = s.String()
	}
	return strings.Join(entries, "\n")
}

// ContainsSameLabelset checks if a vector has samples with the same labelset
// Such a behavior is semantically undefined
// https://github.com/prometheus/prometheus/issues/4562
func (vec Vector) ContainsSameLabelset() bool {
	l := make(map[uint64]struct{}, len(vec))
	for _, s := range vec {
		hash := s.Metric.Hash()
		if _, ok := l[hash]; ok {
			return true
		}
		l[hash] = struct{}{}
	}
	return false
}

// Matrix is a slice of Series that implements sort.Interface and
// has a String method.
type Matrix []Series

func (m Matrix) String() string {
	// TODO(fabxc): sort, or can we rely on order from the querier?
	strs := make([]string, len(m))

	for i, ss := range m {
		strs[i] = ss.String()
	}

	return strings.Join(strs, "\n")
}

// TotalSamples returns the total number of samples in the series within a matrix.
func (m Matrix) TotalSamples() int {
	numSamples := 0
	for _, series := range m {
		numSamples += len(series.Points)
	}
	return numSamples
}

func (m Matrix) Len() int           { return len(m) }
func (m Matrix) Less(i, j int) bool { return labels.Compare(m[i].Metric, m[j].Metric) < 0 }
func (m Matrix) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

// ContainsSameLabelset checks if a matrix has samples with the same labelset
// Such a behavior is semantically undefined
// https://github.com/prometheus/prometheus/issues/4562
func (m Matrix) ContainsSameLabelset() bool {
	l := make(map[uint64]struct{}, len(m))
	for _, ss := range m {
		hash := ss.Metric.Hash()
		if _, ok := l[hash]; ok {
			return true
		}
		l[hash] = struct{}{}
	}
	return false
}
