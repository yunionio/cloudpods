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

package tsdb

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInterval(t *testing.T) {
	Convey("Default interval", t, func() {
		calculator := NewIntervalCalculator(&IntervalOptions{})

		Convey("for 5min", func() {
			tr := NewTimeRange("5m", "now")

			interval := calculator.Calculate(tr, time.Millisecond*1)
			So(interval.Text, ShouldEqual, "200ms")
		})

		Convey("for 15min", func() {
			tr := NewTimeRange("15m", "now")

			interval := calculator.Calculate(tr, time.Millisecond*1)
			So(interval.Text, ShouldEqual, "500ms")
		})

		Convey("for 30min", func() {
			tr := NewTimeRange("30m", "now")

			interval := calculator.Calculate(tr, time.Millisecond*1)
			So(interval.Text, ShouldEqual, "1s")
		})

		Convey("for 1h", func() {
			tr := NewTimeRange("1h", "now")

			interval := calculator.Calculate(tr, time.Millisecond*1)
			So(interval.Text, ShouldEqual, "2s")
		})

		Convey("Round interval", func() {
			So(roundInterval(time.Millisecond*30), ShouldEqual, time.Millisecond*20)
			So(roundInterval(time.Millisecond*45), ShouldEqual, time.Millisecond*50)
		})

		Convey("Format value", func() {
			So(FormatDuration(time.Second*61), ShouldEqual, "1m")
			So(FormatDuration(time.Millisecond*30), ShouldEqual, "30ms")
			So(FormatDuration(time.Hour*23), ShouldEqual, "23h")
			So(FormatDuration(time.Hour*24), ShouldEqual, "1d")
			So(FormatDuration(time.Hour*24*367), ShouldEqual, "1y")
		})
	})
}
