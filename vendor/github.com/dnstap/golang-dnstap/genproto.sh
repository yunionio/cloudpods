#!/bin/sh

go_package() {
	local file pkg line script
	file=$1; shift
	pkg=$1; shift

	line="option go_package = \"$pkg\";"
	grep "^$line\$" $file > /dev/null && return

	script="/^package dnstap/|a|$line|.|w|q|"
	if grep "^option go_package" $file > /dev/null; then
		script="/^option go_package/d|1|${script}"
	fi

	echo "$script" | tr '|' '\n' | ed $file || exit
}

dir=$(dirname $0)
[ -n "$dir" ] && cd $dir

cd dnstap.pb

go_package dnstap.proto "github.com/dnstap/golang-dnstap;dnstap"
protoc --go_out=../../../.. dnstap.proto
