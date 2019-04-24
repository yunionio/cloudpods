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

package ucloud

// https://docs.ucloud.cn/api/summary/regionlist
var UCLOUD_REGION_NAMES = map[string]string{
	"cn-bj1":       "北京一",
	"cn-bj2":       "北京二",
	"cn-sh":        "上海金融云",
	"cn-sh2":       "上海二",
	"cn-gd":        "广州",
	"hk":           "香港",
	"us-ca":        "洛杉矶",
	"us-ws":        "华盛顿",
	"ge-fra":       "法兰克福",
	"th-bkk":       "曼谷",
	"kr-seoul":     "首尔",
	"sg":           "新加坡",
	"tw-tp":        "台北",
	"tw-kh":        "高雄",
	"jpn-tky":      "东京",
	"rus-mosc":     "莫斯科",
	"uae-dubai":    "迪拜",
	"idn-jakarta":  "雅加达",
	"ind-mumbai":   "孟买",
	"bra-saopaulo": "圣保罗",
	"uk-london":    "伦敦",
	"afr-nigeria":  "拉各斯",
	"vn-sng":       "胡志明市",
}

var UCLOUD_ZONE_NAMES = map[string]string{
	"cn-bj1-01":       "北京一可用区A",
	"cn-bj2-02":       "北京二可用区B",
	"cn-bj2-03":       "北京二可用区C",
	"cn-bj2-04":       "北京二可用区D",
	"cn-bj2-05":       "北京二可用区E",
	"cn-sh-01":        "上海一可用区A",
	"cn-sh2-01":       "上海二可用区A",
	"cn-sh2-02":       "上海二可用区B",
	"cn-gd-02":        "广州可用区B",
	"hk-01":           "香港可用区A",
	"hk-02":           "香港可用区B",
	"us-ca-01":        "洛杉矶可用区A",
	"us-ws-01":        "华盛顿可用区A",
	"ge-fra-01":       "法兰克福可用区A",
	"th-bkk-01":       "曼谷可用区A",
	"kr-seoul-01":     "首尔可用区A",
	"sg-01":           "新加坡可用区A",
	"tw-kh-01":        "高雄可用区A",
	"tw-tp-01":        "台北可用区A",
	"jpn-tky-01":      "东京可用区A",
	"rus-mosc-01":     "莫斯科可用区A",
	"uae-dubai-01":    "迪拜可用区A",
	"idn-jakarta-01":  "雅加达可用区A",
	"ind-mumbai-01":   "孟买可用区A",
	"bra-saopaulo-01": "圣保罗可用区A",
	"uk-london-01":    "伦敦可用区A",
	"afr-nigeria-01":  "拉各斯可用区A",
	"vn-sng-01":       "胡志明市可用区A",
}
