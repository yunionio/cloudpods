// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//+build !linux

package dhcp4

import "errors"

// NewSnooperConn creates a Conn that listens on the given UDP ip:port.
//
// Unlike NewConn, NewSnooperConn does not bind to the ip:port,
// enabling the Conn to coexist with other services on the machine.
func NewSnooperConn(addr string) (*Conn, error) {
	return nil, errors.New("snooper Conns not supported on this OS")
}
