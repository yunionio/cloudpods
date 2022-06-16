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

package plugins

type SReceiver struct {
	Contact  string `protobuf:"bytes,1,opt,name=Contact,proto3" json:"Contact,omitempty"`
	DomainId string `protobuf:"bytes,2,opt,name=DomainId,proto3" json:"DomainId,omitempty"`
}

type SendParams struct {
	Receiver       *SReceiver `protobuf:"bytes,1,opt,name=Receiver,proto3" json:"Receiver,omitempty"`
	Topic          string     `protobuf:"bytes,2,opt,name=Topic,proto3" json:"Topic,omitempty"`
	Title          string     `protobuf:"bytes,3,opt,name=Title,proto3" json:"Title,omitempty"`
	Message        string     `protobuf:"bytes,4,opt,name=Message,proto3" json:"Message,omitempty"`
	Priority       string     `protobuf:"bytes,5,opt,name=Priority,proto3" json:"Priority,omitempty"`
	RemoteTemplate string     `protobuf:"bytes,6,opt,name=RemoteTemplate,proto3" json:"RemoteTemplate,omitempty"`
}
