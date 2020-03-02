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

package expressions

type PrimitveType string

const (
	Bool     PrimitveType = "Bool"
	DateTime PrimitveType = "DateTime"
	Double   PrimitveType = "Double"
	String   PrimitveType = "String"
	Null     PrimitveType = "NULL"
)

/*type ConstExp struct {
	Bool bool
	DateTime DateTime
	Double Double
}*/

type ConstExp interface{}

type PropertyExp struct {
	Property string       `json:"property"`
	Type     PrimitveType `json:"type"`
}

type PrimitiveObject struct {
	PropertyExp
	ConstExp
}

type OperatorExp struct {
	Left  *PropertyExp     `json:"left"`
	Right *PrimitiveObject `json:"right"`
}

type LogicalExp struct {
	EQ  *OperatorExp  `json:"eq"`
	IN  *OperatorExp  `json:"in"`
	LT  *OperatorExp  `json:"lt"`
	GT  *OperatorExp  `json:"gt"`
	AND []*LogicalExp `json:"and"`
	OR  []*LogicalExp `json:"or"`
	NOT *LogicalExp   `json:"not"`
}

type ArithmeticExp struct {
	ADD *OperatorExp `json:"add"`
	SUB *OperatorExp `json:"sub"`
}

type FilterExp struct {
	LogicalExp
}

type AlignerExp struct {
	Input *PropertyExp `json:"input"`
}

type MeasureExp struct {
	Mean *AlignerExp `json:"mean"`
	Min  *AlignerExp `json:"min"`
}

type AggregateExp struct {
	MeasureExps []MeasureExp
}
