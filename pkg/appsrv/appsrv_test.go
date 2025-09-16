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

package appsrv

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestSplitPath(t *testing.T) {
	cases := []struct {
		in  string
		out int
	}{
		{in: "/v2.0/tokens/123", out: 3},
		{in: "/v2.0//tokens//123", out: 3},
		{in: "/", out: 0},
		{in: "/v2.0//123//", out: 2},
	}
	for _, p := range cases {
		ret := SplitPath(p.in)
		if len(ret) != p.out {
			t.Error("Split error for ", p.in, " out ", ret)
		}
	}
}

type ApplicationTestSuit struct {
	suite.Suite
	app  *Application
	done chan<- struct{}
}

func (suite *ApplicationTestSuit) SetupTest() {
	suite.app = NewApplication("test", 4, false)
	suite.app.AddHandler("GET", "/delay", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 1)
		Send(w, "delay pong")
	})
	suite.app.AddHandler("GET", "/panic", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("the handler is panic")
	})
	suite.app.AddHandler("GET", "/delaypanic", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 1)
		panic("the handler is delay panic")
	})
	//suite.goServe()
}

func (suite *ApplicationTestSuit) goServe() {
	go func() {
		suite.app.ListenAndServe("0.0.0.0:44444")
	}()
}

func (suite *ApplicationTestSuit) TestHandler() {
	app := suite.app
	assert.True(suite.T(), assert.HTTPBodyContains(suite.T(), app.ServeHTTP, "GET", "/delay", nil, "delay pong"))
	assert.True(suite.T(), assert.HTTPBodyContains(suite.T(), app.ServeHTTP, "GET", "/panic", nil, "the handler is panic"))
	assert.True(suite.T(), assert.HTTPBodyContains(suite.T(), app.ServeHTTP, "GET", "/delaypanic", nil, "the handler is delay panic"))
}

func TestApplicationTestSuite(t *testing.T) {
	suite.Run(t, new(ApplicationTestSuit))
}
