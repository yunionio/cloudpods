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

package seclib2

import "testing"

func TestAes256Cbc(t *testing.T) {
	key, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("generateRadomBytes 32 fail %s", err)
	}
	for _, c := range []string{
		"abc123",
		"a b",
		"a b ",
	} {
		cipher, err := AES_256.CbcEncode([]byte(c), key)
		if err != nil {
			t.Errorf("AES_256.CbcEncode fail %s", err)
		} else {
			d, err := AES_256.CbcDecode(cipher, key)
			if err != nil {
				t.Errorf("AES_256.CbcDecode fail %s", err)
			} else {
				if string(d) != c {
					t.Errorf("expect %s got %s", c, string(d))
				}
			}
		}
	}
}

func TestAes256CbcBase64(t *testing.T) {
	key, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("generateRadomBytes 32 fail %s", err)
	}
	for _, c := range []string{
		"abc123",
		"a b",
		"a b  ",
	} {
		cipher, err := AES_256.CbcEncodeBase64([]byte(c), key)
		if err != nil {
			t.Errorf("AES_256.CbcEncodeBase64 fail %s", err)
		} else {
			d, err := AES_256.CbcDecodeBase64(cipher, key)
			if err != nil {
				t.Errorf("AES_256.CbcDecodeBase64 fail %s", err)
			} else {
				if string(d) != c {
					t.Errorf("expect %s got %s", c, string(d))
				}
			}
		}
	}
}

func TestSm4Cbc(t *testing.T) {
	key, err := GenerateRandomBytes(16)
	if err != nil {
		t.Fatalf("generateRadomBytes 32 fail %s", err)
	}
	for _, c := range []string{
		"abc123",
		"a b",
		"a b ",
	} {
		cipher, err := SM4_128.CbcEncode([]byte(c), key)
		if err != nil {
			t.Errorf("SM4_128.CbcEncode fail %s", err)
		} else {
			d, err := SM4_128.CbcDecode(cipher, key)
			if err != nil {
				t.Errorf("SM4_128.CbcDecode fail %s", err)
			} else {
				if string(d) != c {
					t.Errorf("expect %s got %s", c, string(d))
				}
			}
		}
	}
}

func TestSm4CbcBase64(t *testing.T) {
	key, err := GenerateRandomBytes(16)
	if err != nil {
		t.Fatalf("generateRadomBytes 32 fail %s", err)
	}
	for _, c := range []string{
		"abc123",
		"a b",
		"a b  ",
	} {
		cipher, err := SM4_128.CbcEncodeBase64([]byte(c), key)
		if err != nil {
			t.Errorf("SM4_128.CbcEncodeBase64 fail %s", err)
		} else {
			d, err := SM4_128.CbcDecodeBase64(cipher, key)
			if err != nil {
				t.Errorf("SM4_128.CbcDecodeBase64 fail %s", err)
			} else {
				if string(d) != c {
					t.Errorf("expect %s got %s", c, string(d))
				}
			}
		}
	}
}
