package jws

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type encodedSignatureProxy struct {
	Protected string          `json:"protected,omitempty"`
	Headers   json.RawMessage `json:"header,omitempty"`
	Signature string          `json:"signature,omitempty"`
}

func (sig *encodedSignature) UnmarshalJSON(buf []byte) error {
	var proxy encodedSignatureProxy
	if err := json.Unmarshal(buf, &proxy); err != nil {
		return errors.Wrap(err, `failed to unmarshal into temporary struct`)
	}

	var h Headers
	if len(proxy.Headers) > 0 {
		h = NewHeaders()
		if err := json.Unmarshal(proxy.Headers, h); err != nil {
			return errors.Wrap(err, `failed to unmarshal headers`)
		}
	}

	// XXX: sigh, dream of the day when we kill public fields
	sig.Protected = proxy.Protected
	sig.Signature = proxy.Signature
	sig.Headers = h

	return nil
}
