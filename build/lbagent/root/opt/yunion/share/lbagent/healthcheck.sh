#!/usr/bin/env bash

set -e

o_wait=5
o_req="are you ok?"
o_exp="ok"
o_host=
o_port=

__errmsg() {
	echo "healthcheck: $*" >&2
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		wait=*|\
		req=*|\
		exp=*|\
		host=*|\
		port=*)
			eval "o_$1"
			;;
	esac
	shift
done

if [ -z "$o_host" -o -z "$o_port" ]; then
	__errmsg "--host or --port missing"
	exit 126 # script error are not server error
fi

got="$(
	(stdbuf -o0 printf "%s" "$o_req"; sleep "$o_wait") | \
	nc \
		--verbose \
		--udp \
		--nodns \
		--wait "$o_wait" \
	"$o_host" "$o_port"
)"

if [ "$got" != "$o_exp" ]; then
	__errmsg "result unmatch, got $got, want $o_exp"
	exit 1
fi
