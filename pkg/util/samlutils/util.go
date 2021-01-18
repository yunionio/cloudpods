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

package samlutils

import (
	"bytes"
	"compress/flate"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/ma314smith/signedxml"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

func compressString(in string) string {
	buf := new(bytes.Buffer)
	compressor, _ := flate.NewWriter(buf, 9)
	compressor.Write([]byte(in))
	compressor.Close()
	return buf.String()
}

func decompressString(in string) string {
	buf := new(bytes.Buffer)
	decompressor := flate.NewReader(strings.NewReader(in))
	io.Copy(buf, decompressor)
	decompressor.Close()
	return buf.String()
}

func compress(in []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	compressor, _ := flate.NewWriter(buf, 9)
	_, err := compressor.Write(in)
	if err != nil {
		return nil, errors.Wrap(err, "compressor.Write")
	}
	compressor.Close()
	return buf.Bytes(), nil
}

func decompress(in []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	decompressor := flate.NewReader(bytes.NewReader(in))
	_, err := io.Copy(buf, decompressor)
	if err != nil {
		return nil, errors.Wrap(err, "io.Copy")
	}
	decompressor.Close()
	return buf.Bytes(), nil
}

func SAMLDecode(input string) ([]byte, error) {
	reqBytes, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, errors.Wrap(err, "base64.StdEncoding.DecodeString")
	}
	return func() []byte {
		// Azure no need to decompress
		plainText, err := decompress(reqBytes)
		if err != nil {
			log.Warningf("decompress %s error: %v", string(reqBytes), err)
			return reqBytes
		}
		return plainText
	}(), nil
}

func SAMLEncode(input []byte) (string, error) {
	comp, err := compress(input)
	if err != nil {
		return "", errors.Wrap(err, "compress")
	}
	return base64.StdEncoding.EncodeToString(comp), nil
}

func SAMLForm(action string, attrs map[string]string) string {
	form := strings.Builder{}
	// form.WriteString(`<!DOCTYPE html><html lang="en-US"><body>`)
	form.WriteString(`<form id="saml_submit_form" method="POST" action="`)
	form.WriteString(action)
	form.WriteString(`">`)
	for k, v := range attrs {
		form.WriteString(fmt.Sprintf("<input type=\"hidden\" name=\"%s\" value=\"%s\" />", k, v))
	}
	form.WriteString(`<input type="submit" value="Submit" />`)
	form.WriteString("</form><script><!--\n")
	form.WriteString("document.getElementById('saml_submit_form').submit();\n")
	form.WriteString(`//--></script>`)
	// form.WriteString(`</body></html>`)
	return form.String()
}

func SignXML(xmlstr string, privateKey *rsa.PrivateKey) (string, error) {
	signer, err := signedxml.NewSigner(string(xmlstr))
	if err != nil {
		return "", errors.Wrap(err, "signedxml.NewSigner")
	}
	signed, err := signer.Sign(privateKey)
	if err != nil {
		return "", errors.Wrap(err, "signer.Sign")
	}
	return signed, nil
}

func ValidateXML(signed string) ([]string, error) {
	validator, err := signedxml.NewValidator(signed)
	if err != nil {
		return nil, errors.Wrap(err, "signedxml.NewValidator")
	}
	validXMLs, err := validator.ValidateReferences()
	if err != nil {
		return nil, errors.Wrap(err, "validator.ValidateReferences")
	}
	return validXMLs, nil
}

func GenerateSAMLId() string {
	return "_" + utils.GenRequestId(16)
}
