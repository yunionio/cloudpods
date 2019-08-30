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

const (
	NAT_STAUTS_AVAILABLE     = "available"     //可用
	NAT_STATUS_ALLOCATE      = "allocate"      //创建中
	NAT_STATUS_DEPLOYING     = "deploying"     //配置中
	NAT_STATUS_UNKNOWN       = "unknown"       //未知状态
	NAT_STATUS_FAILED        = "failed"        //创建失败
	NAT_STATUS_DELETED       = "deleted"       //删除
	NAT_STATUS_DELETING      = "deleting"      //删除中
	NAT_STATUS_DELETE_FAILED = "delete_failed" //删除失败

	NAT_SPEC_SMALL  = "small"  //小型
	NAT_SPEC_MIDDLE = "middle" //中型
	NAT_SPEC_LARGE  = "large"  //大型
	NAT_SPEC_XLARGE = "xlarge" //超大型

	QCLOUD_NAT_SPEC_SMALL  = "small"
	QCLOUD_NAT_SPEC_MIDDLE = "middle"
	QCLOUD_NAT_SPEC_LARGE  = "large"
)
