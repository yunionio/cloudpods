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

type KinesisStream struct {
	StreamArn               string
	StreamCreationTimestamp int
	StreamModeDetails       struct {
		StreamMode string
	}
	StreamName   string
	StreamStatus string
	Shards       []struct {
		ShardId string
	}
}

func (self *SRegion) ListStreams() ([]KinesisStream, error) {
	params := map[string]interface{}{
		"MaxResults": "10000",
	}
	ret := []KinesisStream{}
	for {
		part := struct {
			StreamSummaries []KinesisStream
			NextToken       string
		}{}
		err := self.kinesisRequest("ListStreams", "/listStreams", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.StreamSummaries...)
		if len(part.StreamSummaries) == 0 || len(part.NextToken) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) DescribeStream(name string) (*KinesisStream, error) {
	params := map[string]interface{}{
		"StreamName": name,
	}
	ret := struct {
		StreamDescription KinesisStream
	}{}
	err := self.kinesisRequest("DescribeStream", "/describeStream", params, &ret)
	if err != nil {
		return nil, err
	}
	return &ret.StreamDescription, nil
}
