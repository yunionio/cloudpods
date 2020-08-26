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

package jsonutils

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/util/regutils"
)

func normalizeUSCurrency(currency string) string {
	return strings.Replace(currency, ",", "", -1)
}

func normalizeEUCurrency(currency string) string {
	commaPos := strings.IndexByte(currency, ',')
	if commaPos >= 0 {
		return fmt.Sprintf("%s.%s", strings.Replace(currency[:commaPos], ".", "", -1), currency[commaPos+1:])
	} else {
		return strings.Replace(currency, ".", "", -1)
	}
}

func normalizeCurrencyString(currency string) string {
	if regutils.MatchUSCurrency(currency) {
		return normalizeUSCurrency(currency)
	}
	if regutils.MatchEUCurrency(currency) {
		return normalizeEUCurrency(currency)
	}
	return currency
}
