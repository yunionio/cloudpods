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

package qemutils

import (
	"strings"

	"yunion.io/x/pkg/errors"
)

type Cmdline struct {
	options []Option
}

func NewCmdline(content string) (*Cmdline, error) {
	cl := &Cmdline{
		options: make([]Option, 0),
	}
	parts := strings.Split(content, " -")
	for i := range parts {
		part := parts[i]
		segs := strings.Split(part, " ")
		if len(segs) == 0 {
			return nil, errors.Errorf("Invalid part %q", part)
		} else if len(segs) == 1 {
			cl.options = append(cl.options, newOption(segs[0], ""))
		} else {
			cl.options = append(cl.options, newOption(segs[0], strings.Join(segs[1:], " ")))
		}
	}
	return cl, nil
}

type Option struct {
	Key   string
	Value string
}

func newOption(key string, val string) Option {
	val = strings.TrimRight(val, " ")
	return Option{
		Key:   key,
		Value: val,
	}
}

func (o Option) ToString() string {
	if o.Value == "" {
		return o.Key
	}
	return o.Key + " " + o.Value
}

type OptionFilter func(Option) bool

func (cl *Cmdline) FilterOption(filter OptionFilter) {
	opts := make([]Option, 0)
	for _, op := range cl.options {
		if filter(op) {
			continue
		}
		opts = append(opts, op)
	}
	cl.options = opts
}

type OptionChanger func(*Option)

func (cl *Cmdline) ChangeOption(changer OptionChanger) {
	for i, _ := range cl.options {
		changer(&cl.options[i])
	}
}

func (cl *Cmdline) AddOption(opts ...Option) *Cmdline {
	cl.options = append(cl.options, opts...)
	return cl
}

func (cl *Cmdline) ToString() string {
	opts := make([]string, 0)
	for i := range cl.options {
		opts = append(opts, cl.options[i].ToString())
	}
	return strings.Join(opts, " -")
}
