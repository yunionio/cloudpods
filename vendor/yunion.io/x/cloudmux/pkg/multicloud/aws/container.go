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

func (self *SRegion) ListClusters() ([]string, error) {
	params := map[string]interface{}{
		"maxResults": 100,
	}
	ret := []string{}
	for {
		part := struct {
			ClusterArns []string
			NextToken   string
		}{}
		err := self.ecsRequest("ListClusters", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ClusterArns...)
		if len(part.ClusterArns) == 0 || len(part.NextToken) == 0 {
			break
		}
		params["nextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) ListServices(cluster string) ([]string, error) {
	params := map[string]interface{}{
		"cluster":    cluster,
		"maxResults": 100,
	}
	ret := []string{}
	for {
		part := struct {
			ServiceArns []string
			NextToken   string
		}{}
		err := self.ecsRequest("ListServices", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ServiceArns...)
		if len(part.ServiceArns) == 0 || len(part.NextToken) == 0 {
			break
		}
		params["nextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) ListTasks(cluster string) ([]string, error) {
	params := map[string]interface{}{
		"cluster":    cluster,
		"maxResults": 100,
	}
	ret := []string{}
	for {
		part := struct {
			TaskArns  []string
			NextToken string
		}{}
		err := self.ecsRequest("ListTasks", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.TaskArns...)
		if len(part.TaskArns) == 0 || len(part.NextToken) == 0 {
			break
		}
		params["nextToken"] = part.NextToken
	}
	return ret, nil
}
