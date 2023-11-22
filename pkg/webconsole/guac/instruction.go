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
	"fmt"
	"strconv"

	"yunion.io/x/pkg/errors"
)

type Instruction struct {
	Opcode string
	Args   []string
}

func NewInstruction(op string, args ...string) *Instruction {
	return &Instruction{
		Opcode: op,
		Args:   args,
	}
}

func (i *Instruction) String() string {
	ret := fmt.Sprintf("%d.%s", len(i.Opcode), i.Opcode)
	for _, value := range i.Args {
		ret += fmt.Sprintf(",%d.%s", len(value), value)
	}
	ret += ";"
	return ret
}

func parse(buf []byte) ([]*Instruction, []byte, error) {
	ret := []*Instruction{}
	if len(buf) == 0 {
		return ret, []byte{}, nil
	}
	start, opcode, args := 0, "", []string{}
	begin := 0
	for i := 0; i < len(buf); i++ {
		switch buf[i] {
		case ',':
			start = i + 1
		case '.':
			length, err := strconv.Atoi(string(buf[start:i]))
			if err != nil {
				return nil, []byte{}, errors.Wrapf(err, "Atoi(%s)", string(buf[start:i]))
			}
			if i+length+1 > len(buf) {
				return ret, buf[begin:], nil
			}
			str := string(buf[i+1 : i+length+1])
			if len(opcode) == 0 {
				opcode = str
			} else {
				args = append(args, str)
			}
			i += length
		case ';':
			start = i + 1
			begin = i + 1
			instruction := NewInstruction(opcode, args...)
			ret = append(ret, instruction)
			opcode = ""
			args = []string{}
		}
	}
	if buf[len(buf)-1] != ';' {
		return ret, buf[begin:], nil
	}
	return ret, []byte{}, nil
}
