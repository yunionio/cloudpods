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
