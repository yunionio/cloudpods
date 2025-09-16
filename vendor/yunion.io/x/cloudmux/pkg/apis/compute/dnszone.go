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

/*

Architecture For DnsZone


                                                   +-----------+                    +----------------+               +-----------+
                                                   | RecordSet |                    | TrafficPolicy  |               | RecordSet |                                         +-------------+
                                                   | (A)       |                    | (Aliyun)       |               | (TXT)     |                                         |  Vpc1       |
                                                   |           |                    |                |               |           |                                         |  (Aws)      |
                                                   |           |                    |                |               |           |                                         |             |
                  +-----------------------+        |           |                    |                |               |           |               +-----------------+       |             |
    API           |  DnsZone  example.com |        | RecordSet |                    | TrafficPolicy  |               | RecordSet |               | DnsZone abc.app |       |  Vpc2       |
                  |  (Public)             | ------>| (AAAA)    | -----------------> | (Tencent)      | <-------------| (CAA)     | <-------------| (Private)       |-----> |  (Tencent)  |
                  +-----------------------+        |           |                    |                |               |           |               +-----------------+       |             |
                          ^                        |           |                    |                |               |           |                       ^                 |             |
                          |                        |           |                    |                |               |           |                       |                 |  Vpc3       |
                          |                        | RecordSet |                    | TrafficPolicy  |               | RecordSet |                       |                 |  (Aws)      |
                          |                        | (NS)      |                    | (Aws)          |               | (PTR)     |                       |                 +-------------+
                          |                        +-----------+                    +----------------+               +-----------+                       |
                          |                                                                                                                              |
                          |                                                                                                                              |
                  ------------------------------------------------------------------------------------------------------------------------------------------------------------------------
                          |                                                                                                                              |
                          v                                                                                                                              |
                  +-----------------+                                                                                                                    |
                  |                 |                                                                                                                    |
                  |                 |            +----------+                                                                                            v
                  |  example.com <-------------> | Account1 |                                                             +----------+           +---------------+
                  |                 |            | (Aliyun) |                                                             | Account3 | <-------> |     abc.app   |
                  |                 |            +----------+                              +------------+                 | (Aws)    |           |               |
                  |                 |                                                      | Account2   |                 +----------+           |               |
                  |  example.com <-------------------------------------------------------> | (Tencent)  |                                        |               |
   Cache          |                 |                                                      +------------+                                        |               |
                  |                 |                                                                                                            |               |
                  |                 |            +----------+                                                                                    |               |
                  |  example.com <-------------> | Account4 | <--------------------------------------------------------------------------------> |     abc.app   |
                  |                 |            | (Aliyun) |                                                                                    |               |
                  |                 |            +----------+                                                                                    +---------------+
                  +-----------------+

               ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------


                                                *************                           ***************                   *************
                                            ****             ****                   ****               ****           ****             ****
 Public Cloud                                **     Aliyun    **                     **     Tencent     **             **	  Aws      **
                                            ****             ****                   ****               ****           ****             ****
                                                *************                           ***************                   *************


*/

const (
	DNS_ZONE_STATUS_AVAILABLE = "available" // 可用
	DNS_ZONE_STATUS_UNKNOWN   = "unknown"   // 未知
)
