#!/usr/bin/env bash

o=_output/_vet/
ALL="$o/vet.txt"


typi() {
	local typ="$1"
	local fna="$o/vet_$(echo "$typ" | sed -r -e "s/[^a-zA-Z0-9]+/_/g").txt"

	grep -F    "$typ" "$UNKNOWN" >"$fna"
	grep -F -v "$typ" "$UNKNOWN" >""$INTERMEDIATE""
	cp "$INTERMEDIATE" "$UNKNOWN"

	rm -f "$INTERMEDIATE"
	if [ ! -s "$fna" ]; then
		rm -f "$fna"
	fi
}

typ() {
	local UNKNOWN=$o/vet0.txt
	local INTERMEDIATE=$o/vet1.txt

	cp "$ALL" "$UNKNOWN"
	typi "unreachable code"
	typi "composite literal uses unkeyed fields"
	typi "repeats json tag"
	typi "bad syntax for struct tag key"
	typi "bad syntax for struct tag value"
	typi "pairs not separated by spaces"
	typi "bad syntax for struct tag pair"
}

gen() {
	rm -rf "$o"
	mkdir -p "$o"
	go vet ./... 2>"$ALL"

	typ
	ls -l $o/
}

chki() {
	local typ="$1"

	if grep -F "$typ" "$ALL"; then
		exit 1
	fi
}

chk() {
	chki "unreachable code"
	chki "composite literal uses unkeyed fields"
	: chki "repeats json tag"
	: chki "bad syntax for struct tag key"
	: chki "bad syntax for struct tag value"
	chki "pairs not separated by spaces"
	chki "bad syntax for struct tag pair"
}

"$@"
