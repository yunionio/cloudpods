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

package notify

type DingtalkInfo struct {
	AgentId   string
	AppKey    string
	AppSecret string
}

type FailedRecord struct {
	Contact string
	Reason  string
}

type SendParam struct {
	Contact        string
	Topic          string
	Title          string
	Message        string
	Priority       string
	RemoteTemplate string
}

type BatchSendParam struct {
	Contacts       []string
	Title          string
	Message        string
	Priority       string
	RemoteTemplate string
}

type SSenderBase struct {
	//ConfigCache *SConfigCache
	WorkerChan chan struct{}
}

func (self *SSenderBase) Do(f func() error) error {
	self.WorkerChan <- struct{}{}
	defer func() {
		<-self.WorkerChan
	}()
	return f()
}
