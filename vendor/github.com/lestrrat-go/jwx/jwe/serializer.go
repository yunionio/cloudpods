package jwe

import (
	"context"
	"encoding/json"

	"github.com/lestrrat-go/jwx/internal/pool"
	"github.com/pkg/errors"
)

// Compact encodes the given message into a JWE compact serialization format.
func Compact(m *Message, _ ...Option) ([]byte, error) {
	if len(m.recipients) != 1 {
		return nil, errors.New("wrong number of recipients for compact serialization")
	}

	recipient := m.recipients[0]

	// The protected header must be a merge between the message-wide
	// protected header AND the recipient header

	// There's something wrong if m.protectedHeaders is nil, but
	// it could happen
	if m.protectedHeaders == nil {
		return nil, errors.New("invalid protected header")
	}

	hcopy, err := mergeHeaders(context.TODO(), nil, m.protectedHeaders)
	if err != nil {
		return nil, errors.Wrap(err, "failed to copy protected header")
	}
	hcopy, err = mergeHeaders(context.TODO(), hcopy, m.unprotectedHeaders)
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge unprotected header")
	}
	hcopy, err = mergeHeaders(context.TODO(), hcopy, recipient.Headers())
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge recipient header")
	}

	protected, err := hcopy.Encode()
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode header")
	}

	encryptedKey, err := recipient.EncryptedKey().Base64Encode()
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode encryption key")
	}

	iv, err := m.initializationVector.Base64Encode()
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode iv")
	}

	cipher, err := m.cipherText.Base64Encode()
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode cipher text")
	}

	tag, err := m.tag.Base64Encode()
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode tag")
	}

	buf := pool.GetBytesBuffer()
	defer pool.ReleaseBytesBuffer(buf)

	buf.Grow(len(protected) + len(encryptedKey) + len(iv) + len(cipher) + len(tag) + 4)
	buf.Write(protected)
	buf.WriteByte('.')
	buf.Write(encryptedKey)
	buf.WriteByte('.')
	buf.Write(iv)
	buf.WriteByte('.')
	buf.Write(cipher)
	buf.WriteByte('.')
	buf.Write(tag)

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// JSON encodes the message into a JWE JSON serialization format.
func JSON(m *Message, options ...Option) ([]byte, error) {
	var pretty bool
	for _, option := range options {
		switch option.Name() {
		case optkeyPrettyJSONFormat:
			pretty = option.Value().(bool)
		}
	}

	if pretty {
		return json.MarshalIndent(m, "", "  ")
	}
	return json.Marshal(m)
}
