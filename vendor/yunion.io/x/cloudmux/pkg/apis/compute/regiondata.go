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

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var (
	// China
	RegionBeijing = cloudprovider.SGeographicInfo{
		Latitude:    39.90419989999999,
		Longitude:   116.4073963,
		City:        CITY_BEI_JING,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionNingxia = cloudprovider.SGeographicInfo{
		Latitude:    37.198731,
		Longitude:   106.1580937,
		City:        CITY_NING_XIA,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionShanghai = cloudprovider.SGeographicInfo{
		Latitude:    31.17,
		Longitude:   121.47,
		City:        CITY_SHANG_HAI,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionGuangzhou = cloudprovider.SGeographicInfo{
		Latitude:    23.12911,
		Longitude:   113.264385,
		City:        CITY_GUANG_ZHOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionDalian = cloudprovider.SGeographicInfo{
		Latitude:    38.91400300000001,
		Longitude:   121.614682,
		City:        CITY_DA_LIAN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionGuiyang = cloudprovider.SGeographicInfo{
		Latitude:    26.6470035286,
		Longitude:   106.6302113880,
		City:        CITY_GUI_YANG,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionNeimenggu = cloudprovider.SGeographicInfo{
		Latitude:    41.0178713,
		Longitude:   113.094978,
		City:        CITY_NEI_MENG_GU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionQingdao = cloudprovider.SGeographicInfo{
		Latitude:    36.067108,
		Longitude:   120.382607,
		City:        CITY_QING_DAO,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionZhangjiakou = cloudprovider.SGeographicInfo{
		Latitude:    40.767544,
		Longitude:   114.886337,
		City:        CITY_ZHANG_JIA_KOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHuhehaote = cloudprovider.SGeographicInfo{
		Latitude:    40.842358,
		Longitude:   111.749992,
		City:        CITY_HU_HE_HAO_TE,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHangzhou = cloudprovider.SGeographicInfo{
		Latitude:    30.274084,
		Longitude:   120.155067,
		City:        CITY_HANG_ZHOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionShenzhen = cloudprovider.SGeographicInfo{
		Latitude:    22.543097,
		Longitude:   114.057861,
		City:        CITY_SHEN_ZHEN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionChengdu = cloudprovider.SGeographicInfo{
		Latitude:    30.572815,
		Longitude:   104.066803,
		City:        CITY_CHENG_DU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHeyuan = cloudprovider.SGeographicInfo{
		Latitude:    23.7292717,
		Longitude:   114.6965786,
		City:        CITY_HE_YUAN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionWulanchabu = cloudprovider.SGeographicInfo{
		Latitude:    41.0178065,
		Longitude:   113.094978,
		City:        CITY_WU_LAN_CHA_BU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionChongqing = cloudprovider.SGeographicInfo{
		Latitude:    29.431585,
		Longitude:   106.912254,
		City:        CITY_CHONG_QING,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionNanjing = cloudprovider.SGeographicInfo{
		Latitude:    32.0584065670,
		Longitude:   118.7964897811,
		City:        CITY_NAN_JING,
		CountryCode: COUNTRY_CODE_CN,
	}

	// Asia
	RegionTaiwan = cloudprovider.SGeographicInfo{
		Latitude:    25.0443,
		Longitude:   121.509,
		City:        CITY_TAI_WAN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHongkong = cloudprovider.SGeographicInfo{
		Latitude:    22.396427,
		Longitude:   114.109497,
		City:        CITY_HONG_KONG,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionTokyo = cloudprovider.SGeographicInfo{
		Latitude:    35.709026,
		Longitude:   139.731995,
		City:        CITY_TOKYO,
		CountryCode: COUNTRY_CODE_JP,
	}
	RegionOsaka = cloudprovider.SGeographicInfo{
		Latitude:    34.6937378,
		Longitude:   135.5021651,
		City:        CITY_OSAKA,
		CountryCode: COUNTRY_CODE_JP,
	}
	RegionSeoul = cloudprovider.SGeographicInfo{
		Latitude:    34.6937378,
		Longitude:   135.5021651,
		City:        CITY_SEOUL,
		CountryCode: COUNTRY_CODE_KR,
	}
	RegionBusan = cloudprovider.SGeographicInfo{
		Latitude:    35.1,
		Longitude:   129.0403,
		City:        CITY_BUSAN,
		CountryCode: COUNTRY_CODE_KR,
	}

	RegionMumbai = cloudprovider.SGeographicInfo{
		Latitude:    19.07599,
		Longitude:   72.877393,
		City:        CITY_MUMBAI,
		CountryCode: COUNTRY_CODE_IN,
	}
	// 德里
	RegionDelhi = cloudprovider.SGeographicInfo{
		Latitude:    28.61,
		Longitude:   77.23,
		City:        CITY_DELHI,
		CountryCode: COUNTRY_CODE_IN,
	}
	RegionMaharashtra = cloudprovider.SGeographicInfo{
		Latitude:    18.97,
		Longitude:   72.82,
		City:        CITY_MAHARASHTRA,
		CountryCode: COUNTRY_CODE_IN,
	}
	RegionKanchipuram = cloudprovider.SGeographicInfo{
		Latitude:    12.8387,
		Longitude:   79.7016,
		City:        CITY_KANCHIPURAM,
		CountryCode: COUNTRY_CODE_IN,
	}
	RegionHyderabad = cloudprovider.SGeographicInfo{
		Latitude:    17.4123318,
		Longitude:   78.3256457,
		City:        CITY_HYDERABAD,
		CountryCode: COUNTRY_CODE_IN,
	}

	RegionSingapore = cloudprovider.SGeographicInfo{
		Latitude:    1.352083,
		Longitude:   103.819839,
		City:        CITY_SINGAPORE,
		CountryCode: COUNTRY_CODE_SG,
	}
	// 雅加达
	RegionJakarta = cloudprovider.SGeographicInfo{
		Latitude:    -6.175110,
		Longitude:   106.865036,
		City:        CITY_JAKARTA,
		CountryCode: COUNTRY_CODE_ID,
	}
	RegionKualaLumpur = cloudprovider.SGeographicInfo{
		Latitude:    3.139003,
		Longitude:   101.686852,
		City:        CITY_KUALA_LUMPUR,
		CountryCode: COUNTRY_CODE_MY,
	}
	RegionBangkok = cloudprovider.SGeographicInfo{
		Latitude:    13.7563309,
		Longitude:   100.5017651,
		City:        CITY_BANGKOK,
		CountryCode: COUNTRY_CODE_TH,
	}

	RegionSydney = cloudprovider.SGeographicInfo{
		Latitude:    -33.8688197,
		Longitude:   151.2092955,
		City:        CITY_SYDNEY,
		CountryCode: COUNTRY_CODE_AU,
	}
	// 墨尔本
	RegionMelbourne = cloudprovider.SGeographicInfo{
		Latitude:    -37.813611,
		Longitude:   144.963056,
		City:        CITY_MELBOURNE,
		CountryCode: COUNTRY_CODE_AU,
	}
	//亚拉伦拉 澳大利亚
	RegionYarralumla = cloudprovider.SGeographicInfo{
		Latitude:    -35.3016,
		Longitude:   149.078,
		City:        CITY_YARRALUMLA,
		CountryCode: COUNTRY_CODE_AU,
	}

	// Africa
	RegionCapeTown = cloudprovider.SGeographicInfo{
		Latitude:    -33.9152209,
		Longitude:   18.3758904,
		City:        CITY_CAPE_TOWN,
		CountryCode: COUNTRY_CODE_ZA,
	}
	// 比勒陀利亚
	RegionPretoria = cloudprovider.SGeographicInfo{
		Latitude:    -25.716667,
		Longitude:   28.283333,
		City:        CITY_PRETORIA,
		CountryCode: COUNTRY_CODE_ZA,
	}
	RegionJohannesburg = cloudprovider.SGeographicInfo{
		Latitude:    -26.1714537,
		Longitude:   27.8999389,
		City:        CITY_JOHANNESBURG,
		CountryCode: COUNTRY_CODE_ZA,
	}

	// Middleeast
	RegionBahrain = cloudprovider.SGeographicInfo{
		Latitude:    25.9408826,
		Longitude:   50.4474957,
		City:        CITY_BAHRAIN,
		CountryCode: COUNTRY_CODE_BH,
	}
	// 迪拜
	RegionDubai = cloudprovider.SGeographicInfo{
		Latitude:    25.263056,
		Longitude:   55.297222,
		City:        CITY_DUBAI,
		CountryCode: COUNTRY_CODE_AE,
	}
	// 阿布扎比
	RegionAbuDhabi = cloudprovider.SGeographicInfo{
		Latitude:    24.3867414,
		Longitude:   54.3938162,
		City:        CITY_ABU_DHABI,
		CountryCode: COUNTRY_CODE_AE,
	}

	// Europe
	RegionFinland = cloudprovider.SGeographicInfo{
		Latitude:    64.8255731,
		Longitude:   21.5432837,
		City:        CITY_FINLAND,
		CountryCode: COUNTRY_CODE_FI,
	}
	RegionBelgium = cloudprovider.SGeographicInfo{
		Latitude:    50.499734,
		Longitude:   3.9057517,
		City:        CITY_BELGIUM,
		CountryCode: COUNTRY_CODE_BE,
	}
	RegionLondon = cloudprovider.SGeographicInfo{
		Latitude:    51.507351,
		Longitude:   -0.127758,
		City:        CITY_LONDON,
		CountryCode: COUNTRY_CODE_GB,
	}
	RegionHalton = cloudprovider.SGeographicInfo{
		Latitude:    53.3331,
		Longitude:   -2.6957,
		City:        CITY_HALTON,
		CountryCode: COUNTRY_CODE_GB,
	}
	RegionSussex = cloudprovider.SGeographicInfo{
		Latitude:    51,
		Longitude:   0,
		City:        CITY_WEST_SUSSEX,
		CountryCode: COUNTRY_CODE_GB,
	}
	RegionFrankfurt = cloudprovider.SGeographicInfo{
		Latitude:    51.165691,
		Longitude:   10.451526,
		City:        CITY_FRANKFURT,
		CountryCode: COUNTRY_CODE_DE,
	}
	RegionHolland = cloudprovider.SGeographicInfo{
		Latitude:    52.2076831,
		Longitude:   4.1585786,
		City:        CITY_HOLLAND,
		CountryCode: COUNTRY_CODE_NL,
	}
	RegionZurich = cloudprovider.SGeographicInfo{
		Latitude:    47.3774497,
		Longitude:   8.5016958,
		City:        CITY_ZURICH,
		CountryCode: COUNTRY_CODE_CH,
	}
	RegionMilan = cloudprovider.SGeographicInfo{
		Latitude:    45.4627124,
		Longitude:   9.1076929,
		City:        CITY_MILAN,
		CountryCode: COUNTRY_CODE_IT,
	}
	RegionWarsaw = cloudprovider.SGeographicInfo{
		Latitude:    52.233333,
		Longitude:   21.016667,
		City:        CITY_WARSAW,
		CountryCode: COUNTRY_CODE_PL,
	}
	RegionMadrid = cloudprovider.SGeographicInfo{
		Latitude:    40.4378698,
		Longitude:   -3.8196188,
		City:        CITY_MADRID,
		CountryCode: COUNTRY_CODE_ES,
	}
	RegionIreland = cloudprovider.SGeographicInfo{
		Latitude:    53.41291,
		Longitude:   -8.24389,
		City:        CITY_IRELAND,
		CountryCode: COUNTRY_CODE_IE,
	}
	RegionDublin = cloudprovider.SGeographicInfo{
		Latitude:    53.349722,
		Longitude:   -6.260278,
		City:        CITY_DUBLIN,
		CountryCode: COUNTRY_CODE_IE,
	}
	RegionParis = cloudprovider.SGeographicInfo{
		Latitude:    48.856614,
		Longitude:   2.3522219,
		City:        CITY_PARIS,
		CountryCode: COUNTRY_CODE_FR,
	}
	RegionAllier = cloudprovider.SGeographicInfo{
		Latitude:    46.5178,
		Longitude:   3.3592,
		City:        CITY_ALLIER,
		CountryCode: COUNTRY_CODE_FR,
	}
	RegionTarn = cloudprovider.SGeographicInfo{
		Latitude:    43.8656,
		Longitude:   1.505,
		City:        CITY_TARN,
		CountryCode: COUNTRY_CODE_FR,
	}
	RegionStockholm = cloudprovider.SGeographicInfo{
		Latitude:    59.1946,
		Longitude:   18.47,
		City:        CITY_STOCKHOLM,
		CountryCode: COUNTRY_CODE_SE,
	}
	RegionDelmenhorst = cloudprovider.SGeographicInfo{
		Latitude:    53.050556,
		Longitude:   8.631667,
		City:        CITY_DELMENHORST,
		CountryCode: COUNTRY_CODE_DE,
	}
	RegionGeneva = cloudprovider.SGeographicInfo{
		Latitude:    46.201667,
		Longitude:   6.146944,
		City:        CITY_GENEVA,
		CountryCode: COUNTRY_CODE_CH,
	}
	RegionStavanger = cloudprovider.SGeographicInfo{
		Latitude:    58.963333,
		Longitude:   5.718889,
		City:        CITY_STAVANGER,
		CountryCode: COUNTRY_CODE_NO,
	}
	RegionOslo = cloudprovider.SGeographicInfo{
		Latitude:    59.905556,
		Longitude:   10.768333,
		City:        CITY_OSLO,
		CountryCode: COUNTRY_CODE_NO,
	}

	RegionMoscow = cloudprovider.SGeographicInfo{
		Latitude:    55.755825,
		Longitude:   37.617298,
		City:        CITY_MOSCOW,
		CountryCode: COUNTRY_CODE_RU,
	}

	// America
	RegionMontreal = cloudprovider.SGeographicInfo{
		Latitude:    45.5580206,
		Longitude:   -73.8003414,
		City:        CITY_MONTREAL,
		CountryCode: COUNTRY_CODE_CA,
	}
	RegionToronto = cloudprovider.SGeographicInfo{
		Latitude:    43.653225,
		Longitude:   -79.383186,
		City:        CITY_TORONTO,
		CountryCode: COUNTRY_CODE_CA,
	}
	RegionCanadaCentral = cloudprovider.SGeographicInfo{
		Latitude:    56.130366,
		Longitude:   -106.346771,
		City:        CITY_CANADA_CENTRAL,
		CountryCode: COUNTRY_CODE_CA,
	}
	RegionQuebec = cloudprovider.SGeographicInfo{
		Latitude:    52,
		Longitude:   -72,
		City:        CITY_QUEBEC,
		CountryCode: COUNTRY_CODE_CA,
	}

	RegionSaoPaulo = cloudprovider.SGeographicInfo{
		Latitude:    -23.5505199,
		Longitude:   -46.6333094,
		City:        CITY_SAO_PAULO,
		CountryCode: COUNTRY_CODE_BR,
	}
	// 里约热内卢
	RegionRioDeJaneiro = cloudprovider.SGeographicInfo{
		Latitude:    -22.9083,
		Longitude:   -43.1964,
		City:        CITY_RIO_DE_JANEIRO,
		CountryCode: COUNTRY_CODE_BR,
	}

	RegionMexico = cloudprovider.SGeographicInfo{
		Latitude:    55.1182908,
		Longitude:   141.0377645,
		City:        CITY_MEXICO,
		CountryCode: COUNTRY_CODE_MX,
	}

	RegionSantiago = cloudprovider.SGeographicInfo{
		Latitude:    -33.45206,
		Longitude:   -70.676031,
		City:        CITY_SANTIAGO,
		CountryCode: COUNTRY_CODE_CL,
	}

	RegionIowa = cloudprovider.SGeographicInfo{
		Latitude:    41.9328655,
		Longitude:   -94.5106809,
		City:        CITY_IOWA,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionCarolina = cloudprovider.SGeographicInfo{
		Latitude:    33.6194409,
		Longitude:   -82.0475635,
		City:        CITY_SOUTH_CAROLINA,
		CountryCode: COUNTRY_CODE_US,
	}
	// 西雅图，华盛顿州
	RegionWashington = cloudprovider.SGeographicInfo{
		Latitude:    47.6,
		Longitude:   -122.3,
		City:        CITY_WASHINGTON,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionVirginia = cloudprovider.SGeographicInfo{
		Latitude:    37.4315734,
		Longitude:   -78.6568942,
		City:        CITY_VIRGINIA,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionTexas = cloudprovider.SGeographicInfo{
		Latitude:    31,
		Longitude:   -100,
		City:        CITY_TEXAS,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionChicago = cloudprovider.SGeographicInfo{
		Latitude:    41.8819,
		Longitude:   -87.6278,
		City:        CITY_CHICAGO,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionPhoenix = cloudprovider.SGeographicInfo{
		Latitude:    33.4483,
		Longitude:   -112.0739,
		City:        CITY_PHOENIX,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionUtah = cloudprovider.SGeographicInfo{
		Latitude:    39.5,
		Longitude:   -111.5,
		City:        CITY_UTAH,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionSanFrancisco = cloudprovider.SGeographicInfo{
		Latitude:    37.7775,
		Longitude:   -122.4164,
		City:        CITY_SAN_FRANCISCO,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionSiliconValley = cloudprovider.SGeographicInfo{
		Latitude:    37.387474,
		Longitude:   -122.057541,
		City:        CITY_SILICONVALLEY,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionNothVirginia = cloudprovider.SGeographicInfo{
		Latitude:    37.4315734,
		Longitude:   -78.6568942,
		City:        CITY_N_VIRGINIA,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionOregon = cloudprovider.SGeographicInfo{
		Latitude:    43.8041334,
		Longitude:   -120.5542012,
		City:        CITY_OREGON,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionLosAngeles = cloudprovider.SGeographicInfo{
		Latitude:    34.0522342,
		Longitude:   -118.2436849,
		City:        CITY_LOS_ANGELES,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionSaltLakeCity = cloudprovider.SGeographicInfo{
		Latitude:    40.7767168,
		Longitude:   -111.9905243,
		City:        CITY_SALT_LAKE_CITY,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionLasVegas = cloudprovider.SGeographicInfo{
		Latitude:    36.1249185,
		Longitude:   -115.3150811,
		City:        CITY_LAS_VEGAS,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionDallas = cloudprovider.SGeographicInfo{
		Latitude:    32.7844251,
		Longitude:   -96.8913045,
		City:        CITY_DALLAS,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionColumbus = cloudprovider.SGeographicInfo{
		Latitude:    32.0879267,
		Longitude:   34.7622266,
		City:        CITY_TEL_AVIV,
		CountryCode: COUNTRY_CODE_IL,
	}
	RegionNorthCalifornia = cloudprovider.SGeographicInfo{
		Latitude:    38.8375215,
		Longitude:   -120.8958242,
		City:        CITY_N_CALIFORNIA,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionOhio = cloudprovider.SGeographicInfo{
		Latitude:    40.4172871,
		Longitude:   -82.90712300000001,
		City:        CITY_OHIO,
		CountryCode: COUNTRY_CODE_US,
	}

	RegionUSGOVWest = cloudprovider.SGeographicInfo{
		Latitude:    37.09024,
		Longitude:   -95.712891,
		City:        CITY_US_GOV_WEST,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionJioIndiaWest = cloudprovider.SGeographicInfo{
		Latitude:    22.4707,
		Longitude:   70.0577,
		City:        CITY_JAMNAGAR,
		CountryCode: COUNTRY_CODE_IN,
	}
	RegionJioIndiaCentral = cloudprovider.SGeographicInfo{
		Latitude:    21.1466,
		Longitude:   79.0889,
		City:        CITY_NAGPUR,
		CountryCode: COUNTRY_CODE_IN,
	}
)
