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

package validators

// TODO
//
//  - strict type, no implicit conversion
//  - test model validator
//  - invalid default for string choice, range

import (
	"context"
	"net"
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/choices"
)

func TestURLPathRegexp(t *testing.T) {
	cases := []struct {
		in    string
		match bool
	}{
		{in: "", match: true},
		{in: "/", match: true},
		{in: "/p", match: true},
		{in: "/p/", match: true},
		{in: "p", match: false},
		{in: "p/", match: false},
		{in: "/p?", match: false},
		{in: "/p#", match: false},
	}
	for _, c := range cases {
		got := regexpURLPath.MatchString(c.in)
		if got != c.match {
			t.Errorf("%q match, want %v, got %v", c.in, c.match, got)
		}
	}
}

func TestRegHostPort(t *testing.T) {
	inputs := []string{
		"www.yunion.cn",
		"www.yunion.cn:9000",
	}
	for _, in := range inputs {
		if !regHostPort.Match([]byte(in)) {
			t.Errorf("should match: %q", in)
		}
	}
}

type C struct {
	Name      string
	In        string
	Out       string
	Optional  bool
	Default   interface{}
	ValueWant interface{}
	Err       ErrType
}

func testS(t *testing.T, v IValidator, c *C) {
	returnHttpError = false

	j, _ := jsonutils.ParseString(c.In)
	jd := j.(*jsonutils.JSONDict)
	err := v.Validate(context.Background(), jd)
	if err != nil {
		verr, ok := err.(*ValidateError)
		if ok {
			if verr.ErrType != c.Err {
				t.Errorf("error want %q, got %q",
					c.Err, verr.ErrType)
			}
		} else {
			t.Errorf("want error type ValidateError")
		}
	} else {
		if c.Err != ERR_SUCCESS {
			t.Errorf("expect error: %s", c.Err)
		}
	}
	jWant, _ := jsonutils.ParseString(c.Out)
	if !reflect.DeepEqual(j, jWant) {
		t.Errorf("json out want %s, got %s",
			jWant.String(), j.String())
	}
	value := v.getValue()
	if !reflect.DeepEqual(value, c.ValueWant) {
		t.Errorf("value want %#v, got %#v", c.ValueWant, value)
	}
}

func TestStringChoicesValidator(t *testing.T) {
	choices := choices.NewChoices("choice0", "choice1", "100")
	cases := []*C{
		{
			Name:      "missing non-optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  false,
			Err:       ERR_MISSING_KEY,
			ValueWant: "",
		},
		{
			Name:      "missing optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  true,
			ValueWant: "",
		},
		{
			Name:      "missing with default",
			In:        `{}`,
			Out:       `{s: "choice0"}`,
			Default:   "choice0",
			ValueWant: "choice0",
		},
		{
			Name:      "stringified",
			In:        `{"s": 100}`,
			Out:       `{s: "100"}`,
			ValueWant: "100",
		},
		{
			Name:      "stringified invalid choice",
			In:        `{"s": 101}`,
			Out:       `{"s": 101}`,
			Err:       ERR_INVALID_CHOICE,
			ValueWant: "",
		},
		{
			Name:      "good choice",
			In:        `{"s": "choice1"}`,
			Out:       `{"s": "choice1"}`,
			ValueWant: "choice1",
		},
		{
			Name:      "bad choice",
			In:        `{"s": "badchoice"}`,
			Out:       `{"s": "badchoice"}`,
			Err:       ERR_INVALID_CHOICE,
			ValueWant: "",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewStringChoicesValidator("s", choices)
			if c.Default != nil {
				s := c.Default.(string)
				v.Default(s)
			}
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, c)
		})
	}
}

func TestIntChoicesValidator(t *testing.T) {
	choices := []int64{-1, 0, 100}
	cases := []*C{
		{
			Name:      "missing non-optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  false,
			Err:       ERR_MISSING_KEY,
			ValueWant: int64(0),
		},
		{
			Name:      "missing optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  true,
			ValueWant: int64(0),
		},
		{
			Name:      "missing with default",
			In:        `{}`,
			Out:       `{s: -1}`,
			Default:   int64(-1),
			ValueWant: int64(-1),
		},
		{
			Name:      "stringified",
			In:        `{"s": "100"}`,
			Out:       `{s: 100}`,
			ValueWant: int64(100),
		},
		{
			Name:      "stringified invalid choice",
			In:        `{"s": "101"}`,
			Out:       `{"s": "101"}`,
			Err:       ERR_INVALID_CHOICE,
			ValueWant: int64(0),
		},
		{
			Name:      "good choice",
			In:        `{"s": 0}`,
			Out:       `{"s": 0}`,
			ValueWant: int64(0),
		},
		{
			Name:      "bad choice",
			In:        `{"s": 101}`,
			Out:       `{"s": 101}`,
			Err:       ERR_INVALID_CHOICE,
			ValueWant: int64(0),
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewIntChoicesValidator("s", choices)
			if c.Default != nil {
				s := c.Default.(int64)
				v.Default(s)
			}
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, c)
		})
	}
}

func TestStringMultiChoicesValidator(t *testing.T) {
	type MultiChoicesC struct {
		*C
		KeepDup bool
	}
	choices := choices.NewChoices("choice0", "choice1")
	cases := []*MultiChoicesC{
		{
			C: &C{
				Name:      "missing non-optional",
				In:        `{}`,
				Out:       `{}`,
				Optional:  false,
				Err:       ERR_MISSING_KEY,
				ValueWant: "",
			},
		},
		{
			C: &C{
				Name:      "missing optional",
				In:        `{}`,
				Out:       `{}`,
				Optional:  true,
				ValueWant: "",
			},
		},
		{
			C: &C{
				Name:      "missing with default",
				In:        `{}`,
				Out:       `{s: "choice0,choice1"}`,
				Default:   "choice0,choice1",
				ValueWant: "choice0,choice1",
			},
		},
		{
			C: &C{
				Name:      "good choices",
				In:        `{"s": "choice0,choice1"}`,
				Out:       `{"s": "choice0,choice1"}`,
				ValueWant: "choice0,choice1",
			},
		},
		{
			C: &C{
				Name:      "keep dup",
				In:        `{"s": "choice0,choice0,choice1,choice0"}`,
				Out:       `{"s": "choice0,choice0,choice1,choice0"}`,
				ValueWant: "choice0,choice0,choice1,choice0",
			},
			KeepDup: true,
		},
		{
			C: &C{
				Name:      "strip dup",
				In:        `{"s": "choice0,choice0,choice1,choice0"}`,
				Out:       `{"s": "choice0,choice1"}`,
				ValueWant: "choice0,choice1",
			},
		},
		{
			C: &C{
				Name:      "invalid choice",
				In:        `{"s": "choice0,choicex"}`,
				Out:       `{"s": "choice0,choicex"}`,
				Err:       ERR_INVALID_CHOICE,
				ValueWant: "",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewStringMultiChoicesValidator("s", choices).Sep(",").KeepDup(c.KeepDup)
			if c.Default != nil {
				s := c.Default.(string)
				v.Default(s)
			}
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, c.C)
		})
	}
}

func TestBoolValidator(t *testing.T) {
	cases := []*C{
		{
			Name:      "missing non-optional",
			In:        `{}`,
			Out:       `{}`,
			Err:       ERR_MISSING_KEY,
			ValueWant: false,
		},
		{
			Name:      "missing optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  true,
			ValueWant: false,
		},
		{
			Name:      "missing with default",
			In:        `{}`,
			Out:       `{s: true}`,
			Default:   true,
			ValueWant: true,
		},
		{
			Name:      "true",
			In:        `{s: true}`,
			Out:       `{s: true}`,
			ValueWant: true,
		},
		{
			Name:      "false",
			In:        `{s: false}`,
			Out:       `{s: false}`,
			ValueWant: false,
		},
		{
			Name:      `parsed "true"`,
			In:        `{s: "true"}`,
			Out:       `{s: true}`,
			ValueWant: true,
		},
		{
			Name:      `parsed "on"`,
			In:        `{s: "on"}`,
			Out:       `{s: true}`,
			ValueWant: true,
		},
		{
			Name:      `parsed "yes"`,
			In:        `{s: "yes"}`,
			Out:       `{s: true}`,
			ValueWant: true,
		},
		{
			Name:      `parsed "1"`,
			In:        `{s: "1"}`,
			Out:       `{s: true}`,
			ValueWant: true,
		},
		{
			Name:      "parsed invalid",
			In:        `{s: "abc"}`,
			Out:       `{s: "abc"}`,
			Err:       ERR_INVALID_TYPE,
			ValueWant: false,
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewBoolValidator("s")
			if c.Default != nil {
				i := c.Default.(bool)
				v.Default(i)
			}
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, c)
		})
	}
}

func TestRangeValidator(t *testing.T) {
	cases := []*C{
		{
			Name:      "missing non-optional",
			In:        `{}`,
			Out:       `{}`,
			Err:       ERR_MISSING_KEY,
			ValueWant: int64(0),
		},
		{
			Name:      "missing optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  true,
			ValueWant: int64(0),
		},
		{
			Name:      "missing with default",
			In:        `{}`,
			Out:       `{s: 1}`,
			Default:   int64(1),
			ValueWant: int64(1),
		},
		{
			Name:      "parsed",
			In:        `{s: "100"}`,
			Out:       `{s: 100}`,
			ValueWant: int64(100),
		},
		{
			Name:      "parsed invalid int",
			In:        `{s: "abc"}`,
			Out:       `{s: "abc"}`,
			Err:       ERR_INVALID_TYPE,
			ValueWant: int64(0),
		},
		{
			Name:      "parsed not in range",
			In:        `{s: "65536"}`,
			Out:       `{s: "65536"}`,
			Err:       ERR_NOT_IN_RANGE,
			ValueWant: int64(0),
		},
		{
			Name:      "in range",
			In:        `{s: 100}`,
			Out:       `{s: 100}`,
			ValueWant: int64(100),
		},
		{
			Name:      "not in range",
			In:        `{s: 0}`,
			Out:       `{s: 0}`,
			Err:       ERR_NOT_IN_RANGE,
			ValueWant: int64(0),
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewPortValidator("s")
			if c.Default != nil {
				i := c.Default.(int64)
				v.Default(i)
			}
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, c)
		})
	}
}

func TestRegexValidator(t *testing.T) {
	type RegexC struct {
		*C
		AllowEmpty bool
	}
	cases := []*RegexC{
		{
			C: &C{
				Name:      "missing non-optional",
				In:        `{}`,
				Out:       `{}`,
				Err:       ERR_MISSING_KEY,
				ValueWant: "",
			},
		},
		{
			C: &C{
				Name:      "missing optional",
				In:        `{}`,
				Out:       `{}`,
				Optional:  true,
				ValueWant: "",
			},
		},
		{
			C: &C{
				Name:      "missing with default",
				In:        `{}`,
				Out:       `{s: "example.com"}`,
				Default:   "example.com",
				ValueWant: "example.com",
			},
		},
		{
			C: &C{
				Name:      "valid",
				In:        `{s: "a.example.com"}`,
				Out:       `{s: "a.example.com"}`,
				ValueWant: "a.example.com",
			},
		},
		{
			C: &C{
				Name:      "valid (allow empty)",
				In:        `{s: ""}`,
				Out:       `{s: ""}`,
				ValueWant: "",
			},
			AllowEmpty: true,
		},
		{
			C: &C{
				Name:      "invalid",
				In:        `{s: "/.example.com"}`,
				Out:       `{s: "/.example.com"}`,
				ValueWant: "",
				Err:       ERR_INVALID_VALUE,
			},
		},
		{
			C: &C{
				Name:      "invalid (disallow empty)",
				In:        `{s: ""}`,
				Out:       `{s: ""}`,
				ValueWant: "",
				Err:       ERR_INVALID_VALUE,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewDomainNameValidator("s")
			if c.Default != nil {
				i := c.Default.(string)
				v.Default(i)
			}
			if c.Optional {
				v.Optional(true)
			}
			if c.AllowEmpty {
				v.AllowEmpty(true)
			}
			testS(t, v, c.C)
		})
	}
}

func TestHostPortValidator(t *testing.T) {
	type HostPortC struct {
		*C
		AllowEmpty   bool
		OptionalPort bool
	}
	cases := []*HostPortC{
		{
			C: &C{
				Name:      "missing non-optional",
				In:        `{}`,
				Out:       `{}`,
				Err:       ERR_MISSING_KEY,
				ValueWant: "",
			},
		},
		{
			C: &C{
				Name:      "missing optional",
				In:        `{}`,
				Out:       `{}`,
				Optional:  true,
				ValueWant: "",
			},
		},
		{
			C: &C{
				Name:      "missing with default",
				In:        `{}`,
				Out:       `{s: "example.com"}`,
				Default:   "example.com",
				ValueWant: "example.com",
			},
			OptionalPort: true,
		},
		{
			C: &C{
				Name:      "missing with default (has port)",
				In:        `{}`,
				Out:       `{s: "example.com:9000"}`,
				Default:   "example.com:9000",
				ValueWant: "example.com:9000",
			},
		},
		{
			C: &C{
				Name:      "valid",
				In:        `{s: "a.example.com"}`,
				Out:       `{s: "a.example.com"}`,
				ValueWant: "a.example.com",
			},
			OptionalPort: true,
		},
		{
			C: &C{
				Name:      "valid (has port)",
				In:        `{s: "a.example.com:9000"}`,
				Out:       `{s: "a.example.com:9000"}`,
				ValueWant: "a.example.com:9000",
			},
		},
		{
			C: &C{
				Name:      "valid (allow empty)",
				In:        `{s: ""}`,
				Out:       `{s: ""}`,
				ValueWant: "",
			},
			AllowEmpty: true,
		},
		{
			C: &C{
				Name:      "invalid (domain)",
				In:        `{s: "/.example.com:9000"}`,
				Out:       `{s: "/.example.com:9000"}`,
				ValueWant: "",
				Err:       ERR_INVALID_VALUE,
			},
		},
		{
			C: &C{
				Name:      "invalid (port)",
				In:        `{s: "/.example.com:65536"}`,
				Out:       `{s: "/.example.com:65536"}`,
				ValueWant: "",
				Err:       ERR_INVALID_VALUE,
			},
		},
		{
			C: &C{
				Name:      "invalid (no port)",
				In:        `{s: "a.example.com"}`,
				Out:       `{s: "a.example.com"}`,
				ValueWant: "",
				Err:       ERR_INVALID_VALUE,
			},
		},
		{
			C: &C{
				Name:      "invalid (disallow empty)",
				In:        `{s: ""}`,
				Out:       `{s: ""}`,
				ValueWant: "",
				Err:       ERR_INVALID_VALUE,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewHostPortValidator("s")
			if c.Default != nil {
				i := c.Default.(string)
				v.Default(i)
			}
			if c.Optional {
				v.Optional(true)
			}
			if c.OptionalPort {
				v.OptionalPort(true)
			}
			if c.AllowEmpty {
				v.AllowEmpty(true)
			}
			testS(t, v, c.C)
		})
	}
}

func TestIPv4Validator(t *testing.T) {
	var nilIP net.IP
	localIP := net.IPv4(127, 0, 0, 1).To4()
	cases := []*C{
		{
			Name:      "missing non-optional",
			In:        `{}`,
			Out:       `{}`,
			Err:       ERR_MISSING_KEY,
			ValueWant: nilIP,
		},
		{
			Name:      "missing optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  true,
			ValueWant: nilIP,
		},
		{
			Name:      "missing with default",
			In:        `{}`,
			Out:       `{s: "127.0.0.1"}`,
			Default:   localIP,
			ValueWant: localIP,
		},
		{
			Name:      "valid",
			In:        `{s: "127.0.0.1"}`,
			Out:       `{s: "127.0.0.1"}`,
			ValueWant: localIP,
		},
		{
			Name:      "valid (v4 in v6)",
			In:        `{s: "::ffff:127.0.0.1"}`,
			Out:       `{s: "::ffff:127.0.0.1"}`,
			ValueWant: localIP,
		},
		{
			Name:      "invalid",
			In:        `{s: "127.0.0"}`,
			Out:       `{s: "127.0.0"}`,
			Err:       ERR_INVALID_VALUE,
			ValueWant: nilIP,
		},
		{
			Name:      "invalid (empty string)",
			In:        `{s: ""}`,
			Out:       `{s: ""}`,
			Err:       ERR_INVALID_VALUE,
			ValueWant: nilIP,
		},
		{
			Name:      "invalid (wrong type)",
			In:        `{s: 100}`,
			Out:       `{s: 100}`,
			Err:       ERR_INVALID_VALUE,
			ValueWant: nilIP,
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewIPv4AddrValidator("s")
			if c.Default != nil {
				i := c.Default.(net.IP)
				v.Default(i)
			}
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, c)
		})
	}
}

type TestStruct struct {
	Name string
	F0   string
	F1   int
	F2   bool
}

type TestVStruct TestStruct

func (v *TestVStruct) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	switch v.Name {
	case "bad":
		return newInvalidValueError("Name", v.Name)
	case "setDefault":
		v.Name = "defaultVal"
	}
	return nil
}

func TestStructValidator(t *testing.T) {
	type StructC struct {
		*C
		Value interface{}
	}

	cases := []*StructC{
		{
			C: &C{
				Name:      "missing non-optional",
				In:        `{}`,
				Out:       `{}`,
				Err:       ERR_MISSING_KEY,
				ValueWant: &TestStruct{},
			},
			Value: &TestStruct{},
		},
		{
			C: &C{
				Name:      "missing optional",
				In:        `{}`,
				Out:       `{}`,
				Optional:  true,
				ValueWant: &TestStruct{},
			},
			Value: &TestStruct{},
		},
		{
			C: &C{
				Name: "valid",
				In:   `{s: {"F0": "holy", "F1": 100, "F2": true}}`,
				Out:  `{s: {"f0": "holy", "f1": 100, "f2": true}}`,
				ValueWant: &TestStruct{
					F0: "holy",
					F1: 100,
					F2: true,
				},
			},
			Value: &TestStruct{},
		},
		{
			C: &C{
				Name: "valid (missing fields)",
				In:   `{s: {"F0": "holy", "F1": 100}}`,
				Out:  `{s: {"f0": "holy", "f1": 100, "f2": false}}`,
				ValueWant: &TestStruct{
					F0: "holy",
					F1: 100,
					F2: false,
				},
			},
			Value: &TestStruct{},
		},
		{
			C: &C{
				Name: "valid (more fields)",
				In:   `{s: {"F0": "holy", "F1": 100, "F2": true, "Foo": "bar"}}`,
				Out:  `{s: {"f0": "holy", "f1": 100, "f2": true}}`,
				ValueWant: &TestStruct{
					F0: "holy",
					F1: 100,
					F2: true,
				},
			},
			Value: &TestStruct{},
		},
		{
			C: &C{
				Name: "valid (struct says valid)",
				In:   `{s: {"F0": "holy", "F1": 100, "F2": true}}`,
				Out:  `{s: {"f0": "holy", "f1": 100, "f2": true}}`,
				ValueWant: &TestVStruct{
					F0: "holy",
					F1: 100,
					F2: true,
				},
			},
			Value: &TestVStruct{},
		},
		{
			C: &C{
				Name: "valid (with default initialized)",
				In:   `{s: {Name: "setDefault", "F0": "holy", "F1": 100, "F2": false}}`,
				Out:  `{s: {name: "defaultVal", "f0": "holy", "f1": 100, "f2": false}}`,
				ValueWant: &TestVStruct{
					Name: "defaultVal",
					F0:   "holy",
					F1:   100,
					F2:   false,
				},
			},
			Value: &TestVStruct{},
		},
		{
			C: &C{
				Name: "invalid (struct says invalid)",
				In:   `{s: {Name: "bad", "F0": "holy", "F1": 100, "F2": false}}`,
				Out:  `{s: {Name: "bad", "F0": "holy", "F1": 100, "F2": false}}`,
				ValueWant: &TestVStruct{
					Name: "bad",
					F0:   "holy",
					F1:   100,
					F2:   false,
				},
				Err: ERR_INVALID_VALUE,
			},
			Value: &TestVStruct{},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewStructValidator("s", c.Value)
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, c.C)
		})
	}
}
