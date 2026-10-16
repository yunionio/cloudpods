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

package container_registries

import (
	"context"
	"strings"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/drivers/container_registries/client"
)

// validateRegistryAuthPair ensures username and password are both set or both
// empty. Returns (hasAuth, error).
func validateRegistryAuthPair(username, password string) (bool, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" && password == "" {
		return false, nil
	}
	if username == "" || password == "" {
		return false, httperrors.NewInputParameterError("username and password must both be set")
	}
	return true, nil
}

// loginRegistryWithAuth validates credentials against the registry using the
// docker login authentication challenge flow (Bearer / Basic).
func loginRegistryWithAuth(ctx context.Context, urlStr, username, password string) error {
	if err := client.LoginRegistry(ctx, urlStr, client.DockerAuthConfig{
		Username: username,
		Password: password,
	}); err != nil {
		return httperrors.NewInputParameterError("login registry %q failed: %v", urlStr, err)
	}
	return nil
}
