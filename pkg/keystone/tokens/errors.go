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

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	ErrVerMismatch        = errors.Error("[auth] version mismatch")
	ErrProjectDisabled    = errors.Error("[auth] project disabled")
	ErrUserDisabled       = errors.Error("[auth] user disabled")
	ErrInvalidToken       = errors.Error("[auth] invalid token")
	ErrExpiredToken       = errors.Error("[auth] expired token")
	ErrInvalidFernetToken = errors.Error("[auth] invalid fernet token")
	ErrInvalidAuthMethod  = errors.Error("[auth] invalid auth methods")
	ErrUserNotFound       = errors.Error("[auth] user not found")
	ErrDomainDisabled     = errors.Error("[auth] domain is disabled")
	ErrEmptyAuth          = errors.Error("[auth] empty auth request")
	ErrUserNotInProject   = errors.Error("[auth] user not in project")
	ErrInvalidAccessKeyId = errors.Error("[auth] invalid access key id")
	ErrExpiredAccessKey   = errors.Error("[auth] expired access key")
	ErrTokenNotFound      = errors.Error("[auth] token not found")
)

func init() {
	httperrors.RegisterErrorHttpCode(ErrVerMismatch, 401)
	httperrors.RegisterErrorHttpCode(ErrProjectDisabled, 401)
	httperrors.RegisterErrorHttpCode(ErrUserDisabled, 401)
	httperrors.RegisterErrorHttpCode(ErrInvalidToken, 401)
	httperrors.RegisterErrorHttpCode(ErrExpiredToken, 401)
	httperrors.RegisterErrorHttpCode(ErrInvalidFernetToken, 401)
	httperrors.RegisterErrorHttpCode(ErrInvalidAuthMethod, 401)
	httperrors.RegisterErrorHttpCode(ErrUserNotFound, 401)
	httperrors.RegisterErrorHttpCode(ErrDomainDisabled, 401)
	httperrors.RegisterErrorHttpCode(ErrEmptyAuth, 401)
	httperrors.RegisterErrorHttpCode(ErrUserNotInProject, 401)
	httperrors.RegisterErrorHttpCode(ErrInvalidAccessKeyId, 401)
	httperrors.RegisterErrorHttpCode(ErrExpiredAccessKey, 401)
	httperrors.RegisterErrorHttpCode(ErrTokenNotFound, 401)
}
