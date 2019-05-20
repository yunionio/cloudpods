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
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/golang-plus/uuid"
	"github.com/vmihailenco/msgpack"
)

type TScopedPayloadVersion byte

const (
	SProjectScopedPayloadVersion = TScopedPayloadVersion(2)
	SDomainScopedPayloadVersion  = TScopedPayloadVersion(1)
	SUnscopedPayloadVersion      = TScopedPayloadVersion(0)
)

var (
	ErrVerMismatch = errors.New("version mismatch")
)

type ITokenPayload interface {
	Unmarshal(tk []byte) error
	Decode(token *SAuthToken)
	Encode() ([]byte, error)
}

type SUuidPayload struct {
	IsUuid  bool
	Payload string
}

func (up *SUuidPayload) parse(hex string) {
	u, err := uuid.Parse(hex)
	if err != nil {
		up.IsUuid = false
		up.Payload = hex
	} else {
		up.IsUuid = true
		up.Payload = string(u[:])
	}
}

func convertUuidBytesToHex(bs []byte) string {
	u := uuid.UUID{}
	copy(u[:], bs)
	return u.Format(uuid.StyleWithoutDash)
}

func (u *SUuidPayload) getUuid() string {
	if u.IsUuid {
		return convertUuidBytesToHex([]byte(u.Payload))
	} else {
		return string(u.Payload)
	}
}

/*
 * msgpack payload
 *
 * https://github.com/msgpack/msgpack/blob/master/spec.md
 */
type SProjectScopedPayload struct {
	Version   TScopedPayloadVersion
	UserId    SUuidPayload
	Method    byte
	ProjectId SUuidPayload
	ExpiresAt float64
	AuditIds  []string
}

func (p *SProjectScopedPayload) Unmarshal(tk []byte) error {
	err := msgpack.Unmarshal(tk, p)
	if err != nil {
		return err
	}
	if p.Version != SProjectScopedPayloadVersion {
		return ErrVerMismatch
	}
	return nil
}

func (p *SProjectScopedPayload) Decode(token *SAuthToken) {
	token.UserId = p.UserId.getUuid()
	token.Method = authMethodId2Str(p.Method)
	token.ProjectId = p.ProjectId.getUuid()
	token.ExpiresAt = time.Unix(int64(p.ExpiresAt), 0).UTC()
	token.AuditIds = auditBytes2Strings(p.AuditIds)
}

func msgpackEncoder(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf).StructAsArray(true).UseCompactEncoding(true)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *SProjectScopedPayload) Encode() ([]byte, error) {
	return msgpackEncoder(p)
}

type SDomainScopedPayload struct {
	Version   TScopedPayloadVersion
	UserId    SUuidPayload
	Method    byte
	DomainId  SUuidPayload
	ExpiresAt float64
	AuditIds  []string
}

func (p *SDomainScopedPayload) Unmarshal(tk []byte) error {
	err := msgpack.Unmarshal(tk, p)
	if err != nil {
		return err
	}
	if p.Version != SDomainScopedPayloadVersion {
		return ErrVerMismatch
	}
	return nil
}

func (p *SDomainScopedPayload) Decode(token *SAuthToken) {
	token.UserId = p.UserId.getUuid()
	token.Method = authMethodId2Str(p.Method)
	token.DomainId = p.DomainId.getUuid()
	token.ExpiresAt = time.Unix(int64(p.ExpiresAt), 0).UTC()
	token.AuditIds = auditBytes2Strings(p.AuditIds)
}

func (p *SDomainScopedPayload) Encode() ([]byte, error) {
	return msgpackEncoder(p)
}

type SUnscopedPayload struct {
	Version   TScopedPayloadVersion
	UserId    SUuidPayload
	Method    byte
	ExpiresAt float64
	AuditIds  []string
}

func (p *SUnscopedPayload) Unmarshal(tk []byte) error {
	err := msgpack.Unmarshal(tk, p)
	if err != nil {
		return err
	}
	if p.Version != SUnscopedPayloadVersion {
		return ErrVerMismatch
	}
	return nil
}

func (p *SUnscopedPayload) Decode(token *SAuthToken) {
	token.UserId = p.UserId.getUuid()
	token.Method = authMethodId2Str(p.Method)
	token.ExpiresAt = time.Unix(int64(p.ExpiresAt), 0).UTC()
	token.AuditIds = auditBytes2Strings(p.AuditIds)
}

func (p *SUnscopedPayload) Encode() ([]byte, error) {
	return msgpackEncoder(p)
}

func auditString2Bytes(str string) string {
	bt, _ := base64.URLEncoding.DecodeString(str + "==")
	return string(bt)
}

func auditBytes2String(bs string) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString([]byte(bs)), "=")
}

func auditStrings2Bytes(strs []string) []string {
	ret := make([]string, len(strs))
	for i := range strs {
		ret[i] = auditString2Bytes(strs[i])
	}
	return ret
}

func auditBytes2Strings(bs []string) []string {
	ret := make([]string, len(bs))
	for i := range bs {
		ret[i] = auditBytes2String(bs[i])
	}
	return ret
}
