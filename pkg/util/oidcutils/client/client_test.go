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

package client

import (
	"crypto/rsa"
	"testing"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/oidcutils"
)

var (
	key = `{
  "keys": [
    {
      "use": "sig",
      "kty": "RSA",
      "kid": "f0183bc084bb3ab231ad0f9db924fa5b71c7a5ee",
      "alg": "RS256",
      "n": "7KOBiQRunXqOfqJ2IwuYg6sO2_fIqQaeifDFwMgGApUNJuCepwal0Y-pxziXwCYSw_ErYX40mZDomplJf5mf3YOSYlfpM_mhP4w8ozj0rGfx1oE7_8Wn5T2hZhS-xHkdcWulkzjGSCo-oVrm6OCGmj842RCp94cAC6G7Adjs7agf4KdB2d0vY8zwlHxh4WljujL87ORdx8xSzocB0yeGL-Vnd7-j_v96whZlmlURsWi4codmA_Xc5grEeLmGBamjYuUn1FKCdBpHnB4jBcfj-Tw2IsC8iJxIniLgCEEsveHsb6Qt2sl_9WbJN4Ix0-cYVuC0jazIMDori__wFIothw",
      "e": "AQAB"
    },
    {
      "use": "sig",
      "kty": "RSA",
      "kid": "7603bda55f6d88563caf80a235a7a96f0b8d0d56",
      "alg": "RS256",
      "n": "wO7bi63IzvIonpa48easQzcrdpl77qWX7K9O2jjf5KGojpW4T4Dpgxh9lA3bgPrlkNin8mImvGBsrtdWwSbDE9eMbqY2I_qL9T3UNFe7Rhsb8Oj-voEe2LErMkAWgWKwN8QlC0avUB0IMfRGkiaaLnljNaP_agtmLDc1cEwqcUmU0YENHWBZpE8m_3CAsjAuzAbtWokDjRXQ1llSmiOsUpTKB12-Bm9Hl-ebkMhi8OwYBOQ_y3sQCdJcYjIC5Guvfk_t3_acnVamKkx0-avVSAnAub2GqS-jBFGfRgckM8X5UJ7S5SHJXg6kFe2sWaO1WO2Rk25J2i7XIKlIdxTIJQ",
      "e": "AQAB"
    },
    {
      "use": "sig",
      "kty": "RSA",
      "kid": "31315fd7cfb32fef6091f037cba8b356bb734000",
      "alg": "RS256",
      "n": "rrqvVbnEXMtfdf2VxRcejSkaCHLnTT19bWzicA-_a6GlHrw8giPev-BYfYr9PF1XgOgIYu_867DlzQJQ0H93_z3OkfDvtQiafbg4hjI4OXN8-tf5kQnDra79jtWuHQfR1hTE1JGiRpbYgV1yCKvF2f7hNYilVfq0tVgW1q8I--vd8PBQTbM6Ty_vJoBSjApFTnpgGF2kZjWGVGRaxz_G7eyAHGtksHEmRCCOABun0oi0dsuazva9u2OdHo_ghVFSr2R3aBy81a7Xdcttid68ydyEb1EgLzOUuWGPtZ-OVdIsqbtXpr5mO4fR2l1y4g1WUzvSCz7oJofjAQ_580aPyw",
      "e": "AQAB"
    }
  ]
}`
	token = `{"access_token":"eyJhbGciOiJSUzI1NiIsImtpZCI6ImIwNjg1MWU0MmE3ZTI4NjY1ODAwODZlZGUyOTc5M2I4YTY1ZGY0MWQifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4Iiwic3ViIjoiQ2cwd0xUTTROUzB5T0RBNE9TMHdFZ1J0YjJOciIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxNTkzMzI2MzE5LCJpYXQiOjE1OTMyMzk5MTksImF0X2hhc2giOiJpaEJCNzVMUWdNS05uc016UEVPdUNRIiwiZW1haWwiOiJraWxnb3JlQGtpbGdvcmUudHJvdXQiLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IktpbGdvcmUgVHJvdXQifQ.0m1C3HaK42DuZ_Hqs0OqnyMG8Z1AyKhU1vx-a4jXpEbXSshmClIPbBd-T-MZs87_XwBGMqiJ8fNLardAe98bUtmDA1b4nlQWW1MZCEpU-6n0VXmOTMjVje6G1kj3GPWRoeY8qGKRJU3RzWeih946Y1AsES90JNBh9wYt2UvTATlseFFxHgZ_QSAdTiNNogFNOB6lK8V9yUVWbJ2gZMRA1-WtQkWyc0HJKAryDoZdlvrbiOTQUX1RB1cMP1xbDnguZ3AJurdfBDTWbAiKM55dQAck632lTAOFkUve_gtp3dqcm0WORKnaEUeyvXXoTI8A6b-8A6ht5VN_JbJYzC63BQ","token_type":"bearer","expires_in":86399,"id_token":"eyJhbGciOiJSUzI1NiIsImtpZCI6ImIwNjg1MWU0MmE3ZTI4NjY1ODAwODZlZGUyOTc5M2I4YTY1ZGY0MWQifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4Iiwic3ViIjoiQ2cwd0xUTTROUzB5T0RBNE9TMHdFZ1J0YjJOciIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxNTkzMzI2MzE5LCJpYXQiOjE1OTMyMzk5MTksImF0X2hhc2giOiItdEUzMmRqLVE1ODNKcnNCOFpMRnJ3IiwiZW1haWwiOiJraWxnb3JlQGtpbGdvcmUudHJvdXQiLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IktpbGdvcmUgVHJvdXQifQ.wNRb6Cyj35n2L4CjQ2-nj7cHd5YdeSIJLaFQHhzHabh3coErQnLnUOQ1Iu5b_Q1RSyHYzEZqMkMPsydNsjVjGTqzv5jgcoZMEKIkJH2-cysvFgQWvLN5kuhgJ-apJzWIjHEtSxQm6hgKpa5vagPHWjfHmWtBM1lzvB7Nsdy3PSUS3VoqhBcAOuDCk_zrfXO4RUjh9VI8pBfdyCYWUg_Y0-BPI9Viwupo5M-YRp6dgMOl6wddYFW36HzggUsgOPHieuM9rSE6AqlojuiXLs68Xo3ek-lHIdun78Nol_8PpvCaR3pYOImuwR98iaxazeY51rMbBiW7Kd_uiXOs-jx7Vw"}`
)

func TestJWKVerify(t *testing.T) {
	tokenJson, err := jsonutils.ParseString(token)
	if err != nil {
		t.Fatalf("jsonutils.ParseString fail %s", err)
	}
	resp := oidcutils.SOIDCAccessTokenResponse{}
	err = tokenJson.Unmarshal(&resp)
	if err != nil {
		t.Fatalf("Unmarshal SOIDCAccessTokenResponse")
	}

	keySet, err := jwk.ParseString(key)
	if err != nil {
		t.Fatalf("jwk.ParseString fail %s", err)
	}
	for i := range keySet.Keys {
		key := keySet.Keys[i]
		if key.KeyUsage() == "sig" {
			var oKey rsa.PublicKey
			err := key.Raw(&oKey)
			if err != nil {
				t.Fatalf("Meterialize fail %s", err)
			}
			opt := jwt.WithVerify(jwa.RS256, oKey)
			_, err = jwt.ParseString(resp.AccessToken, opt)
			if err != nil {
				t.Logf("jwt.ParseString with keyid %s fail %s", key.KeyID(), err)
			} else {
				t.Logf("jwt.ParseString with keyid %s success", key.KeyID())
			}
		}
	}
}
