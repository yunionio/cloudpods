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

import (
	"testing"

	"yunion.io/x/pkg/util/netutils"
)

type actorJoinedByCase struct {
	sep         string
	trimSpace   bool
	ignoreEmpty bool
	C
}

func (ac *actorJoinedByCase) Validator() *ValidatorByActor {
	actor := NewActorJoinedBy(ac.sep,
		NewActorIPv4Prefix(),
	).TrimSpace(ac.trimSpace).IgnoreEmpty(ac.ignoreEmpty)
	return NewValidatorByActor("s", actor)
}

func TestJoinedByActor(t *testing.T) {
	var valsWant []interface{}
	for _, n := range []string{"10.0.0.0/8", "192.168.0.0/16"} {
		p, _ := netutils.NewIPV4Prefix(n)
		valsWant = append(valsWant, &p)
	}
	cases := []*actorJoinedByCase{
		{
			C: C{
				Name:      "missing non-optional",
				In:        `{}`,
				Out:       `{}`,
				Optional:  false,
				Err:       ERR_MISSING_KEY,
				ValueWant: nil,
			},
		},
		{
			C: C{
				Name:      "missing optional",
				In:        `{}`,
				Out:       `{}`,
				Optional:  true,
				ValueWant: nil,
			},
		},
		{
			sep: ",",
			C: C{
				Name:      "missing with default",
				In:        `{}`,
				Out:       `{s: "10.0.0.0/8,192.168.0.0/16"}`,
				Default:   "10.0.0.0/8,192.168.0.0/16",
				ValueWant: valsWant,
			},
		},
		{
			sep: ",",
			C: C{
				Name:      "good in",
				In:        `{"s": "10.0.0.0/8,192.168.0.0/16"}`,
				Out:       `{"s": "10.0.0.0/8,192.168.0.0/16"}`,
				ValueWant: valsWant,
			},
		},
		{
			sep:         ",",
			ignoreEmpty: true,
			C: C{
				Name:      "good in (nothing)",
				In:        `{"s": ""}`,
				Out:       `{"s": ""}`,
				ValueWant: []interface{}{},
			},
		},
		{
			sep: ",",
			C: C{
				Name: "good in (0.0.0.0/32)",
				In:   `{"s": ""}`,
				Out:  `{"s": ""}`,
				ValueWant: []interface{}{func() *netutils.IPV4Prefix {
					p, _ := netutils.NewIPV4Prefix("0.0.0.0/32")
					return &p
				}()},
			},
		},
		{
			sep:         ",",
			trimSpace:   true,
			ignoreEmpty: true,
			C: C{
				Name:      "good in (ignore empty, trim space)",
				In:        `{"s": ",,, 10.0.0.0/8 , 192.168.0.0/16, ,,"}`,
				Out:       `{"s": "10.0.0.0/8,192.168.0.0/16"}`,
				ValueWant: valsWant,
			},
		},
		{
			sep:         ",",
			ignoreEmpty: false,
			C: C{
				Name:      "bad in (empty)",
				In:        `{"s": ",,, 10.0.0.0/8 , 192.168.0.0/16, ,,"}`,
				Out:       `{"s": ",,, 10.0.0.0/8 , 192.168.0.0/16, ,,"}`,
				Err:       ERR_INVALID_VALUE,
				ValueWant: nil,
			},
		},
		{
			sep:       ",",
			trimSpace: false,
			C: C{
				Name:      "bad in (space)",
				In:        `{"s": ",,, 10.0.0.0/8 , 192.168.0.0/16, ,,"}`,
				Out:       `{"s": ",,, 10.0.0.0/8 , 192.168.0.0/16, ,,"}`,
				Err:       ERR_INVALID_VALUE,
				ValueWant: nil,
			},
		},
		{
			C: C{
				Name:      "bad in (bad value)",
				In:        `{"s": "10.0.0.259/32"}`,
				Out:       `{"s": "10.0.0.259/32"}`,
				Err:       ERR_INVALID_VALUE,
				ValueWant: nil,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.C.Name, func(t *testing.T) {
			v := c.Validator()
			if c.Default != nil {
				s := c.Default.(string)
				v.Default(s)
			}
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, &c.C)
		})
	}
}
