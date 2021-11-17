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

package sqlchemy

const (
	// SQL_OP_AND represents AND operator
	SQL_OP_AND = "AND"
	// SQL_OP_OR represents OR operator
	SQL_OP_OR = "OR"
	// SQL_OP_NOT represents NOT operator
	SQL_OP_NOT = "NOT"
	// SQL_OP_LIKE represents LIKE operator
	SQL_OP_LIKE = "LIKE"
	// SQL_OP_IN represents IN operator
	SQL_OP_IN = "IN"
	// SQL_OP_NOTIN represents NOT IN operator
	SQL_OP_NOTIN = "NOT IN"
	// SQL_OP_EQUAL represents EQUAL operator
	SQL_OP_EQUAL = "="
	// SQL_OP_LT represents < operator
	SQL_OP_LT = "<"
	// SQL_OP_LE represents <= operator
	SQL_OP_LE = "<="
	// SQL_OP_GT represents > operator
	SQL_OP_GT = ">"
	// SQL_OP_GE represents >= operator
	SQL_OP_GE = ">="
	// SQL_OP_BETWEEN represents BETWEEN operator
	SQL_OP_BETWEEN = "BETWEEN"
	// SQL_OP_NOTEQUAL represents NOT EQUAL operator
	SQL_OP_NOTEQUAL = "<>"
)

const (
	// TAG_IGNORE is a field tag that indicates the field is ignored, not represents a table column
	TAG_IGNORE = "ignore"
	// TAG_NAME is a field tag that indicates the column name of this field
	TAG_NAME = "name"
	// TAG_WIDTH is a field tag that indicates the width of the column, like VARCHAR(15)
	// Supported by: mysql
	TAG_WIDTH = "width"
	// TAG_TEXT_LENGTH is a field tag that indicates the length of a text column
	// Supported by: mysql
	TAG_TEXT_LENGTH = "length"
	// TAG_CHARSET is a field tag that indicates the charset of a text column
	// Supported by: mysql
	TAG_CHARSET = "charset"
	// TAG_PRECISION is a field tag that indicates the precision of a float column
	TAG_PRECISION = "precision"
	// TAG_DEFAULT is a field tag that indicates the default value of a column
	TAG_DEFAULT = "default"
	// TAG_UNIQUE is a field tag that indicates the column value is unique
	TAG_UNIQUE = "unique"
	// TAG_INDEX is a field tag that indicates the column is a indexable column
	TAG_INDEX = "index"
	// TAG_PRIMARY is a field tag that indicates the column is part of primary key
	TAG_PRIMARY = "primary"
	// TAG_NULLABLE is a field tag that indicates the column is nullable
	TAG_NULLABLE = "nullable"
	// TAG_AUTOINCREMENT is a field tag that indicates the integer column is auto_increment, the column should must be primary
	TAG_AUTOINCREMENT = "auto_increment"
	// TAG_AUTOVERSION is a field tag that indicates the integer column is used to records the update version of a record
	TAG_AUTOVERSION = "auto_version"
	// TAG_UPDATE_TIMESTAMP is a field tag that indicates the datetime column is the updated_at timestamp
	TAG_UPDATE_TIMESTAMP = "updated_at"
	// TAG_CREATE_TIMESTAMP is a field tag that indicates the datetime column is the created_at timestamp
	TAG_CREATE_TIMESTAMP = "created_at"
	// TAG_ALLOW_ZERO is a field tag that indicates whether the column allow zero value
	TAG_ALLOW_ZERO = "allow_zero"
)
