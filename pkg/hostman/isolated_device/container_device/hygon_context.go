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

package container_device

import "sync"

var (
	hygonCtxMu          sync.Mutex
	hygonPendingGuestId string
	hygonPendingCtrName string
)

// SetHygonContainerContext stores guest/container identity for vdev cache directory naming.
func SetHygonContainerContext(guestId, containerName string) {
	hygonCtxMu.Lock()
	defer hygonCtxMu.Unlock()
	hygonPendingGuestId = guestId
	hygonPendingCtrName = containerName
}

// ClearHygonContainerContext clears pending guest/container identity.
func ClearHygonContainerContext() {
	hygonCtxMu.Lock()
	defer hygonCtxMu.Unlock()
	hygonPendingGuestId = ""
	hygonPendingCtrName = ""
}

func getHygonContainerContext() (guestId, containerName string) {
	hygonCtxMu.Lock()
	defer hygonCtxMu.Unlock()
	return hygonPendingGuestId, hygonPendingCtrName
}
