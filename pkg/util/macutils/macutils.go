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
