package quotas

import (
	"testing"
)

func TestRelation(t *testing.T) {
	keys := []SZonalCloudResourceKeys{
		//Top :=
		{},
		// Domain1 :=
		{
			SRegionalCloudRegionKeys: SRegionalCloudRegionKeys{
				SCloudResourceKeys: SCloudResourceKeys{
					SBaseQuotaKeys: SBaseQuotaKeys{
						DomainId: "domain1",
					},
				},
			},
		},
		// Domain2 :=
		{
			SRegionalCloudRegionKeys: SRegionalCloudRegionKeys{
				SCloudResourceKeys: SCloudResourceKeys{
					SBaseQuotaKeys: SBaseQuotaKeys{
						DomainId: "domain2",
					},
				},
			},
		},
		// Project11 :=
		{
			SRegionalCloudRegionKeys: SRegionalCloudRegionKeys{
				SCloudResourceKeys: SCloudResourceKeys{
					SBaseQuotaKeys: SBaseQuotaKeys{
						DomainId:  "domain1",
						ProjectId: "project1",
					},
				},
			},
		},
		// Project12 :=
		{
			SRegionalCloudRegionKeys: SRegionalCloudRegionKeys{
				SCloudResourceKeys: SCloudResourceKeys{
					SBaseQuotaKeys: SBaseQuotaKeys{
						DomainId:  "domain1",
						ProjectId: "project2",
					},
				},
			},
		},
		// Project21 :=
		{
			SRegionalCloudRegionKeys: SRegionalCloudRegionKeys{
				SCloudResourceKeys: SCloudResourceKeys{
					SBaseQuotaKeys: SBaseQuotaKeys{
						DomainId:  "domain2",
						ProjectId: "project1",
					},
				},
			},
		},
		// Project11Region1 :=
		{
			SRegionalCloudRegionKeys: SRegionalCloudRegionKeys{
				SCloudResourceKeys: SCloudResourceKeys{
					SBaseQuotaKeys: SBaseQuotaKeys{
						DomainId:  "domain1",
						ProjectId: "project1",
					},
				},
				RegionId: "region1",
			},
		},
		// Project11Region2 :=
		{
			SRegionalCloudRegionKeys: SRegionalCloudRegionKeys{
				SCloudResourceKeys: SCloudResourceKeys{
					SBaseQuotaKeys: SBaseQuotaKeys{
						DomainId:  "domain1",
						ProjectId: "project1",
					},
				},
				RegionId: "region2",
			},
		},
		// Project11Aliyun :=
		{
			SRegionalCloudRegionKeys: SRegionalCloudRegionKeys{
				SCloudResourceKeys: SCloudResourceKeys{
					SBaseQuotaKeys: SBaseQuotaKeys{
						DomainId:  "domain1",
						ProjectId: "project1",
					},
					Provider: "Aliyun",
				},
			},
		},
		// Project11AliyunRegion1 :=
		{
			SRegionalCloudRegionKeys: SRegionalCloudRegionKeys{
				SCloudResourceKeys: SCloudResourceKeys{
					SBaseQuotaKeys: SBaseQuotaKeys{
						DomainId:  "domain1",
						ProjectId: "project1",
					},
					Provider: "Aliyun",
				},
				RegionId: "region1",
			},
		},
	}
	want := [][]TQuotaKeysRelation{
		{
			QuotaKeysEqual,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
		},
		{
			QuotaKeysBelong,
			QuotaKeysEqual,
			QuotaKeysExclude,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysExclude,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
		},
		{
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysEqual,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysContain,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
		},
		{
			QuotaKeysBelong,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysEqual,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
			QuotaKeysContain,
		},
		{
			QuotaKeysBelong,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysEqual,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
		},
		{
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysEqual,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
		},
		{
			QuotaKeysBelong,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysEqual,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysContain,
		},
		{
			QuotaKeysBelong,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysEqual,
			QuotaKeysExclude,
			QuotaKeysExclude,
		},
		{
			QuotaKeysBelong,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysEqual,
			QuotaKeysContain,
		},
		{
			QuotaKeysBelong,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysBelong,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysExclude,
			QuotaKeysEqual,
		},
	}
	for i := range keys {
		for j := range keys {
			rel := relation(keys[i], keys[j])
			if rel != want[i][j] {
				t.Errorf("%#v %#v got %s want %s", keys[i], keys[j], rel, want[i][j])
			}
		}
	}
}
