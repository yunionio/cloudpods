package jwe

import (
	"context"
	"sync"

	"github.com/lestrrat-go/jwx/buffer"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/pdebug"
	"github.com/pkg/errors"
)

var encryptCtxPool = sync.Pool{
	New: func() interface{} {
		return &encryptCtx{}
	},
}

func getEncryptCtx() *encryptCtx {
	return encryptCtxPool.Get().(*encryptCtx)
}

func releaseEncryptCtx(ctx *encryptCtx) {
	ctx.contentEncrypter = nil
	ctx.generator = nil
	ctx.keyEncrypters = nil
	ctx.compress = jwa.NoCompress
	encryptCtxPool.Put(ctx)
}

// Encrypt takes the plaintext and encrypts into a JWE message.
func (e encryptCtx) Encrypt(plaintext []byte) (*Message, error) {
	bk, err := e.generator.Generate()
	if err != nil {
		if pdebug.Enabled {
			pdebug.Printf("Failed to generate key: %s", err)
		}
		return nil, errors.Wrap(err, "failed to generate key")
	}
	cek := bk.Bytes()

	if pdebug.Enabled {
		pdebug.Printf("Encrypt: generated cek len = %d", len(cek))
	}

	protected := NewHeaders()
	if err := protected.Set(ContentEncryptionKey, e.contentEncrypter.Algorithm()); err != nil {
		return nil, errors.Wrap(err, `failed to set "enc" in protected header`)
	}

	compression := e.compress
	if compression != jwa.NoCompress {
		if err := protected.Set(CompressionKey, compression); err != nil {
			return nil, errors.Wrap(err, `failed to set "zip" in protected header`)
		}
	}

	// In JWE, multiple recipients may exist -- they receive an
	// encrypted version of the CEK, using their key encryption
	// algorithm of choice.
	recipients := make([]Recipient, len(e.keyEncrypters))
	for i, enc := range e.keyEncrypters {
		r := NewRecipient()
		if err := r.Headers().Set(AlgorithmKey, enc.Algorithm()); err != nil {
			return nil, errors.Wrap(err, "failed to set header")
		}
		if v := enc.KeyID(); v != "" {
			if err := r.Headers().Set(KeyIDKey, v); err != nil {
				return nil, errors.Wrap(err, "failed to set header")
			}
		}
		enckey, err := enc.Encrypt(cek)
		if err != nil {
			if pdebug.Enabled {
				pdebug.Printf("Failed to encrypt key: %s", err)
			}
			return nil, errors.Wrap(err, `failed to encrypt key`)
		}
		if err := r.SetEncryptedKey(enckey.Bytes()); err != nil {
			return nil, errors.Wrap(err, "failed to set encrypted key")
		}
		if hp, ok := enckey.(populater); ok {
			if err := hp.Populate(r.Headers()); err != nil {
				return nil, errors.Wrap(err, "failed to populate")
			}
		}
		if pdebug.Enabled {
			pdebug.Printf("Encrypt: encrypted_key = %x (%d)", enckey.Bytes(), len(enckey.Bytes()))
		}
		recipients[i] = r
	}

	// If there's only one recipient, you want to include that in the
	// protected header
	if len(recipients) == 1 {
		h, err := mergeHeaders(context.TODO(), protected, recipients[0].Headers())
		if err != nil {
			return nil, errors.Wrap(err, "failed to merge protected headers")
		}
		protected = h
	}

	aad, err := protected.Encode()
	if err != nil {
		return nil, errors.Wrap(err, "failed to base64 encode protected headers")
	}

	plaintext, err = compress(plaintext, compression)
	if err != nil {
		return nil, errors.Wrap(err, `failed to compress payload before encryption`)
	}

	// ...on the other hand, there's only one content cipher.
	iv, ciphertext, tag, err := e.contentEncrypter.Encrypt(cek, plaintext, aad)
	if err != nil {
		if pdebug.Enabled {
			pdebug.Printf("Failed to encrypt: %s", err)
		}
		return nil, errors.Wrap(err, "failed to encrypt payload")
	}

	if pdebug.Enabled {
		pdebug.Printf("Encrypt.Encrypt: cek        = %x (%d)", cek, len(cek))
		pdebug.Printf("Encrypt.Encrypt: aad        = %x", aad)
		pdebug.Printf("Encrypt.Encrypt: ciphertext = %x", ciphertext)
		pdebug.Printf("Encrypt.Encrypt: iv         = %x", iv)
		pdebug.Printf("Encrypt.Encrypt: tag        = %x", tag)
	}

	msg := NewMessage()

	decodedAad, err := buffer.FromBase64(aad)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode base64")
	}
	if err := msg.Set(AuthenticatedDataKey, decodedAad.Bytes()); err != nil {
		return nil, errors.Wrapf(err, `failed to set %s`, AuthenticatedDataKey)
	}
	if err := msg.Set(CipherTextKey, ciphertext); err != nil {
		return nil, errors.Wrapf(err, `failed to set %s`, CipherTextKey)
	}
	if err := msg.Set(InitializationVectorKey, iv); err != nil {
		return nil, errors.Wrapf(err, `failed to set %s`, InitializationVectorKey)
	}
	if err := msg.Set(ProtectedHeadersKey, protected); err != nil {
		return nil, errors.Wrapf(err, `failed to set %s`, ProtectedHeadersKey)
	}
	if err := msg.Set(RecipientsKey, recipients); err != nil {
		return nil, errors.Wrapf(err, `failed to set %s`, RecipientsKey)
	}
	if err := msg.Set(TagKey, tag); err != nil {
		return nil, errors.Wrapf(err, `failed to set %s`, TagKey)
	}

	return msg, nil
}
