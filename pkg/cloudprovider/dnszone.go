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

package cloudprovider

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"
)

type TDnsZoneType string
type TDnsPolicyType string

type TDnsType string

type TDnsPolicyValue string

type TDnsTTLRangeType string

type TDnsProductType string

const (
	PublicZone  = TDnsZoneType("PublicZone")
	PrivateZone = TDnsZoneType("PrivateZone")
)

const (
	ContinuousTTlRange = TDnsTTLRangeType("ContinuousTTlRange")
	DiscreteTTlRange   = TDnsTTLRangeType("DiscreteTTlRange")
)

const (
	DnsPolicyTypeSimple           = TDnsPolicyType("Simple")           //简单
	DnsPolicyTypeByCarrier        = TDnsPolicyType("ByCarrier")        //运营商
	DnsPolicyTypeByGeoLocation    = TDnsPolicyType("ByGeoLocation")    //地理区域
	DnsPolicyTypeBySearchEngine   = TDnsPolicyType("BySearchEngine")   //搜索引擎
	DnsPolicyTypeIpRange          = TDnsPolicyType("IpRange")          //自定义IP范围
	DnsPolicyTypeWeighted         = TDnsPolicyType("Weighted")         //加权
	DnsPolicyTypeFailover         = TDnsPolicyType("Failover")         //故障转移
	DnsPolicyTypeMultiValueAnswer = TDnsPolicyType("MultiValueAnswer") //多值应答
	DnsPolicyTypeLatency          = TDnsPolicyType("Latency")          //延迟
)

const (
	DnsTypeA            = TDnsType("A")
	DnsTypeAAAA         = TDnsType("AAAA")
	DnsTypeCAA          = TDnsType("CAA")
	DnsTypeCNAME        = TDnsType("CNAME")
	DnsTypeMX           = TDnsType("MX")
	DnsTypeNS           = TDnsType("NS")
	DnsTypeSRV          = TDnsType("SRV")
	DnsTypeSOA          = TDnsType("SOA")
	DnsTypeTXT          = TDnsType("TXT")
	DnsTypePTR          = TDnsType("PTR")
	DnsTypeDS           = TDnsType("DS")
	DnsTypeDNSKEY       = TDnsType("DNSKEY")
	DnsTypeIPSECKEY     = TDnsType("IPSECKEY")
	DnsTypeNAPTR        = TDnsType("NAPTR")
	DnsTypeSPF          = TDnsType("SPF")
	DnsTypeSSHFP        = TDnsType("SSHFP")
	DnsTypeTLSA         = TDnsType("TLSA")
	DnsTypeREDIRECT_URL = TDnsType("REDIRECT_URL") //显性URL转发
	DnsTypeFORWARD_URL  = TDnsType("FORWARD_URL")  //隐性URL转发
)

const (
	DnsProductEnterpriseUltimate   = TDnsProductType("DP_EnterpriseUltimate")
	DnsProductEnterpriseStandard   = TDnsProductType("DP_EnterpriseStandard")
	DnsProductEnterpriseBasic      = TDnsProductType("DP_EnterpriseBasic")
	DnsProductPersonalProfessional = TDnsProductType("DP_PersonalProfessional")
	DnsProductFree                 = TDnsProductType("DP_Free")
)

var (
	SUPPORTED_DNS_TYPES = []TDnsType{
		DnsTypeA,
		DnsTypeAAAA,
		DnsTypeCAA,
		DnsTypeCNAME,
		DnsTypeMX,
		DnsTypeNS,
		DnsTypeSRV,
		DnsTypeSOA,
		DnsTypeTXT,
		DnsTypePTR,
		DnsTypeDS,
		DnsTypeDNSKEY,
		DnsTypeIPSECKEY,
		DnsTypeNAPTR,
		DnsTypeSPF,
		DnsTypeSSHFP,
		DnsTypeTLSA,
		DnsTypeREDIRECT_URL,
		DnsTypeFORWARD_URL,
	}
)

var (
	DnsPolicyValueEmpty = TDnsPolicyValue("")

	DnsPolicyValueUnicom      = TDnsPolicyValue("unicom")
	DnsPolicyValueTelecom     = TDnsPolicyValue("telecom")
	DnsPolicyValueChinaMobile = TDnsPolicyValue("chinamobile")
	DnsPolicyValueCernet      = TDnsPolicyValue("cernet")
	DnsPolicyValueDrPeng      = TDnsPolicyValue("drpeng")
	DnsPolicyValueBtvn        = TDnsPolicyValue("btvn")

	DnsPolicyValueOversea  = TDnsPolicyValue("oversea")
	DnsPolicyValueMainland = TDnsPolicyValue("mainland")

	DnsPolicyValueBaidu   = TDnsPolicyValue("baidu")
	DnsPolicyValueGoogle  = TDnsPolicyValue("google")
	DnsPolicyValueBing    = TDnsPolicyValue("bing")
	DnsPolicyValueYoudao  = TDnsPolicyValue("youdao")
	DnsPolicyValueSousou  = TDnsPolicyValue("sousou")
	DnsPolicyValueSougou  = TDnsPolicyValue("sougou")
	DnsPolicyValueQihu360 = TDnsPolicyValue("qihu360")
)

var AwsGeoLocations = []TDnsPolicyValue{
	TDnsPolicyValue("Default"),
	TDnsPolicyValue("Andorra"),
	TDnsPolicyValue("United Arab Emirates"),
	TDnsPolicyValue("Afghanistan"),
	TDnsPolicyValue("Antigua and Barbuda"),
	TDnsPolicyValue("Anguilla"),
	TDnsPolicyValue("Albania"),
	TDnsPolicyValue("Armenia"),
	TDnsPolicyValue("Angola"),
	TDnsPolicyValue("Antarctica"),
	TDnsPolicyValue("Argentina"),
	TDnsPolicyValue("American Samoa"),
	TDnsPolicyValue("Austria"),
	TDnsPolicyValue("Australia"),
	TDnsPolicyValue("Aruba"),
	TDnsPolicyValue("Åland"),
	TDnsPolicyValue("Azerbaijan"),
	TDnsPolicyValue("Bosnia and Herzegovina"),
	TDnsPolicyValue("Barbados"),
	TDnsPolicyValue("Bangladesh"),
	TDnsPolicyValue("Belgium"),
	TDnsPolicyValue("Burkina Faso"),
	TDnsPolicyValue("Bulgaria"),
	TDnsPolicyValue("Bahrain"),
	TDnsPolicyValue("Burundi"),
	TDnsPolicyValue("Benin"),
	TDnsPolicyValue("Saint Barthélemy"),
	TDnsPolicyValue("Bermuda"),
	TDnsPolicyValue("Brunei"),
	TDnsPolicyValue("Bolivia"),
	TDnsPolicyValue("Bonaire"),
	TDnsPolicyValue("Brazil"),
	TDnsPolicyValue("Bahamas"),
	TDnsPolicyValue("Bhutan"),
	TDnsPolicyValue("Botswana"),
	TDnsPolicyValue("Belarus"),
	TDnsPolicyValue("Belize"),
	TDnsPolicyValue("Canada"),
	TDnsPolicyValue("Cocos [Keeling] Islands"),
	TDnsPolicyValue("Congo"),
	TDnsPolicyValue("Central African Republic"),
	TDnsPolicyValue("Republic of the Congo"),
	TDnsPolicyValue("Switzerland"),
	TDnsPolicyValue("Ivory Coast"),
	TDnsPolicyValue("Cook Islands"),
	TDnsPolicyValue("Chile"),
	TDnsPolicyValue("Cameroon"),
	TDnsPolicyValue("China"),
	TDnsPolicyValue("Colombia"),
	TDnsPolicyValue("Costa Rica"),
	TDnsPolicyValue("Cuba"),
	TDnsPolicyValue("Cape Verde"),
	TDnsPolicyValue("Curaçao"),
	TDnsPolicyValue("Cyprus"),
	TDnsPolicyValue("Czech Republic"),
	TDnsPolicyValue("Germany"),
	TDnsPolicyValue("Djibouti"),
	TDnsPolicyValue("Denmark"),
	TDnsPolicyValue("Dominica"),
	TDnsPolicyValue("Dominican Republic"),
	TDnsPolicyValue("Algeria"),
	TDnsPolicyValue("Ecuador"),
	TDnsPolicyValue("Estonia"),
	TDnsPolicyValue("Egypt"),
	TDnsPolicyValue("Eritrea"),
	TDnsPolicyValue("Spain"),
	TDnsPolicyValue("Ethiopia"),
	TDnsPolicyValue("Finland"),
	TDnsPolicyValue("Fiji"),
	TDnsPolicyValue("Falkland Islands"),
	TDnsPolicyValue("Federated States of Micronesia"),
	TDnsPolicyValue("Faroe Islands"),
	TDnsPolicyValue("France"),
	TDnsPolicyValue("Gabon"),
	TDnsPolicyValue("United Kingdom"),
	TDnsPolicyValue("Grenada"),
	TDnsPolicyValue("Georgia"),
	TDnsPolicyValue("French Guiana"),
	TDnsPolicyValue("Guernsey"),
	TDnsPolicyValue("Ghana"),
	TDnsPolicyValue("Gibraltar"),
	TDnsPolicyValue("Greenland"),
	TDnsPolicyValue("Gambia"),
	TDnsPolicyValue("Guinea"),
	TDnsPolicyValue("Guadeloupe"),
	TDnsPolicyValue("Equatorial Guinea"),
	TDnsPolicyValue("Greece"),
	TDnsPolicyValue("South Georgia and the South Sandwich Islands"),
	TDnsPolicyValue("Guatemala"),
	TDnsPolicyValue("Guam"),
	TDnsPolicyValue("Guinea-Bissau"),
	TDnsPolicyValue("Guyana"),
	TDnsPolicyValue("Hong Kong"),
	TDnsPolicyValue("Honduras"),
	TDnsPolicyValue("Croatia"),
	TDnsPolicyValue("Haiti"),
	TDnsPolicyValue("Hungary"),
	TDnsPolicyValue("Indonesia"),
	TDnsPolicyValue("Ireland"),
	TDnsPolicyValue("Israel"),
	TDnsPolicyValue("Isle of Man"),
	TDnsPolicyValue("India"),
	TDnsPolicyValue("British Indian Ocean Territory"),
	TDnsPolicyValue("Iraq"),
	TDnsPolicyValue("Iran"),
	TDnsPolicyValue("Iceland"),
	TDnsPolicyValue("Italy"),
	TDnsPolicyValue("Jersey"),
	TDnsPolicyValue("Jamaica"),
	TDnsPolicyValue("Hashemite Kingdom of Jordan"),
	TDnsPolicyValue("Japan"),
	TDnsPolicyValue("Kenya"),
	TDnsPolicyValue("Kyrgyzstan"),
	TDnsPolicyValue("Cambodia"),
	TDnsPolicyValue("Kiribati"),
	TDnsPolicyValue("Comoros"),
	TDnsPolicyValue("Saint Kitts and Nevis"),
	TDnsPolicyValue("North Korea"),
	TDnsPolicyValue("Republic of Korea"),
	TDnsPolicyValue("Kuwait"),
	TDnsPolicyValue("Cayman Islands"),
	TDnsPolicyValue("Kazakhstan"),
	TDnsPolicyValue("Laos"),
	TDnsPolicyValue("Lebanon"),
	TDnsPolicyValue("Saint Lucia"),
	TDnsPolicyValue("Liechtenstein"),
	TDnsPolicyValue("Sri Lanka"),
	TDnsPolicyValue("Liberia"),
	TDnsPolicyValue("Lesotho"),
	TDnsPolicyValue("Lithuania"),
	TDnsPolicyValue("Luxembourg"),
	TDnsPolicyValue("Latvia"),
	TDnsPolicyValue("Libya"),
	TDnsPolicyValue("Morocco"),
	TDnsPolicyValue("Monaco"),
	TDnsPolicyValue("Republic of Moldova"),
	TDnsPolicyValue("Montenegro"),
	TDnsPolicyValue("Saint Martin"),
	TDnsPolicyValue("Madagascar"),
	TDnsPolicyValue("Marshall Islands"),
	TDnsPolicyValue("Macedonia"),
	TDnsPolicyValue("Mali"),
	TDnsPolicyValue("Myanmar [Burma]"),
	TDnsPolicyValue("Mongolia"),
	TDnsPolicyValue("Macao"),
	TDnsPolicyValue("Northern Mariana Islands"),
	TDnsPolicyValue("Martinique"),
	TDnsPolicyValue("Mauritania"),
	TDnsPolicyValue("Montserrat"),
	TDnsPolicyValue("Malta"),
	TDnsPolicyValue("Mauritius"),
	TDnsPolicyValue("Maldives"),
	TDnsPolicyValue("Malawi"),
	TDnsPolicyValue("Mexico"),
	TDnsPolicyValue("Malaysia"),
	TDnsPolicyValue("Mozambique"),
	TDnsPolicyValue("Namibia"),
	TDnsPolicyValue("New Caledonia"),
	TDnsPolicyValue("Niger"),
	TDnsPolicyValue("Norfolk Island"),
	TDnsPolicyValue("Nigeria"),
	TDnsPolicyValue("Nicaragua"),
	TDnsPolicyValue("Netherlands"),
	TDnsPolicyValue("Norway"),
	TDnsPolicyValue("Nepal"),
	TDnsPolicyValue("Nauru"),
	TDnsPolicyValue("Niue"),
	TDnsPolicyValue("New Zealand"),
	TDnsPolicyValue("Oman"),
	TDnsPolicyValue("Panama"),
	TDnsPolicyValue("Peru"),
	TDnsPolicyValue("French Polynesia"),
	TDnsPolicyValue("Papua New Guinea"),
	TDnsPolicyValue("Philippines"),
	TDnsPolicyValue("Pakistan"),
	TDnsPolicyValue("Poland"),
	TDnsPolicyValue("Saint Pierre and Miquelon"),
	TDnsPolicyValue("Pitcairn Islands"),
	TDnsPolicyValue("Puerto Rico"),
	TDnsPolicyValue("Palestine"),
	TDnsPolicyValue("Portugal"),
	TDnsPolicyValue("Palau"),
	TDnsPolicyValue("Paraguay"),
	TDnsPolicyValue("Qatar"),
	TDnsPolicyValue("Réunion"),
	TDnsPolicyValue("Romania"),
	TDnsPolicyValue("Serbia"),
	TDnsPolicyValue("Russia"),
	TDnsPolicyValue("Rwanda"),
	TDnsPolicyValue("Saudi Arabia"),
	TDnsPolicyValue("Solomon Islands"),
	TDnsPolicyValue("Seychelles"),
	TDnsPolicyValue("Sudan"),
	TDnsPolicyValue("Sweden"),
	TDnsPolicyValue("Singapore"),
	TDnsPolicyValue("Saint Helena"),
	TDnsPolicyValue("Slovenia"),
	TDnsPolicyValue("Svalbard and Jan Mayen"),
	TDnsPolicyValue("Slovakia"),
	TDnsPolicyValue("Sierra Leone"),
	TDnsPolicyValue("San Marino"),
	TDnsPolicyValue("Senegal"),
	TDnsPolicyValue("Somalia"),
	TDnsPolicyValue("Suriname"),
	TDnsPolicyValue("South Sudan"),
	TDnsPolicyValue("São Tomé and Príncipe"),
	TDnsPolicyValue("El Salvador"),
	TDnsPolicyValue("Sint Maarten"),
	TDnsPolicyValue("Syria"),
	TDnsPolicyValue("Swaziland"),
	TDnsPolicyValue("Turks and Caicos Islands"),
	TDnsPolicyValue("Chad"),
	TDnsPolicyValue("French Southern Territories"),
	TDnsPolicyValue("Togo"),
	TDnsPolicyValue("Thailand"),
	TDnsPolicyValue("Tajikistan"),
	TDnsPolicyValue("Tokelau"),
	TDnsPolicyValue("East Timor"),
	TDnsPolicyValue("Turkmenistan"),
	TDnsPolicyValue("Tunisia"),
	TDnsPolicyValue("Tonga"),
	TDnsPolicyValue("Turkey"),
	TDnsPolicyValue("Trinidad and Tobago"),
	TDnsPolicyValue("Tuvalu"),
	TDnsPolicyValue("Taiwan"),
	TDnsPolicyValue("Tanzania"),
	TDnsPolicyValue("Ukraine"),
	TDnsPolicyValue("Uganda"),
	TDnsPolicyValue("U.S. Minor Outlying Islands"),
	TDnsPolicyValue("United States"),
	TDnsPolicyValue("Alaska"),
	TDnsPolicyValue("Alabama"),
	TDnsPolicyValue("Arkansas"),
	TDnsPolicyValue("Arizona"),
	TDnsPolicyValue("California"),
	TDnsPolicyValue("Colorado"),
	TDnsPolicyValue("Connecticut"),
	TDnsPolicyValue("District of Columbia"),
	TDnsPolicyValue("Delaware"),
	TDnsPolicyValue("Florida"),
	TDnsPolicyValue("Georgia"),
	TDnsPolicyValue("Hawaii"),
	TDnsPolicyValue("Iowa"),
	TDnsPolicyValue("Idaho"),
	TDnsPolicyValue("Illinois"),
	TDnsPolicyValue("Indiana"),
	TDnsPolicyValue("Kansas"),
	TDnsPolicyValue("Kentucky"),
	TDnsPolicyValue("Louisiana"),
	TDnsPolicyValue("Massachusetts"),
	TDnsPolicyValue("Maryland"),
	TDnsPolicyValue("Maine"),
	TDnsPolicyValue("Michigan"),
	TDnsPolicyValue("Minnesota"),
	TDnsPolicyValue("Missouri"),
	TDnsPolicyValue("Mississippi"),
	TDnsPolicyValue("Montana"),
	TDnsPolicyValue("North Carolina"),
	TDnsPolicyValue("North Dakota"),
	TDnsPolicyValue("Nebraska"),
	TDnsPolicyValue("New Hampshire"),
	TDnsPolicyValue("New Jersey"),
	TDnsPolicyValue("New Mexico"),
	TDnsPolicyValue("Nevada"),
	TDnsPolicyValue("New York"),
	TDnsPolicyValue("Ohio"),
	TDnsPolicyValue("Oklahoma"),
	TDnsPolicyValue("Oregon"),
	TDnsPolicyValue("Pennsylvania"),
	TDnsPolicyValue("Rhode Island"),
	TDnsPolicyValue("South Carolina"),
	TDnsPolicyValue("South Dakota"),
	TDnsPolicyValue("Tennessee"),
	TDnsPolicyValue("Texas"),
	TDnsPolicyValue("Utah"),
	TDnsPolicyValue("Virginia"),
	TDnsPolicyValue("Vermont"),
	TDnsPolicyValue("Washington"),
	TDnsPolicyValue("Wisconsin"),
	TDnsPolicyValue("West Virginia"),
	TDnsPolicyValue("Wyoming"),
	TDnsPolicyValue("Uruguay"),
	TDnsPolicyValue("Uzbekistan"),
	TDnsPolicyValue("Vatican City"),
	TDnsPolicyValue("Saint Vincent and the Grenadines"),
	TDnsPolicyValue("Venezuela"),
	TDnsPolicyValue("British Virgin Islands"),
	TDnsPolicyValue("U.S. Virgin Islands"),
	TDnsPolicyValue("Vietnam"),
	TDnsPolicyValue("Vanuatu"),
	TDnsPolicyValue("Wallis and Futuna"),
	TDnsPolicyValue("Samoa"),
	TDnsPolicyValue("Kosovo"),
	TDnsPolicyValue("Yemen"),
	TDnsPolicyValue("Mayotte"),
	TDnsPolicyValue("South Africa"),
	TDnsPolicyValue("Zambia"),
	TDnsPolicyValue("Zimbabwe"),
	TDnsPolicyValue("Africa"),
	TDnsPolicyValue("Antarctica"),
	TDnsPolicyValue("Asia"),
	TDnsPolicyValue("Europe"),
	TDnsPolicyValue("North America"),
	TDnsPolicyValue("Oceania"),
	TDnsPolicyValue("South America"),
}

var AwsRegions = []TDnsPolicyValue{
	TDnsPolicyValue("us-east-2"),
	TDnsPolicyValue("us-east-1"),
	TDnsPolicyValue("us-west-1"),
	TDnsPolicyValue("us-west-2"),
	TDnsPolicyValue("af-south-1"),
	TDnsPolicyValue("ap-east-1"),
	TDnsPolicyValue("ap-south-1"),
	TDnsPolicyValue("ap-northeast-3"),
	TDnsPolicyValue("ap-northeast-2"),
	TDnsPolicyValue("ap-southeast-1"),
	TDnsPolicyValue("ap-southeast-2"),
	TDnsPolicyValue("ap-northeast-1"),
	TDnsPolicyValue("ca-central-1"),
	TDnsPolicyValue("cn-north-1"),
	TDnsPolicyValue("cn-northwest-1"),
	TDnsPolicyValue("eu-central-1"),
	TDnsPolicyValue("eu-west-1"),
	TDnsPolicyValue("eu-west-2"),
	TDnsPolicyValue("eu-south-1"),
	TDnsPolicyValue("eu-west-3"),
	TDnsPolicyValue("eu-north-1"),
	TDnsPolicyValue("me-south-1"),
	TDnsPolicyValue("sa-east-1"),
	TDnsPolicyValue("us-gov-east-1"),
	TDnsPolicyValue("us-gov-west-1"),
}

var AwsFailovers = []TDnsPolicyValue{
	TDnsPolicyValue("PRIMARY"),
	TDnsPolicyValue("SECONDARY"),
}

type TTlRange struct {
	RangeType   TDnsTTLRangeType
	TTLMinValue int64
	TTLMaxValue int64
	AllowedTTLs []int64 // sorted
}

func (ttlR TTlRange) GetSuppportedTTL(ttl int64) int64 {
	if ttlR.RangeType == ContinuousTTlRange {
		if ttl < ttlR.TTLMinValue {
			return ttlR.TTLMinValue
		}
		if ttl > ttlR.TTLMaxValue {
			return ttlR.TTLMaxValue
		}
		return ttl
	}
	if ttlR.RangeType == DiscreteTTlRange {
		if ttl < ttlR.AllowedTTLs[0] {
			return ttlR.AllowedTTLs[0]
		}
		for i := 0; i < len(ttlR.AllowedTTLs)-1; i++ {
			if ttl > ttlR.AllowedTTLs[i] && ttl < ttlR.AllowedTTLs[i+1] {
				if ttl-ttlR.AllowedTTLs[i] < ttlR.AllowedTTLs[i+1]-ttl {
					return ttlR.AllowedTTLs[i]
				}
				return ttlR.AllowedTTLs[i+1]
			}
		}
		return ttlR.AllowedTTLs[len(ttlR.AllowedTTLs)-1]
	}
	return ttl
}

var TtlRangeAliyunEnterpriseUltimate = TTlRange{RangeType: ContinuousTTlRange, TTLMinValue: 1, TTLMaxValue: 86400}
var TtlRangeAliyunEnterpriseStandard = TTlRange{RangeType: ContinuousTTlRange, TTLMinValue: 60, TTLMaxValue: 86400}
var TtlRangeAliyunPersonal = TTlRange{RangeType: ContinuousTTlRange, TTLMinValue: 600, TTLMaxValue: 86400}
var TtlRangeAliyunFree = TTlRange{RangeType: ContinuousTTlRange, TTLMinValue: 600, TTLMaxValue: 86400}

var TtlRangeAliyunPvtz = TTlRange{RangeType: DiscreteTTlRange, AllowedTTLs: []int64{5, 10, 15, 20, 30, 60, 120, 300, 600, 1800, 3600, 43200, 86400}}

var TtlRangeQcloudEnterpriseUltimate = TTlRange{RangeType: ContinuousTTlRange, TTLMinValue: 1, TTLMaxValue: 604800}
var TtlRangeQcloudEnterpriseStandard = TTlRange{RangeType: ContinuousTTlRange, TTLMinValue: 30, TTLMaxValue: 604800}
var TtlRangeQcloudEnterpriseBasic = TTlRange{RangeType: ContinuousTTlRange, TTLMinValue: 60, TTLMaxValue: 604800}
var TtlRangeQcloudPersonalProfessional = TTlRange{RangeType: ContinuousTTlRange, TTLMinValue: 120, TTLMaxValue: 604800}
var TtlRangeQcloudFree = TTlRange{RangeType: ContinuousTTlRange, TTLMinValue: 600, TTLMaxValue: 604800}

type SPrivateZoneVpc struct {
	Id       string
	RegionId string
}

type SDnsZoneCreateOptions struct {
	Name     string
	Desc     string
	ZoneType TDnsZoneType
	Vpcs     []SPrivateZoneVpc
	Options  *jsonutils.JSONDict
}

func IsSupportPolicyValue(v1 TDnsPolicyValue, arr []TDnsPolicyValue) bool {
	isIn, _ := utils.InArray(v1, arr)
	return isIn
}

type DnsRecordSet struct {
	Id         string
	ExternalId string

	Enabled       bool
	DnsName       string
	DnsType       TDnsType
	DnsValue      string
	Status        string
	Ttl           int64
	MxPriority    int64
	PolicyType    TDnsPolicyType
	PolicyValue   TDnsPolicyValue
	PolicyOptions *jsonutils.JSONDict
}

func (r DnsRecordSet) GetGlobalId() string {
	return r.ExternalId
}

func (r DnsRecordSet) GetName() string {
	return r.DnsName
}

func (r DnsRecordSet) GetDnsName() string {
	return r.DnsName
}

func (r DnsRecordSet) GetDnsValue() string {
	return r.DnsValue
}

func (r DnsRecordSet) GetPolicyType() TDnsPolicyType {
	return r.PolicyType
}

func (r DnsRecordSet) GetPolicyOptions() *jsonutils.JSONDict {
	return r.PolicyOptions
}

func (r DnsRecordSet) GetPolicyValue() TDnsPolicyValue {
	return r.PolicyValue
}

func (r DnsRecordSet) GetStatus() string {
	return r.Status
}

func (r DnsRecordSet) GetTTL() int64 {
	return r.Ttl
}

func (r DnsRecordSet) GetDnsType() TDnsType {
	return r.DnsType
}

func (r DnsRecordSet) GetEnabled() bool {
	return r.Enabled
}

func (r DnsRecordSet) GetMxPriority() int64 {
	return r.MxPriority
}

func (record DnsRecordSet) Equals(r DnsRecordSet) bool {
	if strings.ToLower(record.DnsName) != strings.ToLower(r.DnsName) {
		return false
	}
	if record.DnsType != r.DnsType {
		return false
	}
	if record.DnsValue != r.DnsValue {
		return false
	}
	if record.PolicyType != r.PolicyType {
		return false
	}
	if record.PolicyValue != r.PolicyValue {
		return false
	}
	if record.Ttl != r.Ttl {
		return false
	}
	if record.Enabled != r.Enabled {
		return false
	}
	if record.DnsType == DnsTypeMX && record.MxPriority != r.MxPriority {
		return false
	}
	return IsPolicyOptionEquals(record.PolicyOptions, r.PolicyOptions)
}

func (record DnsRecordSet) String() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", record.DnsType, record.DnsName, record.DnsValue, record.PolicyType, record.PolicyValue)
}

func IsPolicyOptionEquals(o1, o2 *jsonutils.JSONDict) bool {
	if o1 == nil {
		o1 = jsonutils.NewDict()
	}
	if o2 == nil {
		o2 = jsonutils.NewDict()
	}
	return o1.Equals(o2)
}

func CompareDnsRecordSet(iRecords []ICloudDnsRecordSet, local []DnsRecordSet, debug bool) ([]DnsRecordSet, []DnsRecordSet, []DnsRecordSet, []DnsRecordSet) {
	common, added, removed, updated := []DnsRecordSet{}, []DnsRecordSet{}, []DnsRecordSet{}, []DnsRecordSet{}

	localMaps := map[string]DnsRecordSet{}
	remoteMaps := map[string]DnsRecordSet{}
	for i := range iRecords {
		record := DnsRecordSet{
			ExternalId: iRecords[i].GetGlobalId(),

			DnsName:       iRecords[i].GetDnsName(),
			DnsType:       iRecords[i].GetDnsType(),
			DnsValue:      iRecords[i].GetDnsValue(),
			Status:        iRecords[i].GetStatus(),
			Enabled:       iRecords[i].GetEnabled(),
			Ttl:           iRecords[i].GetTTL(),
			MxPriority:    iRecords[i].GetMxPriority(),
			PolicyType:    iRecords[i].GetPolicyType(),
			PolicyValue:   iRecords[i].GetPolicyValue(),
			PolicyOptions: iRecords[i].GetPolicyOptions(),
		}
		remoteMaps[record.String()] = record
	}
	for i := range local {
		localMaps[local[i].String()] = local[i]
	}

	for key, record := range localMaps {
		remoteRecord, ok := remoteMaps[key]
		if ok {
			record.ExternalId = remoteRecord.ExternalId
			if !record.Equals(remoteRecord) {
				updated = append(updated, record)
			} else {
				common = append(common, record)
			}
		} else {
			added = append(added, record)
		}
	}

	for key, record := range remoteMaps {
		_, ok := localMaps[key]
		if !ok {
			removed = append(removed, record)
		}
	}

	return common, added, removed, updated
}
