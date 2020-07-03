// This file is auto-generated. DO NOT EDIT

package jwk

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/lestrrat-go/iter/mapiter"
	"github.com/lestrrat-go/jwx/internal/base64"
	"github.com/lestrrat-go/jwx/internal/iter"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/pkg/errors"
)

const (
	ECDSACrvKey = "crv"
	ECDSADKey   = "d"
	ECDSAXKey   = "x"
	ECDSAYKey   = "y"
)

type ECDSAPrivateKey interface {
	Key
	FromRaw(*ecdsa.PrivateKey) error
	Crv() jwa.EllipticCurveAlgorithm
	D() []byte
	X() []byte
	Y() []byte
	PublicKey() (ECDSAPublicKey, error)
}

type ecdsaPrivateKey struct {
	algorithm              *string // https://tools.ietf.org/html/rfc7517#section-4.4
	crv                    *jwa.EllipticCurveAlgorithm
	d                      []byte
	keyID                  *string           // https://tools.ietf.org/html/rfc7515#section-4.1.4
	keyUsage               *string           // https://tools.ietf.org/html/rfc7517#section-4.2
	keyops                 *KeyOperationList // https://tools.ietf.org/html/rfc7517#section-4.3
	x                      []byte
	x509CertChain          *CertificateChain // https://tools.ietf.org/html/rfc7515#section-4.1.6
	x509CertThumbprint     *string           // https://tools.ietf.org/html/rfc7515#section-4.1.7
	x509CertThumbprintS256 *string           // https://tools.ietf.org/html/rfc7515#section-4.1.8
	x509URL                *string           // https://tools.ietf.org/html/rfc7515#section-4.1.5
	y                      []byte
	privateParams          map[string]interface{}
}

type ecdsaPrivateKeyMarshalProxy struct {
	XkeyType                jwa.KeyType                 `json:"kty"`
	Xalgorithm              *string                     `json:"alg,omitempty"`
	Xcrv                    *jwa.EllipticCurveAlgorithm `json:"crv,omitempty"`
	Xd                      *string                     `json:"d,omitempty"`
	XkeyID                  *string                     `json:"kid,omitempty"`
	XkeyUsage               *string                     `json:"use,omitempty"`
	Xkeyops                 *KeyOperationList           `json:"key_ops,omitempty"`
	Xx                      *string                     `json:"x,omitempty"`
	Xx509CertChain          *CertificateChain           `json:"x5c,omitempty"`
	Xx509CertThumbprint     *string                     `json:"x5t,omitempty"`
	Xx509CertThumbprintS256 *string                     `json:"x5t#S256,omitempty"`
	Xx509URL                *string                     `json:"x5u,omitempty"`
	Xy                      *string                     `json:"y,omitempty"`
}

func (h ecdsaPrivateKey) KeyType() jwa.KeyType {
	return jwa.EC
}

func (h *ecdsaPrivateKey) Algorithm() string {
	if h.algorithm != nil {
		return *(h.algorithm)
	}
	return ""
}

func (h *ecdsaPrivateKey) Crv() jwa.EllipticCurveAlgorithm {
	if h.crv != nil {
		return *(h.crv)
	}
	return jwa.InvalidEllipticCurve
}

func (h *ecdsaPrivateKey) D() []byte {
	return h.d
}

func (h *ecdsaPrivateKey) KeyID() string {
	if h.keyID != nil {
		return *(h.keyID)
	}
	return ""
}

func (h *ecdsaPrivateKey) KeyUsage() string {
	if h.keyUsage != nil {
		return *(h.keyUsage)
	}
	return ""
}

func (h *ecdsaPrivateKey) KeyOps() KeyOperationList {
	if h.keyops != nil {
		return *(h.keyops)
	}
	return nil
}

func (h *ecdsaPrivateKey) X() []byte {
	return h.x
}

func (h *ecdsaPrivateKey) X509CertChain() []*x509.Certificate {
	if h.x509CertChain != nil {
		return h.x509CertChain.Get()
	}
	return nil
}

func (h *ecdsaPrivateKey) X509CertThumbprint() string {
	if h.x509CertThumbprint != nil {
		return *(h.x509CertThumbprint)
	}
	return ""
}

func (h *ecdsaPrivateKey) X509CertThumbprintS256() string {
	if h.x509CertThumbprintS256 != nil {
		return *(h.x509CertThumbprintS256)
	}
	return ""
}

func (h *ecdsaPrivateKey) X509URL() string {
	if h.x509URL != nil {
		return *(h.x509URL)
	}
	return ""
}

func (h *ecdsaPrivateKey) Y() []byte {
	return h.y
}

func (h *ecdsaPrivateKey) iterate(ctx context.Context, ch chan *HeaderPair) {
	defer close(ch)

	var pairs []*HeaderPair
	pairs = append(pairs, &HeaderPair{Key: "kty", Value: jwa.EC})
	if h.algorithm != nil {
		pairs = append(pairs, &HeaderPair{Key: AlgorithmKey, Value: *(h.algorithm)})
	}
	if h.crv != nil {
		pairs = append(pairs, &HeaderPair{Key: ECDSACrvKey, Value: *(h.crv)})
	}
	if h.d != nil {
		pairs = append(pairs, &HeaderPair{Key: ECDSADKey, Value: h.d})
	}
	if h.keyID != nil {
		pairs = append(pairs, &HeaderPair{Key: KeyIDKey, Value: *(h.keyID)})
	}
	if h.keyUsage != nil {
		pairs = append(pairs, &HeaderPair{Key: KeyUsageKey, Value: *(h.keyUsage)})
	}
	if h.keyops != nil {
		pairs = append(pairs, &HeaderPair{Key: KeyOpsKey, Value: *(h.keyops)})
	}
	if h.x != nil {
		pairs = append(pairs, &HeaderPair{Key: ECDSAXKey, Value: h.x})
	}
	if h.x509CertChain != nil {
		pairs = append(pairs, &HeaderPair{Key: X509CertChainKey, Value: *(h.x509CertChain)})
	}
	if h.x509CertThumbprint != nil {
		pairs = append(pairs, &HeaderPair{Key: X509CertThumbprintKey, Value: *(h.x509CertThumbprint)})
	}
	if h.x509CertThumbprintS256 != nil {
		pairs = append(pairs, &HeaderPair{Key: X509CertThumbprintS256Key, Value: *(h.x509CertThumbprintS256)})
	}
	if h.x509URL != nil {
		pairs = append(pairs, &HeaderPair{Key: X509URLKey, Value: *(h.x509URL)})
	}
	if h.y != nil {
		pairs = append(pairs, &HeaderPair{Key: ECDSAYKey, Value: h.y})
	}
	for k, v := range h.privateParams {
		pairs = append(pairs, &HeaderPair{Key: k, Value: v})
	}
	for _, pair := range pairs {
		select {
		case <-ctx.Done():
			return
		case ch <- pair:
		}
	}
}

func (h *ecdsaPrivateKey) PrivateParams() map[string]interface{} {
	return h.privateParams
}

func (h *ecdsaPrivateKey) Get(name string) (interface{}, bool) {
	switch name {
	case KeyTypeKey:
		return h.KeyType(), true
	case AlgorithmKey:
		if h.algorithm == nil {
			return nil, false
		}
		return *(h.algorithm), true
	case ECDSACrvKey:
		if h.crv == nil {
			return nil, false
		}
		return *(h.crv), true
	case ECDSADKey:
		if h.d == nil {
			return nil, false
		}
		return h.d, true
	case KeyIDKey:
		if h.keyID == nil {
			return nil, false
		}
		return *(h.keyID), true
	case KeyUsageKey:
		if h.keyUsage == nil {
			return nil, false
		}
		return *(h.keyUsage), true
	case KeyOpsKey:
		if h.keyops == nil {
			return nil, false
		}
		return *(h.keyops), true
	case ECDSAXKey:
		if h.x == nil {
			return nil, false
		}
		return h.x, true
	case X509CertChainKey:
		if h.x509CertChain == nil {
			return nil, false
		}
		return *(h.x509CertChain), true
	case X509CertThumbprintKey:
		if h.x509CertThumbprint == nil {
			return nil, false
		}
		return *(h.x509CertThumbprint), true
	case X509CertThumbprintS256Key:
		if h.x509CertThumbprintS256 == nil {
			return nil, false
		}
		return *(h.x509CertThumbprintS256), true
	case X509URLKey:
		if h.x509URL == nil {
			return nil, false
		}
		return *(h.x509URL), true
	case ECDSAYKey:
		if h.y == nil {
			return nil, false
		}
		return h.y, true
	default:
		v, ok := h.privateParams[name]
		return v, ok
	}
}

func (h *ecdsaPrivateKey) Set(name string, value interface{}) error {
	switch name {
	case "kty":
		return nil
	case AlgorithmKey:
		switch v := value.(type) {
		case string:
			h.algorithm = &v
		case fmt.Stringer:
			tmp := v.String()
			h.algorithm = &tmp
		default:
			return errors.Errorf(`invalid type for %s key: %T`, AlgorithmKey, value)
		}
		return nil
	case ECDSACrvKey:
		if v, ok := value.(jwa.EllipticCurveAlgorithm); ok {
			h.crv = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, ECDSACrvKey, value)
	case ECDSADKey:
		if v, ok := value.([]byte); ok {
			h.d = v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, ECDSADKey, value)
	case KeyIDKey:
		if v, ok := value.(string); ok {
			h.keyID = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, KeyIDKey, value)
	case KeyUsageKey:
		if v, ok := value.(string); ok {
			h.keyUsage = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, KeyUsageKey, value)
	case KeyOpsKey:
		var acceptor KeyOperationList
		if err := acceptor.Accept(value); err != nil {
			return errors.Wrapf(err, `invalid value for %s key`, KeyOpsKey)
		}
		h.keyops = &acceptor
		return nil
	case ECDSAXKey:
		if v, ok := value.([]byte); ok {
			h.x = v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, ECDSAXKey, value)
	case X509CertChainKey:
		var acceptor CertificateChain
		if err := acceptor.Accept(value); err != nil {
			return errors.Wrapf(err, `invalid value for %s key`, X509CertChainKey)
		}
		h.x509CertChain = &acceptor
		return nil
	case X509CertThumbprintKey:
		if v, ok := value.(string); ok {
			h.x509CertThumbprint = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, X509CertThumbprintKey, value)
	case X509CertThumbprintS256Key:
		if v, ok := value.(string); ok {
			h.x509CertThumbprintS256 = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, X509CertThumbprintS256Key, value)
	case X509URLKey:
		if v, ok := value.(string); ok {
			h.x509URL = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, X509URLKey, value)
	case ECDSAYKey:
		if v, ok := value.([]byte); ok {
			h.y = v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, ECDSAYKey, value)
	default:
		if h.privateParams == nil {
			h.privateParams = map[string]interface{}{}
		}
		h.privateParams[name] = value
	}
	return nil
}

func (h *ecdsaPrivateKey) UnmarshalJSON(buf []byte) error {
	var proxy ecdsaPrivateKeyMarshalProxy
	if err := json.Unmarshal(buf, &proxy); err != nil {
		return errors.Wrap(err, `failed to unmarshal ecdsaPrivateKey`)
	}
	if proxy.XkeyType != jwa.EC {
		return errors.Errorf(`invalid kty value for ECDSAPrivateKey (%s)`, proxy.XkeyType)
	}
	h.algorithm = proxy.Xalgorithm
	h.crv = proxy.Xcrv
	if proxy.Xd == nil {
		return errors.New(`required field d is missing`)
	}
	if h.d = nil; proxy.Xd != nil {
		decoded, err := base64.DecodeString(*(proxy.Xd))
		if err != nil {
			return errors.Wrap(err, `failed to decode base64 value for d`)
		}
		h.d = decoded
	}
	h.keyID = proxy.XkeyID
	h.keyUsage = proxy.XkeyUsage
	h.keyops = proxy.Xkeyops
	if proxy.Xx == nil {
		return errors.New(`required field x is missing`)
	}
	if h.x = nil; proxy.Xx != nil {
		decoded, err := base64.DecodeString(*(proxy.Xx))
		if err != nil {
			return errors.Wrap(err, `failed to decode base64 value for x`)
		}
		h.x = decoded
	}
	h.x509CertChain = proxy.Xx509CertChain
	h.x509CertThumbprint = proxy.Xx509CertThumbprint
	h.x509CertThumbprintS256 = proxy.Xx509CertThumbprintS256
	h.x509URL = proxy.Xx509URL
	if proxy.Xy == nil {
		return errors.New(`required field y is missing`)
	}
	if h.y = nil; proxy.Xy != nil {
		decoded, err := base64.DecodeString(*(proxy.Xy))
		if err != nil {
			return errors.Wrap(err, `failed to decode base64 value for y`)
		}
		h.y = decoded
	}
	var m map[string]interface{}
	if err := json.Unmarshal(buf, &m); err != nil {
		return errors.Wrap(err, `failed to parse privsate parameters`)
	}
	delete(m, `kty`)
	delete(m, AlgorithmKey)
	delete(m, ECDSACrvKey)
	delete(m, ECDSADKey)
	delete(m, KeyIDKey)
	delete(m, KeyUsageKey)
	delete(m, KeyOpsKey)
	delete(m, ECDSAXKey)
	delete(m, X509CertChainKey)
	delete(m, X509CertThumbprintKey)
	delete(m, X509CertThumbprintS256Key)
	delete(m, X509URLKey)
	delete(m, ECDSAYKey)
	h.privateParams = m
	return nil
}

func (h ecdsaPrivateKey) MarshalJSON() ([]byte, error) {
	var proxy ecdsaPrivateKeyMarshalProxy
	proxy.XkeyType = jwa.EC
	proxy.Xalgorithm = h.algorithm
	proxy.Xcrv = h.crv
	if len(h.d) > 0 {
		v := base64.EncodeToString(h.d)
		proxy.Xd = &v
	}
	proxy.XkeyID = h.keyID
	proxy.XkeyUsage = h.keyUsage
	proxy.Xkeyops = h.keyops
	if len(h.x) > 0 {
		v := base64.EncodeToString(h.x)
		proxy.Xx = &v
	}
	proxy.Xx509CertChain = h.x509CertChain
	proxy.Xx509CertThumbprint = h.x509CertThumbprint
	proxy.Xx509CertThumbprintS256 = h.x509CertThumbprintS256
	proxy.Xx509URL = h.x509URL
	if len(h.y) > 0 {
		v := base64.EncodeToString(h.y)
		proxy.Xy = &v
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(proxy); err != nil {
		return nil, errors.Wrap(err, `failed to encode proxy to JSON`)
	}
	hasContent := buf.Len() > 3 // encoding/json always adds a newline, so "{}\n" is the empty hash
	if l := len(h.privateParams); l > 0 {
		buf.Truncate(buf.Len() - 2)
		keys := make([]string, 0, l)
		for k := range h.privateParams {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if hasContent || i > 0 {
				fmt.Fprintf(&buf, `,`)
			}
			fmt.Fprintf(&buf, `%s:`, strconv.Quote(k))
			if err := enc.Encode(h.privateParams[k]); err != nil {
				return nil, errors.Wrapf(err, `failed to encode private param %s`, k)
			}
		}
		fmt.Fprintf(&buf, `}`)
	}
	return buf.Bytes(), nil
}

func (h *ecdsaPrivateKey) Iterate(ctx context.Context) HeaderIterator {
	ch := make(chan *HeaderPair)
	go h.iterate(ctx, ch)
	return mapiter.New(ch)
}

func (h *ecdsaPrivateKey) Walk(ctx context.Context, visitor HeaderVisitor) error {
	return iter.WalkMap(ctx, h, visitor)
}

func (h *ecdsaPrivateKey) AsMap(ctx context.Context) (map[string]interface{}, error) {
	return iter.AsMap(ctx, h)
}

type ECDSAPublicKey interface {
	Key
	FromRaw(*ecdsa.PublicKey) error
	Crv() jwa.EllipticCurveAlgorithm
	X() []byte
	Y() []byte
}

type ecdsaPublicKey struct {
	algorithm              *string // https://tools.ietf.org/html/rfc7517#section-4.4
	crv                    *jwa.EllipticCurveAlgorithm
	keyID                  *string           // https://tools.ietf.org/html/rfc7515#section-4.1.4
	keyUsage               *string           // https://tools.ietf.org/html/rfc7517#section-4.2
	keyops                 *KeyOperationList // https://tools.ietf.org/html/rfc7517#section-4.3
	x                      []byte
	x509CertChain          *CertificateChain // https://tools.ietf.org/html/rfc7515#section-4.1.6
	x509CertThumbprint     *string           // https://tools.ietf.org/html/rfc7515#section-4.1.7
	x509CertThumbprintS256 *string           // https://tools.ietf.org/html/rfc7515#section-4.1.8
	x509URL                *string           // https://tools.ietf.org/html/rfc7515#section-4.1.5
	y                      []byte
	privateParams          map[string]interface{}
}

type ecdsaPublicKeyMarshalProxy struct {
	XkeyType                jwa.KeyType                 `json:"kty"`
	Xalgorithm              *string                     `json:"alg,omitempty"`
	Xcrv                    *jwa.EllipticCurveAlgorithm `json:"crv,omitempty"`
	XkeyID                  *string                     `json:"kid,omitempty"`
	XkeyUsage               *string                     `json:"use,omitempty"`
	Xkeyops                 *KeyOperationList           `json:"key_ops,omitempty"`
	Xx                      *string                     `json:"x,omitempty"`
	Xx509CertChain          *CertificateChain           `json:"x5c,omitempty"`
	Xx509CertThumbprint     *string                     `json:"x5t,omitempty"`
	Xx509CertThumbprintS256 *string                     `json:"x5t#S256,omitempty"`
	Xx509URL                *string                     `json:"x5u,omitempty"`
	Xy                      *string                     `json:"y,omitempty"`
}

func (h ecdsaPublicKey) KeyType() jwa.KeyType {
	return jwa.EC
}

func (h *ecdsaPublicKey) Algorithm() string {
	if h.algorithm != nil {
		return *(h.algorithm)
	}
	return ""
}

func (h *ecdsaPublicKey) Crv() jwa.EllipticCurveAlgorithm {
	if h.crv != nil {
		return *(h.crv)
	}
	return jwa.InvalidEllipticCurve
}

func (h *ecdsaPublicKey) KeyID() string {
	if h.keyID != nil {
		return *(h.keyID)
	}
	return ""
}

func (h *ecdsaPublicKey) KeyUsage() string {
	if h.keyUsage != nil {
		return *(h.keyUsage)
	}
	return ""
}

func (h *ecdsaPublicKey) KeyOps() KeyOperationList {
	if h.keyops != nil {
		return *(h.keyops)
	}
	return nil
}

func (h *ecdsaPublicKey) X() []byte {
	return h.x
}

func (h *ecdsaPublicKey) X509CertChain() []*x509.Certificate {
	if h.x509CertChain != nil {
		return h.x509CertChain.Get()
	}
	return nil
}

func (h *ecdsaPublicKey) X509CertThumbprint() string {
	if h.x509CertThumbprint != nil {
		return *(h.x509CertThumbprint)
	}
	return ""
}

func (h *ecdsaPublicKey) X509CertThumbprintS256() string {
	if h.x509CertThumbprintS256 != nil {
		return *(h.x509CertThumbprintS256)
	}
	return ""
}

func (h *ecdsaPublicKey) X509URL() string {
	if h.x509URL != nil {
		return *(h.x509URL)
	}
	return ""
}

func (h *ecdsaPublicKey) Y() []byte {
	return h.y
}

func (h *ecdsaPublicKey) iterate(ctx context.Context, ch chan *HeaderPair) {
	defer close(ch)

	var pairs []*HeaderPair
	pairs = append(pairs, &HeaderPair{Key: "kty", Value: jwa.EC})
	if h.algorithm != nil {
		pairs = append(pairs, &HeaderPair{Key: AlgorithmKey, Value: *(h.algorithm)})
	}
	if h.crv != nil {
		pairs = append(pairs, &HeaderPair{Key: ECDSACrvKey, Value: *(h.crv)})
	}
	if h.keyID != nil {
		pairs = append(pairs, &HeaderPair{Key: KeyIDKey, Value: *(h.keyID)})
	}
	if h.keyUsage != nil {
		pairs = append(pairs, &HeaderPair{Key: KeyUsageKey, Value: *(h.keyUsage)})
	}
	if h.keyops != nil {
		pairs = append(pairs, &HeaderPair{Key: KeyOpsKey, Value: *(h.keyops)})
	}
	if h.x != nil {
		pairs = append(pairs, &HeaderPair{Key: ECDSAXKey, Value: h.x})
	}
	if h.x509CertChain != nil {
		pairs = append(pairs, &HeaderPair{Key: X509CertChainKey, Value: *(h.x509CertChain)})
	}
	if h.x509CertThumbprint != nil {
		pairs = append(pairs, &HeaderPair{Key: X509CertThumbprintKey, Value: *(h.x509CertThumbprint)})
	}
	if h.x509CertThumbprintS256 != nil {
		pairs = append(pairs, &HeaderPair{Key: X509CertThumbprintS256Key, Value: *(h.x509CertThumbprintS256)})
	}
	if h.x509URL != nil {
		pairs = append(pairs, &HeaderPair{Key: X509URLKey, Value: *(h.x509URL)})
	}
	if h.y != nil {
		pairs = append(pairs, &HeaderPair{Key: ECDSAYKey, Value: h.y})
	}
	for k, v := range h.privateParams {
		pairs = append(pairs, &HeaderPair{Key: k, Value: v})
	}
	for _, pair := range pairs {
		select {
		case <-ctx.Done():
			return
		case ch <- pair:
		}
	}
}

func (h *ecdsaPublicKey) PrivateParams() map[string]interface{} {
	return h.privateParams
}

func (h *ecdsaPublicKey) Get(name string) (interface{}, bool) {
	switch name {
	case KeyTypeKey:
		return h.KeyType(), true
	case AlgorithmKey:
		if h.algorithm == nil {
			return nil, false
		}
		return *(h.algorithm), true
	case ECDSACrvKey:
		if h.crv == nil {
			return nil, false
		}
		return *(h.crv), true
	case KeyIDKey:
		if h.keyID == nil {
			return nil, false
		}
		return *(h.keyID), true
	case KeyUsageKey:
		if h.keyUsage == nil {
			return nil, false
		}
		return *(h.keyUsage), true
	case KeyOpsKey:
		if h.keyops == nil {
			return nil, false
		}
		return *(h.keyops), true
	case ECDSAXKey:
		if h.x == nil {
			return nil, false
		}
		return h.x, true
	case X509CertChainKey:
		if h.x509CertChain == nil {
			return nil, false
		}
		return *(h.x509CertChain), true
	case X509CertThumbprintKey:
		if h.x509CertThumbprint == nil {
			return nil, false
		}
		return *(h.x509CertThumbprint), true
	case X509CertThumbprintS256Key:
		if h.x509CertThumbprintS256 == nil {
			return nil, false
		}
		return *(h.x509CertThumbprintS256), true
	case X509URLKey:
		if h.x509URL == nil {
			return nil, false
		}
		return *(h.x509URL), true
	case ECDSAYKey:
		if h.y == nil {
			return nil, false
		}
		return h.y, true
	default:
		v, ok := h.privateParams[name]
		return v, ok
	}
}

func (h *ecdsaPublicKey) Set(name string, value interface{}) error {
	switch name {
	case "kty":
		return nil
	case AlgorithmKey:
		switch v := value.(type) {
		case string:
			h.algorithm = &v
		case fmt.Stringer:
			tmp := v.String()
			h.algorithm = &tmp
		default:
			return errors.Errorf(`invalid type for %s key: %T`, AlgorithmKey, value)
		}
		return nil
	case ECDSACrvKey:
		if v, ok := value.(jwa.EllipticCurveAlgorithm); ok {
			h.crv = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, ECDSACrvKey, value)
	case KeyIDKey:
		if v, ok := value.(string); ok {
			h.keyID = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, KeyIDKey, value)
	case KeyUsageKey:
		if v, ok := value.(string); ok {
			h.keyUsage = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, KeyUsageKey, value)
	case KeyOpsKey:
		var acceptor KeyOperationList
		if err := acceptor.Accept(value); err != nil {
			return errors.Wrapf(err, `invalid value for %s key`, KeyOpsKey)
		}
		h.keyops = &acceptor
		return nil
	case ECDSAXKey:
		if v, ok := value.([]byte); ok {
			h.x = v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, ECDSAXKey, value)
	case X509CertChainKey:
		var acceptor CertificateChain
		if err := acceptor.Accept(value); err != nil {
			return errors.Wrapf(err, `invalid value for %s key`, X509CertChainKey)
		}
		h.x509CertChain = &acceptor
		return nil
	case X509CertThumbprintKey:
		if v, ok := value.(string); ok {
			h.x509CertThumbprint = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, X509CertThumbprintKey, value)
	case X509CertThumbprintS256Key:
		if v, ok := value.(string); ok {
			h.x509CertThumbprintS256 = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, X509CertThumbprintS256Key, value)
	case X509URLKey:
		if v, ok := value.(string); ok {
			h.x509URL = &v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, X509URLKey, value)
	case ECDSAYKey:
		if v, ok := value.([]byte); ok {
			h.y = v
			return nil
		}
		return errors.Errorf(`invalid value for %s key: %T`, ECDSAYKey, value)
	default:
		if h.privateParams == nil {
			h.privateParams = map[string]interface{}{}
		}
		h.privateParams[name] = value
	}
	return nil
}

func (h *ecdsaPublicKey) UnmarshalJSON(buf []byte) error {
	var proxy ecdsaPublicKeyMarshalProxy
	if err := json.Unmarshal(buf, &proxy); err != nil {
		return errors.Wrap(err, `failed to unmarshal ecdsaPublicKey`)
	}
	if proxy.XkeyType != jwa.EC {
		return errors.Errorf(`invalid kty value for ECDSAPublicKey (%s)`, proxy.XkeyType)
	}
	h.algorithm = proxy.Xalgorithm
	h.crv = proxy.Xcrv
	h.keyID = proxy.XkeyID
	h.keyUsage = proxy.XkeyUsage
	h.keyops = proxy.Xkeyops
	if proxy.Xx == nil {
		return errors.New(`required field x is missing`)
	}
	if h.x = nil; proxy.Xx != nil {
		decoded, err := base64.DecodeString(*(proxy.Xx))
		if err != nil {
			return errors.Wrap(err, `failed to decode base64 value for x`)
		}
		h.x = decoded
	}
	h.x509CertChain = proxy.Xx509CertChain
	h.x509CertThumbprint = proxy.Xx509CertThumbprint
	h.x509CertThumbprintS256 = proxy.Xx509CertThumbprintS256
	h.x509URL = proxy.Xx509URL
	if proxy.Xy == nil {
		return errors.New(`required field y is missing`)
	}
	if h.y = nil; proxy.Xy != nil {
		decoded, err := base64.DecodeString(*(proxy.Xy))
		if err != nil {
			return errors.Wrap(err, `failed to decode base64 value for y`)
		}
		h.y = decoded
	}
	var m map[string]interface{}
	if err := json.Unmarshal(buf, &m); err != nil {
		return errors.Wrap(err, `failed to parse privsate parameters`)
	}
	delete(m, `kty`)
	delete(m, AlgorithmKey)
	delete(m, ECDSACrvKey)
	delete(m, KeyIDKey)
	delete(m, KeyUsageKey)
	delete(m, KeyOpsKey)
	delete(m, ECDSAXKey)
	delete(m, X509CertChainKey)
	delete(m, X509CertThumbprintKey)
	delete(m, X509CertThumbprintS256Key)
	delete(m, X509URLKey)
	delete(m, ECDSAYKey)
	h.privateParams = m
	return nil
}

func (h ecdsaPublicKey) MarshalJSON() ([]byte, error) {
	var proxy ecdsaPublicKeyMarshalProxy
	proxy.XkeyType = jwa.EC
	proxy.Xalgorithm = h.algorithm
	proxy.Xcrv = h.crv
	proxy.XkeyID = h.keyID
	proxy.XkeyUsage = h.keyUsage
	proxy.Xkeyops = h.keyops
	if len(h.x) > 0 {
		v := base64.EncodeToString(h.x)
		proxy.Xx = &v
	}
	proxy.Xx509CertChain = h.x509CertChain
	proxy.Xx509CertThumbprint = h.x509CertThumbprint
	proxy.Xx509CertThumbprintS256 = h.x509CertThumbprintS256
	proxy.Xx509URL = h.x509URL
	if len(h.y) > 0 {
		v := base64.EncodeToString(h.y)
		proxy.Xy = &v
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(proxy); err != nil {
		return nil, errors.Wrap(err, `failed to encode proxy to JSON`)
	}
	hasContent := buf.Len() > 3 // encoding/json always adds a newline, so "{}\n" is the empty hash
	if l := len(h.privateParams); l > 0 {
		buf.Truncate(buf.Len() - 2)
		keys := make([]string, 0, l)
		for k := range h.privateParams {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if hasContent || i > 0 {
				fmt.Fprintf(&buf, `,`)
			}
			fmt.Fprintf(&buf, `%s:`, strconv.Quote(k))
			if err := enc.Encode(h.privateParams[k]); err != nil {
				return nil, errors.Wrapf(err, `failed to encode private param %s`, k)
			}
		}
		fmt.Fprintf(&buf, `}`)
	}
	return buf.Bytes(), nil
}

func (h *ecdsaPublicKey) Iterate(ctx context.Context) HeaderIterator {
	ch := make(chan *HeaderPair)
	go h.iterate(ctx, ch)
	return mapiter.New(ch)
}

func (h *ecdsaPublicKey) Walk(ctx context.Context, visitor HeaderVisitor) error {
	return iter.WalkMap(ctx, h, visitor)
}

func (h *ecdsaPublicKey) AsMap(ctx context.Context) (map[string]interface{}, error) {
	return iter.AsMap(ctx, h)
}
