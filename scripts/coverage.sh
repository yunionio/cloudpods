#!/usr/bin/env bash

# Copyright 2019 Yunion
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

covermode=${COVERMODE:-atomic}
coverdir=$(mktemp -d /tmp/coverage.XXXXXXXXXX)
profile="${coverdir}/profile.out"

function get_test_pkgs() {
    go list ./... | egrep -v 'host-image|hostimage'
}

function generate_coverage_data() {
    echo "mode: $covermode" > "$profile"
    go test -coverprofile="$profile" -covermode="$covermode" $(get_test_pkgs)
}

function push_to_codecov() {
    if [ -z "$CODECOV_TOKEN" ]; then
        echo "You must set CODECOV_TOKEN"
        exit 1
    fi
    echo "Push $profile to codecov.io"
    curl -s https://codecov.io/bash | bash -s -- -c -F aFlag -f "$profile"
}

generate_coverage_data
go tool cover -func "$profile"

case "${1-}" in
    --html)
        go tool cover -html "$profile"
        ;;
    --codecov)
        push_to_codecov
        ;;
esac
