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

package forwarder

import (
	"google.golang.org/grpc"

	"yunion.io/x/pkg/errors"

	pb "yunion.io/x/onecloud/pkg/hostman/guestman/forwarder/api"
)

func NewClient(target string) (pb.ForwarderClient, error) {
	cc, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "grpc Dial")
	}
	c := pb.NewForwarderClient(cc)
	return c, nil
}
