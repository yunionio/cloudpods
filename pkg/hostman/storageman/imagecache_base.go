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

package storageman

import (
	"context"

	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
)

type IImageCache interface {
	GetPath() string
	GetName() string
	Load() bool
	Acquire(ctx context.Context, zone, srcUrl, format, checksum string) bool
	Release()
	Remove(ctx context.Context) error
	GetImageId() string

	GetDesc() *remotefile.SImageDesc
}
