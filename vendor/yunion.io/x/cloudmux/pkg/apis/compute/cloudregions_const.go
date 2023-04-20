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

package compute

const (
	CLOUD_REGION_STATUS_INSERVER     = "inservice"
	CLOUD_REGION_STATUS_OUTOFSERVICE = "outofservice"

	// 中国
	CITY_QING_DAO       = "Qingdao"      //青岛
	CITY_BEI_JING       = "Beijing"      //北京
	CITY_ZHANG_JIA_KOU  = "Zhangjiakou"  //张家口
	CITY_HU_HE_HAO_TE   = "Huhehaote"    //呼和浩特
	CITY_HANG_ZHOU      = "Hangzhou"     //杭州
	CITY_SHANG_HAI      = "Shanghai"     //上海
	CITY_SHEN_ZHEN      = "Shenzhen"     //深圳
	CITY_HONG_KONG      = "Hongkong"     //香港
	CITY_NING_XIA       = "Ningxia"      //宁夏
	CITY_GUANG_ZHOU     = "Guangzhou"    //广州
	CITY_TAI_WAN        = "Taiwan"       //台湾
	CITY_GUI_YANG       = "Guiyang"      //贵阳
	CITY_TAIPEI         = "Taipei"       //台北市
	CITY_KAOHSIUNG      = "Kaohsiung"    //高雄市
	CITY_CHENG_DU       = "Chengdu"      //成都
	CITY_HE_YUAN        = "HeYuan"       //河源
	CITY_CHONG_QING     = "Chongqing"    //重庆
	CITY_LAN_ZHOU       = "Lanzhou"      //兰州
	CITY_TAI_YUAN       = "Taiyuan"      //太原
	CITY_TIAN_JIN       = "Tianjin"      //天津
	CITY_WU_LU_MU_QI    = "Wulumuqi"     //乌鲁木齐
	CITY_NAN_NING       = "Nanning"      //南宁
	CITY_ZHENG_ZHOU     = "Zhengzhou"    //郑州
	CITY_KUN_MING       = "Kunming"      //昆明
	CITY_XI_AN          = "Xian"         //西安
	CITY_HAI_KOU        = "Haikou"       //海口
	CITY_WU_HU          = "Wuhu"         //芜湖
	CITY_FU_ZHOU        = "Fuzhou"       //福州
	CITY_WU_HAN         = "Wuhan"        //武汉
	CITY_CHANG_SHA      = "Changsha"     //长沙
	CITY_SU_ZHOU        = "Suzhou"       //苏州
	CITY_BAO_DING       = "Baoding"      //保定
	CITY_NAN_JING       = "Nanjing"      //南京
	CITY_FO_SHAN        = "Foshan"       //佛山
	CITY_QUAN_ZHOU      = "Quanzhou"     //泉州
	CITY_NEI_MENG_GU    = "Neimenggu"    //内蒙古
	CITY_WU_LAN_CHA_BU  = "Wulanchabu"   //乌兰察布
	CITY_JI_NAM         = "JiNan"        //济南
	CITY_CHANG_CHUN     = "ChangChun"    //长春
	CITY_XIANG_YANG     = "XiangYang"    //襄阳
	CITY_NAN_CHANG      = "NanChang"     //南昌
	CITY_SHEN_YANG      = "ShenYang"     //沈阳
	CITY_SHI_JIA_ZHUANG = "ShiJiaZhuang" //石家庄
	CITY_XIA_MEN        = "XiaMen"       //厦门
	CITY_HUAI_NAN       = "HuaiNan"      //淮南
	CITY_SU_QIAN        = "SuQian"       //宿迁
	CITY_DA_LIAN        = "Dalian"       //大连

	// 日本
	CITY_TOKYO = "Tokyo" //东京
	CITY_OSAKA = "Osaka" //大阪市

	// 新加坡
	CITY_SINGAPORE = "Singapore" //新加坡

	// 澳大利亚
	CITY_SYDNEY     = "Sydney"     //悉尼
	CITY_YARRALUMLA = "Yarralumla" //亚拉伦拉
	CITY_MELBOURNE  = "Melbourne"  //墨尔本

	//芬兰
	CITY_FINLAND = "Finland"

	//比利时
	CITY_BELGIUM = "Belgium" //比利时

	//瑞士
	CITY_ZURICH = "Zurich" //苏黎世
	CITY_GENEVA = "Geneva" //日内瓦

	// 挪威
	CITY_STAVANGER = "Stavanger" //斯塔万格
	CITY_OSLO      = "Oslo"      // 奥斯陆

	// 马来西亚
	CITY_KUALA_LUMPUR = "Kuala Lumpur" //吉隆坡

	// 印度尼西亚
	CITY_JAKARTA = "Jakarta" //雅加达

	// 印度
	CITY_DELHI       = "Delhi"       // 德里
	CITY_MUMBAI      = "Mumbai"      // 孟买
	CITY_KANCHIPURAM = "Kanchipuram" // 甘吉布勒姆
	CITY_MAHARASHTRA = "Maharashtra" // 马哈拉施特拉邦
	CITY_JAMNAGAR    = "Jamnagar"    // 贾姆讷格尔
	CITY_NAGPUR      = "Nagpur"      // 那格浦尔
	CITY_HYDERABAD   = "Hyderabad"   // 海得拉巴

	// 美国
	CITY_VIRGINIA       = "Virginia"       //弗吉尼亚
	CITY_SILICONVALLEY  = "Siliconvalley"  //硅谷
	CITY_OHIO           = "Ohio"           //俄亥俄州
	CITY_N_VIRGINIA     = "N. Virginia"    //北弗吉尼亚
	CITY_N_CALIFORNIA   = "N. California"  //北加州
	CITY_OREGON         = "Oregon"         //俄勒冈州
	CITY_LOS_ANGELES    = "Los Angeles"    //洛杉矶
	CITY_SAN_FRANCISCO  = "San Francisco"  //旧金山
	CITY_UTAH           = "Utah"           //犹他州
	CITY_WASHINGTON     = "Washington"     //华盛顿
	CITY_TEXAS          = "Texas"          //德克萨斯
	CITY_CHICAGO        = "Chicago"        //芝加哥
	CITY_IOWA           = "Iowa"           //爱荷华
	CITY_US_GOV_WEST    = "us-gov-west"    //???
	CITY_SOUTH_CAROLINA = "South Carolina" //南卡罗来纳州
	CITY_SALT_LAKE_CITY = "Salt Lake City" //盐湖城
	CITY_LAS_VEGAS      = "Las Vegas"      //拉斯维加斯
	CITY_PHOENIX        = "Phoenix"        //菲尼克斯
	CITY_DALLAS         = "Dallas"         //达拉斯
	CITY_COLUMBUS       = "Columbus"       //哥伦布

	// 英国
	CITY_LONDON      = "London"      //伦敦
	CITY_HALTON      = "Halton"      //哈尔顿
	CITY_WEST_SUSSEX = "West Sussex" //西苏塞克斯

	// 阿拉伯联合酋长国
	CITY_DUBAI     = "Dubai"     //迪拜
	CITY_ABU_DHABI = "Abu Dhabi" // 阿布扎比

	// 德国
	CITY_FRANKFURT   = "Frankfurt"   // 法兰克福
	CITY_DELMENHORST = "Delmenhorst" // 代尔门霍斯特

	// 韩国
	CITY_SEOUL = "Seoul" //首尔
	CITY_BUSAN = "Busan" //釜山

	// 加拿大
	CITY_CANADA_CENTRAL = "Canada Central" //加拿大中部
	CITY_QUEBEC         = "Quebec"         //魁北克市
	CITY_TORONTO        = "Toronto"        //多伦多
	CITY_MONTREAL       = "Montreal"       //蒙特利尔

	// 爱尔兰
	CITY_IRELAND = "Ireland" //爱尔兰
	CITY_DUBLIN  = "Dublin"  //都柏林

	// 法国
	CITY_PARIS  = "Paris"  //巴黎
	CITY_ALLIER = "Allier" //阿利埃河
	CITY_TARN   = "Tarn"   //塔恩

	// 瑞典
	CITY_STOCKHOLM = "Stockholm" //斯德哥尔摩

	// 波兰
	CITY_WARSAW = "Warsaw" // 华沙

	// 巴西
	CITY_SAO_PAULO      = "Sao Paulo"      //圣保罗
	CITY_RIO_DE_JANEIRO = "Rio de Janeiro" // 里约热内卢

	// 智利
	CITY_SANTIAGO = "Santiago" // 圣地亚哥

	// 墨西哥
	CITY_MEXICO = "Mexico" // 墨西哥

	// 荷兰
	CITY_HOLLAND = "Holland" //荷兰

	// 南非
	CITY_PRETORIA     = "Pretoria"     //比勒陀利亚
	CITY_CAPE_TOWN    = "Cape Town"    //开普敦
	CITY_JOHANNESBURG = "Johannesburg" //约翰内斯堡

	// 泰国
	CITY_BANGKOK = "Bangkok" //曼谷

	// 俄罗斯
	CITY_MOSCOW = "Moscow" //莫斯科

	// 尼日利亚
	CITY_LAGOS = "Lagos" //拉哥斯

	// 巴林王国 (中东国家)
	CITY_BAHRAIN = "Bahrain" // 巴林

	// 越南
	CITY_HO_CHI_MINH = "Ho Chi Minh" //???

	// 以色列
	CITY_TEL_AVIV = "Tel Aviv" // 拉斯维夫

	// 意大利
	CITY_MILAN = "Milan" // 米兰

	// 西班牙
	CITY_MADRID = "Madrid" // 马德里

	COUNTRY_CODE_CN = "CN" //中国
	COUNTRY_CODE_JP = "JP" //日本
	COUNTRY_CODE_SG = "SG" //新加坡
	COUNTRY_CODE_AU = "AU" //澳大利亚
	COUNTRY_CODE_MY = "MY" //马来西亚
	COUNTRY_CODE_ID = "ID" //印度尼西亚
	COUNTRY_CODE_IN = "IN" //印度
	COUNTRY_CODE_US = "US" //美国
	COUNTRY_CODE_GB = "GB" //英国
	COUNTRY_CODE_AE = "AE" //阿拉伯联合酋长国
	COUNTRY_CODE_DE = "DE" //德国
	COUNTRY_CODE_KR = "KR" //韩国
	COUNTRY_CODE_CA = "CA" //加拿大
	COUNTRY_CODE_IE = "IE" //爱尔兰
	COUNTRY_CODE_FR = "FR" //法国
	COUNTRY_CODE_SE = "SE" //瑞典
	COUNTRY_CODE_BR = "BR" //巴西
	COUNTRY_CODE_NL = "NL" //荷兰
	COUNTRY_CODE_ZA = "ZA" //南非
	COUNTRY_CODE_TH = "TH" //泰国
	COUNTRY_CODE_RU = "RU" //俄罗斯
	COUNTRY_CODE_NG = "NG" //尼日利亚
	COUNTRY_CODE_VN = "VN" //越南
	COUNTRY_CODE_CH = "CH" //瑞士
	COUNTRY_CODE_NO = "NO" //挪威
	COUNTRY_CODE_MX = "MX" //墨西哥
	COUNTRY_CODE_CL = "CL" //智利
	COUNTRY_CODE_BH = "BH" //巴林
	COUNTRY_CODE_PL = "PL" //波兰
	COUNTRY_CODE_FI = "FI" //芬兰
	COUNTRY_CODE_BE = "BE" //比利时
	COUNTRY_CODE_IL = "IL" //以色列
	COUNTRY_CODE_IT = "IT" //意大利
	COUNTRY_CODE_ES = "ES" //西班牙
)
