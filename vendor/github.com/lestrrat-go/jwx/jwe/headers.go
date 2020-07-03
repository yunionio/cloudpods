package jwe

import (
	"context"
	"encoding/json"

	"github.com/lestrrat-go/iter/mapiter"
	"github.com/lestrrat-go/jwx/buffer"
	"github.com/lestrrat-go/jwx/internal/iter"
	"github.com/pkg/errors"
)

type isZeroer interface {
	isZero() bool
}

func (h *stdHeaders) isZero() bool {
	return h.agreementPartyUInfo == nil &&
		h.agreementPartyVInfo == nil &&
		h.algorithm == nil &&
		h.compression == nil &&
		h.contentEncryption == nil &&
		h.contentType == nil &&
		h.critical == nil &&
		h.ephemeralPublicKey == nil &&
		h.jwk == nil &&
		h.jwkSetURL == nil &&
		h.keyID == nil &&
		h.typ == nil &&
		h.x509CertChain == nil &&
		h.x509CertThumbprint == nil &&
		h.x509CertThumbprintS256 == nil &&
		h.x509URL == nil &&
		len(h.privateParams) == 0
}

// Iterate returns a channel that successively returns all the
// header name and values.
func (h *stdHeaders) Iterate(ctx context.Context) Iterator {
	ch := make(chan *HeaderPair)
	go h.iterate(ctx, ch)
	return mapiter.New(ch)
}

func (h *stdHeaders) Walk(ctx context.Context, visitor Visitor) error {
	return iter.WalkMap(ctx, h, visitor)
}

func (h *stdHeaders) AsMap(ctx context.Context) (map[string]interface{}, error) {
	return iter.AsMap(ctx, h)
}

func (h *stdHeaders) Encode() ([]byte, error) {
	buf, err := json.Marshal(h)
	if err != nil {
		return nil, errors.Wrap(err, `failed to marshal headers to JSON prior to encoding`)
	}

	buf, err = buffer.Buffer(buf).Base64Encode()
	if err != nil {
		return nil, errors.Wrap(err, "failed to base64 encode encoded header")
	}
	return buf, nil
}

func (h *stdHeaders) Decode(buf []byte) error {
	// base64 json string -> json object representation of header
	b, err := buffer.FromBase64(buf)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal base64 encoded buffer")
	}

	if err := json.Unmarshal(b.Bytes(), h); err != nil {
		return errors.Wrap(err, "failed to unmarshal buffer")
	}

	return nil
}
