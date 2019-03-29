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

package macutils

import (
	"encoding/xml"
	"fmt"
)

type SPlist struct {
	XMLName xml.Name `xml:"plist"`
	Dict    SDict    `xml:"dict"`
}

type SDict struct {
	Key    []string `xml:"key"`
	String []string `xml:"string"`
}

func ParsePlist(sinfo []byte) map[string]string {
	v := &SPlist{}
	ret := map[string]string{}
	if err := xml.Unmarshal(sinfo, v); err != nil {
		return ret
	}
	lenK := len(v.Dict.Key)
	lenS := len(v.Dict.String)
	if lenK > lenS {
		lenK = lenS
	}
	for i := 0; i < lenK; i++ {
		ret[v.Dict.Key[i]] = v.Dict.String[i]
	}
	return ret
}

func LaunchdRun(label, script string) string {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>Label</key>
        <string>%s</string>
        <key>ProgramArguments</key>
        <array>
            <string>/bin/sh</string>
            <string>%s</string>
        </array>
        <key>RunAtLoad</key>
        <true/>
        <key>KeepAlive</key>
        <dict>
            <key>SuccessfulExit</key>
            <false/>
            <key>OtherJobEnabled</key>
            <string>com.apple.opendirectoryd</string>
        </dict>
    </dict>
</plist>
`
	return fmt.Sprintf(content, label, script)
}
