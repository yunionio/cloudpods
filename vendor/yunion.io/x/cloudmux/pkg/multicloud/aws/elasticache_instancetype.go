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

package aws

type SElastiCacheInstanceType struct {
	Vcpu      int
	MemeoryMb float32
}

var redisInstanceType = map[string]SElastiCacheInstanceType{
	"cache.t4g.micro":  {Vcpu: 2, MemeoryMb: 0.5 * 1024},
	"cache.t4g.small":  {Vcpu: 2, MemeoryMb: 1403},
	"cache.t4g.medium": {Vcpu: 2, MemeoryMb: 3164},

	"cache.t3.micro":  {Vcpu: 2, MemeoryMb: 0.5 * 1024},
	"cache.t3.small":  {Vcpu: 2, MemeoryMb: 1403},
	"cache.t3.medium": {Vcpu: 2, MemeoryMb: 3164},

	"cache.t2.micro":  {Vcpu: 1, MemeoryMb: 0.5 * 1024},
	"cache.t2.small":  {Vcpu: 1, MemeoryMb: 1587},
	"cache.t2.medium": {Vcpu: 2, MemeoryMb: 3297},

	"cache.t4.micro": {Vcpu: 2, MemeoryMb: 0.5 * 1024},
	"cache.t4.small": {Vcpu: 2, MemeoryMb: 1403},

	"cache.r6g.large":    {Vcpu: 0, MemeoryMb: 13384},
	"cache.r6g.xlarge":   {Vcpu: 0, MemeoryMb: 26952},
	"cache.r6g.2xlarge":  {Vcpu: 0, MemeoryMb: 52.82 * 1024},
	"cache.r6g.4xlarge":  {Vcpu: 0, MemeoryMb: 105.81 * 1024},
	"cache.r6g.8xlarge":  {Vcpu: 0, MemeoryMb: 209.55 * 1024},
	"cache.r6g.12xlarge": {Vcpu: 0, MemeoryMb: 317.77 * 1024},
	"cache.r6g.16xlarge": {Vcpu: 0, MemeoryMb: 419.09 * 1024},

	"cache.m6g.large":    {Vcpu: 0, MemeoryMb: 6.38 * 1024},
	"cache.m6g.xlarge":   {Vcpu: 0, MemeoryMb: 12.93 * 1024},
	"cache.m6g.2xlarge":  {Vcpu: 0, MemeoryMb: 26.04 * 1024},
	"cache.m6g.4xlarge":  {Vcpu: 0, MemeoryMb: 52.26 * 1024},
	"cache.m6g.8xlarge":  {Vcpu: 0, MemeoryMb: 105.81 * 1024},
	"cache.m6g.12xlarge": {Vcpu: 0, MemeoryMb: 157.12 * 1024},
	"cache.m6g.16xlarge": {Vcpu: 0, MemeoryMb: 209.55 * 1024},

	"cache.r5.large":    {Vcpu: 0, MemeoryMb: 13.07 * 1024},
	"cache.r5.xlarge":   {Vcpu: 0, MemeoryMb: 26.32 * 1024},
	"cache.r5.2xlarge":  {Vcpu: 0, MemeoryMb: 52.82 * 1024},
	"cache.r5.4xlarge":  {Vcpu: 0, MemeoryMb: 105.81 * 1024},
	"cache.r5.12xlarge": {Vcpu: 0, MemeoryMb: 317.77 * 1024},
	"cache.r5.24xlarge": {Vcpu: 0, MemeoryMb: 635.61 * 1024},

	"cache.r4.large":    {Vcpu: 0, MemeoryMb: 12.3 * 1024},
	"cache.r4.xlarge":   {Vcpu: 0, MemeoryMb: 25.05 * 1024},
	"cache.r4.2xlarge":  {Vcpu: 0, MemeoryMb: 50.47 * 1024},
	"cache.r4.4xlarge":  {Vcpu: 0, MemeoryMb: 101.38 * 1024},
	"cache.r4.8xlarge":  {Vcpu: 0, MemeoryMb: 203.06 * 1024},
	"cache.r4.16xlarge": {Vcpu: 0, MemeoryMb: 407 * 1024},

	"cache.m4.large":    {Vcpu: 0, MemeoryMb: 6 * 1024},
	"cache.m4.xlarge":   {Vcpu: 0, MemeoryMb: 14 * 1024},
	"cache.m4.2xlarge":  {Vcpu: 0, MemeoryMb: 30 * 1024},
	"cache.m4.4xlarge":  {Vcpu: 0, MemeoryMb: 61 * 1024},
	"cache.m4.10xlarge": {Vcpu: 0, MemeoryMb: 155 * 1024},
}

type ElastiCacheEngineVersion struct {
	CacheParameterGroupFamily     string `xml:"CacheParameterGroupFamily"`
	Engine                        string `xml:"Engine"`
	CacheEngineVersionDescription string `xml:"CacheEngineVersionDescription"`
	EngineVersion                 string `xml:"EngineVersion"`
}

func (self *SRegion) GetElastiCacheEngineVersion(engine string) ([]ElastiCacheEngineVersion, error) {
	params := map[string]string{
		//"DefaultOnly":               "true",
		"Engine": engine,
	}
	ret := []ElastiCacheEngineVersion{}
	for {
		part := struct {
			CacheEngineVersions []ElastiCacheEngineVersion `xml:"CacheEngineVersions>CacheEngineVersion"`
			Marker              string                     `xml:"Marker"`
		}{}
		err := self.ecRequest("DescribeCacheEngineVersions", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.CacheEngineVersions...)
		if len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}
