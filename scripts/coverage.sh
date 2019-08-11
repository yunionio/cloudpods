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

set -o errexit
set -o pipefail

function push_to_codecov() {
    if [ -z "$CODECOV_TOKEN" ]; then
        echo "You must set CODECOV_TOKEN"
        exit 1
    fi
    echo "Push $profile to codecov.io"
    curl -s https://codecov.io/bash | bash -s -- -c -F aFlag -f "$profile"
}


covermode=${COVERMODE:-atomic}
coverdir=$(mktemp -d /tmp/coverage.XXXXXXXXXX)
profile="${coverdir}/profile.out"
if [ -z "$pkgs" ]; then
	pkgs="$(go list -mod vendor ./... | grep -vE 'host-image|hostimage')"
fi

echo "mode: $covermode" >"$profile"
echo "$pkgs" | xargs -n 8 --no-run-if-empty echo \
	| while read batch; do \
		go test -v \
			-coverprofile="$profile.tmp" \
			-covermode="$covermode" \
			-mod vendor \
			-ldflags '-w' \
			$batch; \
		tail -n +2 "$profile.tmp" >>"$profile"; \
		rm -f "$profile.tmp"; \
	done

case "${1-}" in
    --html)
        go tool cover -html "$profile"
        ;;
    --codecov)
        if ! push_to_codecov; then
		echo "ignored: push to codecov failed" >&2
	fi
	;;
    *)
	go tool cover -func "$profile"
        ;;
esac
