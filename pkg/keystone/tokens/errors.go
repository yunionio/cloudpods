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

package tokens

import "yunion.io/x/pkg/errors"

const (
	ErrVerMismatch        = errors.Error("version mismatch")
	ErrProjectDisabled    = errors.Error("project disabled")
	ErrUserDisabled       = errors.Error("user disabled")
	ErrInvalidToken       = errors.Error("invalid token")
	ErrExpiredToken       = errors.Error("expired token")
	ErrInvalidFernetToken = errors.Error("invalid fernet token")
	ErrInvalidAuthMethod  = errors.Error("invalid auth methods")
	ErrUserNotFound       = errors.Error("user not found")
	ErrDomainDisabled     = errors.Error("domain is disabled")
	ErrEmptyAuth          = errors.Error("empty auth request")
	ErrUserNotInProject   = errors.Error("user not in project")
	ErrInvalidAccessKeyId = errors.Error("invalid access key id")
	ErrExpiredAccessKey   = errors.Error("expired access key")
)
