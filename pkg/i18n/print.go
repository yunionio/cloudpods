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

package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"yunion.io/x/onecloud/locales"
)

func P(tag language.Tag, id string, a ...interface{}) string {
	p := message.NewPrinter(tag, message.Catalog(locales.Catalog))
	return p.Sprintf(id, a...)
}
