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
		Latitude:    39.904,
		Longitude:   116.407,
		City:        CITY_BEI_JING,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionNingxia = cloudprovider.SGeographicInfo{
		Latitude:    37.199,
		Longitude:   106.158,
		City:        CITY_NING_XIA,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionShanghai = cloudprovider.SGeographicInfo{
		Latitude:    31.17,
		Longitude:   121.47,
		City:        CITY_SHANG_HAI,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionQingYang = cloudprovider.SGeographicInfo{
		Latitude:    35.73,
		Longitude:   107.61,
		City:        CITY_QING_YANG,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionGuangzhou = cloudprovider.SGeographicInfo{
		Latitude:    23.129,
		Longitude:   113.264,
		City:        CITY_GUANG_ZHOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionSuzhou = cloudprovider.SGeographicInfo{
		Latitude:    31.328,
		Longitude:   120.479,
		City:        CITY_SU_ZHOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionDalian = cloudprovider.SGeographicInfo{
		Latitude:    38.914,
		Longitude:   121.615,
		City:        CITY_DA_LIAN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionGuiyang = cloudprovider.SGeographicInfo{
		Latitude:    26.647,
		Longitude:   106.63,
		City:        CITY_GUI_YANG,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHaerbin = cloudprovider.SGeographicInfo{
		Latitude:    45.757,
		Longitude:   126.57,
		City:        CITY_HAE_BIN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionZhengzhou = cloudprovider.SGeographicInfo{
		Latitude:    34.743,
		Longitude:   113.498,
		City:        CITY_ZHENG_ZHOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionNeimenggu = cloudprovider.SGeographicInfo{
		Latitude:    41.018,
		Longitude:   113.095,
		City:        CITY_NEI_MENG_GU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionQingdao = cloudprovider.SGeographicInfo{
		Latitude:    36.067,
		Longitude:   120.383,
		City:        CITY_QING_DAO,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionBaoDing = cloudprovider.SGeographicInfo{
		Latitude:    38.871,
		Longitude:   115.393,
		City:        CITY_BAO_DING,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionZhangjiakou = cloudprovider.SGeographicInfo{
		Latitude:    40.768,
		Longitude:   114.886,
		City:        CITY_ZHANG_JIA_KOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHuhehaote = cloudprovider.SGeographicInfo{
		Latitude:    40.842,
		Longitude:   111.75,
		City:        CITY_HU_HE_HAO_TE,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHangzhou = cloudprovider.SGeographicInfo{
		Latitude:    30.274,
		Longitude:   120.155,
		City:        CITY_HANG_ZHOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionWuhan = cloudprovider.SGeographicInfo{
		Latitude:    30.568,
		Longitude:   114.136,
		City:        CITY_WU_HAN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionFoshan = cloudprovider.SGeographicInfo{
		Latitude:    23.009,
		Longitude:   113.024,
		City:        CITY_FO_SHAN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionXian = cloudprovider.SGeographicInfo{
		Latitude:    34.26,
		Longitude:   108.802,
		City:        CITY_XI_AN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionLanzhou = cloudprovider.SGeographicInfo{
		Latitude:    36.078,
		Longitude:   103.596,
		City:        CITY_LAN_ZHOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionWuhu = cloudprovider.SGeographicInfo{
		Latitude:    31.3285,
		Longitude:   118.312,
		City:        CITY_WU_HU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionShenzhen = cloudprovider.SGeographicInfo{
		Latitude:    22.543,
		Longitude:   114.058,
		City:        CITY_SHEN_ZHEN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionKunming = cloudprovider.SGeographicInfo{
		Latitude:    25.021,
		Longitude:   102.659,
		City:        CITY_KUN_MING,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHefei = cloudprovider.SGeographicInfo{
		Latitude:    31.855,
		Longitude:   117.204,
		City:        CITY_HE_FEI,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionFujian = cloudprovider.SGeographicInfo{
		Latitude:    25.925,
		Longitude:   116.98,
		City:        CITY_FU_JIAN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionJinzhong = cloudprovider.SGeographicInfo{
		Latitude:    37.699,
		Longitude:   112.662,
		City:        CITY_JIN_ZHONG,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionNanning = cloudprovider.SGeographicInfo{
		Latitude:    22.822,
		Longitude:   108.204,
		City:        CITY_NAN_NING,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionChengzhou = cloudprovider.SGeographicInfo{
		Latitude:    25.777,
		Longitude:   112.975,
		City:        CITY_CHENG_ZHOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionChangsha = cloudprovider.SGeographicInfo{
		Latitude:    28.176,
		Longitude:   112.86,
		City:        CITY_CHANG_SHA,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionJiangxi = cloudprovider.SGeographicInfo{
		Latitude:    27.274,
		Longitude:   114.713,
		City:        CITY_JIANG_XI,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHunan = cloudprovider.SGeographicInfo{
		Latitude:    27.274,
		Longitude:   114.713,
		City:        CITY_JIANG_XI,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHaikou = cloudprovider.SGeographicInfo{
		Latitude:    20.012,
		Longitude:   110.236,
		City:        CITY_HAI_KOU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionTianjin = cloudprovider.SGeographicInfo{
		Latitude:    39.125,
		Longitude:   117.131,
		City:        CITY_TIAN_JIN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionChengdu = cloudprovider.SGeographicInfo{
		Latitude:    30.573,
		Longitude:   104.067,
		City:        CITY_CHENG_DU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHeyuan = cloudprovider.SGeographicInfo{
		Latitude:    23.729,
		Longitude:   114.697,
		City:        CITY_HE_YUAN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionWulanchabu = cloudprovider.SGeographicInfo{
		Latitude:    41.018,
		Longitude:   113.095,
		City:        CITY_WU_LAN_CHA_BU,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionChongqing = cloudprovider.SGeographicInfo{
		Latitude:    29.432,
		Longitude:   106.912,
		City:        CITY_CHONG_QING,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionNanjing = cloudprovider.SGeographicInfo{
		Latitude:    32.058,
		Longitude:   118.796,
		City:        CITY_NAN_JING,
		CountryCode: COUNTRY_CODE_CN,
	}

	// Asia
	RegionTaiwan = cloudprovider.SGeographicInfo{
		Latitude:    25.044,
		Longitude:   121.509,
		City:        CITY_TAI_WAN,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionHongkong = cloudprovider.SGeographicInfo{
		Latitude:    22.396,
		Longitude:   114.109,
		City:        CITY_HONG_KONG,
		CountryCode: COUNTRY_CODE_CN,
	}
	RegionTokyo = cloudprovider.SGeographicInfo{
		Latitude:    35.709,
		Longitude:   139.732,
		City:        CITY_TOKYO,
		CountryCode: COUNTRY_CODE_JP,
	}
	RegionOsaka = cloudprovider.SGeographicInfo{
		Latitude:    34.694,
		Longitude:   135.502,
		City:        CITY_OSAKA,
		CountryCode: COUNTRY_CODE_JP,
	}
	RegionSeoul = cloudprovider.SGeographicInfo{
		Latitude:    34.694,
		Longitude:   135.502,
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
		Latitude:    19.076,
		Longitude:   72.877,
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
		Latitude:    17.412,
		Longitude:   78.326,
		City:        CITY_HYDERABAD,
		CountryCode: COUNTRY_CODE_IN,
	}

	RegionSingapore = cloudprovider.SGeographicInfo{
		Latitude:    1.352,
		Longitude:   103.82,
		City:        CITY_SINGAPORE,
		CountryCode: COUNTRY_CODE_SG,
	}
	// 雅加达
	RegionJakarta = cloudprovider.SGeographicInfo{
		Latitude:    -6.175,
		Longitude:   106.865,
		City:        CITY_JAKARTA,
		CountryCode: COUNTRY_CODE_ID,
	}
	RegionKualaLumpur = cloudprovider.SGeographicInfo{
		Latitude:    3.139,
		Longitude:   101.687,
		City:        CITY_KUALA_LUMPUR,
		CountryCode: COUNTRY_CODE_MY,
	}
	RegionBangkok = cloudprovider.SGeographicInfo{
		Latitude:    13.756,
		Longitude:   100.502,
		City:        CITY_BANGKOK,
		CountryCode: COUNTRY_CODE_TH,
	}

	RegionSydney = cloudprovider.SGeographicInfo{
		Latitude:    -33.869,
		Longitude:   151.209,
		City:        CITY_SYDNEY,
		CountryCode: COUNTRY_CODE_AU,
	}
	// 墨尔本
	RegionMelbourne = cloudprovider.SGeographicInfo{
		Latitude:    -37.814,
		Longitude:   144.963,
		City:        CITY_MELBOURNE,
		CountryCode: COUNTRY_CODE_AU,
	}
	//亚拉伦拉 澳大利亚
	RegionYarralumla = cloudprovider.SGeographicInfo{
		Latitude:    -35.302,
		Longitude:   149.078,
		City:        CITY_YARRALUMLA,
		CountryCode: COUNTRY_CODE_AU,
	}

	// Africa
	RegionCapeTown = cloudprovider.SGeographicInfo{
		Latitude:    -33.915,
		Longitude:   18.376,
		City:        CITY_CAPE_TOWN,
		CountryCode: COUNTRY_CODE_ZA,
	}
	// 比勒陀利亚
	RegionPretoria = cloudprovider.SGeographicInfo{
		Latitude:    -25.717,
		Longitude:   28.283,
		City:        CITY_PRETORIA,
		CountryCode: COUNTRY_CODE_ZA,
	}
	RegionJohannesburg = cloudprovider.SGeographicInfo{
		Latitude:    -26.171,
		Longitude:   27.9,
		City:        CITY_JOHANNESBURG,
		CountryCode: COUNTRY_CODE_ZA,
	}

	// Middleeast
	RegionBahrain = cloudprovider.SGeographicInfo{
		Latitude:    25.941,
		Longitude:   50.447,
		City:        CITY_BAHRAIN,
		CountryCode: COUNTRY_CODE_BH,
	}
	// 迪拜
	RegionDubai = cloudprovider.SGeographicInfo{
		Latitude:    25.263,
		Longitude:   55.297,
		City:        CITY_DUBAI,
		CountryCode: COUNTRY_CODE_AE,
	}
	// 阿布扎比
	RegionAbuDhabi = cloudprovider.SGeographicInfo{
		Latitude:    24.387,
		Longitude:   54.394,
		City:        CITY_ABU_DHABI,
		CountryCode: COUNTRY_CODE_AE,
	}

	// Europe
	RegionFinland = cloudprovider.SGeographicInfo{
		Latitude:    64.826,
		Longitude:   21.543,
		City:        CITY_FINLAND,
		CountryCode: COUNTRY_CODE_FI,
	}
	RegionBelgium = cloudprovider.SGeographicInfo{
		Latitude:    50.5,
		Longitude:   3.906,
		City:        CITY_BELGIUM,
		CountryCode: COUNTRY_CODE_BE,
	}
	RegionLondon = cloudprovider.SGeographicInfo{
		Latitude:    51.507,
		Longitude:   -0.128,
		City:        CITY_LONDON,
		CountryCode: COUNTRY_CODE_GB,
	}
	RegionHalton = cloudprovider.SGeographicInfo{
		Latitude:    53.333,
		Longitude:   -2.696,
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
		Latitude:    51.166,
		Longitude:   10.452,
		City:        CITY_FRANKFURT,
		CountryCode: COUNTRY_CODE_DE,
	}
	RegionHolland = cloudprovider.SGeographicInfo{
		Latitude:    52.208,
		Longitude:   4.159,
		City:        CITY_HOLLAND,
		CountryCode: COUNTRY_CODE_NL,
	}
	RegionZurich = cloudprovider.SGeographicInfo{
		Latitude:    47.377,
		Longitude:   8.502,
		City:        CITY_ZURICH,
		CountryCode: COUNTRY_CODE_CH,
	}
	RegionMilan = cloudprovider.SGeographicInfo{
		Latitude:    45.463,
		Longitude:   9.108,
		City:        CITY_MILAN,
		CountryCode: COUNTRY_CODE_IT,
	}
	RegionWarsaw = cloudprovider.SGeographicInfo{
		Latitude:    52.233,
		Longitude:   21.017,
		City:        CITY_WARSAW,
		CountryCode: COUNTRY_CODE_PL,
	}
	RegionMadrid = cloudprovider.SGeographicInfo{
		Latitude:    40.438,
		Longitude:   -3.82,
		City:        CITY_MADRID,
		CountryCode: COUNTRY_CODE_ES,
	}
	RegionIreland = cloudprovider.SGeographicInfo{
		Latitude:    53.413,
		Longitude:   -8.244,
		City:        CITY_IRELAND,
		CountryCode: COUNTRY_CODE_IE,
	}
	RegionDublin = cloudprovider.SGeographicInfo{
		Latitude:    53.35,
		Longitude:   -6.26,
		City:        CITY_DUBLIN,
		CountryCode: COUNTRY_CODE_IE,
	}
	RegionParis = cloudprovider.SGeographicInfo{
		Latitude:    48.857,
		Longitude:   2.352,
		City:        CITY_PARIS,
		CountryCode: COUNTRY_CODE_FR,
	}
	RegionTurin = cloudprovider.SGeographicInfo{
		Latitude:    45.07,
		Longitude:   7.63,
		City:        CITY_TURIN,
		CountryCode: COUNTRY_CODE_IT,
	}
	RegionAllier = cloudprovider.SGeographicInfo{
		Latitude:    46.518,
		Longitude:   3.359,
		City:        CITY_ALLIER,
		CountryCode: COUNTRY_CODE_FR,
	}
	RegionTarn = cloudprovider.SGeographicInfo{
		Latitude:    43.866,
		Longitude:   1.505,
		City:        CITY_TARN,
		CountryCode: COUNTRY_CODE_FR,
	}
	RegionStockholm = cloudprovider.SGeographicInfo{
		Latitude:    59.195,
		Longitude:   18.47,
		City:        CITY_STOCKHOLM,
		CountryCode: COUNTRY_CODE_SE,
	}
	RegionDelmenhorst = cloudprovider.SGeographicInfo{
		Latitude:    53.051,
		Longitude:   8.632,
		City:        CITY_DELMENHORST,
		CountryCode: COUNTRY_CODE_DE,
	}
	RegionGeneva = cloudprovider.SGeographicInfo{
		Latitude:    46.202,
		Longitude:   6.147,
		City:        CITY_GENEVA,
		CountryCode: COUNTRY_CODE_CH,
	}
	RegionStavanger = cloudprovider.SGeographicInfo{
		Latitude:    58.963,
		Longitude:   5.719,
		City:        CITY_STAVANGER,
		CountryCode: COUNTRY_CODE_NO,
	}
	RegionOslo = cloudprovider.SGeographicInfo{
		Latitude:    59.906,
		Longitude:   10.768,
		City:        CITY_OSLO,
		CountryCode: COUNTRY_CODE_NO,
	}

	RegionMoscow = cloudprovider.SGeographicInfo{
		Latitude:    55.756,
		Longitude:   37.617,
		City:        CITY_MOSCOW,
		CountryCode: COUNTRY_CODE_RU,
	}

	// America
	RegionMontreal = cloudprovider.SGeographicInfo{
		Latitude:    45.558,
		Longitude:   -73.8,
		City:        CITY_MONTREAL,
		CountryCode: COUNTRY_CODE_CA,
	}
	RegionToronto = cloudprovider.SGeographicInfo{
		Latitude:    43.653,
		Longitude:   -79.383,
		City:        CITY_TORONTO,
		CountryCode: COUNTRY_CODE_CA,
	}
	RegionCanadaCentral = cloudprovider.SGeographicInfo{
		Latitude:    56.13,
		Longitude:   -106.347,
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
		Latitude:    -23.551,
		Longitude:   -46.633,
		City:        CITY_SAO_PAULO,
		CountryCode: COUNTRY_CODE_BR,
	}
	// 里约热内卢
	RegionRioDeJaneiro = cloudprovider.SGeographicInfo{
		Latitude:    -22.908,
		Longitude:   -43.196,
		City:        CITY_RIO_DE_JANEIRO,
		CountryCode: COUNTRY_CODE_BR,
	}

	RegionMexico = cloudprovider.SGeographicInfo{
		Latitude:    55.118,
		Longitude:   141.038,
		City:        CITY_MEXICO,
		CountryCode: COUNTRY_CODE_MX,
	}

	RegionSantiago = cloudprovider.SGeographicInfo{
		Latitude:    -33.452,
		Longitude:   -70.676,
		City:        CITY_SANTIAGO,
		CountryCode: COUNTRY_CODE_CL,
	}

	RegionIowa = cloudprovider.SGeographicInfo{
		Latitude:    41.933,
		Longitude:   -94.511,
		City:        CITY_IOWA,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionCarolina = cloudprovider.SGeographicInfo{
		Latitude:    33.619,
		Longitude:   -82.047,
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
		Latitude:    37.432,
		Longitude:   -78.657,
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
		Latitude:    41.882,
		Longitude:   -87.628,
		City:        CITY_CHICAGO,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionPhoenix = cloudprovider.SGeographicInfo{
		Latitude:    33.448,
		Longitude:   -112.074,
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
		Latitude:    37.778,
		Longitude:   -122.416,
		City:        CITY_SAN_FRANCISCO,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionSiliconValley = cloudprovider.SGeographicInfo{
		Latitude:    37.387,
		Longitude:   -122.058,
		City:        CITY_SILICONVALLEY,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionNothVirginia = cloudprovider.SGeographicInfo{
		Latitude:    37.432,
		Longitude:   -78.657,
		City:        CITY_N_VIRGINIA,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionOregon = cloudprovider.SGeographicInfo{
		Latitude:    43.804,
		Longitude:   -120.554,
		City:        CITY_OREGON,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionLosAngeles = cloudprovider.SGeographicInfo{
		Latitude:    34.052,
		Longitude:   -118.244,
		City:        CITY_LOS_ANGELES,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionSaltLakeCity = cloudprovider.SGeographicInfo{
		Latitude:    40.777,
		Longitude:   -111.991,
		City:        CITY_SALT_LAKE_CITY,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionLasVegas = cloudprovider.SGeographicInfo{
		Latitude:    36.125,
		Longitude:   -115.315,
		City:        CITY_LAS_VEGAS,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionDallas = cloudprovider.SGeographicInfo{
		Latitude:    32.784,
		Longitude:   -96.891,
		City:        CITY_DALLAS,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionColumbus = cloudprovider.SGeographicInfo{
		Latitude:    32.088,
		Longitude:   34.762,
		City:        CITY_TEL_AVIV,
		CountryCode: COUNTRY_CODE_IL,
	}
	RegionDamman = cloudprovider.SGeographicInfo{
		Latitude:    26.36,
		Longitude:   49.91,
		City:        CITY_DAMMAN,
		CountryCode: COUNTRY_CODE_AE,
	}
	RegionDoha = cloudprovider.SGeographicInfo{
		Latitude:    25.28,
		Longitude:   51.43,
		City:        CITY_DOHA,
		CountryCode: COUNTRY_CODE_QA,
	}
	RegionNorthCalifornia = cloudprovider.SGeographicInfo{
		Latitude:    38.838,
		Longitude:   -120.896,
		City:        CITY_N_CALIFORNIA,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionOhio = cloudprovider.SGeographicInfo{
		Latitude:    40.417,
		Longitude:   -82.907,
		City:        CITY_OHIO,
		CountryCode: COUNTRY_CODE_US,
	}

	RegionUSGOVWest = cloudprovider.SGeographicInfo{
		Latitude:    37.09,
		Longitude:   -95.713,
		City:        CITY_US_GOV_WEST,
		CountryCode: COUNTRY_CODE_US,
	}
	RegionJioIndiaWest = cloudprovider.SGeographicInfo{
		Latitude:    22.471,
		Longitude:   70.058,
		City:        CITY_JAMNAGAR,
		CountryCode: COUNTRY_CODE_IN,
	}
	RegionJioIndiaCentral = cloudprovider.SGeographicInfo{
		Latitude:    21.147,
		Longitude:   79.09,
		City:        CITY_NAGPUR,
		CountryCode: COUNTRY_CODE_IN,
	}
)
