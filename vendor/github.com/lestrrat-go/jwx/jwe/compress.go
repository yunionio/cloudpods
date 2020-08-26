package jwe

import (
	"bytes"
	"compress/flate"
	"io/ioutil"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/pkg/errors"
)

func uncompress(plaintext []byte) ([]byte, error) {
	return ioutil.ReadAll(flate.NewReader(bytes.NewReader(plaintext)))
}

func compress(plaintext []byte, alg jwa.CompressionAlgorithm) ([]byte, error) {
	if alg == jwa.NoCompress {
		return plaintext, nil
	}

	var output bytes.Buffer
	w, _ := flate.NewWriter(&output, 1)
	in := plaintext
	for len(in) > 0 {
		n, err := w.Write(in)
		if err != nil {
			return nil, errors.Wrap(err, `failed to write to compression writer`)
		}
		in = in[n:]
	}
	if err := w.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close compression writer")
	}
	return output.Bytes(), nil
}
