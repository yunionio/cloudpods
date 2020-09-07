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

package saml // import "yunion.io/x/onecloud/pkg/cloudid/saml"

/*

                 +-----------------+                                                                                             +----------------+
                 | CloudId Service |                                                                                             | Region Service |
                 +-----------------+                                                                                             +----------------+


                                                                                                                                                       +----------+
                                                                                                                                                       |EnableSaml|
                +-------------------+                                                                                            +------------+        +----------+
                | Cloudaccount      |                                                                                            |Cloudaccount|
                | (enabled|disable) |                                                                                            +------------+        +-----------+
                +----________-------+                                                                                                                  |DisableSaml|
             _______/        \_______                                                                                                                  +-----------+
         ___/                        \____
   +--------------+                 +--------------+
   | SamlProvider |                 | SamlProvider |
   | (available)  |                 | (not match)  |
   +--------------+                 +--------------+






             Saml Check CronJob Task

                     +-------+
                     | Start |
                     +---|---+
                         |
                         |
                         v
              +------------------------+       Yes         +----------------------------------------+      No          +---------------------+
              | Is account enable saml | ----------------> | Is account has available saml provider |----------------> | Create saml provider|
              +------------------------+                   +----------------------------------------+                  +---------------------+
                         |                                                     |                                                  |
                         | No                                                  | Yes                                              |
                         |                                                     |                                                  |
                         v                                                     |                                                  |
                     +------+                                                  |                                                  |
                     | End  | <-----------------------------------------------<----------------------------------------------------
                     +------+





            Saml Auth Login

                     +-------+
                     | Start |
                     +-------+
                         |
                         |
                         v
               +-----------------------+       Yes     +-----------------------------------------------+
               |Is account enable saml |-------------> |Prepare tmp Role and set expired time for user |
               +-----------------------+               +-----------------------------------------------+
                         | No                                               |
                         |                                                  |
                         |                                                  |
                         |                                                  v
                         |                                         +------------------+
                         |                                         | Auth for console |
                         |                                         +------------------+
                         |                                                  |
                         |                                                  |
                         v                                                  |
                      +------+                                              |
                      | End  |<----------------------------------------------
                      +------+
































*/
