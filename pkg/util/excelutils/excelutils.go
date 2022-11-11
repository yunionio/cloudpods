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

package excelutils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	DEFAULT_SHEET = "Sheet1"
)

func decimalBaseMaxWidth(decNum int, base int) int {
	if decNum == 0 {
		return 1
	}
	width := 0
	for decNum > 0 {
		decNum = decNum / base
		width += 1
	}
	return width
}

func decimalBaseN(decNum int, base int, width int) (int, int) {
	b := 1
	for i := 0; i < width-1; i += 1 {
		decNum = decNum / base
		b = b * base
	}
	return decNum, b
}

func decimal2Base(decNum int, base int) []int {
	width := decimalBaseMaxWidth(decNum, base)
	ret := make([]int, width)
	for i := width; i > 0; i -= 1 {
		ith, divider := decimalBaseN(decNum, base, i)
		decNum -= ith * divider
		ret[width-i] = ith
	}
	return ret
}

func decimal2Alphabet(decNum int) string {
	var buf bytes.Buffer
	b26 := decimal2Base(decNum, 26)
	for i := 0; i < len(b26); i += 1 {
		if i == 0 && len(b26) > 1 {
			buf.WriteByte(byte('A' + b26[i] - 1))
		} else {
			buf.WriteByte(byte('A' + b26[i]))
		}
	}
	return buf.String()
}

func exportHeader(xlsx *excelize.File, texts []string, rowIndex int) {
	for i := 0; i < len(texts); i += 1 {
		cell := fmt.Sprintf("%s%d", decimal2Alphabet(i), rowIndex)
		xlsx.SetCellValue(DEFAULT_SHEET, cell, texts[i])
	}
}

func exportRow(xlsx *excelize.File, data jsonutils.JSONObject, keys []string, rowIndex int) {
	for i := 0; i < len(keys); i += 1 {
		var valStr string
		var val jsonutils.JSONObject
		if strings.Contains(keys[i], ".") {
			val, _ = data.GetIgnoreCases(strings.Split(keys[i], ".")...)
		} else {
			val, _ = data.GetIgnoreCases(keys[i])
		}
		if val != nil {
			// hack, make floating point number prettier
			if fval, ok := val.(*jsonutils.JSONFloat); ok {
				f, _ := fval.Float()
				valStr = stringutils2.PrettyFloat(f, 2)
			} else {
				valStr, _ = val.GetString()
			}
		}
		cell := fmt.Sprintf("%s%d", decimal2Alphabet(i), rowIndex)
		xlsx.SetCellValue(DEFAULT_SHEET, cell, valStr)
	}
}

func Export(data []jsonutils.JSONObject, keys []string, texts []string, writer io.Writer) error {
	xlsx := excelize.NewFile()

	exportHeader(xlsx, texts, 1)
	for i := 0; i < len(data); i += 1 {
		exportRow(xlsx, data[i], keys, i+2)
	}

	return xlsx.Write(writer)
}

func ExportFile(data []jsonutils.JSONObject, keys []string, texts []string, filename string) error {
	writer, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer writer.Close()

	return Export(data, keys, texts, writer)
}
