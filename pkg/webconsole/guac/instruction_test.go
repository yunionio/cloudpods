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

package guac

import (
	"testing"

	"yunion.io/x/log"
)

// 4.args,13.VERSION_1_5_0,8.hostname,4.port,6.domain,8.username,8.password,5.width,6.height,3.dpi,15.initial-program,11.color-depth,13.disable-audio,15.enable-printing,12.printer-name,12.enable-drive,10.drive-name,10.drive-path,17.create-drive-path,16.disable-download,14.disable-upload,7.console,13.console-audio,13.server-layout,8.security,11.ignore-cert,12.disable-auth,10.remote-app,14.remote-app-dir,15.remote-app-args,15.static-channels,11.client-name,16.enable-wallpaper,14.enable-theming,21.enable-font-smoothing,23.enable-full-window-drag,26.enable-desktop-composition,22.enable-menu-animations,22.disable-bitmap-caching,25.disable-offscreen-caching,21.disable-glyph-caching,16.preconnection-id,18.preconnection-blob,8.timezone,11.enable-sftp,13.sftp-hostname,13.sftp-host-key,9.sftp-port,13.sftp-username,13.sftp-password,16.sftp-private-key,15.sftp-passphrase,14.sftp-directory,19.sftp-root-directory,26.sftp-server-alive-interval,21.sftp-disable-download,19.sftp-disable-upload,14.recording-path,14.recording-name,24.recording-exclude-output,23.recording-exclude-mouse,23.recording-exclude-touch,22.recording-include-keys,21.create-recording-path,13.resize-method,18.enable-audio-input,12.enable-touch,9.read-only,16.gateway-hostname,12.gateway-port,14.gateway-domain,16.gateway-username,16.gateway-password,17.load-balance-info,12.disable-copy,13.disable-paste,15.wol-send-packet,12.wol-mac-addr,18.wol-broadcast-addr,12.wol-udp-port,13.wol-wait-time,14.force-lossless,19.normalize-clipboard;
func TestInstruction(t *testing.T) {
	cases := []struct {
		instructionStr string
		left           string
		want           []*Instruction
	}{
		{"5.ready,37.$59a9eff3-ef1a-4758-8be2-0f1d06b4c8be;", "", []*Instruction{NewInstruction("ready", "$59a9eff3-ef1a-4758-8be2-0f1d06b4c8be")}},
		{"5.audio,1.1,31.audio/L16;rate=44100,channels=2;4.size,1.0,4.1648,3.991;4.size,2.-1,2.11,2.16;3.img,1.3,2.12,2.-1,9.image/png,1.0,1.0;4.blob,1.3,232.iVBORw0KGgoAAAANSUhEUgAAAAsAAAAQCAYAAADAvYV+AAAABmJLR0QA/wD/AP+gvaeTAAAAYklEQVQokY2RQQ4AIQgDW+L/v9y9qCEsIJ4QZggoJAnDYwAwFQwASI4EO8FEMH95CRYTnfCDOyGFK6GEM6GFo7AqKI4sSSsCJH1X+roFkKdjueABX/On77lz2uGtr6pj9okfTeJQAYVaxnMAAAAASUVORK5CYII=;3.end,1.3;6.cursor,1.0,1.0,2.-1,1.0,1.0,2.11,2.16;", "", []*Instruction{
			NewInstruction("audio", "1", "audio/L16;rate=44100,channels=2"),
			NewInstruction("size", "0", "1648", "991"),
			NewInstruction("size", "-1", "11", "16"),
			NewInstruction("img", "3", "12", "-1", "image/png", "0", "0"),
			NewInstruction("blob", "3", "iVBORw0KGgoAAAANSUhEUgAAAAsAAAAQCAYAAADAvYV+AAAABmJLR0QA/wD/AP+gvaeTAAAAYklEQVQokY2RQQ4AIQgDW+L/v9y9qCEsIJ4QZggoJAnDYwAwFQwASI4EO8FEMH95CRYTnfCDOyGFK6GEM6GFo7AqKI4sSSsCJH1X+roFkKdjueABX/On77lz2uGtr6pj9okfTeJQAYVaxnMAAAAASUVORK5CYII="),
			NewInstruction("end", "3"),
			NewInstruction("cursor", "0", "0", "-1", "0", "0", "11", "16"),
		}},
		{"4.copy,2.-2,1.0,1.0,2.64,2.64,2.14,1.0,3.640,3.576;4.copy,2.-2,1.0,1.0,2.64,2.64,2.14,1.0,3.704,3.576;4.copy,2.-2,1.0,1.0,2.64,2.64,2.14,1.0,3.7",
			"4.copy,2.-2,1.0,1.0,2.64,2.64,2.14,1.0,3.7",
			[]*Instruction{
				NewInstruction("copy", "-2", "0", "0", "64", "64", "14", "0", "640", "576"),
				NewInstruction("copy", "-2", "0", "0", "64", "64", "14", "0", "704", "576"),
			}},
		{"4.copy,2.-6,1.0,1.0,2.64,2.64,2.14,1.0,3.640,3.576;4.copy,2.-6,1", "4.copy,2.-6,1", []*Instruction{
			NewInstruction("copy", "-6", "0", "0", "64", "64", "14", "0", "640", "576"),
		}},
	}

	for _, c := range cases {
		t.Run("test instruction", func(t *testing.T) {
			instructions, left, err := parse([]byte(c.instructionStr))
			if err != nil {
				t.Fatalf("parse %s error: %v", c.instructionStr, err)
			}
			if c.left != string(left) {
				log.Fatalf("want %s left, got %s", c.left, string(left))
			}
			if len(instructions) != len(c.want) {
				log.Fatalf("want %d instructions, parse %d", len(c.want), len(instructions))
			}
			for i := range instructions {
				if instructions[i].String() != c.want[i].String() {
					log.Fatalf("%s not equals %s", instructions[i].String(), c.want[i].String())
				}
			}
		})
	}
}
