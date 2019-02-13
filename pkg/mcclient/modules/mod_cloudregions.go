package modules

import (
	"yunion.io/x/jsonutils"

	"sort"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCloudregionManager struct {
	ResourceManager
}

var (
	Cloudregions SCloudregionManager
)

type sCityCounter struct {
	City    string
	Regions int
}

type cityCounters []sCityCounter

func (cc cityCounters) Len() int      { return len(cc) }
func (cc cityCounters) Swap(i, j int) { cc[i], cc[j] = cc[j], cc[i] }
func (cc cityCounters) Less(i, j int) bool {
	if cc[i].Regions != cc[j].Regions {
		return cc[i].Regions > cc[j].Regions
	}
	return cc[i].City < cc[j].City
}

func (this *SCloudregionManager) GetRegionCities(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	paramsDict := params.(*jsonutils.JSONDict)
	paramsDict.Set("limit", jsonutils.NewInt(0))

	listResult, err := this.List(session, params)
	if err != nil {
		return nil, err
	}

	cities := make(map[string]int)
	for i := range listResult.Data {
		cityStr, _ := listResult.Data[i].GetString("city")
		if len(cityStr) > 0 {
			if _, ok := cities[cityStr]; ok {
				cities[cityStr] += 1
			} else {
				cities[cityStr] = 1
			}
		}
	}

	cityList := make([]sCityCounter, len(cities))
	i := 0
	for k, v := range cities {
		cityList[i] = sCityCounter{City: k, Regions: v}
		i += 1
	}

	sort.Sort(cityCounters(cityList))

	return jsonutils.Marshal(cityList), nil
}

func init() {
	Cloudregions = SCloudregionManager{
		NewComputeManager("cloudregion", "cloudregions",
			[]string{"ID", "Name", "Enabled", "Status", "Provider",
				"Latitude", "Longitude", "City", "Country_Code",
				"vpc_count", "zone_count", "guest_count", "guest_increment_count",
				"External_Id"},
			[]string{}),
	}

	registerCompute(&Cloudregions)
}
